import asyncio
import logging
import signal
from aiokafka import AIOKafkaProducer

from config import load_config
from binance_client import BinanceClient, watch_trades, watch_ticker
from aggregator import OptimizedAggregator

# Optimisation des performances avec uvloop (event loop plus rapide)
try:
    import uvloop
    uvloop.install()
except ImportError:
    pass

# Configuration du logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

async def create_kafka_producer(config):
    """
    Crée et démarre le producer Kafka avec optimisations

    Paramètres de performance:
    - linger_ms=100: Batching des messages pour réduire les requêtes
    - compression: Compression des données (lz4, gzip, etc.)
    - acks="all": Attendre confirmation de tous les replicas
    - enable_idempotence: Éviter les doublons en cas de retry
    """
    producer = AIOKafkaProducer(
        bootstrap_servers=config.kafka_bootstrap,
        client_id=config.client_id,
        value_serializer=lambda v: v,  # Déjà sérialisé avec orjson
        key_serializer=lambda k: k,    # Clés en bytes
        linger_ms=100,                  # Batching pour performances
        compression_type=config.compression,
        acks="all",                     # Garantie de durabilité
        enable_idempotence=True,        # Pas de doublons
        max_request_size=2_097_152,     # 2 MB max par requête
        request_timeout_ms=60000,       # Timeout 60s
    )
    await producer.start()
    return producer

async def main():
    """
    Fonction principale du collecteur

    Étapes:
    1. Charger la configuration depuis les variables d'environnement
    2. Se connecter à Binance et récupérer les symboles filtrés
    3. Créer le producer Kafka et l'agrégateur (si activé)
    4. Lancer les workers WebSocket pour chaque symbole
    5. Démarrer le monitoring périodique
    6. Gérer l'arrêt propre sur signal SIGINT/SIGTERM
    """
    logger.info("=== Démarrage du Collecteur Crypto ===")

    # Charger la configuration depuis l'environnement
    config = load_config()
    logger.info(f"Configuration: quotes={config.quote_currencies}, min_volume={config.min_volume}")

    # Se connecter à Binance et récupérer les symboles filtrés
    client = BinanceClient(config)
    symbols = await client.get_all_symbols()

    if not symbols:
        logger.error("Aucun symbole trouvé")
        return

    # Créer le producer Kafka et l'agrégateur si activé
    producer = await create_kafka_producer(config)
    aggregator = OptimizedAggregator(producer, config) if config.enable_aggregation else None

    # Créer les tâches WebSocket pour chaque symbole
    tasks = []
    semaphore = asyncio.Semaphore(config.max_concurrent)

    # Démarrer le sweeper pour l'agrégateur (ferme les fenêtres expirées)
    if aggregator:
        tasks.append(asyncio.create_task(aggregator.sweeper()))

    # Créer un worker pour chaque symbole et feature activée
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
    
    # Tâche de monitoring périodique pour suivre les performances
    async def monitor():
        """Affiche les statistiques toutes les 5 minutes"""
        while True:
            await asyncio.sleep(300)
            if aggregator:
                logger.info(f"Stats: {aggregator.stats}, fenêtres actives: {len(aggregator.windows)}")

    tasks.append(asyncio.create_task(monitor()))

    # Gestion de l'arrêt propre du collecteur
    stop = asyncio.Event()

    def shutdown():
        """Callback appelé lors de la réception d'un signal d'arrêt"""
        logger.info("Signal d'arrêt reçu...")
        stop.set()

    # Enregistrer les handlers pour SIGINT (Ctrl+C) et SIGTERM (docker stop)
    loop = asyncio.get_running_loop()
    for sig in (signal.SIGINT, signal.SIGTERM):
        try:
            loop.add_signal_handler(sig, shutdown)
        except NotImplementedError:
            # Pas supporté sur Windows
            pass

    # Attendre le signal d'arrêt
    try:
        await stop.wait()
    finally:
        # Arrêt propre de tous les composants
        logger.info("Arrêt en cours...")

        # Annuler toutes les tâches WebSocket
        for task in tasks:
            task.cancel()
        await asyncio.gather(*tasks, return_exceptions=True)

        # Fermer l'agrégateur (publier les barres en cours)
        if aggregator:
            await aggregator.stop()

        # Fermer le producer Kafka
        await producer.stop()

        # Fermer les connexions Binance
        await client.close()

        logger.info("Arrêt terminé proprement")

if __name__ == "__main__":
    """
    Point d'entrée du programme
    Lance la boucle asyncio et gère les erreurs globales
    """
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.info("Programme interrompu par l'utilisateur")
    except Exception as e:
        logger.error(f"Erreur fatale: {e}", exc_info=True)