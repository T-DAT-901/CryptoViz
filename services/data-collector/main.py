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


async def run_backfill_if_needed(config, symbols):
    enable_backfill = os.getenv("ENABLE_BACKFILL", "false").lower() == "true"

    if not enable_backfill:
        logger.info("⏭️  Backfill désactivé")
        return

    logger.info("🔍 Vérif si backfill nécessaire...")

    lookback_days = int(os.getenv("BACKFILL_LOOKBACK_DAYS", "365"))
    backfill_timeframes_str = os.getenv("BACKFILL_TIMEFRAMES", "1m,5m,15m,1h,4h,1d")
    backfill_timeframes = []

    # Garde que les timeframes supportés par l'API REST (pas de secondes)
    for tf in backfill_timeframes_str.split(","):
        tf = tf.strip()
        if tf and not tf.endswith('s'):
            backfill_timeframes.append(tf)
        elif tf and tf.endswith('s') and not tf.endswith('ms'):
            logger.warning(f"⏭️  {tf} ignoré (REST API supporte pas les secondes)")

    collector = HistoricalCollector(config)

    try:
        await collector.setup()

        # Check si déjà fait en lisant le fichier d'état
        state_file = Path(config.backfill_state_file)
        if state_file.exists():
            logger.info("📝 Fichier d'état trouvé, check si complet...")

            with open(state_file, 'r') as f:
                state = json.load(f)

            target_ts = int((datetime.utcnow() - timedelta(hours=1)).timestamp() * 1000)
            all_complete = True

            for symbol in symbols:
                for tf in backfill_timeframes:
                    if tf.endswith('s'):
                        continue
                    last_ts = state.get(symbol, {}).get(tf)
                    if last_ts is None or last_ts < target_ts:
                        all_complete = False
                        break
                if not all_complete:
                    break

            if all_complete:
                logger.info("✅ Backfill déjà complet")
                return

        logger.info("🚀 Go backfill historique...")
        logger.info(f"   Période: {lookback_days} jours")
        logger.info(f"   Timeframes: {backfill_timeframes}")
        logger.info(f"   Symboles: {len(symbols)}")
        logger.info("⏳ Ça peut prendre plusieurs heures...")

        await collector.backfill_all(symbols, backfill_timeframes, lookback_days)

        logger.info("✅ Backfill terminé")

    except Exception as e:
        logger.error(f"❌ Erreur backfill: {e}", exc_info=True)
        logger.warning("⚠️  On passe au temps réel quand même")
    finally:
        await collector.cleanup()


async def run_realtime_collector(config, symbols, client):
    logger.info("🔴 Go collecte temps réel...")

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

    logger.info(f"✅ {len(tasks)} tâches démarrées pour {len(symbols)} symboles")

    # Affiche des stats toutes les 5 min
    async def monitor():
        while True:
            await asyncio.sleep(300)
            if aggregator:
                logger.info(f"📊 Stats: {aggregator.stats}, fenêtres actives: {len(aggregator.windows)}")

    tasks.append(asyncio.create_task(monitor()))

    # Gestion arrêt propre
    stop = asyncio.Event()

    def shutdown():
        logger.info("🛑 Signal d'arrêt reçu...")
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
        logger.info("⏹️  Arrêt en cours...")

        # Cancel toutes les tâches
        for task in tasks:
            task.cancel()
        await asyncio.gather(*tasks, return_exceptions=True)

        # Ferme tout proprement
        if aggregator:
            await aggregator.stop()
        await producer.stop()
        await client.close()

        logger.info("✅ Arrêt propre OK")


async def main():
    logger.info("=" * 50)
    logger.info("  📊 CryptoViz Data Collector")
    logger.info("=" * 50)

    config = load_config()
    logger.info(f"Config: quotes={config.quote_currencies}, min_volume={config.min_volume}")

    # Récup les symboles depuis Binance avec les filtres
    client = BinanceClient(config)
    symbols = await client.get_all_symbols()

    if not symbols:
        logger.error("❌ Aucun symbole trouvé")
        return

    logger.info(f"📋 {len(symbols)} symboles sélectionnés")

    # 1. Backfill historique si besoin
    await run_backfill_if_needed(config, symbols)

    # 2. Collecte temps réel
    await run_realtime_collector(config, symbols, client)


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.info("Interrompu par user")
    except Exception as e:
        logger.error(f"Erreur fatale: {e}", exc_info=True)
