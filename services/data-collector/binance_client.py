import logging
import asyncio
import time
import orjson
import ccxt.pro as ccxtpro
from typing import List
from aiokafka import AIOKafkaProducer

from config import Config

logger = logging.getLogger(__name__)
EXCHANGE_NAME = "BINANCE"


def create_kafka_headers(message_type: str, exchange: str, symbol: str, schema: str, **kwargs):
    # Headers standardisés pour tous les messages Kafka
    headers = [
        ("source", b"collector-ws"),
        ("type", message_type.encode()),
        ("exchange", exchange.encode()),
        ("symbol", symbol.replace("/", "").encode()),
        ("schema", schema.encode()),
    ]

    # Ajoute les headers optionnels (ex: closed, timeframe)
    for key, value in kwargs.items():
        if isinstance(value, bytes):
            headers.append((key, value))
        elif isinstance(value, str):
            headers.append((key, value.encode()))
        elif isinstance(value, bool):
            headers.append((key, b"true" if value else b"false"))

    return headers


class BinanceClient:

    def __init__(self, config: Config):
        self.config = config
        self.exchange = ccxtpro.binance({
            "enableRateLimit": True,
            "options": {"defaultType": config.default_type},
        })

        # Auth si clés API fournies
        if config.api_key and config.api_secret:
            self.exchange.apiKey = config.api_key
            self.exchange.secret = config.api_secret

    async def get_all_symbols(self) -> List[str]:
        # Récup et filtre les symboles selon la config (quote currencies, volume min, max symbols)
        try:
            logger.info("Chargement des marchés Binance...")
            await self.exchange.load_markets()

            logger.info("Récupération des tickers pour filtrage par volume...")
            tickers = await self.exchange.fetch_tickers()

            all_symbols = []
            filtered_symbols = []
            quote_stats = {}

            # Parcourt tous les marchés Binance
            for symbol, market in self.exchange.markets.items():
                # Garde que les marchés actifs et spot
                if not market.get('active', True) or market.get('type') != 'spot':
                    continue

                all_symbols.append(symbol)

                # Stats par devise de cotation
                quote_currency = market.get('quote', 'UNKNOWN')
                quote_stats[quote_currency] = quote_stats.get(quote_currency, 0) + 1

                # Filtre par devises de cotation
                if self.config.quote_currencies and quote_currency not in self.config.quote_currencies:
                    continue

                # Filtre par volume minimum 24h
                if self.config.min_volume:
                    ticker = tickers.get(symbol)
                    if not ticker or not ticker.get('quoteVolume') or ticker['quoteVolume'] < self.config.min_volume:
                        continue

                filtered_symbols.append(symbol)

            # Logs informatifs
            logger.info(f"Total marchés spot actifs: {len(all_symbols)}")
            logger.info(f"Symboles après filtrage: {len(filtered_symbols)}")

            logger.info("Répartition par devise de cotation:")
            for quote, count in sorted(quote_stats.items(), key=lambda x: x[1], reverse=True)[:10]:
                logger.info(f"   {quote}: {count} paires")

            # Tri par volume décroissant
            if self.config.min_volume and tickers:
                filtered_symbols.sort(
                    key=lambda s: tickers.get(s, {}).get('quoteVolume', 0),
                    reverse=True
                )
                logger.info(f"Top 10 par volume: {filtered_symbols[:10]}")

            # Limite au nombre max de symboles
            if self.config.max_symbols and len(filtered_symbols) > self.config.max_symbols:
                filtered_symbols = filtered_symbols[:self.config.max_symbols]
                logger.info(f"Limité à {self.config.max_symbols} symboles")

            return filtered_symbols

        except Exception as e:
            logger.error(f"Erreur récupération symboles: {e}")
            # Fallback si ça plante
            fallback = ['BTC/USDT', 'ETH/USDT', 'BNB/USDT', 'SOL/USDT', 'XRP/USDT']
            logger.info(f"Utilisation fallback: {fallback}")
            return fallback

    async def close(self):
        await self.exchange.close()


