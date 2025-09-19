# ==================== Structure simplifiée ====================
#
# services/data-collector/
# ├── main.py              # Point d'entrée + orchestration (~100 lignes)
# ├── config.py            # Configuration centralisée (~50 lignes)
# ├── binance_client.py    # Binance + gestion symboles (~150 lignes)  
# └── aggregator.py        # Agrégation + utils (~200 lignes)

# ==================== config.py ====================
import os
import logging
from typing import List, Optional
from dataclasses import dataclass

@dataclass
class Config:
    """Configuration centralisée de l'application"""
    # Symboles
    quote_currencies: Optional[List[str]]
    min_volume: Optional[float] 
    max_symbols: Optional[int]
    
    # Features
    enable_trades: bool
    enable_ticker: bool
    enable_aggregation: bool
    
    # Kafka
    kafka_bootstrap: str
    topic_trades: str
    topic_ticker: str
    output_prefix: str
    compression: Optional[str]
    client_id: str
    
    # Agrégation
    timeframes: List[str]
    grace_ms: int
    sweep_sec: float
    
    # Binance
    api_key: Optional[str]
    api_secret: Optional[str]
    default_type: str
    
    # Performance
    max_concurrent: int

def load_config() -> Config:
    """Charge la configuration depuis les variables d'environnement"""
    quote_currencies = os.getenv("QUOTE_CURRENCIES", "").strip()
    quote_filter = [q.strip().upper() for q in quote_currencies.split(",") if q.strip()] if quote_currencies else None
    
    min_volume = None
    if os.getenv("MIN_VOLUME"):
        try:
            min_volume = float(os.getenv("MIN_VOLUME"))
        except ValueError:
            logging.warning("MIN_VOLUME invalide, ignoré")
    
    max_symbols = None
    if os.getenv("MAX_SYMBOLS"):
        try:
            max_symbols = int(os.getenv("MAX_SYMBOLS"))
        except ValueError:
            logging.warning("MAX_SYMBOLS invalide, ignoré")
    
    return Config(
        quote_currencies=quote_filter,
        min_volume=min_volume,
        max_symbols=max_symbols,
        enable_trades=os.getenv("ENABLE_TRADES", "true").lower() == "true",
        enable_ticker=os.getenv("ENABLE_TICKER", "false").lower() == "true",
        enable_aggregation=os.getenv("ENABLE_AGGREGATION", "true").lower() == "true",
        kafka_bootstrap=os.getenv("KAFKA_BROKERS", "localhost:9092"),
        topic_trades=os.getenv("TOPIC_TRADES", "crypto.raw.trades"),
        topic_ticker=os.getenv("TOPIC_TICKER", "crypto.ticker"),
        output_prefix=os.getenv("OUTPUT_PREFIX", "crypto.aggregated."),
        compression=os.getenv("KAFKA_COMPRESSION", "lz4").lower() or None,
        client_id=os.getenv("KAFKA_CLIENT_ID", "collector-optimized"),
        timeframes=[t.strip() for t in os.getenv("TIMEFRAMES", "5s,1m,15m,1h").split(",") if t.strip()],
        grace_ms=int(os.getenv("GRACE_MS", "2000")),
        sweep_sec=float(os.getenv("SWEEP_SEC", "1.0")),
        api_key=os.getenv("BINANCE_API_KEY"),
        api_secret=os.getenv("BINANCE_SECRET_KEY"),
        default_type=os.getenv("BINANCE_DEFAULT_TYPE", "spot"),
        max_concurrent=int(os.getenv("MAX_CONCURRENT", "50")),
    )

# ==================== binance_client.py ====================
import logging
import asyncio
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

# ==================== aggregator.py ====================
import orjson
import time
import datetime
import asyncio
import logging
from typing import Dict, Tuple, List
from aiokafka import AIOKafkaProducer

from config import Config

logger = logging.getLogger(__name__)

def parse_timeframe(s: str) -> int:
    """Parse timeframe string vers millisecondes"""
    s = s.strip().lower()
    try:
        if s.endswith("ms"): return int(s[:-2])
        if s.endswith("s"):  return int(s[:-1]) * 1000
        if s.endswith("m"):  return int(s[:-1]) * 60_000
        if s.endswith("h"):  return int(s[:-1]) * 3_600_000
        if s.endswith("d"):  return int(s[:-1]) * 86_400_000
        raise ValueError(f"Format timeframe non reconnu: {s}")
    except ValueError as e:
        logger.error(f"Erreur parsing timeframe '{s}': {e}")
        return 60_000

