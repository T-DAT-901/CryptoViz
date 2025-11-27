#!/bin/bash

# =============================================================================
# CryptoViz - Database Migration Runner
# =============================================================================
# Runs database migrations (idempotent - safe to run multiple times)
# All SQL files use CREATE OR REPLACE / IF NOT EXISTS patterns

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

# Wait for TimescaleDB to be ready
wait_for_db() {
    local max_attempts=30
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        if docker exec cryptoviz-timescaledb pg_isready -U postgres -d cryptoviz > /dev/null 2>&1; then
            return 0
        fi
        warn "Waiting for TimescaleDB... (attempt $attempt/$max_attempts)"
        sleep 2
        attempt=$((attempt + 1))
    done

    echo "ERROR: TimescaleDB not ready after $max_attempts attempts"
    return 1
}

# Run migrations
run_migrations() {
    log "Running database migrations..."

    # Backfill tracking (update_backfill_progress, get_last_backfill_timestamp, etc.)
    docker exec -i cryptoviz-timescaledb psql -U postgres -d cryptoviz < database/04-backfill-tracking.sql

    # News deduplication (unique URL constraint)
    docker exec -i cryptoviz-timescaledb psql -U postgres -d cryptoviz < database/05-news-dedup.sql

    log "✓ Migrations complete"
}

# Main
main() {
    wait_for_db
    run_migrations
}

main "$@"