async def watch_trades(
    symbol: str,
    client: BinanceClient,
    producer: AIOKafkaProducer,
    topic: str,
    aggregator,
    semaphore: asyncio.Semaphore
):
    # Worker qui écoute les trades d'un symbole via WebSocket et les publie dans Kafka
    last_id = None
    error_count = 0
    max_errors = 10

    # Étale les connexions dans le temps pour pas surcharger
    async with semaphore:
        await asyncio.sleep(0.1)

    while True:
        try:
            # Attend les nouveaux trades via WebSocket
            trades = await client.exchange.watch_trades(symbol)
            if not trades:
                continue

            # Prend le dernier trade uniquement
            trade = trades[-1]
            tid = trade.get("id")

            # Évite les doublons
            if tid and tid == last_id:
                continue
            last_id = tid

            # Extraction des données
            ts = int(trade["timestamp"]) if trade.get("timestamp") else int(time.time() * 1000)
            price = float(trade["price"])
            amount = float(trade["amount"])

            # Message Kafka
            payload = {
                "type": "trade",
                "exchange": EXCHANGE_NAME,
                "symbol": symbol,
                "event_ts": ts,
                "price": price,
                "amount": amount,
                "side": trade.get("side"),
                "trade_id": tid,
            }

            # Clé pour partitionnement (même symbole = même partition)
            key = f"{EXCHANGE_NAME}|{symbol.replace('/', '')}".encode()

            # Headers standardisés
            headers = create_kafka_headers("trade", EXCHANGE_NAME, symbol, "trade_v1")

            # Publie dans Kafka
            await producer.send_and_wait(topic, orjson.dumps(payload), key=key, headers=headers)

            # Alimente l'agrégateur si activé
            if aggregator:
                await aggregator.on_trade(EXCHANGE_NAME, symbol, ts, price, amount)

            error_count = 0

        except Exception as e:
            error_count += 1
            if error_count <= 3:
                logger.warning(f"[trades:{symbol}] Erreur {error_count}/{max_errors}: {e}")

            # Arrête après trop d'erreurs
            if error_count >= max_errors:
                logger.error(f"[trades:{symbol}] Trop d'erreurs, arrêt du worker")
                break

            # Backoff exponentiel (2s, 4s, 8s, ... max 60s)
            await asyncio.sleep(min(error_count * 2, 60))


async def watch_ticker(
    symbol: str,
    client: BinanceClient,
    producer: AIOKafkaProducer,
    topic: str,
    semaphore: asyncio.Semaphore
):
    # Worker qui écoute le ticker d'un symbole (bid, ask, volumes 24h)
    error_count = 0
    max_errors = 10

    # Étale les connexions
    async with semaphore:
        await asyncio.sleep(0.1)

    while True:
        try:
            # Attend les updates du ticker
            ticker = await client.exchange.watch_ticker(symbol)

            # Message Kafka
            payload = {
                "type": "ticker",
                "exchange": EXCHANGE_NAME,
                "symbol": symbol,
                "event_ts": int(ticker["timestamp"]) if ticker.get("timestamp") else None,
                "bid": float(ticker["bid"]) if ticker.get("bid") else None,
                "ask": float(ticker["ask"]) if ticker.get("ask") else None,
                "last": float(ticker["last"]) if ticker.get("last") else None,
                "baseVolume": float(ticker["baseVolume"]) if ticker.get("baseVolume") else None,
                "quoteVolume": float(ticker["quoteVolume"]) if ticker.get("quoteVolume") else None,
            }

            # Clé pour partitionnement
            key = f"{EXCHANGE_NAME}|{symbol.replace('/', '')}".encode()

            # Headers standardisés
            headers = create_kafka_headers("ticker", EXCHANGE_NAME, symbol, "ticker_v1")

            # Publie dans Kafka
            await producer.send_and_wait(topic, orjson.dumps(payload), key=key, headers=headers)
            error_count = 0

        except Exception as e:
            error_count += 1
            if error_count <= 3:
                logger.warning(f"[ticker:{symbol}] Erreur {error_count}/{max_errors}: {e}")

            # Arrête après trop d'erreurs
            if error_count >= max_errors:
                logger.error(f"[ticker:{symbol}] Trop d'erreurs, arrêt du worker")
                break

            # Backoff exponentiel
            await asyncio.sleep(min(error_count * 2, 60))
