import orjson
import time
import datetime
import asyncio
import logging
from typing import Dict, Tuple, List
from aiokafka import AIOKafkaProducer

from config import Config

logger = logging.getLogger(__name__)


def create_bar_headers(exchange: str, symbol: str, timeframe: str, closed: bool):
    return [
        ("source", b"collector-ws"),
        ("type", b"bar"),
        ("exchange", exchange.encode()),
        ("symbol", symbol.replace("/", "").encode()),
        ("timeframe", timeframe.encode()),
        ("closed", b"true" if closed else b"false"),
        ("schema", b"bar_v2"),
    ]


def parse_timeframe(s: str) -> int:
    # Convertit un timeframe en millisecondes (ex: "5s" -> 5000)
    s = s.strip().lower()
    try:
        if s.endswith("ms"):
            return int(s[:-2])
        if s.endswith("s"):
            return int(s[:-1]) * 1000
        if s.endswith("m"):
            return int(s[:-1]) * 60_000
        if s.endswith("h"):
            return int(s[:-1]) * 3_600_000
        if s.endswith("d"):
            return int(s[:-1]) * 86_400_000
        raise ValueError(f"Format timeframe non reconnu: {s}")
    except ValueError as e:
        logger.error(f"Erreur parsing timeframe '{s}': {e}")
        return 60_000  # Défaut: 1 minute


def floor_ts_correct(ts_ms: int, tf_ms: int) -> int:
    # Arrondit un timestamp au début de sa fenêtre temporelle
    ts_sec = ts_ms // 1000
    tf_sec = tf_ms // 1000

    dt = datetime.datetime.fromtimestamp(ts_sec, tz=datetime.timezone.utc)

    # Cas courants optimisés
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
        # Fallback pour timeframes custom
        seconds_since_midnight = dt.hour * 3600 + dt.minute * 60 + dt.second
        aligned_seconds = (seconds_since_midnight // tf_sec) * tf_sec
        hours = aligned_seconds // 3600
        minutes = (aligned_seconds % 3600) // 60
        seconds = aligned_seconds % 60
        aligned = dt.replace(hour=hours, minute=minutes, second=seconds, microsecond=0)

    return int(aligned.timestamp() * 1000)


class OptimizedAggregator:

    def __init__(self, producer: AIOKafkaProducer, config: Config):
        self.producer = producer
        self.config = config

        # Conversion des timeframes en ms
        self.tf_ms = {lbl: parse_timeframe(lbl) for lbl in config.timeframes}

        # Fenêtres actives: (exchange, symbol, timeframe) -> barre
        self.windows: Dict[Tuple[str, str, str], dict] = {}

        self._stop = asyncio.Event()

        # Stats
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
        logger.info(f"Stats finales: {self.stats}")

    async def sweeper(self):
        # Boucle qui ferme les fenêtres expirées
        while not self._stop.is_set():
            await asyncio.sleep(self.config.sweep_sec)
            now_ms = int(time.time() * 1000)

            # Identifie les fenêtres à fermer
            to_close = []
            for key, bar in list(self.windows.items()):
                tf_ms = self.tf_ms[bar["timeframe"]]
                window_end = bar["window_start"] + tf_ms

                # Ferme si expirée + grace period écoulé
                if not bar["closed"] and now_ms >= window_end + self.config.grace_ms:
                    to_close.append(key)

            # Ferme et publie
            for key in to_close:
                bar = self.windows.get(key)
                if bar and not bar["closed"]:
                    await self._emit_bar(bar, closed=True)
                    self.stats['bars_closed'] += 1

    async def on_trade(self, exchange: str, symbol: str, ts: int, price: float, amount: float):
        # Reçoit un trade et met à jour toutes les fenêtres temporelles
        self.stats['trades_processed'] += 1

        try:
            # Update chaque timeframe
            for tf_label, tf_ms in self.tf_ms.items():
                # Calcule le début de fenêtre pour ce trade
                win_start = floor_ts_correct(ts, tf_ms)
                key = (exchange, symbol, tf_label)
                bar = self.windows.get(key)

                # Nouvelle fenêtre ou changement de période
                if (bar is None) or (bar["window_start"] != win_start):
                    # Ferme l'ancienne si elle existe
                    if bar is not None and not bar["closed"]:
                        await self._emit_bar(bar, closed=True)
                        self.stats['bars_closed'] += 1

                    # Crée une nouvelle fenêtre
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
                    # Update la fenêtre existante
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
        # Publie une barre dans Kafka
        try:
            # Prépare le message
            output_bar = dict(bar)
            output_bar["closed"] = closed
            output_bar["duration_ms"] = bar["last_trade_ts"] - bar["first_trade_ts"]

            # Topic basé sur le timeframe
            topic = f"{self.config.output_prefix}{bar['timeframe']}"

            # Clé pour partitionnement
            key = f"{bar['exchange']}|{bar['symbol'].replace('/','')}|{bar['timeframe']}|{bar['window_start']}".encode()

            # Headers standardisés
            headers = create_bar_headers(bar["exchange"], bar["symbol"], bar["timeframe"], closed)

            # Publie dans Kafka
            await self.producer.send_and_wait(topic, orjson.dumps(output_bar), key=key, headers=headers)

            if closed:
                bar["closed"] = True

        except Exception as e:
            self.stats['errors'] += 1
            logger.error(f"Erreur émission barre {bar.get('symbol', 'unknown')}: {e}")
