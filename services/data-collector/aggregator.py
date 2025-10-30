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
    """
    Convertit une chaîne de timeframe en millisecondes
    Exemples: '5s' -> 5000, '1m' -> 60000, '1h' -> 3600000
    """
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
        return 60_000  # Valeur par défaut: 1 minute


def floor_ts_correct(ts_ms: int, tf_ms: int) -> int:
    """
    Aligne un timestamp sur le début de sa fenêtre temporelle
    Exemples:
      - 14:32:47 avec timeframe 1m -> 14:32:00
      - 14:32:47 avec timeframe 5m -> 14:30:00
    """
    ts_sec = ts_ms // 1000
    tf_sec = tf_ms // 1000

    dt = datetime.datetime.fromtimestamp(ts_sec, tz=datetime.timezone.utc)

    # Cas spécifiques pour les timeframes courants
    if tf_sec == 1:  # 1 seconde
        aligned = dt.replace(microsecond=0)
    elif tf_sec == 5:  # 5 secondes
        aligned = dt.replace(second=(dt.second // 5) * 5, microsecond=0)
    elif tf_sec == 30:  # 30 secondes
        aligned = dt.replace(second=(dt.second // 30) * 30, microsecond=0)
    elif tf_sec == 60:  # 1 minute
        aligned = dt.replace(second=0, microsecond=0)
    elif tf_sec == 300:  # 5 minutes
        aligned = dt.replace(minute=(dt.minute // 5) * 5, second=0, microsecond=0)
    elif tf_sec == 900:  # 15 minutes
        aligned = dt.replace(minute=(dt.minute // 15) * 15, second=0, microsecond=0)
    elif tf_sec == 3600:  # 1 heure
        aligned = dt.replace(minute=0, second=0, microsecond=0)
    elif tf_sec == 14400:  # 4 heures
        aligned = dt.replace(hour=(dt.hour // 4) * 4, minute=0, second=0, microsecond=0)
    elif tf_sec == 86400:  # 1 jour
        aligned = dt.replace(hour=0, minute=0, second=0, microsecond=0)
    else:
        # Fallback pour les timeframes personnalisés
        seconds_since_midnight = dt.hour * 3600 + dt.minute * 60 + dt.second
        aligned_seconds = (seconds_since_midnight // tf_sec) * tf_sec
        hours = aligned_seconds // 3600
        minutes = (aligned_seconds % 3600) // 60
        seconds = aligned_seconds % 60
        aligned = dt.replace(hour=hours, minute=minutes, second=seconds, microsecond=0)

    return int(aligned.timestamp() * 1000)


class OptimizedAggregator:
    """
    Agrégateur de trades en barres OHLCV

    Principe:
    - Reçoit les trades en temps réel
    - Les agrège dans des fenêtres temporelles (5s, 1m, 15m, 1h)
    - Publie les barres complètes dans Kafka
    """

    def __init__(self, producer: AIOKafkaProducer, config: Config):
        self.producer = producer
        self.config = config

        # Conversion des timeframes en millisecondes
        self.tf_ms = {lbl: parse_timeframe(lbl) for lbl in config.timeframes}

        # Stockage des fenêtres en cours: (exchange, symbol, timeframe) -> barre
        self.windows: Dict[Tuple[str, str, str], dict] = {}

        self._stop = asyncio.Event()

        # Statistiques de fonctionnement
        self.stats = {
            'bars_created': 0,      # Nouvelles barres créées
            'bars_updated': 0,      # Barres mises à jour
            'bars_closed': 0,       # Barres finalisées et publiées
            'trades_processed': 0,  # Nombre total de trades traités
            'errors': 0             # Erreurs rencontrées
        }

        logger.info(f"Agrégateur initialisé avec timeframes: {config.timeframes}")

    async def stop(self):
        """Arrête l'agrégateur et affiche les statistiques finales"""
        self._stop.set()
        logger.info(f"Stats finales: {self.stats}")

    async def sweeper(self):
        """
        Tâche de nettoyage périodique
        Ferme les fenêtres expirées qui n'ont pas reçu de nouveaux trades
        """
        while not self._stop.is_set():
            await asyncio.sleep(self.config.sweep_sec)
            now_ms = int(time.time() * 1000)

            # Identifier les fenêtres à fermer
            to_close = []
            for key, bar in list(self.windows.items()):
                tf_ms = self.tf_ms[bar["timeframe"]]
                window_end = bar["window_start"] + tf_ms

                # Fermer si la fenêtre est expirée + délai de grâce écoulé
                if not bar["closed"] and now_ms >= window_end + self.config.grace_ms:
                    to_close.append(key)

            # Fermer et publier les fenêtres expirées
            for key in to_close:
                bar = self.windows.get(key)
                if bar and not bar["closed"]:
                    await self._emit_bar(bar, closed=True)
                    self.stats['bars_closed'] += 1

    async def on_trade(self, exchange: str, symbol: str, ts: int, price: float, amount: float):
        """
        Traite un nouveau trade et met à jour toutes les fenêtres temporelles

        Args:
            exchange: Nom de l'exchange (ex: BINANCE)
            symbol: Paire de trading (ex: BTC/USDT)
            ts: Timestamp du trade en millisecondes
            price: Prix du trade
            amount: Quantité échangée
        """
        self.stats['trades_processed'] += 1

        try:
            # Mettre à jour chaque timeframe
            for tf_label, tf_ms in self.tf_ms.items():
                # Calculer le début de la fenêtre pour ce trade
                win_start = floor_ts_correct(ts, tf_ms)
                key = (exchange, symbol, tf_label)
                bar = self.windows.get(key)

                # Nouvelle fenêtre ou changement de période
                if (bar is None) or (bar["window_start"] != win_start):
                    # Fermer l'ancienne fenêtre si elle existe
                    if bar is not None and not bar["closed"]:
                        await self._emit_bar(bar, closed=True)
                        self.stats['bars_closed'] += 1

                    # Créer une nouvelle fenêtre
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
                    # Mettre à jour la fenêtre existante
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
        """
        Publie une barre dans Kafka

        Args:
            bar: Dictionnaire contenant les données OHLCV
            closed: True si la barre est finalisée, False pour mise à jour intermédiaire
        """
        try:
            # Préparer le message
            output_bar = dict(bar)
            output_bar["closed"] = closed
            output_bar["duration_ms"] = bar["last_trade_ts"] - bar["first_trade_ts"]

            # Topic Kafka basé sur le timeframe
            topic = f"{self.config.output_prefix}{bar['timeframe']}"

            # Clé pour le partitionnement Kafka
            key = f"{bar['exchange']}|{bar['symbol'].replace('/','')}|{bar['timeframe']}|{bar['window_start']}".encode()

            # Headers pour métadonnées
            headers = [
                ("source", b"collector-ws"),
                ("type", b"bar"),
                ("exchange", bar["exchange"].encode()),
                ("symbol", bar["symbol"].replace("/", "").encode()),
                ("timeframe", bar["timeframe"].encode()),
                ("closed", b"true" if closed else b"false"),
                ("schema", b"bar_v2"),
            ]

            # Publier dans Kafka
            await self.producer.send_and_wait(topic, orjson.dumps(output_bar), key=key, headers=headers)

            if closed:
                bar["closed"] = True

        except Exception as e:
            self.stats['errors'] += 1
            logger.error(f"Erreur émission barre {bar.get('symbol', 'unknown')}: {e}")
