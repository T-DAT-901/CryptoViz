-- =============================================================================
-- CryptoViz TimescaleDB Initialization Script
-- =============================================================================

-- Activer l'extension TimescaleDB
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =============================================================================
-- NON-TIME-SERIES TABLES (Regular PostgreSQL tables)
-- =============================================================================

-- Table pour les devises (crypto et fiat) - créée en premier pour la référence
CREATE TABLE IF NOT EXISTS currencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100),
    symbol VARCHAR(20),
    type VARCHAR(10) NOT NULL, -- 'crypto' or 'fiat'
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT chk_currency_type CHECK (type IN ('crypto', 'fiat'))
);

-- Table pour les utilisateurs
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    currency_id UUID REFERENCES currencies(id) NOT NULL,
    username VARCHAR(255),
    password_encrypted VARCHAR(255),
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    email VARCHAR(255),
    phone VARCHAR(50),
    address TEXT,
    zipcode INTEGER,
    city VARCHAR(100),
    country VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- TIME-SERIES TABLES (TimescaleDB Hypertables)
-- =============================================================================

-- Table pour les trades individuels (haute fréquence)
CREATE TABLE IF NOT EXISTS trades (
    event_ts TIMESTAMPTZ NOT NULL,
    exchange VARCHAR(20) NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    trade_id VARCHAR(50) NOT NULL,
    price DECIMAL(20,8) NOT NULL,
    amount DECIMAL(20,8) NOT NULL,
    side VARCHAR(4) NOT NULL, -- 'buy' or 'sell'
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT chk_side CHECK (side IN ('buy', 'sell'))
);

-- Créer l'hypertable pour les trades
SELECT create_hypertable('trades', 'event_ts',
    partitioning_column => 'exchange',
    number_partitions => 10,
    if_not_exists => TRUE
);

