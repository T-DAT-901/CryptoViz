import asyncio
import logging
import signal
import orjson
from aiokafka import AIOKafkaProducer

from config import load_config
from binance_client import BinanceClient, watch_trades, watch_ticker
from aggregator import OptimizedAggregator

# uvloop pour performances
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
    """Crée et démarre le producer Kafka"""
    producer = AIOKafkaProducer(
        bootstrap_servers=config.kafka_bootstrap,
        client_id=config.client_id,
        value_serializer=lambda v: v,
        key_serializer=lambda k: k,
        linger_ms=100,
        compression_type=config.compression,
        acks="all",
        enable_idempotence=True,
        max_request_size=2_097_152,
        request_timeout_ms=60000,
    )
    await producer.start()
    return producer

async def main():
    """Fonction principale"""
    logger.info("=== Démarrage du Collector Crypto Optimisé ===")
    
    # Configuration
    config = load_config()
    logger.info(f"Configuration: quotes={config.quote_currencies}, min_volume={config.min_volume}")
    
    # Initialiser les composants
    client = BinanceClient(config)
    symbols = await client.get_all_symbols()
    
    if not symbols:
        logger.error("Aucun symbole trouvé")
        return
    
    producer = await create_kafka_producer(config)
    aggregator = OptimizedAggregator(producer, config) if config.enable_aggregation else None
    
    # Créer les tâches
    tasks = []
    semaphore = asyncio.Semaphore(config.max_concurrent)
    
    if aggregator:
        tasks.append(asyncio.create_task(aggregator.sweeper()))
    
    for symbol in symbols:
        if config.enable_trades:
            tasks.append(asyncio.create_task(
                watch_trades(symbol, client, producer, config.topic_trades, aggregator, semaphore)
            ))
        if config.enable_ticker:
            tasks.append(asyncio.create_task(
                watch_ticker(symbol, client, producer, config.topic_ticker, semaphore)
            ))
    
    logger.info(f"{len(tasks)} tâches démarrées pour {len(symbols)} symboles")
    
    # Monitoring périodique
    async def monitor():
        while True:
            await asyncio.sleep(300)  # 5 min
            if aggregator:
                logger.info(f"Stats: {aggregator.stats}, fenêtres actives: {len(aggregator.windows)}")
    
    tasks.append(asyncio.create_task(monitor()))
    
    # Gestion arrêt
    stop = asyncio.Event()
    
    def shutdown():
        logger.info("Signal d'arrêt reçu...")
        stop.set()
    
    loop = asyncio.get_running_loop()
    for sig in (signal.SIGINT, signal.SIGTERM):
        try:
            loop.add_signal_handler(sig, shutdown)
        except NotImplementedError:
            pass
    
    try:
        await stop.wait()
    finally:
        logger.info("Arrêt en cours...")
        for task in tasks:
            task.cancel()
        await asyncio.gather(*tasks, return_exceptions=True)
        
        if aggregator:
            await aggregator.stop()
        await producer.stop()
        await client.close()
        
        logger.info("Arrêt terminé proprement")

if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.info("Programme interrompu")
    except Exception as e:
        logger.error(f"Erreur fatale: {e}", exc_info=True)