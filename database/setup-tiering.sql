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
-- NOTE: add_data_node() is Enterprise-only - wrapped in exception handler for Community Edition
DO $$
BEGIN
    -- Try to add data node (only works in Enterprise Edition)
    PERFORM add_data_node(
        'minio_tiering',
        host => 'minio',
        port => 9000,
        database => 'cryptoviz',
        if_not_exists => TRUE
    );
    RAISE NOTICE 'Data node configured (Enterprise Edition detected)';
EXCEPTION
    WHEN undefined_function THEN
        RAISE NOTICE 'Skipping add_data_node (Community Edition - function not available)';
    WHEN OTHERS THEN
        RAISE WARNING 'Could not configure data node: %', SQLERRM;
END $$;

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
-- NOTE: Uses batching to prevent memory exhaustion on large datasets
CREATE OR REPLACE FUNCTION tier_old_candles()
RETURNS void AS $$
DECLARE
    rows_moved_total INTEGER := 0;
    rows_moved_batch INTEGER;
    rows_moved_interval INTEGER;
    batch_size INTEGER := 10000;  -- Process 10k rows at a time to limit memory usage
BEGIN
    -- Tier 1m candles older than 7 days (HOT_RETENTION_1M)
    RAISE NOTICE 'Starting tiering for 1m candles (batch size: %)', batch_size;
    rows_moved_interval := 0;
    LOOP
        WITH moved AS (
            DELETE FROM candles
            WHERE ctid IN (
                SELECT ctid FROM candles
                WHERE timeframe = '1m' AND window_start < NOW() - INTERVAL '7 days'
                LIMIT batch_size
            )
            RETURNING *
        )
        INSERT INTO cold_storage.candles SELECT * FROM moved;

        GET DIAGNOSTICS rows_moved_batch = ROW_COUNT;
        EXIT WHEN rows_moved_batch = 0;

        rows_moved_interval := rows_moved_interval + rows_moved_batch;
        rows_moved_total := rows_moved_total + rows_moved_batch;
        RAISE NOTICE '  Tiered % rows (1m batch) - interval total: %', rows_moved_batch, rows_moved_interval;

        -- Small delay to allow WAL to flush and prevent memory buildup
        PERFORM pg_sleep(0.1);
    END LOOP;
    RAISE NOTICE 'Completed 1m: % rows to cold storage', rows_moved_interval;

    -- Tier 5m candles older than 14 days (HOT_RETENTION_5M)
    RAISE NOTICE 'Starting tiering for 5m candles (batch size: %)', batch_size;
    rows_moved_interval := 0;
    LOOP
        WITH moved AS (
            DELETE FROM candles
            WHERE ctid IN (
                SELECT ctid FROM candles
                WHERE timeframe = '5m' AND window_start < NOW() - INTERVAL '14 days'
                LIMIT batch_size
            )
            RETURNING *
        )
        INSERT INTO cold_storage.candles SELECT * FROM moved;

        GET DIAGNOSTICS rows_moved_batch = ROW_COUNT;
        EXIT WHEN rows_moved_batch = 0;

        rows_moved_interval := rows_moved_interval + rows_moved_batch;
        rows_moved_total := rows_moved_total + rows_moved_batch;
        RAISE NOTICE '  Tiered % rows (5m batch) - interval total: %', rows_moved_batch, rows_moved_interval;

        PERFORM pg_sleep(0.1);
    END LOOP;
    RAISE NOTICE 'Completed 5m: % rows to cold storage', rows_moved_interval;

    -- Tier 15m candles older than 30 days (HOT_RETENTION_15M)
    RAISE NOTICE 'Starting tiering for 15m candles (batch size: %)', batch_size;
    rows_moved_interval := 0;
    LOOP
        WITH moved AS (
            DELETE FROM candles
            WHERE ctid IN (
                SELECT ctid FROM candles
                WHERE timeframe = '15m' AND window_start < NOW() - INTERVAL '30 days'
                LIMIT batch_size
            )
            RETURNING *
        )
        INSERT INTO cold_storage.candles SELECT * FROM moved;

        GET DIAGNOSTICS rows_moved_batch = ROW_COUNT;
        EXIT WHEN rows_moved_batch = 0;

        rows_moved_interval := rows_moved_interval + rows_moved_batch;
        rows_moved_total := rows_moved_total + rows_moved_batch;
        RAISE NOTICE '  Tiered % rows (15m batch) - interval total: %', rows_moved_batch, rows_moved_interval;

        PERFORM pg_sleep(0.1);
    END LOOP;
    RAISE NOTICE 'Completed 15m: % rows to cold storage', rows_moved_interval;

    -- Tier 1h candles older than 90 days (HOT_RETENTION_1H)
    RAISE NOTICE 'Starting tiering for 1h candles (batch size: %)', batch_size;
    rows_moved_interval := 0;
    LOOP
        WITH moved AS (
            DELETE FROM candles
            WHERE ctid IN (
                SELECT ctid FROM candles
                WHERE timeframe = '1h' AND window_start < NOW() - INTERVAL '90 days'
                LIMIT batch_size
            )
            RETURNING *
        )
        INSERT INTO cold_storage.candles SELECT * FROM moved;

        GET DIAGNOSTICS rows_moved_batch = ROW_COUNT;
        EXIT WHEN rows_moved_batch = 0;

        rows_moved_interval := rows_moved_interval + rows_moved_batch;
        rows_moved_total := rows_moved_total + rows_moved_batch;
        RAISE NOTICE '  Tiered % rows (1h batch) - interval total: %', rows_moved_batch, rows_moved_interval;

        PERFORM pg_sleep(0.1);
    END LOOP;
    RAISE NOTICE 'Completed 1h: % rows to cold storage', rows_moved_interval;

    -- 1d candles stay in hot storage (HOT_RETENTION_1D=36500 days = 100 years)
    -- No tiering for 1d interval

    -- Remove obsolete intervals (1s, 5s, 4h) from cold storage if they exist
    DELETE FROM cold_storage.candles WHERE timeframe IN ('1s', '5s', '4h');

    RAISE NOTICE 'TOTAL: Tiered % candle rows to cold storage', rows_moved_total;