-- Table pour les bougies OHLCV (candles)
CREATE TABLE IF NOT EXISTS candles (
    window_start TIMESTAMPTZ NOT NULL,
    window_end TIMESTAMPTZ NOT NULL,
    exchange VARCHAR(20) NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    timeframe VARCHAR(5) NOT NULL, -- '1m', '5m', '15m', '1h', '1d'
    open DECIMAL(20,8) NOT NULL,
    high DECIMAL(20,8) NOT NULL,
    low DECIMAL(20,8) NOT NULL,
    close DECIMAL(20,8) NOT NULL,
    volume DECIMAL(20,8) NOT NULL,
    trade_count INTEGER,
    closed BOOLEAN NOT NULL,
    first_trade_ts TIMESTAMPTZ,
    last_trade_ts TIMESTAMPTZ,
    duration_ms BIGINT,
    source VARCHAR(20), -- 'collector-ws' or 'collector-historical'
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Créer l'hypertable avec partitioning par symbole
SELECT create_hypertable('candles', 'window_start',
    partitioning_column => 'symbol',
    number_partitions => 50,
    if_not_exists => TRUE
);

-- Table pour les indicateurs techniques
CREATE TABLE IF NOT EXISTS indicators (
    time TIMESTAMPTZ NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    timeframe VARCHAR(5) NOT NULL,
    indicator_type VARCHAR(20) NOT NULL, -- 'rsi', 'macd', 'bollinger', 'momentum'
    value DECIMAL(20,8),
    value_signal DECIMAL(20,8), -- Pour MACD signal
    value_histogram DECIMAL(20,8), -- Pour MACD histogram
    upper_band DECIMAL(20,8), -- Pour Bollinger upper
    lower_band DECIMAL(20,8), -- Pour Bollinger lower
    middle_band DECIMAL(20,8), -- Pour Bollinger middle
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Créer l'hypertable pour les indicateurs
SELECT create_hypertable('indicators', 'time',
    partitioning_column => 'symbol',
    number_partitions => 20,
    if_not_exists => TRUE
);

-- Table pour les actualités crypto
CREATE TABLE IF NOT EXISTS news (
    time TIMESTAMPTZ NOT NULL,
    title TEXT NOT NULL,
    content TEXT,
    source VARCHAR(100) NOT NULL,
    url TEXT NOT NULL,
    sentiment_score DECIMAL(5,2), -- Score de sentiment entre -1 et 1
    symbols JSONB, -- Array des symboles mentionnés: ["BTC/USDT", "ETH/USDT"]
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (time, source, url)
);

-- Créer l'hypertable pour les news
SELECT create_hypertable('news', 'time',
    if_not_exists => TRUE
);

-- =============================================================================
-- COMPRESSION / COLUMNSTORE (requise avant les politiques)
-- =============================================================================

ALTER TABLE trades SET (
    timescaledb.compress = TRUE,
    timescaledb.compress_orderby = 'event_ts',
    timescaledb.compress_segmentby = 'exchange'
);

ALTER TABLE candles SET (
    timescaledb.compress = TRUE,
    timescaledb.compress_orderby = 'window_start',
    timescaledb.compress_segmentby = 'symbol'
);

ALTER TABLE indicators SET (
    timescaledb.compress = TRUE,
    timescaledb.compress_orderby = 'time',
    timescaledb.compress_segmentby = 'symbol'
);

ALTER TABLE news SET (
    timescaledb.compress = TRUE,
    timescaledb.compress_orderby = 'time',
    timescaledb.compress_segmentby = 'source'
);

-- =============================================================================
-- INDEX OPTIMISÉS
-- =============================================================================

-- Index pour trades
CREATE INDEX IF NOT EXISTS idx_trades_exchange_symbol_time
    ON trades (exchange, symbol, event_ts DESC);

CREATE INDEX IF NOT EXISTS idx_trades_symbol_time
    ON trades (symbol, event_ts DESC);

CREATE INDEX IF NOT EXISTS idx_trades_time
    ON trades (event_ts DESC);

-- Index pour candles
CREATE INDEX IF NOT EXISTS idx_candles_exchange_symbol_timeframe_window
    ON candles (exchange, symbol, timeframe, window_start DESC);

CREATE INDEX IF NOT EXISTS idx_candles_symbol_timeframe_window
    ON candles (symbol, timeframe, window_start DESC);

CREATE INDEX IF NOT EXISTS idx_candles_timeframe_window
    ON candles (timeframe, window_start DESC);

CREATE INDEX IF NOT EXISTS idx_candles_window_start
    ON candles (window_start DESC);

-- Index pour indicators
CREATE INDEX IF NOT EXISTS idx_indicators_symbol_time
    ON indicators (symbol, time DESC);

CREATE INDEX IF NOT EXISTS idx_indicators_type_time
    ON indicators (indicator_type, time DESC);

CREATE INDEX IF NOT EXISTS idx_indicators_symbol_type_timeframe_time
    ON indicators (symbol, indicator_type, timeframe, time DESC);

-- Index pour news
CREATE INDEX IF NOT EXISTS idx_news_time
    ON news (time DESC);

CREATE INDEX IF NOT EXISTS idx_news_symbols
    ON news USING GIN (symbols);

CREATE INDEX IF NOT EXISTS idx_news_source_time
    ON news (source, time DESC);

-- =============================================================================
-- POLITIQUES DE COMPRESSION
-- =============================================================================

-- Compression pour trades après 7 jours (haute fréquence)
SELECT add_compression_policy('trades', INTERVAL '7 days', if_not_exists => TRUE);

-- Compression pour candles après 7 jours
SELECT add_compression_policy('candles', INTERVAL '7 days', if_not_exists => TRUE);

-- Compression pour indicators après 7 jours
SELECT add_compression_policy('indicators', INTERVAL '7 days', if_not_exists => TRUE);

-- Compression pour news après 30 jours
SELECT add_compression_policy('news', INTERVAL '30 days', if_not_exists => TRUE);

-- =============================================================================
-- POLITIQUES DE RÉTENTION
-- =============================================================================

-- Rétention pour trades (24 heures pour haute fréquence)
SELECT add_retention_policy('trades', INTERVAL '24 hours', if_not_exists => TRUE);

-- Rétention pour candles selon le timeframe
-- Note: Candles 5s gardées 7 jours, 1m gardées 30 jours, etc.
SELECT add_retention_policy('candles', INTERVAL '2 years', if_not_exists => TRUE);

-- Rétention pour les indicateurs (1 an)
SELECT add_retention_policy('indicators', INTERVAL '1 year', if_not_exists => TRUE);

-- Rétention pour les news (6 mois)
SELECT add_retention_policy('news', INTERVAL '6 months', if_not_exists => TRUE);

-- =============================================================================
-- CONTINUOUS AGGREGATES (TimescaleDB 2.x+)
-- =============================================================================
-- Continuous aggregates provide real-time incremental updates
-- Unlike regular materialized views, they refresh automatically and efficiently

-- Continuous aggregate for hourly OHLCV data
-- Aggregates 1m, 5m, and 15m candles into hourly buckets
CREATE MATERIALIZED VIEW IF NOT EXISTS candles_hourly_summary
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', window_start) AS hour,
    exchange,
    symbol,
    first(open, window_start) AS open,
    max(high) AS high,
    min(low) AS low,
    last(close, window_start) AS close,
    sum(volume) AS volume,
    sum(trade_count) AS trades_count
FROM candles
WHERE timeframe IN ('1m', '5m', '15m')
GROUP BY hour, exchange, symbol;

-- Index pour la continuous aggregate
CREATE INDEX IF NOT EXISTS idx_candles_hourly_summary_symbol_hour
    ON candles_hourly_summary (symbol, hour DESC);

-- Add automatic refresh policy for continuous aggregate
-- Refreshes data from 3 hours ago to 1 hour ago, every hour
-- This ensures near real-time updates while avoiding refreshing very recent data
SELECT add_continuous_aggregate_policy('candles_hourly_summary',
    start_offset => INTERVAL '3 hours',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour',
    if_not_exists => TRUE);

-- Continuous aggregate for latest indicators per symbol
-- Uses time bucketing and last() to get most recent indicator values
CREATE MATERIALIZED VIEW IF NOT EXISTS indicators_latest
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 minute', time) AS bucket,
    symbol,
    indicator_type,
    timeframe,
    last(value, time) AS value,
    last(value_signal, time) AS value_signal,
    last(value_histogram, time) AS value_histogram,
    last(upper_band, time) AS upper_band,
    last(lower_band, time) AS lower_band,
    last(middle_band, time) AS middle_band
FROM indicators
GROUP BY bucket, symbol, indicator_type, timeframe;

-- Add automatic refresh policy for indicators
-- Refreshes every 5 minutes for near real-time indicator updates
SELECT add_continuous_aggregate_policy('indicators_latest',
    start_offset => INTERVAL '1 hour',
    end_offset => INTERVAL '1 minute',
    schedule_interval => INTERVAL '5 minutes',
    if_not_exists => TRUE);

-- =============================================================================
-- FONCTIONS UTILITAIRES
-- =============================================================================

-- Fonction pour nettoyer les anciennes données selon le timeframe
CREATE OR REPLACE FUNCTION cleanup_old_data()
RETURNS void AS $$
BEGIN
    -- Les trades sont automatiquement supprimés par la retention policy (24h)
    -- Note: Hot/cold tiering is now configured in setup-tiering.sql with .env variables
    -- This function provides additional cleanup for total retention limits

    -- Supprimer les candles 1m plus anciennes que 37 jours (7 hot + 30 cold)
    DELETE FROM candles
    WHERE timeframe = '1m'
    AND window_start < NOW() - INTERVAL '37 days';

    -- Supprimer les candles 5m plus anciennes que 104 jours (14 hot + 90 cold)
    DELETE FROM candles
    WHERE timeframe = '5m'
    AND window_start < NOW() - INTERVAL '104 days';

    -- Supprimer les candles 15m plus anciennes que 210 jours (30 hot + 180 cold)
    DELETE FROM candles
    WHERE timeframe = '15m'
    AND window_start < NOW() - INTERVAL '210 days';

    -- Supprimer les candles 1h plus anciennes que 820 jours (90 hot + 730 cold)
    DELETE FROM candles
    WHERE timeframe = '1h'
    AND window_start < NOW() - INTERVAL '820 days';

    -- Les candles 1d sont conservées indéfiniment (low volume, high value)

    -- Supprimer les anciennes données des intervalles obsolètes (1s, 5s, 4h)
    DELETE FROM candles WHERE timeframe IN ('1s', '5s', '4h');

    RAISE NOTICE 'Cleanup completed';
END;
$$ LANGUAGE plpgsql;

-- Fonction pour calculer les statistiques de performance
CREATE OR REPLACE FUNCTION get_crypto_stats(p_symbol VARCHAR, p_timeframe VARCHAR, p_exchange VARCHAR DEFAULT 'BINANCE')
RETURNS TABLE (
    symbol VARCHAR,
    timeframe VARCHAR,
    latest_price DECIMAL,
    price_change_24h DECIMAL,
    price_change_pct_24h DECIMAL,
    volume_24h DECIMAL,
    high_24h DECIMAL,
    low_24h DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        p_symbol::VARCHAR,
        p_timeframe::VARCHAR,
        (SELECT close FROM candles
         WHERE symbol = p_symbol AND timeframe = p_timeframe AND exchange = p_exchange
         ORDER BY window_start DESC LIMIT 1) AS latest_price,

        (SELECT close FROM candles
         WHERE symbol = p_symbol AND timeframe = p_timeframe AND exchange = p_exchange
         ORDER BY window_start DESC LIMIT 1) -
        (SELECT close FROM candles
         WHERE symbol = p_symbol AND timeframe = p_timeframe AND exchange = p_exchange
         AND window_start <= NOW() - INTERVAL '24 hours'
         ORDER BY window_start DESC LIMIT 1) AS price_change_24h,

        ((SELECT close FROM candles
          WHERE symbol = p_symbol AND timeframe = p_timeframe AND exchange = p_exchange
          ORDER BY window_start DESC LIMIT 1) -
         (SELECT close FROM candles
          WHERE symbol = p_symbol AND timeframe = p_timeframe AND exchange = p_exchange
          AND window_start <= NOW() - INTERVAL '24 hours'
          ORDER BY window_start DESC LIMIT 1)) * 100.0 /
        (SELECT close FROM candles
         WHERE symbol = p_symbol AND timeframe = p_timeframe AND exchange = p_exchange
         AND window_start <= NOW() - INTERVAL '24 hours'
         ORDER BY window_start DESC LIMIT 1) AS price_change_pct_24h,

        (SELECT sum(volume) FROM candles
         WHERE symbol = p_symbol AND timeframe = p_timeframe AND exchange = p_exchange
         AND window_start >= NOW() - INTERVAL '24 hours') AS volume_24h,

        (SELECT max(high) FROM candles
         WHERE symbol = p_symbol AND timeframe = p_timeframe AND exchange = p_exchange
         AND window_start >= NOW() - INTERVAL '24 hours') AS high_24h,

        (SELECT min(low) FROM candles
         WHERE symbol = p_symbol AND timeframe = p_timeframe AND exchange = p_exchange
         AND window_start >= NOW() - INTERVAL '24 hours') AS low_24h;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- JOBS AUTOMATIQUES
-- =============================================================================

-- Note: Continuous aggregates (candles_hourly_summary, indicators_latest)
-- refresh automatically via their built-in policies. No manual refresh needed!

-- Job pour nettoyer les anciennes données toutes les heures
SELECT add_job('cleanup_old_data', '1 hour', if_not_exists => TRUE);

-- =============================================================================
-- DONNÉES DE TEST (OPTIONNEL)
-- =============================================================================

-- Insérer quelques devises de test
INSERT INTO currencies (name, symbol, type) VALUES
    ('Bitcoin', 'BTC', 'crypto'),
    ('Ethereum', 'ETH', 'crypto'),
    ('Tether', 'USDT', 'crypto'),
    ('US Dollar', 'USD', 'fiat'),
    ('Euro', 'EUR', 'fiat')
ON CONFLICT DO NOTHING;

-- Insérer quelques trades de test
INSERT INTO trades (event_ts, exchange, symbol, trade_id, price, amount, side) VALUES
    (NOW() - INTERVAL '1 minute', 'BINANCE', 'BTC/USDT', 'T123456', 45000.00, 0.025, 'buy'),
    (NOW() - INTERVAL '30 seconds', 'BINANCE', 'BTC/USDT', 'T123457', 45050.00, 0.015, 'sell'),
    (NOW() - INTERVAL '45 seconds', 'BINANCE', 'ETH/USDT', 'T123458', 3000.00, 1.5, 'buy')
ON CONFLICT DO NOTHING;

-- Insérer quelques candles de test
INSERT INTO candles (window_start, window_end, exchange, symbol, timeframe, open, high, low, close, volume, trade_count, closed) VALUES
    (NOW() - INTERVAL '1 minute', NOW(), 'BINANCE', 'BTC/USDT', '1m', 45000.00, 45100.00, 44900.00, 45050.00, 1.5, 152, true),
    (NOW() - INTERVAL '2 minutes', NOW() - INTERVAL '1 minute', 'BINANCE', 'BTC/USDT', '1m', 44950.00, 45000.00, 44800.00, 45000.00, 2.1, 198, true),
    (NOW() - INTERVAL '3 minutes', NOW() - INTERVAL '2 minutes', 'BINANCE', 'BTC/USDT', '1m', 44800.00, 44950.00, 44700.00, 44950.00, 1.8, 175, true),
    (NOW() - INTERVAL '1 minute', NOW(), 'BINANCE', 'ETH/USDT', '1m', 3000.00, 3010.00, 2990.00, 3005.00, 5.2, 89, true),
    (NOW() - INTERVAL '2 minutes', NOW() - INTERVAL '1 minute', 'BINANCE', 'ETH/USDT', '1m', 2995.00, 3000.00, 2985.00, 3000.00, 4.8, 76, true)
ON CONFLICT DO NOTHING;

-- =============================================================================
-- PERMISSIONS ET SÉCURITÉ
-- =============================================================================

-- Créer un utilisateur pour l'application
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'cryptoviz_app') THEN
        CREATE ROLE cryptoviz_app WITH LOGIN PASSWORD 'cryptoviz_secure_password';
    END IF;
END
$$;

-- Accorder les permissions nécessaires
GRANT USAGE ON SCHEMA public TO cryptoviz_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO cryptoviz_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO cryptoviz_app;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO cryptoviz_app;

-- =============================================================================
-- FINALISATION
-- =============================================================================

-- Analyser les tables pour optimiser les statistiques
ANALYZE users;
ANALYZE currencies;
ANALYZE trades;
ANALYZE candles;
ANALYZE indicators;
ANALYZE news;

-- Afficher un résumé de l'initialisation
DO $$
BEGIN
    RAISE NOTICE '=============================================================================';
    RAISE NOTICE 'CryptoViz Database Initialization Complete';
    RAISE NOTICE '=============================================================================';
    RAISE NOTICE 'Regular tables created: users, currencies';
    RAISE NOTICE 'Hypertables created: trades, candles, indicators, news';
    RAISE NOTICE 'Hypertables configured with time-based partitioning';
    RAISE NOTICE 'Compression and retention policies applied';
    RAISE NOTICE 'Materialized views created for performance';
    RAISE NOTICE 'Utility functions and automated jobs configured';
    RAISE NOTICE 'Test data inserted';
    RAISE NOTICE '=============================================================================';
    RAISE NOTICE 'Schema aligned with Kafka message structure:';
    RAISE NOTICE '  - trades: crypto.raw.trades';
    RAISE NOTICE '  - candles: crypto.aggregated.*';
    RAISE NOTICE '  - indicators: crypto.indicators.*';
    RAISE NOTICE '  - news: crypto.news';
    RAISE NOTICE '=============================================================================';
END
$$;
