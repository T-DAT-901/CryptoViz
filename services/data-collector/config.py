import os
import logging
from typing import List, Optional
from dataclasses import dataclass


@dataclass
class Config:
    # Filtrage symboles
    quote_currencies: Optional[List[str]]
    min_volume: Optional[float]
    max_symbols: Optional[int]

    # Features
    enable_trades: bool
    enable_ticker: bool
    enable_aggregation: bool

    # Kafka
    kafka_bootstrap: str
    topic_trades: str
    topic_ticker: str
    output_prefix: str
    compression: Optional[str]
    client_id: str

    # Agrégation
    timeframes: List[str]
    grace_ms: int
    sweep_sec: float

    # Binance
    api_key: Optional[str]
    api_secret: Optional[str]
    default_type: str

    # Perf
    max_concurrent: int


def is_valid_timeframe(tf: str) -> bool:
    # Vérifie qu'un timeframe a le bon format (5s, 1m, 1h, etc.)
    tf = tf.strip().lower()
    return tf and (tf.endswith('ms') or tf.endswith('s') or tf.endswith('m') or
                   tf.endswith('h') or tf.endswith('d'))


def load_config() -> Config:
    # Parse devises de cotation
    quote_currencies = os.getenv("QUOTE_CURRENCIES", "").strip()
    quote_filter = None
    if quote_currencies:
        quote_filter = [q.strip().upper() for q in quote_currencies.split(",") if q.strip()]

    # Parse volume min
    min_volume = None
    if os.getenv("MIN_VOLUME"):
        try:
            min_volume = float(os.getenv("MIN_VOLUME"))
        except ValueError:
            logging.warning("Invalid MIN_VOLUME, ignoring")

    # Parse nb max symboles
    max_symbols = None
    if os.getenv("MAX_SYMBOLS"):
        try:
            max_symbols = int(os.getenv("MAX_SYMBOLS"))
        except ValueError:
            logging.warning("Invalid MAX_SYMBOLS, ignoring")

    # Parse et valide les timeframes
    timeframes_str = os.getenv("TIMEFRAMES", "1m,5m,15m,1h,1d")
    timeframes = []
    for tf in timeframes_str.split(","):
        tf = tf.strip()
        if tf and is_valid_timeframe(tf):
            timeframes.append(tf)
        elif tf:
            logging.warning(f"Invalid timeframe ignored: {tf}")

    return Config(
        quote_currencies=quote_filter,
        min_volume=min_volume,
        max_symbols=max_symbols,
        enable_trades=os.getenv("ENABLE_TRADES", "true").lower() == "true",
        enable_ticker=os.getenv("ENABLE_TICKER", "false").lower() == "true",
        enable_aggregation=os.getenv("ENABLE_AGGREGATION", "true").lower() == "true",
        kafka_bootstrap=os.getenv("KAFKA_BROKERS", "localhost:9092"),
        topic_trades=os.getenv("TOPIC_TRADES", "crypto.raw.trades"),
        topic_ticker=os.getenv("TOPIC_TICKER", "crypto.ticker"),
        output_prefix=os.getenv("OUTPUT_PREFIX", "crypto.aggregated."),
        compression=os.getenv("KAFKA_COMPRESSION", "lz4").lower() or None,
        client_id=os.getenv("KAFKA_CLIENT_ID", "collector-optimized"),
        timeframes=timeframes,
        grace_ms=int(os.getenv("GRACE_MS", "2000")),
        sweep_sec=float(os.getenv("SWEEP_SEC", "1.0")),
        api_key=os.getenv("BINANCE_API_KEY"),
        api_secret=os.getenv("BINANCE_SECRET_KEY"),
        default_type=os.getenv("BINANCE_DEFAULT_TYPE", "spot"),
        max_concurrent=int(os.getenv("MAX_CONCURRENT", "100")),
    )
