import asyncio
import logging
import json
import os
from datetime import datetime, timedelta
from pathlib import Path
from typing import List, Dict, Optional
import time

import ccxt.async_support as ccxt
from aiokafka import AIOKafkaProducer
import asyncpg
import orjson

from config import load_config

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


def create_historical_headers(exchange: str, symbol: str, timeframe: str):
    # Headers standardisés pour les données historiques
    return [
        ("source", b"collector-historical"),
        ("type", b"bar"),
        ("exchange", exchange.encode()),
        ("symbol", symbol.replace("/", "").encode()),
        ("timeframe", timeframe.encode()),
        ("closed", b"true"),
        ("schema", b"bar_v2"),
    ]


class ProgressTracker:
    """
    Database-backed backfill progress tracker.

    Tracks historical data backfill progress in TimescaleDB instead of JSON files.
    This provides better reliability, queryability, and supports dynamic lookback period changes.
    """

    def __init__(self, db_pool: asyncpg.Pool, exchange: str = "BINANCE"):
        """
        Initialize progress tracker with database connection pool.

        Args:
            db_pool: asyncpg connection pool
            exchange: Exchange name (default: BINANCE)
        """
        self.pool = db_pool
        self.exchange = exchange
        logger.info("✓ ProgressTracker initialized with database backend")

    async def get_last_timestamp(self, symbol: str, timeframe: str) -> Optional[int]:
        """
        Get the last successfully fetched timestamp for a symbol/timeframe.

        Returns:
            Last timestamp in milliseconds, or None if no backfill exists
        """
        try:
            async with self.pool.acquire() as conn:
                result = await conn.fetchval(
                    """
                    SELECT get_last_backfill_timestamp($1, $2, $3)
                    """,
                    symbol, timeframe, self.exchange
                )

                if result:
                    # Convert timestamp to milliseconds
                    return int(result.timestamp() * 1000)
                return None

        except Exception as e:
            logger.error(f"Error getting last timestamp for {symbol} {timeframe}: {e}")
            return None

    async def update_progress(
        self,
        symbol: str,
        timeframe: str,
        current_ts: int,
        start_ts: int,
        end_ts: int,
        status: str = 'in_progress',
        candles_count: int = 0,
        error_message: Optional[str] = None
    ):
        """
        Update backfill progress in database (idempotent).

        Args:
            symbol: Trading pair (e.g., 'BTC/USDT')
            timeframe: Timeframe (e.g., '1m', '1h', '1d')
            current_ts: Current position timestamp in milliseconds
            start_ts: Backfill start timestamp in milliseconds
            end_ts: Backfill end timestamp in milliseconds
            status: Status ('pending', 'in_progress', 'completed', 'failed')
            candles_count: Number of candles fetched in this batch
            error_message: Error message if status is 'failed'
        """
        try:
            # Convert milliseconds to datetime
            current_position = datetime.fromtimestamp(current_ts / 1000)
            backfill_start = datetime.fromtimestamp(start_ts / 1000)
            backfill_end = datetime.fromtimestamp(end_ts / 1000)

            async with self.pool.acquire() as conn:
                await conn.execute(
                    """
                    SELECT update_backfill_progress($1, $2, $3, $4, $5, $6, $7, $8, $9)
                    """,
                    symbol,
                    timeframe,
                    self.exchange,
                    backfill_start,
                    backfill_end,
                    current_position,
                    status,
                    candles_count,
                    error_message
                )

        except Exception as e:
            logger.error(f"Error updating progress for {symbol} {timeframe}: {e}")

    async def is_complete(self, symbol: str, timeframe: str, target_timestamp: int) -> bool:
        """
        Check if backfill is complete for a symbol/timeframe.

        Args:
            symbol: Trading pair
            timeframe: Timeframe
            target_timestamp: Target end timestamp in milliseconds

        Returns:
            True if backfill is completed and reached target timestamp
        """
        try:
            async with self.pool.acquire() as conn:
                result = await conn.fetchrow(
                    """
                    SELECT status, current_position_ts
                    FROM backfill_progress
                    WHERE symbol = $1
                      AND timeframe = $2
                      AND exchange = $3
                    """,
                    symbol, timeframe, self.exchange
                )

                if not result:
                    return False

                status = result['status']
                current_position_ts = result['current_position_ts']

                if status != 'completed':
                    return False

                # Check if we reached the target timestamp
                current_ts_ms = int(current_position_ts.timestamp() * 1000)
                return current_ts_ms >= target_timestamp

        except Exception as e:
            logger.error(f"Error checking completion for {symbol} {timeframe}: {e}")
            return False


