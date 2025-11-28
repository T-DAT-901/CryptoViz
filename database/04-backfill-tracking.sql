-- =============================================================================
-- Backfill Progress Tracking Table
-- =============================================================================
-- This table tracks the progress of historical data backfilling for each
-- symbol/timeframe combination. It replaces the JSON-based tracking system.

CREATE TABLE IF NOT EXISTS backfill_progress (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    timeframe VARCHAR(5) NOT NULL,
    exchange VARCHAR(20) NOT NULL DEFAULT 'BINANCE',

    -- Backfill date range (what we're trying to fill)
    backfill_start_ts TIMESTAMPTZ NOT NULL,  -- Oldest date we want
    backfill_end_ts TIMESTAMPTZ NOT NULL,    -- Newest date we want (usually NOW)

    -- Current progress (where we are in the backfill)
    current_position_ts TIMESTAMPTZ NOT NULL, -- Last timestamp successfully fetched

    -- Status tracking
    status VARCHAR(20) NOT NULL DEFAULT 'pending',  -- pending, in_progress, completed, failed

    -- Statistics
    total_candles_fetched INTEGER DEFAULT 0,
    total_batches_processed INTEGER DEFAULT 0,

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ,

    -- Error tracking
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    last_error_at TIMESTAMPTZ,

    -- Ensure one progress record per symbol/timeframe/exchange combination
    CONSTRAINT uq_backfill_progress UNIQUE (symbol, timeframe, exchange)
);

-- Index for querying by status
CREATE INDEX IF NOT EXISTS idx_backfill_progress_status
ON backfill_progress (status);

-- Index for querying by symbol
CREATE INDEX IF NOT EXISTS idx_backfill_progress_symbol
ON backfill_progress (symbol, timeframe);

-- =============================================================================
-- Helper Function: Update Backfill Progress (Idempotent)
-- =============================================================================
-- This function updates backfill progress. If the record doesn't exist, it creates it.
-- If it exists, it updates the current position and statistics.

CREATE OR REPLACE FUNCTION update_backfill_progress(
    p_symbol VARCHAR(20),
    p_timeframe VARCHAR(5),
    p_exchange VARCHAR(20),
    p_backfill_start_ts TIMESTAMPTZ,
    p_backfill_end_ts TIMESTAMPTZ,
    p_current_position_ts TIMESTAMPTZ,
    p_status VARCHAR(20),
    p_candles_count INTEGER DEFAULT 0,
    p_error_message TEXT DEFAULT NULL
) RETURNS void AS $$
BEGIN
    INSERT INTO backfill_progress (
        symbol,
        timeframe,
        exchange,
        backfill_start_ts,
        backfill_end_ts,
        current_position_ts,
        status,
        total_candles_fetched,
        total_batches_processed,
        updated_at,
        completed_at,
        error_message,
        retry_count,
        last_error_at
    ) VALUES (
        p_symbol,
        p_timeframe,
        p_exchange,
        p_backfill_start_ts,
        p_backfill_end_ts,
        p_current_position_ts,
        p_status,
        p_candles_count,
        1, -- first batch
        NOW(),
        CASE WHEN p_status = 'completed' THEN NOW() ELSE NULL END,
        p_error_message,
        CASE WHEN p_error_message IS NOT NULL THEN 1 ELSE 0 END,
        CASE WHEN p_error_message IS NOT NULL THEN NOW() ELSE NULL END
    )
    ON CONFLICT (symbol, timeframe, exchange) DO UPDATE SET
        -- Update backfill range if extended (for dynamic lookback period changes)
        backfill_start_ts = LEAST(EXCLUDED.backfill_start_ts, backfill_progress.backfill_start_ts),
        backfill_end_ts = GREATEST(EXCLUDED.backfill_end_ts, backfill_progress.backfill_end_ts),

        -- Update current position
        current_position_ts = EXCLUDED.current_position_ts,

        -- Update status
        status = EXCLUDED.status,

        -- Increment statistics
        total_candles_fetched = backfill_progress.total_candles_fetched + EXCLUDED.total_candles_fetched,
        total_batches_processed = backfill_progress.total_batches_processed + 1,

        -- Update timestamps
        updated_at = NOW(),
        completed_at = CASE
            WHEN EXCLUDED.status = 'completed' THEN NOW()
            WHEN EXCLUDED.status = 'in_progress' THEN NULL
            ELSE backfill_progress.completed_at
        END,

        -- Update error tracking
        error_message = CASE
            WHEN p_error_message IS NOT NULL THEN p_error_message
            ELSE backfill_progress.error_message
        END,
        retry_count = CASE
            WHEN p_error_message IS NOT NULL THEN backfill_progress.retry_count + 1
            ELSE backfill_progress.retry_count
        END,
        last_error_at = CASE
            WHEN p_error_message IS NOT NULL THEN NOW()
            ELSE backfill_progress.last_error_at
        END;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- Helper Function: Get Last Backfill Timestamp
-- =============================================================================
-- Returns the last successfully fetched timestamp for a symbol/timeframe.
-- Returns NULL if no backfill has started yet.

CREATE OR REPLACE FUNCTION get_last_backfill_timestamp(
    p_symbol VARCHAR(20),
    p_timeframe VARCHAR(5),
    p_exchange VARCHAR(20) DEFAULT 'BINANCE'
) RETURNS TIMESTAMPTZ AS $$
DECLARE
    v_timestamp TIMESTAMPTZ;
BEGIN
    SELECT current_position_ts INTO v_timestamp
    FROM backfill_progress
    WHERE symbol = p_symbol
      AND timeframe = p_timeframe
      AND exchange = p_exchange
      AND status IN ('in_progress', 'completed');

    RETURN v_timestamp;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- Helper Function: Check if Backfill Needed
-- =============================================================================
-- Checks if backfill is needed for a symbol/timeframe by comparing the
-- database state with the desired lookback period.
-- Returns TRUE if backfill needed, FALSE otherwise.

CREATE OR REPLACE FUNCTION check_backfill_needed(
    p_symbol VARCHAR(20),
    p_timeframe VARCHAR(5),
    p_exchange VARCHAR(20),
    p_lookback_days INTEGER
) RETURNS TABLE(
    needs_backfill BOOLEAN,
    db_start_ts TIMESTAMPTZ,
    db_end_ts TIMESTAMPTZ,
    target_start_ts TIMESTAMPTZ,
    status VARCHAR(20),
    gap_days INTEGER
) AS $$
DECLARE
    v_target_start_ts TIMESTAMPTZ;
BEGIN
    -- Calculate target start based on lookback days
    v_target_start_ts := date_trunc('day', NOW()) - (p_lookback_days || ' days')::INTERVAL;

    RETURN QUERY
    SELECT
        -- Need backfill if: no record, failed, config extended, or gap detected (system downtime)
        COALESCE(bp.status != 'completed', TRUE) OR
        COALESCE(bp.backfill_start_ts > v_target_start_ts, TRUE) OR
        COALESCE(bp.current_position_ts < NOW() - INTERVAL '2 minutes', TRUE) AS needs_backfill,

        bp.backfill_start_ts AS db_start_ts,
        bp.backfill_end_ts AS db_end_ts,
        v_target_start_ts AS target_start_ts,
        COALESCE(bp.status, 'not_started') AS status,

        -- Calculate gap in days if config was extended
        CASE
            WHEN bp.backfill_start_ts IS NOT NULL AND bp.backfill_start_ts > v_target_start_ts
            THEN EXTRACT(DAY FROM bp.backfill_start_ts - v_target_start_ts)::INTEGER
            ELSE 0
        END AS gap_days
    FROM (
        SELECT p_symbol AS symbol, p_timeframe AS timeframe, p_exchange AS exchange
    ) params
    LEFT JOIN backfill_progress bp
        ON bp.symbol = params.symbol
        AND bp.timeframe = params.timeframe
        AND bp.exchange = params.exchange;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- Comments
-- =============================================================================

COMMENT ON TABLE backfill_progress IS
'Tracks historical data backfill progress for each symbol/timeframe combination. Replaces JSON-based tracking for reliability and queryability.';

COMMENT ON COLUMN backfill_progress.backfill_start_ts IS
'The oldest timestamp we want to backfill to (based on BACKFILL_LOOKBACK_DAYS config)';

COMMENT ON COLUMN backfill_progress.current_position_ts IS
'The last timestamp successfully fetched and stored in the database';

COMMENT ON COLUMN backfill_progress.status IS
'Current status: pending (not started), in_progress (actively backfilling), completed (all data fetched), failed (error occurred)';

COMMENT ON FUNCTION update_backfill_progress IS
'Idempotent function to update backfill progress. Creates new record or updates existing one. Automatically detects config extensions.';

COMMENT ON FUNCTION get_last_backfill_timestamp IS
'Returns the last successfully fetched timestamp for a symbol/timeframe, or NULL if no backfill exists.';

COMMENT ON FUNCTION check_backfill_needed IS
'Checks if backfill is needed by comparing database state with desired lookback period. Detects: 1) config changes (e.g., 40 → 365 days), 2) system downtime gaps (>2 minutes since last data).';