def floor_ts_correct(ts_ms: int, tf_ms: int) -> int:
    """Aligne correctement le timestamp sur les fenêtres temporelles réelles"""
    ts_sec = ts_ms // 1000
    tf_sec = tf_ms // 1000
    
    dt = datetime.datetime.fromtimestamp(ts_sec, tz=datetime.timezone.utc)
    
    if tf_sec == 1:
        aligned = dt.replace(microsecond=0)
    elif tf_sec == 5:
        aligned = dt.replace(second=(dt.second // 5) * 5, microsecond=0)
    elif tf_sec == 30:
        aligned = dt.replace(second=(dt.second // 30) * 30, microsecond=0)
    elif tf_sec == 60:
        aligned = dt.replace(second=0, microsecond=0)
    elif tf_sec == 300:
        aligned = dt.replace(minute=(dt.minute // 5) * 5, second=0, microsecond=0)
    elif tf_sec == 900:
        aligned = dt.replace(minute=(dt.minute // 15) * 15, second=0, microsecond=0)
    elif tf_sec == 3600:
        aligned = dt.replace(minute=0, second=0, microsecond=0)
    elif tf_sec == 14400:
        aligned = dt.replace(hour=(dt.hour // 4) * 4, minute=0, second=0, microsecond=0)
    elif tf_sec == 86400:
        aligned = dt.replace(hour=0, minute=0, second=0, microsecond=0)
    else:
        # Fallback
        seconds_since_midnight = dt.hour * 3600 + dt.minute * 60 + dt.second
        aligned_seconds = (seconds_since_midnight // tf_sec) * tf_sec
        hours = aligned_seconds // 3600
        minutes = (aligned_seconds % 3600) // 60
        seconds = aligned_seconds % 60
        aligned = dt.replace(hour=hours, minute=minutes, second=seconds, microsecond=0)
    
    return int(aligned.timestamp() * 1000)

class OptimizedAggregator:
    """Agrégateur optimisé avec alignement temporel correct"""
    
    def __init__(self, producer: AIOKafkaProducer, config: Config):
        self.producer = producer
        self.config = config
        self.tf_ms = {lbl: parse_timeframe(lbl) for lbl in config.timeframes}
        self.windows: Dict[Tuple[str, str, str], dict] = {}
        self._stop = asyncio.Event()
        
        # Statistiques
        self.stats = {
            'bars_created': 0,
            'bars_updated': 0, 
            'bars_closed': 0,
            'trades_processed': 0,
            'errors': 0
        }
        
        logger.info(f"Agrégateur initialisé avec timeframes: {config.timeframes}")

    async def stop(self):
        self._stop.set()
        logger.info(f"Stats finales agrégateur: {self.stats}")

    async def sweeper(self):
        """Nettoie les fenêtres expirées"""
        while not self._stop.is_set():
            await asyncio.sleep(self.config.sweep_sec)
            now_ms = int(time.time() * 1000)
            
            to_close = []
            for key, bar in list(self.windows.items()):
                tf_ms = self.tf_ms[bar["timeframe"]]
                window_end = bar["window_start"] + tf_ms
                
                if not bar["closed"] and now_ms >= window_end + self.config.grace_ms:
                    to_close.append(key)
            
            for key in to_close:
                bar = self.windows.get(key)
                if bar and not bar["closed"]:
                    await self._emit_bar(bar, closed=True)
                    self.stats['bars_closed'] += 1

    async def on_trade(self, exchange: str, symbol: str, ts: int, price: float, amount: float):
        """Traite un trade et met à jour les fenêtres temporelles"""
        self.stats['trades_processed'] += 1
        
        try:
            for tf_label, tf_ms in self.tf_ms.items():
                win_start = floor_ts_correct(ts, tf_ms)
                key = (exchange, symbol, tf_label)
                bar = self.windows.get(key)

                if (bar is None) or (bar["window_start"] != win_start):
                    # Fermer l'ancienne fenêtre
                    if bar is not None and not bar["closed"]:
                        await self._emit_bar(bar, closed=True)
                        self.stats['bars_closed'] += 1

                    # Créer nouvelle fenêtre
                    bar = {
                        "type": "bar",
                        "exchange": exchange,
                        "symbol": symbol,
                        "timeframe": tf_label,
                        "window_start": win_start,
                        "window_end": win_start + tf_ms,
                        "open": price,
                        "high": price,
                        "low": price,
                        "close": price,
                        "volume": amount,
                        "trade_count": 1,
                        "closed": False,
                        "first_trade_ts": ts,
                        "last_trade_ts": ts,
                    }
                    self.windows[key] = bar
                    self.stats['bars_created'] += 1
                    
                else:
                    # Mettre à jour fenêtre existante
                    if price > bar["high"]:
                        bar["high"] = price
                    if price < bar["low"]:
                        bar["low"] = price
                    bar["close"] = price
                    bar["volume"] += amount
                    bar["trade_count"] += 1
                    bar["last_trade_ts"] = ts
                    self.stats['bars_updated'] += 1
                    
        except Exception as e:
            self.stats['errors'] += 1
            logger.error(f"Erreur traitement trade {symbol}: {e}")

    async def _emit_bar(self, bar: dict, closed: bool):
        """Émet une barre vers Kafka"""
        try:
            output_bar = dict(bar)
            output_bar["closed"] = closed
            output_bar["duration_ms"] = bar["last_trade_ts"] - bar["first_trade_ts"]
            
            topic = f"{self.config.output_prefix}{bar['timeframe']}"
            key = f"{bar['exchange']}|{bar['symbol'].replace('/','')}|{bar['timeframe']}|{bar['window_start']}".encode()
            headers = [
                ("source", b"collector-ws"),
                ("type", b"bar"),
                ("exchange", bar["exchange"].encode()),
                ("symbol", bar["symbol"].replace("/", "").encode()),
                ("timeframe", bar["timeframe"].encode()),
                ("closed", b"true" if closed else b"false"),
                ("schema", b"bar_v2"),
            ]
            
            await self.producer.send_and_wait(topic, orjson.dumps(output_bar), key=key, headers=headers)
            
            if closed:
                bar["closed"] = True
                
        except Exception as e:
            self.stats['errors'] += 1
            logger.error(f"Erreur émission barre {bar.get('symbol', 'unknown')}: {e}")