class HistoricalCollector:
    # Collecteur de données historiques via REST API (batch de 1000 barres)

    def __init__(self, config):
        self.config = config
        self.exchange = None
        self.producer = None
        self.db_pool = None
        self.tracker = None

        # Rate limiting pour pas se faire ban
        self.rate_limiter = asyncio.Semaphore(20)
        self.request_delay = 0.05

    async def setup(self):
        # Init database pool
        db_host = os.getenv("TIMESCALE_HOST", "localhost")
        db_port = os.getenv("TIMESCALE_PORT", "7432")
        db_name = os.getenv("TIMESCALE_DB", "cryptoviz")
        db_user = os.getenv("TIMESCALE_USER", "postgres")
        db_password = os.getenv("TIMESCALE_PASSWORD", "secure_password_here")

        try:
            self.db_pool = await asyncpg.create_pool(
                host=db_host,
                port=db_port,
                database=db_name,
                user=db_user,
                password=db_password,
                min_size=2,
                max_size=10,
                command_timeout=60
            )
            logger.info("✓ Database connection pool created")

            # Init progress tracker with database
            self.tracker = ProgressTracker(self.db_pool)

        except Exception as e:
            logger.error(f"Failed to connect to database: {e}")
            raise

        # Init exchange Binance
        self.exchange = ccxt.binance({
            'apiKey': self.config.api_key,
            'secret': self.config.api_secret,
            'enableRateLimit': True,
            'options': {'defaultType': self.config.default_type}
        })

        # Init producer Kafka
        self.producer = AIOKafkaProducer(
            bootstrap_servers=self.config.kafka_bootstrap,
            value_serializer=lambda v: v,
            key_serializer=lambda k: k,
            compression_type=self.config.compression,
            linger_ms=100,
            acks="all",
            enable_idempotence=True,
        )
        await self.producer.start()
        logger.info("✓ Kafka producer started")

    async def cleanup(self):
        if self.producer:
            await self.producer.stop()
        if self.exchange:
            await self.exchange.close()
        if self.db_pool:
            await self.db_pool.close()
            logger.info("✓ Database connection pool closed")

    def get_symbols_from_collector(self) -> List[str]:
        # Récup la liste des symboles (pour l'instant vide, on pourrait partager via Redis)
        return []

    async def fetch_ohlcv_batch(
        self,
        symbol: str,
        timeframe: str,
        since: int,
        limit: int = 1000
    ) -> List:
        # Récupère un batch d'OHLCV (max 1000 barres)
        async with self.rate_limiter:
            try:
                await asyncio.sleep(self.request_delay)
                ohlcv = await self.exchange.fetch_ohlcv(
                    symbol,
                    timeframe=timeframe,
                    since=since,
                    limit=limit
                )
                return ohlcv
            except Exception as e:
                logger.error(f"Error fetching {symbol} {timeframe}: {e}")
                return []

    async def backfill_symbol_timeframe(
        self,
        symbol: str,
        timeframe: str,
        start_date: datetime,
        end_date: datetime
    ):
        # Backfill complet d'un symbole pour un timeframe
        start_ts = int(start_date.timestamp() * 1000)
        end_ts = int(end_date.timestamp() * 1000)

        # Check si déjà complet
        if await self.tracker.is_complete(symbol, timeframe, end_ts):
            logger.info(f"{symbol} {timeframe} already complete")
            return

        # Reprend depuis la dernière position
        last_ts = await self.tracker.get_last_timestamp(symbol, timeframe)
        current_ts = last_ts or start_ts

        topic = f"{self.config.output_prefix}{timeframe}"
        total_bars = 0
        batch_bars = 0

        logger.info(f"Starting backfill for {symbol} {timeframe}: {datetime.fromtimestamp(current_ts/1000)} -> {end_date}")

        try:
            while current_ts < end_ts:
                # Récup un batch de 1000 barres
                ohlcv = await self.fetch_ohlcv_batch(symbol, timeframe, current_ts, limit=1000)

                if not ohlcv:
                    logger.warning(f"No data for {symbol} {timeframe} since {current_ts}")
                    break

                batch_bars = 0

                # Publie chaque barre dans Kafka
                for bar in ohlcv:
                    timestamp, open_price, high, low, close, volume = bar

                    # Dépasse pas la date de fin
                    if timestamp > end_ts:
                        break

                    # Message (même format que le temps réel)
                    message = {
                        'type': 'bar',
                        'exchange': 'BINANCE',
                        'symbol': symbol,
                        'timeframe': timeframe,
                        'window_start': timestamp,
                        'window_end': timestamp + (ohlcv[1][0] - timestamp if len(ohlcv) > 1 else 60000),
                        'open': open_price,
                        'high': high,
                        'low': low,
                        'close': close,
                        'volume': volume,
                        'trade_count': 0,  # Pas dispo en historique
                        'closed': True,
                        'first_trade_ts': timestamp,
                        'last_trade_ts': timestamp,
                        'duration_ms': 0,
                        'source': 'historical'
                    }

                    # Publie dans Kafka
                    key = f"BINANCE|{symbol.replace('/', '')}|{timeframe}|{timestamp}".encode()
                    value = orjson.dumps(message)

                    # Headers standardisés
                    headers = create_historical_headers("BINANCE", symbol, timeframe)

                    await self.producer.send(topic, value=value, key=key, headers=headers)

                    total_bars += 1
                    batch_bars += 1
                    current_ts = timestamp

                # Update la progression (async call with all required parameters)
                await self.tracker.update_progress(
                    symbol=symbol,
                    timeframe=timeframe,
                    current_ts=current_ts,
                    start_ts=start_ts,
                    end_ts=end_ts,
                    status='in_progress',
                    candles_count=batch_bars
                )

                # Si moins de 1000 barres, on a fini
                if len(ohlcv) < 1000:
                    break

                # Avance au timestamp suivant
                current_ts = ohlcv[-1][0] + 1

            # Mark as completed
            await self.tracker.update_progress(
                symbol=symbol,
                timeframe=timeframe,
                current_ts=current_ts,
                start_ts=start_ts,
                end_ts=end_ts,
                status='completed',
                candles_count=0
            )

            logger.info(f"✓ {symbol} {timeframe} completed: {total_bars} bars collected")

        except Exception as e:
            # Mark as failed
            logger.error(f"✗ {symbol} {timeframe} failed: {e}")
            await self.tracker.update_progress(
                symbol=symbol,
                timeframe=timeframe,
                current_ts=current_ts,
                start_ts=start_ts,
                end_ts=end_ts,
                status='failed',
                candles_count=0,
                error_message=str(e)
            )

    async def backfill_all(self, symbols: List[str], timeframes: List[str], lookback_days: int = 365):
        # Lance le backfill pour tous les symboles et timeframes
        end_date = datetime.utcnow() - timedelta(hours=1)
        start_date = end_date - timedelta(days=lookback_days)

        logger.info("=" * 50)
        logger.info("📊 Starting Historical Backfill")
        logger.info("=" * 50)
        logger.info(f"Period: {start_date} -> {end_date}")
        logger.info(f"Symbols: {len(symbols)}")
        logger.info(f"Timeframes: {timeframes}")

        # Crée toutes les tâches
        tasks = []
        for symbol in symbols:
            for timeframe in timeframes:
                task = asyncio.create_task(
                    self.backfill_symbol_timeframe(symbol, timeframe, start_date, end_date)
                )
                tasks.append(task)

        # Exécute toutes les tâches
        logger.info(f"Launching {len(tasks)} backfill tasks...")
        await asyncio.gather(*tasks, return_exceptions=True)

        logger.info("=" * 50)
        logger.info("✓ Historical Backfill Complete")
        logger.info("=" * 50)


