-- =============================================================================
-- News Deduplication - Unique URL + Time Constraint
-- =============================================================================
-- TimescaleDB hypertables require partitioning column (time) in unique indexes.
-- This prevents same URL at same timestamp. Combined with Redis dedup (48h TTL).

-- First, remove existing duplicates (keep earliest by ctid)
DELETE FROM news a USING news b
WHERE a.ctid > b.ctid AND a.url = b.url AND a.time = b.time;

-- Add unique constraint on URL + time (required for TimescaleDB hypertables)
CREATE UNIQUE INDEX IF NOT EXISTS idx_news_unique_url_time ON news (url, time);

-- Log completion
DO $$
BEGIN
    RAISE NOTICE 'Migration complete';
END $$;
