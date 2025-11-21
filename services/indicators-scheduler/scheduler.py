import os
import time
import logging
import threading
from datetime import datetime
import psycopg2
from apscheduler.schedulers.blocking import BlockingScheduler
from apscheduler.triggers.cron import CronTrigger

# Configuration logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Configuration PostgreSQL
DB_CONFIG = {
    'host': os.getenv('PGHOST', 'timescaledb'),
    'port': int(os.getenv('PGPORT', 5432)),
    'database': os.getenv('PGDATABASE', 'cryptoviz'),
    'user': os.getenv('PGUSER', 'postgres'),
    'password': os.getenv('PGPASSWORD', 'postgres')
}

def execute_sql(query):
    """Exécute une requête SQL"""
    try:
        conn = psycopg2.connect(**DB_CONFIG)
        conn.autocommit = True
        cursor = conn.cursor()
        cursor.execute(query)
        conn.close()
        return True
    except Exception as e:
        logger.error(f"Error executing SQL: {e}")
        return False

def refresh_indicators_1s():
    logger.info("Running refresh_indicators_1s()")
    execute_sql("CALL refresh_indicators_1s();")

def refresh_indicators_5s():
    logger.info("Running refresh_indicators_5s()")
    execute_sql("CALL refresh_indicators_5s();")

def refresh_indicators_1m():
    logger.info("Running refresh_indicators_1m()")
    execute_sql("CALL refresh_indicators_1m();")

def refresh_indicators_5m():
    logger.info("Running refresh_indicators_5m()")
    execute_sql("CALL refresh_indicators_5m();")

def refresh_indicators_15m():
    logger.info("Running refresh_indicators_15m()")
    execute_sql("CALL refresh_indicators_15m();")

def refresh_indicators_1h():
    logger.info("Running refresh_indicators_1h()")
    execute_sql("CALL refresh_indicators_1h();")

def refresh_indicators_1d():
    logger.info("Running refresh_indicators_1d()")
    execute_sql("CALL refresh_indicators_1d();")

def wait_for_historical_data():
    """Attend que le data-collector ait complètement fini le backfill historique"""
    logger.info("=" * 60)
    logger.info("WAITING FOR HISTORICAL BACKFILL TO COMPLETE")
    logger.info("=" * 60)
    logger.info("Monitoring collector-historical data (realtime data will continue during wait)")
    logger.info("Checking every 10 seconds...")

    max_wait_minutes = 60  # Attendre jusqu'à 1 heure pour le backfill complet
    checks = 0
    max_checks = (max_wait_minutes * 60) // 10  # Vérifier toutes les 10 secondes

    previous_historical_count = 0
    stable_count = 0
    min_historical_candles = 1000  # Minimum pour considérer que le backfill a vraiment commencé

    while checks < max_checks:
        try:
            conn = psycopg2.connect(**DB_CONFIG)
            cursor = conn.cursor()

            # Compter les candles par source
            cursor.execute("""
                SELECT
                    source,
                    COUNT(*) as count
                FROM candles
                GROUP BY source
                ORDER BY source;
            """)
            results = cursor.fetchall()

            # Extraire les counts par source
            historical_count = 0
            realtime_count = 0
            total_count = 0

            for source, count in results:
                total_count += count
                if source and 'historical' in source.lower():
                    historical_count = count
                elif source and ('ws' in source.lower() or 'realtime' in source.lower()):
                    realtime_count = count

            conn.close()

            # Afficher les stats
            logger.info(f"Current data: {total_count} candles total")
            if historical_count > 0:
                logger.info(f"  - Historical: {historical_count} candles")
            if realtime_count > 0:
                logger.info(f"  - Realtime: {realtime_count} candles")

            # Détecter si le backfill historique est terminé
            # Le backfill est terminé si les données historiques restent stables pendant 30 secondes
            # (les données temps réel peuvent continuer à arriver)
            if historical_count >= min_historical_candles:
                if historical_count == previous_historical_count:
                    stable_count += 1
                    logger.info(f"Historical data stable for {stable_count * 10} seconds...")
                    if stable_count >= 3:  # 3 checks * 10 secondes = 30 secondes stable
                        logger.info(f"✓ Historical backfill complete!")
                        logger.info(f"  - {historical_count} historical candles collected")
                        logger.info(f"  - {realtime_count} realtime candles")
                        logger.info("=" * 60)
                        return True
                else:
                    stable_count = 0  # Reset si les données historiques changent
                    delta = historical_count - previous_historical_count
                    logger.info(f"Backfill in progress... (+{delta} historical candles)")

                previous_historical_count = historical_count
            elif historical_count > 0:
                logger.info(f"Waiting for more data... ({historical_count}/{min_historical_candles} candles)")
                previous_historical_count = historical_count

            time.sleep(10)
            checks += 1

        except Exception as e:
            logger.warning(f"Error checking backfill status: {e}")
            time.sleep(10)
            checks += 1

    logger.warning(f"Timeout waiting for backfill after {max_wait_minutes} minutes")
    logger.warning("Starting scheduler anyway with available data")
    return False

