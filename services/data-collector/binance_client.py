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


class BinanceClient:
    """Client Binance pour récupérer les symboles et se connecter aux WebSockets"""

    def __init__(self, config: Config):
        self.config = config
        self.exchange = ccxtpro.binance({
            "enableRateLimit": True,
            "options": {"defaultType": config.default_type},
        })

        # Authentification si clés API fournies
        if config.api_key and config.api_secret:
            self.exchange.apiKey = config.api_key
            self.exchange.secret = config.api_secret

    async def get_all_symbols(self) -> List[str]:
        """
        Récupère et filtre les symboles de trading

        Filtres appliqués:
        1. Seulement les marchés actifs et spot
        2. Devises de cotation autorisées (QUOTE_CURRENCIES)
        3. Volume minimum 24h (MIN_VOLUME)
        4. Tri par volume décroissant
        5. Limitation au top N symboles (MAX_SYMBOLS)

        Returns:
            Liste des symboles au format "BASE/QUOTE" (ex: "BTC/USDT")
        """
        try:
            logger.info("Chargement des marchés Binance...")
            await self.exchange.load_markets()

            logger.info("Récupération des tickers pour filtrage par volume...")
            tickers = await self.exchange.fetch_tickers()

            all_symbols = []
            filtered_symbols = []
            quote_stats = {}

            # Parcourir tous les marchés Binance
            for symbol, market in self.exchange.markets.items():
                # Ignorer les marchés inactifs et non-spot
                if not market.get('active', True) or market.get('type') != 'spot':
                    continue

                all_symbols.append(symbol)

                # Statistiques par devise de cotation
                quote_currency = market.get('quote', 'UNKNOWN')
                quote_stats[quote_currency] = quote_stats.get(quote_currency, 0) + 1

                # Filtre 1: Devises de cotation autorisées
                if self.config.quote_currencies and quote_currency not in self.config.quote_currencies:
                    continue

                # Filtre 2: Volume minimum 24h
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

            # Limitation au nombre max de symboles
            if self.config.max_symbols and len(filtered_symbols) > self.config.max_symbols:
                filtered_symbols = filtered_symbols[:self.config.max_symbols]
                logger.info(f"Limité à {self.config.max_symbols} symboles")

            return filtered_symbols

        except Exception as e:
            logger.error(f"Erreur récupération symboles: {e}")
            # Fallback en cas d'erreur
            fallback = ['BTC/USDT', 'ETH/USDT', 'BNB/USDT', 'SOL/USDT', 'XRP/USDT']
            logger.info(f"Utilisation fallback: {fallback}")
            return fallback

    async def close(self):
        """Ferme proprement les connexions WebSocket"""
        await self.exchange.close()


# ====================
# Workers WebSocket
# ====================

async def watch_trades(
    symbol: str,
    client: BinanceClient,
    producer: AIOKafkaProducer,
    topic: str,
    aggregator,
    semaphore: asyncio.Semaphore
):
    """
    Worker qui surveille les trades d'un symbole via WebSocket

    Args:
        symbol: Paire de trading (ex: "BTC/USDT")
        client: Instance du client Binance
        producer: Producer Kafka pour publier les trades
        topic: Topic Kafka de destination
        aggregator: Agrégateur pour créer les barres OHLCV (peut être None)
        semaphore: Sémaphore pour limiter les connexions simultanées
    """
    last_id = None
    error_count = 0
    max_errors = 10

    # Utiliser le sémaphore pour étaler les connexions dans le temps
    # Le sémaphore se libère après le délai, permettant à d'autres de se connecter
    async with semaphore:
        await asyncio.sleep(0.1)  # Délai de 100ms entre les connexions

    while True:
        try:
            # Attendre les nouveaux trades via WebSocket
            trades = await client.exchange.watch_trades(symbol)
            if not trades:
                continue

            # Prendre le dernier trade uniquement
            trade = trades[-1]
            tid = trade.get("id")

            # Éviter les doublons
            if tid and tid == last_id:
                continue
            last_id = tid

            # Extraction des données
            ts = int(trade["timestamp"]) if trade.get("timestamp") else int(time.time() * 1000)
            price = float(trade["price"])
            amount = float(trade["amount"])

            # Préparer le message Kafka
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

            # Clé Kafka pour le partitionnement (un symbole = toujours la même partition)
            key = f"{EXCHANGE_NAME}|{symbol.replace('/', '')}".encode()

            # Headers Kafka pour métadonnées
            headers = [
                ("source", b"collector-ws"),
                ("exchange", EXCHANGE_NAME.encode()),
                ("symbol", symbol.replace("/", "").encode()),
                ("type", b"trade"),
                ("schema", b"trade_v1"),
            ]

            # Publier dans Kafka
            await producer.send_and_wait(topic, orjson.dumps(payload), key=key, headers=headers)

            # Alimenter l'agrégateur si activé
            if aggregator:
                await aggregator.on_trade(EXCHANGE_NAME, symbol, ts, price, amount)

            error_count = 0  # Reset compteur d'erreurs

        except Exception as e:
            error_count += 1
            if error_count <= 3:
                logger.warning(f"[trades:{symbol}] Erreur {error_count}/{max_errors}: {e}")

            # Arrêter après trop d'erreurs
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
    """
    Worker qui surveille le ticker d'un symbole via WebSocket

    Le ticker contient: bid, ask, last price, volumes 24h

    Args:
        symbol: Paire de trading (ex: "BTC/USDT")
        client: Instance du client Binance
        producer: Producer Kafka pour publier les tickers
        topic: Topic Kafka de destination
        semaphore: Sémaphore pour limiter les connexions simultanées
    """
    error_count = 0
    max_errors = 10

    # Utiliser le sémaphore pour étaler les connexions dans le temps
    async with semaphore:
        await asyncio.sleep(0.1)  # Délai de 100ms entre les connexions

    while True:
        try:
            # Attendre les mises à jour du ticker via WebSocket
            ticker = await client.exchange.watch_ticker(symbol)

            # Préparer le message Kafka
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

            # Clé Kafka pour le partitionnement
            key = f"{EXCHANGE_NAME}|{symbol.replace('/', '')}".encode()

            # Headers Kafka pour métadonnées
            headers = [
                ("source", b"collector-ws"),
                ("exchange", EXCHANGE_NAME.encode()),
                ("symbol", symbol.replace("/", "").encode()),
                ("type", b"ticker"),
                ("schema", b"ticker_v1"),
            ]

            # Publier dans Kafka
            await producer.send_and_wait(topic, orjson.dumps(payload), key=key, headers=headers)
            error_count = 0  # Reset compteur d'erreurs

        except Exception as e:
            error_count += 1
            if error_count <= 3:
                logger.warning(f"[ticker:{symbol}] Erreur {error_count}/{max_errors}: {e}")

            # Arrêter après trop d'erreurs
            if error_count >= max_errors:
                logger.error(f"[ticker:{symbol}] Trop d'erreurs, arrêt du worker")
                break

            # Backoff exponentiel
            await asyncio.sleep(min(error_count * 2, 60))