async def main():
    config = load_config()

    # Charge les params du backfill
    lookback_days = int(os.getenv("BACKFILL_LOOKBACK_DAYS", "365"))
    backfill_timeframes_str = os.getenv("BACKFILL_TIMEFRAMES", "1m,5m,15m,1h,4h,1d")
    backfill_timeframes = []

    # Filtre les timeframes: vire ceux en secondes (pas supportés par REST API)
    for tf in backfill_timeframes_str.split(","):
        tf = tf.strip()
        if tf and not tf.endswith('s'):
            backfill_timeframes.append(tf)
        elif tf and tf.endswith('s') and not tf.endswith('ms'):
            logger.warning(f"Ignoring timeframe '{tf}' (REST API does not support seconds)")

    logger.info(f"Backfill configuration: {lookback_days} days, timeframes: {backfill_timeframes}")

    collector = HistoricalCollector(config)

    try:
        await collector.setup()

        # Récup la liste des symboles (même logique que temps réel)
        from binance_client import BinanceClient
        client = BinanceClient(config)
        symbols = await client.get_all_symbols()
        await client.close()

        if not symbols:
            logger.error("No symbols found")
            return

        logger.info(f"Symbols to backfill: {symbols[:10]}... ({len(symbols)} total)")

        # Lance le backfill
        await collector.backfill_all(symbols, backfill_timeframes, lookback_days)

    except Exception as e:
        logger.error(f"Backfill error: {e}", exc_info=True)
    finally:
        await collector.cleanup()


if __name__ == "__main__":
    asyncio.run(main())