def calculate_historical_indicators():
    """Calcule les indicateurs sur toutes les données historiques"""
    logger.info("=" * 60)
    logger.info("CALCULATING INDICATORS ON HISTORICAL DATA")
    logger.info("=" * 60)

    try:
        conn = psycopg2.connect(**DB_CONFIG)
        conn.autocommit = True
        cursor = conn.cursor()

        # Vérifier combien de candles existent
        cursor.execute("SELECT COUNT(*) FROM candles;")
        candles_count = cursor.fetchone()[0]
        logger.info(f"Found {candles_count} candles in database")

        # Calculer les indicateurs sur TOUTES les candles historiques (version optimisée)
        logger.info("Calling OPTIMIZED backfill procedure...")
        logger.info("This uses SQL window functions for fast batch processing...")
        cursor.execute("CALL backfill_historical_indicators_optimized('1m');")

        # Vérifier combien d'indicateurs ont été créés
        cursor.execute("SELECT COUNT(*) FROM indicators;")
        indicators_count = cursor.fetchone()[0]

        conn.close()

        logger.info(f"✓ Historical indicators calculated: {indicators_count} indicators created")
        logger.info("=" * 60)
        return True

    except Exception as e:
        logger.error(f"Error calculating historical indicators: {e}")
        return False

def main():
    logger.info("=" * 60)
    logger.info("CryptoViz Indicators Scheduler Starting")
    logger.info("=" * 60)

    # Attendre que la DB soit prête
    logger.info("Waiting for database to be ready...")
    max_retries = 30
    for i in range(max_retries):
        try:
            conn = psycopg2.connect(**DB_CONFIG)
            conn.close()
            logger.info("Database connection successful")
            break
        except Exception as e:
            if i == max_retries - 1:
                logger.error(f"Could not connect to database after {max_retries} attempts")
                return
            time.sleep(2)

    # Attendre les données historiques et lancer le calcul en arrière-plan
    if wait_for_historical_data():
        logger.info("Launching historical indicators calculation in background...")
        background_thread = threading.Thread(
            target=calculate_historical_indicators,
            daemon=True,
            name="HistoricalIndicatorsCalculator"
        )
        background_thread.start()
        logger.info("Background calculation started, proceeding with realtime scheduler...")
    else:
        logger.warning("Starting scheduler without historical indicators calculation")

    # Créer le scheduler
    scheduler = BlockingScheduler()

    # Ajouter les jobs
    scheduler.add_job(refresh_indicators_1s, 'interval', seconds=1, id='indicators_1s')
    scheduler.add_job(refresh_indicators_5s, 'interval', seconds=5, id='indicators_5s')
    scheduler.add_job(refresh_indicators_1m, 'interval', minutes=1, id='indicators_1m')
    scheduler.add_job(refresh_indicators_5m, 'interval', minutes=5, id='indicators_5m')
    scheduler.add_job(refresh_indicators_15m, 'interval', minutes=15, id='indicators_15m')
    scheduler.add_job(refresh_indicators_1h, 'interval', hours=1, id='indicators_1h')
    scheduler.add_job(refresh_indicators_1d, CronTrigger(hour=0, minute=0), id='indicators_1d')

    logger.info("Scheduled jobs:")
    logger.info("  - 1s:  every 1 second")
    logger.info("  - 5s:  every 5 seconds")
    logger.info("  - 1m:  every 1 minute")
    logger.info("  - 5m:  every 5 minutes")
    logger.info("  - 15m: every 15 minutes")
    logger.info("  - 1h:  every hour")
    logger.info("  - 1d:  every day at midnight")
    logger.info("=" * 60)

    # Démarrer le scheduler
    try:
        scheduler.start()
    except (KeyboardInterrupt, SystemExit):
        logger.info("Scheduler stopped")

if __name__ == '__main__':
    main()
