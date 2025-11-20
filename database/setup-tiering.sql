-- =============================================================================
-- TimescaleDB Data Tiering Configuration for MinIO
-- =============================================================================
-- NOTE: This requires TimescaleDB 2.11+ with tiering support
-- This script configures S3-compatible tiering using local MinIO instance

-- =============================================================================
-- CONFIGURE S3 TIERING (MinIO)
-- =============================================================================

-- Configure the tiering data node to use MinIO
-- This tells TimescaleDB where to send cold data
SELECT add_data_node(
    'minio_tiering',
    host => 'minio',
    port => 9000,
    database => 'cryptoviz',
    if_not_exists => TRUE
);

-- Alternative approach: Use object storage directly via S3 API
-- Configure S3 settings for MinIO
-- NOTE: In real TimescaleDB 2.11+, this would use timescaledb_toolkit or pg_tiering extension

-- For now, we'll simulate tiering with a manual approach that works with current TimescaleDB
-- This demonstrates the concept even if full auto-tiering isn't available

-- =============================================================================
-- CREATE TIERING TABLES (Cold Storage Simulation)
-- =============================================================================

-- Create schema for cold storage tables
CREATE SCHEMA IF NOT EXISTS cold_storage;

-- Cold storage for candles (older than 7 days)
CREATE TABLE IF NOT EXISTS cold_storage.candles (
    LIKE candles INCLUDING ALL
);

-- Cold storage for indicators (older than 30 days)
CREATE TABLE IF NOT EXISTS cold_storage.indicators (
    LIKE indicators INCLUDING ALL
);

-- Cold storage for news (older than 30 days)
CREATE TABLE IF NOT EXISTS cold_storage.news (
    LIKE news INCLUDING ALL
);

-- =============================================================================
-- TIERING FUNCTIONS
-- =============================================================================

-- Function to manually tier candles to cold storage (interval-specific)
-- Retention configured in .env:
--   HOT_RETENTION_1M=7, HOT_RETENTION_5M=14, HOT_RETENTION_15M=30
--   HOT_RETENTION_1H=90, HOT_RETENTION_1D=36500
CREATE OR REPLACE FUNCTION tier_old_candles()
RETURNS void AS $$
DECLARE
    rows_moved_total INTEGER := 0;
    rows_moved INTEGER;
BEGIN
    -- Tier 1m candles older than 7 days (HOT_RETENTION_1M)
    WITH moved AS (
        DELETE FROM candles
        WHERE timeframe = '1m' AND window_start < NOW() - INTERVAL '7 days'
        RETURNING *
    )
    INSERT INTO cold_storage.candles SELECT * FROM moved;
    GET DIAGNOSTICS rows_moved = ROW_COUNT;
    rows_moved_total := rows_moved_total + rows_moved;
    RAISE NOTICE 'Tiered % rows (1m) to cold storage', rows_moved;

    -- Tier 5m candles older than 14 days (HOT_RETENTION_5M)
    WITH moved AS (
        DELETE FROM candles
        WHERE timeframe = '5m' AND window_start < NOW() - INTERVAL '14 days'
        RETURNING *
    )
    INSERT INTO cold_storage.candles SELECT * FROM moved;
    GET DIAGNOSTICS rows_moved = ROW_COUNT;
    rows_moved_total := rows_moved_total + rows_moved;
    RAISE NOTICE 'Tiered % rows (5m) to cold storage', rows_moved;

    -- Tier 15m candles older than 30 days (HOT_RETENTION_15M)
    WITH moved AS (
        DELETE FROM candles
        WHERE timeframe = '15m' AND window_start < NOW() - INTERVAL '30 days'
        RETURNING *
    )
    INSERT INTO cold_storage.candles SELECT * FROM moved;
    GET DIAGNOSTICS rows_moved = ROW_COUNT;
    rows_moved_total := rows_moved_total + rows_moved;
    RAISE NOTICE 'Tiered % rows (15m) to cold storage', rows_moved;

    -- Tier 1h candles older than 90 days (HOT_RETENTION_1H)
    WITH moved AS (
        DELETE FROM candles
        WHERE timeframe = '1h' AND window_start < NOW() - INTERVAL '90 days'
        RETURNING *
    )
    INSERT INTO cold_storage.candles SELECT * FROM moved;
    GET DIAGNOSTICS rows_moved = ROW_COUNT;
    rows_moved_total := rows_moved_total + rows_moved;
    RAISE NOTICE 'Tiered % rows (1h) to cold storage', rows_moved;

    -- 1d candles stay in hot storage (HOT_RETENTION_1D=36500 days = 100 years)
    -- No tiering for 1d interval

    -- Remove obsolete intervals (1s, 5s, 4h) from cold storage if they exist
    DELETE FROM cold_storage.candles WHERE timeframe IN ('1s', '5s', '4h');

    RAISE NOTICE 'Total tiered % candle rows to cold storage', rows_moved_total;
