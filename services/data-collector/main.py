import asyncio
import logging
import signal
import os
import json
from datetime import datetime, timedelta
from pathlib import Path
from aiokafka import AIOKafkaProducer

from config import load_config
from binance_client import BinanceClient, watch_trades, watch_ticker
from aggregator import OptimizedAggregator
from historical_collector import HistoricalCollector

# Utilise uvloop si dispo pour meilleures perfs
try:
    import uvloop
    uvloop.install()
except ImportError:
    pass

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


async def create_kafka_producer(config):
    # Producer Kafka optimisé avec batching et compression
    producer = AIOKafkaProducer(
        bootstrap_servers=config.kafka_bootstrap,
        client_id=config.client_id,
        value_serializer=lambda v: v,
        key_serializer=lambda k: k,
        linger_ms=100,  # Groupe les messages pendant 100ms
        compression_type=config.compression,
        acks="all",
        enable_idempotence=True,
        max_request_size=2_097_152,
        request_timeout_ms=60000,
    )
    await producer.start()
    return producer


async def check_backfill_needed(db_pool, symbols, timeframes, lookback_days):
    """
    Smart backfill check using database backend.

    Detects:
    - First-time backfill (no records)
    - Failed backfills that need retry
    - Config changes (lookback period extended, e.g., 40 → 365 days)

    Returns:
        List of (symbol, timeframe, gap_days) tuples that need backfill
    """
    needs_backfill = []

    try:
        async with db_pool.acquire() as conn:
            for symbol in symbols:
                for timeframe in timeframes:
                    result = await conn.fetchrow(
                        """
                        SELECT * FROM check_backfill_needed($1, $2, $3, $4)
                        """,
                        symbol, timeframe, 'BINANCE', lookback_days
                    )

                    if result and result['needs_backfill']:
                        gap_days = result['gap_days']
                        status = result['status']

                        if gap_days > 0:
                            logger.info(
                                f"Config change detected for {symbol} {timeframe}: "
                                f"extending backfill by {gap_days} days "
                                f"(status: {status})"
                            )
                        else:
                            logger.info(
                                f"Backfill needed for {symbol} {timeframe} "
                                f"(status: {status})"
                            )

                        needs_backfill.append((symbol, timeframe, gap_days))

        if needs_backfill:
            logger.info(f"Found {len(needs_backfill)} symbol/timeframe combinations needing backfill")
        else:
            logger.info("All backfills complete and up-to-date")

        return needs_backfill

    except Exception as e:
        logger.error(f"Error checking backfill status: {e}")
        # If database check fails, assume backfill is needed (fail-safe)
        return [(s, tf, 0) for s in symbols for tf in timeframes]


async def run_backfill_if_needed(config, symbols):
    enable_backfill = os.getenv("ENABLE_BACKFILL", "false").lower() == "true"

    if not enable_backfill:
        logger.info("Backfill disabled, skipping")
        return

    logger.info("Checking if backfill is needed...")

    lookback_days = int(os.getenv("BACKFILL_LOOKBACK_DAYS", "365"))
    backfill_timeframes_str = os.getenv("BACKFILL_TIMEFRAMES", "1m,5m,15m,1h,4h,1d")
    backfill_timeframes = []

    # Garde que les timeframes supportés par l'API REST (pas de secondes)
    for tf in backfill_timeframes_str.split(","):
        tf = tf.strip()
        if tf and not tf.endswith('s'):
            backfill_timeframes.append(tf)
        elif tf and tf.endswith('s') and not tf.endswith('ms'):
            logger.warning(f"Ignoring timeframe '{tf}' (REST API does not support seconds)")

    collector = HistoricalCollector(config)

    try:
        await collector.setup()

        # Smart database-backed backfill check
        # This automatically detects config changes (e.g., 40 → 365 day extension)
        needs_backfill = await check_backfill_needed(
            collector.db_pool,
            symbols,
            backfill_timeframes,
            lookback_days
        )

        if not needs_backfill:
            logger.info("Backfill already complete and up-to-date")
            return

        logger.info(
            f"Starting historical backfill: {lookback_days} days, "
            f"{len(symbols)} symbols, timeframes: {backfill_timeframes}"
        )
        logger.info("This may take several hours...")

        await collector.backfill_all(symbols, backfill_timeframes, lookback_days)

        logger.info("✓ Backfill completed successfully")

    except Exception as e:
        logger.error(f"Backfill error: {e}", exc_info=True)
        logger.warning("Continuing with realtime collection despite backfill error")
    finally:
        await collector.cleanup()


