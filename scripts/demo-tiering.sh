#!/bin/bash
# =============================================================================
# CryptoViz Data Tiering Demo Script
# =============================================================================
# This script demonstrates the data tiering functionality for presentations

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Database connection details
DB_HOST="${TIMESCALE_HOST:-localhost}"
DB_PORT="${TIMESCALE_PORT:-7432}"
DB_NAME="${TIMESCALE_DB:-cryptoviz}"
DB_USER="${TIMESCALE_USER:-postgres}"

# Function to execute SQL and display results
execute_sql() {
    local sql="$1"
    local description="$2"

    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}${description}${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo

    docker exec -it cryptoviz-timescaledb psql -U "$DB_USER" -d "$DB_NAME" -c "$sql"
    echo
}

# Function to pause and wait for user
pause() {
    echo -e "${YELLOW}Press Enter to continue...${NC}"
    read
}

clear
echo -e "${GREEN}"
cat << "EOF"
╔═══════════════════════════════════════════════════════════════════════════╗
║                                                                           ║
║          CryptoViz Data Tiering Demonstration                            ║
║          POC: Local MinIO Object Storage                                 ║
║                                                                           ║
╚═══════════════════════════════════════════════════════════════════════════╝
EOF
echo -e "${NC}"
echo
echo "This demo showcases how CryptoViz handles data lifecycle management:"
echo "  • HOT data (recent):  Fast SSD storage"
echo "  • COLD data (old):    Cheap object storage (MinIO/S3)"
echo
pause

# Step 1: Initial state
execute_sql "SELECT * FROM get_tiering_stats();" \
    "Step 1: Check initial data distribution (Hot vs Cold)"
pause

# Step 2: Generate old test data
echo -e "${GREEN}Step 2: Generating old test data (30 days ago)...${NC}"
echo "This simulates historical data that should be tiered to cold storage"
echo
execute_sql "SELECT generate_old_test_data();" \
    "Generating 720 old candle records (30 days × 24 hours)"
pause

# Step 3: Check data before tiering
execute_sql "SELECT * FROM get_tiering_stats();" \
    "Step 3: Data distribution BEFORE tiering"
echo "Notice: All data is currently in HOT storage (main tables)"
pause

# Step 4: Trigger tiering manually
echo -e "${GREEN}Step 4: Triggering data tiering...${NC}"
echo "Moving data older than 7 days to cold storage (MinIO)"
echo
execute_sql "SELECT tier_old_candles();" \
    "Tiering old candles to cold storage"
pause

# Step 5: Check data after tiering
execute_sql "SELECT * FROM get_tiering_stats();" \
    "Step 5: Data distribution AFTER tiering"
echo "Notice: Old data moved to COLD storage, recent data remains in HOT storage"
pause

# Step 6: Query hot data (fast)
execute_sql "
SELECT
    timeframe,
    COUNT(*) as records,
    MIN(window_start) as oldest,
    MAX(window_start) as newest
FROM candles
GROUP BY timeframe;" \
    "Step 6: Query HOT data only (fast - from SSD)"
pause

# Step 7: Query cold data
execute_sql "
SELECT
    timeframe,
    COUNT(*) as records,
    MIN(window_start) as oldest,
    MAX(window_start) as newest
FROM cold_storage.candles
GROUP BY timeframe;" \
    "Step 7: Query COLD data only (slower - from MinIO/S3)"
pause

# Step 8: Query unified view (hot + cold)
execute_sql "
SELECT
    timeframe,
    COUNT(*) as records,
    MIN(window_start) as oldest,
    MAX(window_start) as newest
FROM all_candles
GROUP BY timeframe;" \
    "Step 8: Query ALL data (hot + cold combined - transparent to application)"
pause

# Step 9: Show MinIO storage
echo -e "${GREEN}Step 9: Check MinIO Object Storage${NC}"
echo "MinIO Console URL: http://localhost:9001"
echo "Login: minioadmin / minioadmin"
echo "Bucket: cryptoviz-cold-storage"
echo
echo "In production, this would be AWS S3, reducing storage costs by 85%+"
pause

# Step 10: Show automatic scheduling
execute_sql "
SELECT
    job_id,
    application_name,
    schedule_interval,
    next_start
FROM timescaledb_information.jobs
WHERE application_name LIKE 'tier_%';" \
    "Step 10: Automatic tiering schedule (runs daily at 3 AM)"
pause

# Final summary
clear
echo -e "${GREEN}"
cat << "EOF"
╔═══════════════════════════════════════════════════════════════════════════╗
║                       Demo Summary                                        ║
╚═══════════════════════════════════════════════════════════════════════════╝
EOF
echo -e "${NC}"
echo
echo "✓ HOT Storage (SSD):  Recent data for fast queries"
echo "✓ COLD Storage (S3):  Historical data for cost optimization"
echo "✓ Transparent Access: Application queries work seamlessly"
echo "✓ Auto Scheduling:    Data tiers automatically every day"
echo
echo -e "${BLUE}Architecture Benefits:${NC}"
echo "  • 85%+ cost reduction for old data"
echo "  • Maintains performance for recent queries"
echo "  • Scales to petabytes of historical data"
echo "  • Production-ready (swap MinIO → AWS S3)"
echo
echo -e "${BLUE}Current Setup:${NC}"
echo "  • Local MinIO for demo"
echo "  • Same codebase works with real S3"
echo "  • Zero application code changes"
echo
execute_sql "SELECT * FROM get_tiering_stats();" \
    "Final data distribution"
echo
echo -e "${GREEN}Demo complete!${NC}"
echo
echo "To reset and run again:"
echo "  docker-compose down -v"
echo "  docker-compose up -d"
echo "  ./scripts/demo-tiering.sh"
echo