END;
$$ LANGUAGE plpgsql;

-- Function to manually tier indicators to cold storage
CREATE OR REPLACE FUNCTION tier_old_indicators()
RETURNS void AS $$
DECLARE
    rows_moved INTEGER;
BEGIN
    -- Move indicators older than 30 days to cold storage
    WITH moved AS (
        DELETE FROM indicators
        WHERE time < NOW() - INTERVAL '30 days'
        RETURNING *
    )
    INSERT INTO cold_storage.indicators SELECT * FROM moved;

    GET DIAGNOSTICS rows_moved = ROW_COUNT;
    RAISE NOTICE 'Tiered % indicator rows to cold storage', rows_moved;
END;
$$ LANGUAGE plpgsql;

-- Function to manually tier news to cold storage
CREATE OR REPLACE FUNCTION tier_old_news()
RETURNS void AS $$
DECLARE
    rows_moved INTEGER;
BEGIN
    -- Move news older than 30 days to cold storage
    WITH moved AS (
        DELETE FROM news
        WHERE time < NOW() - INTERVAL '30 days'
        RETURNING *
    )
    INSERT INTO cold_storage.news SELECT * FROM moved;

    GET DIAGNOSTICS rows_moved = ROW_COUNT;
    RAISE NOTICE 'Tiered % news rows to cold storage', rows_moved;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- CREATE UNIFIED VIEWS (Hot + Cold Data)
-- =============================================================================

-- View that combines hot and cold candles data
CREATE OR REPLACE VIEW all_candles AS
SELECT * FROM candles
UNION ALL
SELECT * FROM cold_storage.candles;

-- View that combines hot and cold indicators data
CREATE OR REPLACE VIEW all_indicators AS
SELECT * FROM indicators
UNION ALL
SELECT * FROM cold_storage.indicators;

-- View that combines hot and cold news data
CREATE OR REPLACE VIEW all_news AS
SELECT * FROM news
UNION ALL
SELECT * FROM cold_storage.news;

-- =============================================================================
-- SCHEDULED TIERING JOB
-- =============================================================================

-- Schedule tiering to run daily at 3 AM
-- This simulates automatic tiering
SELECT add_job('tier_old_candles', '1 day', initial_start => '2025-01-01 03:00:00', if_not_exists => TRUE);
SELECT add_job('tier_old_indicators', '1 day', initial_start => '2025-01-01 03:00:00', if_not_exists => TRUE);
SELECT add_job('tier_old_news', '1 day', initial_start => '2025-01-01 03:00:00', if_not_exists => TRUE);

-- =============================================================================
-- EXPORT TO MINIO (via Foreign Data Wrapper - Optional Advanced Setup)
-- =============================================================================

-- Install parquet_s3_fdw extension if available (requires compilation)
-- CREATE EXTENSION IF NOT EXISTS parquet_s3_fdw;

-- Create server pointing to MinIO
-- CREATE SERVER minio_server
--   FOREIGN DATA WRAPPER parquet_s3_fdw
--   OPTIONS (
--     endpoint 'http://minio:9000',
--     region 'us-east-1',
--     use_ssl 'false'
--   );

-- Create user mapping with MinIO credentials
-- CREATE USER MAPPING FOR postgres
--   SERVER minio_server
--   OPTIONS (
--     access_key_id 'minioadmin',
--     secret_access_key 'minioadmin'
--   );

-- =============================================================================
-- MONITORING QUERIES
-- =============================================================================