async def run_realtime_collector(config, symbols, client):
    logger.info("Starting realtime collection...")

    producer = await create_kafka_producer(config)
    aggregator = OptimizedAggregator(producer, config) if config.enable_aggregation else None

    tasks = []
    semaphore = asyncio.Semaphore(config.max_concurrent)

    # Le sweeper ferme les fenêtres d'agrégation expirées
    if aggregator:
        tasks.append(asyncio.create_task(aggregator.sweeper()))

    # Un worker WebSocket par symbole et par feature
    for symbol in symbols:
        if config.enable_trades:
            tasks.append(asyncio.create_task(
                watch_trades(symbol, client, producer, config.topic_trades, aggregator, semaphore)
            ))
        if config.enable_ticker:
            tasks.append(asyncio.create_task(
                watch_ticker(symbol, client, producer, config.topic_ticker, semaphore)
            ))

    logger.info(f"✓ Started {len(tasks)} tasks for {len(symbols)} symbols")

    # Affiche des stats toutes les 5 min
    async def monitor():
        while True:
            await asyncio.sleep(300)
            if aggregator:
                logger.info(f"Stats: {aggregator.stats}, active windows: {len(aggregator.windows)}")

    tasks.append(asyncio.create_task(monitor()))

    # Gestion arrêt propre
    stop = asyncio.Event()

    def shutdown():
        logger.info("Shutdown signal received, stopping...")
        stop.set()

    # Handlers pour Ctrl+C et docker stop
    loop = asyncio.get_running_loop()
    for sig in (signal.SIGINT, signal.SIGTERM):
        try:
            loop.add_signal_handler(sig, shutdown)
        except NotImplementedError:
            pass  # Windows supporte pas

    try:
        await stop.wait()
    finally:
        logger.info("Stopping collector...")

        # Cancel toutes les tâches
        for task in tasks:
            task.cancel()
        await asyncio.gather(*tasks, return_exceptions=True)

        # Ferme tout proprement
        if aggregator:
            await aggregator.stop()
        await producer.stop()
        await client.close()

        logger.info("✓ Collector stopped cleanly")


async def main():
    logger.info("=" * 50)
    logger.info("🚀 CryptoViz Data Collector - Starting")
    logger.info("=" * 50)

    config = load_config()
    logger.info(f"Configuration loaded: quotes={config.quote_currencies}, min_volume={config.min_volume}, max_symbols={config.max_symbols}")

    # Récup les symboles depuis Binance avec les filtres
    client = BinanceClient(config)
    symbols = await client.get_all_symbols()

    if not symbols:
        logger.error("No symbols found, exiting")
        return

    logger.info(f"✓ Selected {len(symbols)} symbols for collection")

    # Start realtime collection immediately (non-blocking)
    logger.info("Starting realtime collection (priority)")
    realtime_task = asyncio.create_task(
        run_realtime_collector(config, symbols, client)
    )

    # Run backfill in background if enabled (non-blocking)
    enable_backfill = os.getenv("ENABLE_BACKFILL", "false").lower() == "true"
    if enable_backfill:
        logger.info("Starting historical backfill in background")
        backfill_task = asyncio.create_task(
            run_backfill_if_needed(config, symbols)
        )
    else:
        logger.info("Backfill disabled, skipping")
        backfill_task = None

    # Wait for realtime collection (runs indefinitely)
    # Backfill runs in parallel and completes independently
    await realtime_task


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.info("Interrupted by user")
    except Exception as e:
        logger.error(f"Fatal error: {e}", exc_info=True)
