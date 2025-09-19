import logging
import asyncio
import orjson
import ccxt.pro as ccxtpro
from typing import List, Optional
from aiokafka import AIOKafkaProducer

from config import Config

logger = logging.getLogger(__name__)
EXCHANGE_NAME = "BINANCE"

class BinanceClient:
    """Client Binance avec gestion des symboles et workers"""
    
    def __init__(self, config: Config):
        self.config = config
        self.exchange = ccxtpro.binance({
            "enableRateLimit": True,
            "options": {"defaultType": config.default_type},
        })
        
        if config.api_key and config.api_secret:
            self.exchange.apiKey = config.api_key
            self.exchange.secret = config.api_secret
    
    async def get_all_symbols(self) -> List[str]:
        """Récupère tous les symboles avec filtrage"""
        try:
            logger.info("Chargement des marchés Binance...")
            await self.exchange.load_markets()
            
            logger.info("Récupération des tickers pour filtrage par volume...")
            tickers = await self.exchange.fetch_tickers()
            
            all_symbols = []
            filtered_symbols = []
            quote_stats = {}
            
            for symbol, market in self.exchange.markets.items():
                if not market.get('active', True) or market.get('type') != 'spot':
                    continue
                    
                all_symbols.append(symbol)
                
                quote_currency = market.get('quote', 'UNKNOWN')
                quote_stats[quote_currency] = quote_stats.get(quote_currency, 0) + 1
                
                should_include = True
                
                if self.config.quote_currencies and quote_currency not in self.config.quote_currencies:
                    should_include = False
                    continue
                
                if self.config.min_volume and should_include:
                    ticker = tickers.get(symbol)
                    if not ticker or not ticker.get('quoteVolume') or ticker['quoteVolume'] < self.config.min_volume:
                        should_include = False
                
                if should_include:
                    filtered_symbols.append(symbol)
            
            logger.info(f"Total marchés spot actifs: {len(all_symbols)}")
            logger.info(f"Symboles après filtrage: {len(filtered_symbols)}")
            
            # Stats par devise
            logger.info("Répartition par devise de cotation:")
            for quote, count in sorted(quote_stats.items(), key=lambda x: x[1], reverse=True)[:10]:
                logger.info(f"   {quote}: {count} paires")
            
            # Tri par volume et limitation
            if self.config.min_volume and tickers:
                filtered_symbols.sort(key=lambda s: tickers.get(s, {}).get('quoteVolume', 0), reverse=True)
                logger.info(f"Top 10 par volume: {filtered_symbols[:10]}")
            
            if self.config.max_symbols and len(filtered_symbols) > self.config.max_symbols:
                filtered_symbols = filtered_symbols[:self.config.max_symbols]
                logger.info(f"Limité à {self.config.max_symbols} symboles")
            
            return filtered_symbols
            
        except Exception as e:
            logger.error(f"Erreur récupération symboles: {e}")
            fallback = ['BTC/USDT', 'ETH/USDT', 'BNB/USDT', 'ADA/USDT', 'XRP/USDT']
            logger.info(f"Utilisation fallback: {fallback}")
            return fallback
    
    async def close(self):
        """Ferme les connexions"""
        await self.exchange.close()

# Workers
async def watch_trades(symbol: str, client: BinanceClient, producer: AIOKafkaProducer, 
                      topic: str, aggregator, semaphore: asyncio.Semaphore):
    """Worker pour surveiller les trades"""
    async with semaphore:
        last_id = None
        error_count = 0
        max_errors = 10
        
        while True:
            try:
                trades = await client.exchange.watch_trades(symbol)
                if not trades:
                    continue
                    
                trade = trades[-1]
                tid = trade.get("id")
                if tid and tid == last_id:
                    continue
                last_id = tid

                ts = int(trade["timestamp"]) if trade.get("timestamp") else int(time.time() * 1000)
                price = float(trade["price"])
                amount = float(trade["amount"])

                # Publier le trade
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
                
                key = f"{EXCHANGE_NAME}|{symbol.replace('/', '')}".encode()
                headers = [
                    ("source", b"collector-ws"),
                    ("exchange", EXCHANGE_NAME.encode()),
                    ("symbol", symbol.replace("/", "").encode()),
                    ("type", b"trade"),
                    ("schema", b"trade_v1"),
                ]
                
                await producer.send_and_wait(topic, orjson.dumps(payload), key=key, headers=headers)

                # Alimenter l'agrégateur
                if aggregator:
                    await aggregator.on_trade(EXCHANGE_NAME, symbol, ts, price, amount)
                
                error_count = 0

            except Exception as e:
                error_count += 1
                if error_count <= 3:
                    logger.warning(f"[trades:{symbol}] Erreur {error_count}/{max_errors}: {e}")
                
                if error_count >= max_errors:
                    logger.error(f"[trades:{symbol}] Trop d'erreurs, arrêt")
                    break
                    
                await asyncio.sleep(min(error_count * 2, 60))

async def watch_ticker(symbol: str, client: BinanceClient, producer: AIOKafkaProducer, 
                      topic: str, semaphore: asyncio.Semaphore):
    """Worker pour surveiller les tickers"""
    async with semaphore:
        error_count = 0
        max_errors = 10
        
        while True:
            try:
                ticker = await client.exchange.watch_ticker(symbol)
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
                
                key = f"{EXCHANGE_NAME}|{symbol.replace('/', '')}".encode()
                headers = [
                    ("source", b"collector-ws"),
                    ("exchange", EXCHANGE_NAME.encode()),
                    ("symbol", symbol.replace("/", "").encode()),
                    ("type", b"ticker"),
                    ("schema", b"ticker_v1"),
                ]
                
                await producer.send_and_wait(topic, orjson.dumps(payload), key=key, headers=headers)
                error_count = 0
                
            except Exception as e:
                error_count += 1
                if error_count <= 3:
                    logger.warning(f"[ticker:{symbol}] Erreur {error_count}/{max_errors}: {e}")
                
                if error_count >= max_errors:
                    logger.error(f"[ticker:{symbol}] Trop d'erreurs, arrêt")
                    break
                    
                await asyncio.sleep(min(error_count * 2, 60))