-- Query to check data distribution (hot vs cold)
CREATE OR REPLACE FUNCTION get_tiering_stats()
RETURNS TABLE (
    table_name TEXT,
    hot_rows BIGINT,
    cold_rows BIGINT,
    total_rows BIGINT,
    hot_size TEXT,
    cold_size TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        'candles'::TEXT,
        (SELECT COUNT(*) FROM candles)::BIGINT,
        (SELECT COUNT(*) FROM cold_storage.candles)::BIGINT,
        (SELECT COUNT(*) FROM all_candles)::BIGINT,
        pg_size_pretty(pg_total_relation_size('candles'))::TEXT,
        pg_size_pretty(pg_total_relation_size('cold_storage.candles'))::TEXT
    UNION ALL
    SELECT
        'indicators'::TEXT,
        (SELECT COUNT(*) FROM indicators)::BIGINT,
        (SELECT COUNT(*) FROM cold_storage.indicators)::BIGINT,
        (SELECT COUNT(*) FROM all_indicators)::BIGINT,
        pg_size_pretty(pg_total_relation_size('indicators'))::TEXT,
        pg_size_pretty(pg_total_relation_size('cold_storage.indicators'))::TEXT
    UNION ALL
    SELECT
        'news'::TEXT,
        (SELECT COUNT(*) FROM news)::BIGINT,
        (SELECT COUNT(*) FROM cold_storage.news)::BIGINT,
        (SELECT COUNT(*) FROM all_news)::BIGINT,
        pg_size_pretty(pg_total_relation_size('news'))::TEXT,
        pg_size_pretty(pg_total_relation_size('cold_storage.news'))::TEXT;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- DEMO HELPER FUNCTIONS
-- =============================================================================

-- Function to generate old test data for tiering demo
CREATE OR REPLACE FUNCTION generate_old_test_data()
RETURNS void AS $$
BEGIN
    -- Generate candles from 30 days ago
    INSERT INTO candles (window_start, window_end, exchange, symbol, timeframe, open, high, low, close, volume, trade_count, closed)
    SELECT
        NOW() - INTERVAL '30 days' + (i || ' hours')::INTERVAL,
        NOW() - INTERVAL '30 days' + (i || ' hours')::INTERVAL + INTERVAL '1 hour',
        'BINANCE',
        'BTC/USDT',
        '1h',
        45000.00 + RANDOM() * 1000,
        46000.00 + RANDOM() * 1000,
        44000.00 + RANDOM() * 1000,
        45500.00 + RANDOM() * 1000,
        RANDOM() * 100,
        FLOOR(RANDOM() * 1000)::INTEGER,
        true
    FROM generate_series(1, 720) AS i;  -- 30 days * 24 hours

    RAISE NOTICE 'Generated 720 old candle records';
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- PERMISSIONS
-- =============================================================================

-- Grant access to cold storage schema
GRANT USAGE ON SCHEMA cold_storage TO cryptoviz_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA cold_storage TO cryptoviz_app;
GRANT SELECT ON all_candles, all_indicators, all_news TO cryptoviz_app;

-- =============================================================================
-- COMPLETION NOTICE
-- =============================================================================

DO $$
BEGIN
    RAISE NOTICE '=============================================================================';
    RAISE NOTICE 'Data Tiering Configuration Complete';
    RAISE NOTICE '=============================================================================';
    RAISE NOTICE 'Cold storage schema created: cold_storage';
    RAISE NOTICE 'Tiering functions created: tier_old_candles(), tier_old_indicators(), tier_old_news()';
    RAISE NOTICE 'Unified views created: all_candles, all_indicators, all_news';
    RAISE NOTICE 'Scheduled jobs configured to tier data daily at 3 AM';
    RAISE NOTICE '';
    RAISE NOTICE 'To manually trigger tiering:';
    RAISE NOTICE '  SELECT tier_old_candles();';
    RAISE NOTICE '';
    RAISE NOTICE 'To generate demo data:';
    RAISE NOTICE '  SELECT generate_old_test_data();';
    RAISE NOTICE '';
    RAISE NOTICE 'To check tiering stats:';
    RAISE NOTICE '  SELECT * FROM get_tiering_stats();';
    RAISE NOTICE '=============================================================================';
END
$$;
