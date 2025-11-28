-- =============================================================================
-- News Deduplication - Unique Title + Time Constraint
-- =============================================================================
-- URL dedup doesn't work due to UTM tracking params creating different URLs for same article.
-- Title + time constraint catches duplicates regardless of URL variations.

-- Drop old URL-based index if it exists
DROP INDEX IF EXISTS idx_news_unique_url_time;

-- Remove existing duplicates by title+time (keep earliest by ctid)
DELETE FROM news a USING news b
WHERE a.ctid > b.ctid AND a.title = b.title AND a.time = b.time;

-- Add unique constraint on title + time
CREATE UNIQUE INDEX IF NOT EXISTS idx_news_unique_title_time ON news (title, time);
