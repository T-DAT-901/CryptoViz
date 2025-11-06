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
    # Suit la progression du backfill pour pouvoir reprendre en cas d'interruption

    def __init__(self, state_file: str = "backfill_state.json"):
        self.state_file = Path(state_file)
        self.state: Dict[str, Dict[str, int]] = {}
        self._load_state()

    def _load_state(self):
        if self.state_file.exists():
            try:
                with open(self.state_file, 'r') as f:
                    self.state = json.load(f)
                logger.info(f"État chargé: {len(self.state)} symboles déjà en cours")
            except Exception as e:
                logger.warning(f"Impossible de charger l'état: {e}")
                self.state = {}
        else:
            self.state = {}

    def _save_state(self):
        try:
            with open(self.state_file, 'w') as f:
                json.dump(self.state, f, indent=2)
        except Exception as e:
            logger.error(f"Impossible de sauvegarder l'état: {e}")

    def get_last_timestamp(self, symbol: str, timeframe: str) -> Optional[int]:
        return self.state.get(symbol, {}).get(timeframe)

    def update_progress(self, symbol: str, timeframe: str, timestamp: int):
        if symbol not in self.state:
            self.state[symbol] = {}
        self.state[symbol][timeframe] = timestamp
        self._save_state()

    def is_complete(self, symbol: str, timeframe: str, target_timestamp: int) -> bool:
        last = self.get_last_timestamp(symbol, timeframe)
        return last is not None and last >= target_timestamp


class HistoricalCollector:
    # Collecteur de données historiques via REST API (batch de 1000 barres)

    def __init__(self, config):
        self.config = config
        self.exchange = None
        self.producer = None
        self.tracker = ProgressTracker(config.backfill_state_file)

        # Rate limiting pour pas se faire ban
        self.rate_limiter = asyncio.Semaphore(20)
        self.request_delay = 0.05

    async def setup(self):
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
        logger.info("Producer Kafka démarré")

    async def cleanup(self):
        if self.producer:
            await self.producer.stop()
        if self.exchange:
            await self.exchange.close()

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
                logger.error(f"Erreur fetch {symbol} {timeframe}: {e}")
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
        if self.tracker.is_complete(symbol, timeframe, end_ts):
            logger.info(f"✓ {symbol} {timeframe} déjà complet")
            return

        # Reprend depuis la dernière position
        current_ts = self.tracker.get_last_timestamp(symbol, timeframe) or start_ts

        topic = f"{self.config.output_prefix}{timeframe}"
        total_bars = 0

        logger.info(f"Démarrage backfill {symbol} {timeframe}: {datetime.fromtimestamp(current_ts/1000)} -> {end_date}")

        while current_ts < end_ts:
            # Récup un batch de 1000 barres
            ohlcv = await self.fetch_ohlcv_batch(symbol, timeframe, current_ts, limit=1000)

            if not ohlcv:
                logger.warning(f"Aucune donnée pour {symbol} {timeframe} depuis {current_ts}")
                break

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
                current_ts = timestamp

            # Update la progression
            self.tracker.update_progress(symbol, timeframe, current_ts)

            # Si moins de 1000 barres, on a fini
            if len(ohlcv) < 1000:
                break

            # Avance au timestamp suivant
            current_ts = ohlcv[-1][0] + 1

        logger.info(f"✓ {symbol} {timeframe} terminé: {total_bars} barres collectées")

    async def backfill_all(self, symbols: List[str], timeframes: List[str], lookback_days: int = 365):
        # Lance le backfill pour tous les symboles et timeframes
        end_date = datetime.utcnow() - timedelta(hours=1)
        start_date = end_date - timedelta(days=lookback_days)

        logger.info(f"=== Démarrage Backfill Historique ===")
        logger.info(f"Période: {start_date} -> {end_date}")
        logger.info(f"Symboles: {len(symbols)}")
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
        logger.info(f"Lancement de {len(tasks)} tâches de backfill...")
        await asyncio.gather(*tasks, return_exceptions=True)

        logger.info("=== Backfill Terminé ===")


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
            logger.warning(f"Timeframe {tf} ignoré (REST API supporte pas les secondes)")

    logger.info(f"Configuration backfill: {lookback_days} jours, timeframes: {backfill_timeframes}")

    collector = HistoricalCollector(config)

    try:
        await collector.setup()

        # Récup la liste des symboles (même logique que temps réel)
        from binance_client import BinanceClient
        client = BinanceClient(config)
        symbols = await client.get_all_symbols()
        await client.close()

        if not symbols:
            logger.error("Aucun symbole trouvé")
            return

        logger.info(f"Symboles à backfill: {symbols[:10]}... ({len(symbols)} total)")

        # Lance le backfill
        await collector.backfill_all(symbols, backfill_timeframes, lookback_days)

    except Exception as e:
        logger.error(f"Erreur lors du backfill: {e}", exc_info=True)
    finally:
        await collector.cleanup()


if __name__ == "__main__":
    asyncio.run(main())
