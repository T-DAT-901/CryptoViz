import os
import logging
from typing import List, Optional
from dataclasses import dataclass


@dataclass
class Config:
    """Configuration centralisée de l'application"""

    # === Filtrage des symboles ===
    quote_currencies: Optional[List[str]]  # Devises de cotation autorisées (ex: ['USDT', 'BUSD'])
    min_volume: Optional[float]            # Volume minimum 24h (ex: 5000000 = 5M$)
    max_symbols: Optional[int]             # Nombre maximum de symboles à collecter

    # === Features activées ===
    enable_trades: bool        # Collecter les trades (transactions)
    enable_ticker: bool        # Collecter les tickers (bid/ask/volumes)
    enable_aggregation: bool   # Activer l'agrégation en barres OHLCV

    # === Configuration Kafka ===
    kafka_bootstrap: str    # Adresse du broker Kafka
    topic_trades: str       # Topic pour les trades bruts
    topic_ticker: str       # Topic pour les tickers
    output_prefix: str      # Préfixe pour les topics agrégés (ex: "crypto.aggregated.")
    compression: Optional[str]  # Type de compression ('lz4', 'gzip', etc.)
    client_id: str          # Identifiant du client Kafka

    # === Configuration agrégation ===
    timeframes: List[str]   # Timeframes à générer (ex: ['5s', '1m', '15m', '1h'])
    grace_ms: int           # Délai d'attente avant de fermer une fenêtre (ms)
    sweep_sec: float        # Intervalle de nettoyage des fenêtres expirées (s)

    # === Configuration Binance ===
    api_key: Optional[str]     # Clé API Binance (optionnelle)
    api_secret: Optional[str]  # Secret API Binance (optionnel)
    default_type: str          # Type de marché ('spot', 'future', etc.)

    # === Performance ===
    max_concurrent: int    # Nombre max de connexions WebSocket simultanées


def load_config() -> Config:
    """Charge la configuration depuis les variables d'environnement"""

    # Parsing des devises de cotation
    quote_currencies = os.getenv("QUOTE_CURRENCIES", "").strip()
    quote_filter = None
    if quote_currencies:
        quote_filter = [q.strip().upper() for q in quote_currencies.split(",") if q.strip()]

    # Parsing du volume minimum
    min_volume = None
    if os.getenv("MIN_VOLUME"):
        try:
            min_volume = float(os.getenv("MIN_VOLUME"))
        except ValueError:
            logging.warning("MIN_VOLUME invalide, ignoré")

    # Parsing du nombre max de symboles
    max_symbols = None
    if os.getenv("MAX_SYMBOLS"):
        try:
            max_symbols = int(os.getenv("MAX_SYMBOLS"))
        except ValueError:
            logging.warning("MAX_SYMBOLS invalide, ignoré")

    return Config(
        # Filtrage
        quote_currencies=quote_filter,
        min_volume=min_volume,
        max_symbols=max_symbols,

        # Features
        enable_trades=os.getenv("ENABLE_TRADES", "true").lower() == "true",
        enable_ticker=os.getenv("ENABLE_TICKER", "false").lower() == "true",
        enable_aggregation=os.getenv("ENABLE_AGGREGATION", "true").lower() == "true",

        # Kafka
        kafka_bootstrap=os.getenv("KAFKA_BROKERS", "localhost:9092"),
        topic_trades=os.getenv("TOPIC_TRADES", "crypto.raw.trades"),
        topic_ticker=os.getenv("TOPIC_TICKER", "crypto.ticker"),
        output_prefix=os.getenv("OUTPUT_PREFIX", "crypto.aggregated."),
        compression=os.getenv("KAFKA_COMPRESSION", "lz4").lower() or None,
        client_id=os.getenv("KAFKA_CLIENT_ID", "collector-optimized"),

        # Agrégation
        timeframes=[t.strip() for t in os.getenv("TIMEFRAMES", "5s,1m,15m,1h").split(",") if t.strip()],
        grace_ms=int(os.getenv("GRACE_MS", "2000")),
        sweep_sec=float(os.getenv("SWEEP_SEC", "1.0")),

        # Binance
        api_key=os.getenv("BINANCE_API_KEY"),
        api_secret=os.getenv("BINANCE_SECRET_KEY"),
        default_type=os.getenv("BINANCE_DEFAULT_TYPE", "spot"),

        # Performance
        max_concurrent=int(os.getenv("MAX_CONCURRENT", "100")),
    )
