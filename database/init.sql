-- =============================================================================
-- CryptoViz TimescaleDB Initialization Script
-- =============================================================================

-- Activer l'extension TimescaleDB
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- =============================================================================
-- TABLES PRINCIPALES
-- =============================================================================

-- Table pour les données crypto brutes et agrégées
CREATE TABLE IF NOT EXISTS crypto_data (
    time TIMESTAMPTZ NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    interval_type VARCHAR(5) NOT NULL, -- '1s', '5s', '1m', '15m', '1h', '4h', '1d'
    open DECIMAL(20,8) NOT NULL,
    high DECIMAL(20,8) NOT NULL,
    low DECIMAL(20,8) NOT NULL,
    close DECIMAL(20,8) NOT NULL,
    volume DECIMAL(20,8) NOT NULL,
    quote_volume DECIMAL(20,8),
    trades_count INTEGER,
    taker_buy_base_volume DECIMAL(20,8),
    taker_buy_quote_volume DECIMAL(20,8),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Créer l'hypertable avec partitioning par symbole
SELECT create_hypertable('crypto_data', 'time',
    partitioning_column => 'symbol',
    number_partitions => 50,
    if_not_exists => TRUE
);

-- Table pour les indicateurs techniques
CREATE TABLE IF NOT EXISTS crypto_indicators (
    time TIMESTAMPTZ NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    interval_type VARCHAR(5) NOT NULL,
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
SELECT create_hypertable('crypto_indicators', 'time',
    partitioning_column => 'symbol',
    number_partitions => 20,
    if_not_exists => TRUE
);

-- Table pour les actualités crypto
CREATE TABLE IF NOT EXISTS crypto_news (
    id SERIAL PRIMARY KEY,
    time TIMESTAMPTZ NOT NULL,
    title TEXT NOT NULL,
    content TEXT,
    source VARCHAR(100),
    url TEXT,
    sentiment_score DECIMAL(5,2), -- Score de sentiment entre -1 et 1
    symbols TEXT[], -- Array des symboles mentionnés
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Créer l'hypertable pour les news
SELECT create_hypertable('crypto_news', 'time',
    if_not_exists => TRUE
);

-- =============================================================================
-- INDEX OPTIMISÉS
-- =============================================================================

-- Index pour crypto_data
CREATE INDEX IF NOT EXISTS idx_crypto_data_symbol_time
    ON crypto_data (symbol, time DESC);

CREATE INDEX IF NOT EXISTS idx_crypto_data_interval_time
    ON crypto_data (interval_type, time DESC);

CREATE INDEX IF NOT EXISTS idx_crypto_data_symbol_interval_time
    ON crypto_data (symbol, interval_type, time DESC);

-- Index pour crypto_indicators
CREATE INDEX IF NOT EXISTS idx_crypto_indicators_symbol_time
    ON crypto_indicators (symbol, time DESC);

CREATE INDEX IF NOT EXISTS idx_crypto_indicators_type_time
    ON crypto_indicators (indicator_type, time DESC);

CREATE INDEX IF NOT EXISTS idx_crypto_indicators_symbol_type_time
    ON crypto_indicators (symbol, indicator_type, time DESC);

-- Index pour crypto_news
CREATE INDEX IF NOT EXISTS idx_crypto_news_time
    ON crypto_news (time DESC);

CREATE INDEX IF NOT EXISTS idx_crypto_news_symbols
    ON crypto_news USING GIN (symbols);

CREATE INDEX IF NOT EXISTS idx_crypto_news_source_time
    ON crypto_news (source, time DESC);

-- =============================================================================
-- POLITIQUES DE COMPRESSION
-- =============================================================================

-- Compression pour crypto_data après 7 jours
SELECT add_compression_policy('crypto_data', INTERVAL '7 days', if_not_exists => TRUE);

-- Compression pour crypto_indicators après 7 jours
SELECT add_compression_policy('crypto_indicators', INTERVAL '7 days', if_not_exists => TRUE);

-- Compression pour crypto_news après 30 jours
SELECT add_compression_policy('crypto_news', INTERVAL '30 days', if_not_exists => TRUE);

-- =============================================================================
-- POLITIQUES DE RÉTENTION
-- =============================================================================

-- Rétention des données brutes 1s (24 heures)
SELECT add_retention_policy('crypto_data', INTERVAL '24 hours',
    if_not_exists => TRUE,
    schedule_interval => INTERVAL '1 hour'
) WHERE EXISTS (
    SELECT 1 FROM crypto_data WHERE interval_type = '1s' LIMIT 1
);

-- Rétention générale pour toutes les données (2 ans)
SELECT add_retention_policy('crypto_data', INTERVAL '2 years', if_not_exists => TRUE);

-- Rétention pour les indicateurs (1 an)
SELECT add_retention_policy('crypto_indicators', INTERVAL '1 year', if_not_exists => TRUE);

-- Rétention pour les news (6 mois)
SELECT add_retention_policy('crypto_news', INTERVAL '6 months', if_not_exists => TRUE);

-- =============================================================================
-- VUES MATÉRIALISÉES POUR AGRÉGATIONS
-- =============================================================================

-- Vue pour les données OHLCV par heure
CREATE MATERIALIZED VIEW IF NOT EXISTS crypto_hourly_summary AS
SELECT
    time_bucket('1 hour', time) AS hour,
    symbol,
    first(open, time) AS open,
    max(high) AS high,
    min(low) AS low,
    last(close, time) AS close,
    sum(volume) AS volume,
    count(*) AS trades_count
FROM crypto_data
WHERE interval_type IN ('1m', '5m', '15m')
GROUP BY hour, symbol
ORDER BY hour DESC, symbol;

-- Index pour la vue matérialisée
CREATE INDEX IF NOT EXISTS idx_crypto_hourly_summary_symbol_hour
    ON crypto_hourly_summary (symbol, hour DESC);

-- Vue pour les indicateurs récents
CREATE MATERIALIZED VIEW IF NOT EXISTS crypto_indicators_latest AS
SELECT DISTINCT ON (symbol, indicator_type, interval_type)
    symbol,
    indicator_type,
    interval_type,
    time,
    value,
    value_signal,
    value_histogram,
    upper_band,
    lower_band,
    middle_band
FROM crypto_indicators
ORDER BY symbol, indicator_type, interval_type, time DESC;

-- =============================================================================
-- FONCTIONS UTILITAIRES
-- =============================================================================

-- Fonction pour nettoyer les anciennes données selon l'intervalle
CREATE OR REPLACE FUNCTION cleanup_old_data()
RETURNS void AS $$
BEGIN
    -- Supprimer les données 1s plus anciennes que 24h
    DELETE FROM crypto_data
    WHERE interval_type = '1s'
    AND time < NOW() - INTERVAL '24 hours';

    -- Supprimer les données 5s plus anciennes que 7 jours
    DELETE FROM crypto_data
    WHERE interval_type = '5s'
    AND time < NOW() - INTERVAL '7 days';

    -- Supprimer les données 1m plus anciennes que 30 jours
    DELETE FROM crypto_data
    WHERE interval_type = '1m'
    AND time < NOW() - INTERVAL '30 days';

    -- Supprimer les données 15m plus anciennes que 6 mois
    DELETE FROM crypto_data
    WHERE interval_type = '15m'
    AND time < NOW() - INTERVAL '6 months';

    RAISE NOTICE 'Cleanup completed';
END;
$$ LANGUAGE plpgsql;

-- Fonction pour calculer les statistiques de performance
CREATE OR REPLACE FUNCTION get_crypto_stats(p_symbol VARCHAR, p_interval VARCHAR)
RETURNS TABLE (
    symbol VARCHAR,
    interval_type VARCHAR,
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
        p_interval::VARCHAR,
        (SELECT close FROM crypto_data
         WHERE symbol = p_symbol AND interval_type = p_interval
         ORDER BY time DESC LIMIT 1) AS latest_price,

        (SELECT close FROM crypto_data
         WHERE symbol = p_symbol AND interval_type = p_interval
         ORDER BY time DESC LIMIT 1) -
        (SELECT close FROM crypto_data
         WHERE symbol = p_symbol AND interval_type = p_interval
         AND time <= NOW() - INTERVAL '24 hours'
         ORDER BY time DESC LIMIT 1) AS price_change_24h,

        ((SELECT close FROM crypto_data
          WHERE symbol = p_symbol AND interval_type = p_interval
          ORDER BY time DESC LIMIT 1) -
         (SELECT close FROM crypto_data
          WHERE symbol = p_symbol AND interval_type = p_interval
          AND time <= NOW() - INTERVAL '24 hours'
          ORDER BY time DESC LIMIT 1)) * 100.0 /
        (SELECT close FROM crypto_data
         WHERE symbol = p_symbol AND interval_type = p_interval
         AND time <= NOW() - INTERVAL '24 hours'
         ORDER BY time DESC LIMIT 1) AS price_change_pct_24h,

        (SELECT sum(volume) FROM crypto_data
         WHERE symbol = p_symbol AND interval_type = p_interval
         AND time >= NOW() - INTERVAL '24 hours') AS volume_24h,

        (SELECT max(high) FROM crypto_data
         WHERE symbol = p_symbol AND interval_type = p_interval
         AND time >= NOW() - INTERVAL '24 hours') AS high_24h,

        (SELECT min(low) FROM crypto_data
         WHERE symbol = p_symbol AND interval_type = p_interval
         AND time >= NOW() - INTERVAL '24 hours') AS low_24h;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- JOBS AUTOMATIQUES
-- =============================================================================

-- Job pour rafraîchir les vues matérialisées toutes les 5 minutes
SELECT add_job('refresh_crypto_views', '5 minutes', if_not_exists => TRUE);

-- Procédure pour rafraîchir les vues
CREATE OR REPLACE PROCEDURE refresh_crypto_views()
LANGUAGE plpgsql AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY crypto_hourly_summary;
    REFRESH MATERIALIZED VIEW CONCURRENTLY crypto_indicators_latest;
    COMMIT;
END;
$$;

-- Job pour nettoyer les anciennes données toutes les heures
SELECT add_job('cleanup_old_data', '1 hour', if_not_exists => TRUE);

-- =============================================================================
-- DONNÉES DE TEST (OPTIONNEL)
-- =============================================================================

-- Insérer quelques données de test pour vérifier le fonctionnement
INSERT INTO crypto_data (time, symbol, interval_type, open, high, low, close, volume) VALUES
    (NOW() - INTERVAL '1 minute', 'BTCUSDT', '1m', 45000.00, 45100.00, 44900.00, 45050.00, 1.5),
    (NOW() - INTERVAL '2 minutes', 'BTCUSDT', '1m', 44950.00, 45000.00, 44800.00, 45000.00, 2.1),
    (NOW() - INTERVAL '3 minutes', 'BTCUSDT', '1m', 44800.00, 44950.00, 44700.00, 44950.00, 1.8),
    (NOW() - INTERVAL '1 minute', 'ETHUSDT', '1m', 3000.00, 3010.00, 2990.00, 3005.00, 5.2),
    (NOW() - INTERVAL '2 minutes', 'ETHUSDT', '1m', 2995.00, 3000.00, 2985.00, 3000.00, 4.8)
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
ANALYZE crypto_data;
ANALYZE crypto_indicators;
ANALYZE crypto_news;

-- Afficher un résumé de l'initialisation
DO $$
BEGIN
    RAISE NOTICE '=============================================================================';
    RAISE NOTICE 'CryptoViz Database Initialization Complete';
    RAISE NOTICE '=============================================================================';
    RAISE NOTICE 'Tables created: crypto_data, crypto_indicators, crypto_news';
    RAISE NOTICE 'Hypertables configured with partitioning';
    RAISE NOTICE 'Compression and retention policies applied';
    RAISE NOTICE 'Materialized views created for performance';
    RAISE NOTICE 'Utility functions and jobs configured';
    RAISE NOTICE 'Test data inserted';
    RAISE NOTICE '=============================================================================';
END
$$;