END;
$$ LANGUAGE plpgsql;

-- Function to manually tier indicators to cold storage
-- NOTE: Uses batching to prevent memory exhaustion on large datasets
CREATE OR REPLACE FUNCTION tier_old_indicators()
RETURNS void AS $$
DECLARE
    rows_moved_total INTEGER := 0;
    rows_moved_batch INTEGER;
    batch_size INTEGER := 10000;  -- Process 10k rows at a time to limit memory usage
BEGIN
    RAISE NOTICE 'Starting tiering for indicators (batch size: %)', batch_size;

    -- Move indicators older than 30 days to cold storage
    LOOP
        WITH moved AS (
            DELETE FROM indicators
            WHERE ctid IN (
                SELECT ctid FROM indicators
                WHERE time < NOW() - INTERVAL '30 days'
                LIMIT batch_size
            )
            RETURNING *
        )
        INSERT INTO cold_storage.indicators SELECT * FROM moved;

        GET DIAGNOSTICS rows_moved_batch = ROW_COUNT;
        EXIT WHEN rows_moved_batch = 0;

        rows_moved_total := rows_moved_total + rows_moved_batch;
        RAISE NOTICE '  Tiered % indicator rows (batch) - total: %', rows_moved_batch, rows_moved_total;

        PERFORM pg_sleep(0.1);
    END LOOP;

    RAISE NOTICE 'Completed indicators: % rows to cold storage', rows_moved_total;
END;
$$ LANGUAGE plpgsql;

-- Function to manually tier news to cold storage
-- NOTE: Uses batching to prevent memory exhaustion on large datasets
CREATE OR REPLACE FUNCTION tier_old_news()
RETURNS void AS $$
DECLARE
    rows_moved_total INTEGER := 0;
    rows_moved_batch INTEGER;
    batch_size INTEGER := 10000;  -- Process 10k rows at a time to limit memory usage
BEGIN
    RAISE NOTICE 'Starting tiering for news (batch size: %)', batch_size;

    -- Move news older than 30 days to cold storage
    LOOP
        WITH moved AS (
            DELETE FROM news
            WHERE ctid IN (
                SELECT ctid FROM news
                WHERE time < NOW() - INTERVAL '30 days'
                LIMIT batch_size
            )
            RETURNING *
        )
        INSERT INTO cold_storage.news SELECT * FROM moved;

        GET DIAGNOSTICS rows_moved_batch = ROW_COUNT;
        EXIT WHEN rows_moved_batch = 0;

        rows_moved_total := rows_moved_total + rows_moved_batch;
        RAISE NOTICE '  Tiered % news rows (batch) - total: %', rows_moved_batch, rows_moved_total;

        PERFORM pg_sleep(0.1);
    END LOOP;

    RAISE NOTICE 'Completed news: % rows to cold storage', rows_moved_total;
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
-- NOTE: TimescaleDB add_job() is Enterprise-only
-- Using pg_cron extension instead (Community Edition compatible)

-- Ensure pg_cron extension is available
DO $$
BEGIN
    -- Try to create pg_cron extension if not exists
    CREATE EXTENSION IF NOT EXISTS pg_cron;
    RAISE NOTICE 'pg_cron extension enabled for job scheduling';
EXCEPTION
    WHEN OTHERS THEN
        RAISE WARNING 'pg_cron extension not available - tiering jobs must be scheduled manually';
        RAISE WARNING 'To schedule manually: SELECT tier_old_candles(); (run daily via cron/scheduler)';
END $$;

-- Schedule tiering jobs using pg_cron (Community Edition compatible)
-- Runs daily at 3 AM
DO $$
BEGIN
    -- Schedule candles tiering
    PERFORM cron.schedule('tier-candles-job', '0 3 * * *', 'SELECT tier_old_candles()');
    RAISE NOTICE 'Scheduled tier_old_candles() to run daily at 3 AM';

    -- Schedule indicators tiering
    PERFORM cron.schedule('tier-indicators-job', '0 3 * * *', 'SELECT tier_old_indicators()');
    RAISE NOTICE 'Scheduled tier_old_indicators() to run daily at 3 AM';

    -- Schedule news tiering
    PERFORM cron.schedule('tier-news-job', '0 3 * * *', 'SELECT tier_old_news()');
    RAISE NOTICE 'Scheduled tier_old_news() to run daily at 3 AM';
EXCEPTION
    WHEN undefined_function THEN
        RAISE WARNING 'pg_cron not available - jobs not scheduled';
        RAISE NOTICE 'Manual scheduling required: Call tier_old_candles(), tier_old_indicators(), tier_old_news() daily';
    WHEN OTHERS THEN
        RAISE WARNING 'Could not schedule tiering jobs: %', SQLERRM;
        RAISE NOTICE 'You can manually run: SELECT tier_old_candles(); SELECT tier_old_indicators(); SELECT tier_old_news();';
END $$;

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
