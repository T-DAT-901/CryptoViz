# Backend-Go Database Models Refactoring

**Date**: 2025-01-06
**Status**: ✅ Complete
**Breaking Changes**: Yes - All model names and table references updated

---

## Summary

The backend-go models have been completely refactored to match the actual TimescaleDB implementation in `database/init.sql`. This includes:

- ✅ Updated table names to match schema
- ✅ Updated column names (`interval_type` → `timeframe`, added `exchange`, etc.)
- ✅ Added support for unified views (hot + cold storage tiering)
- ✅ Added support for continuous aggregates
- ✅ Created new models for `Trade`, `User`, `Currency`
- ✅ Removed GORM AutoMigrate (using `init.sql` instead)

---

## File Changes

### 1. New Model Files Created

| File | Model | Table | Type |
|------|-------|-------|------|
| `models/candles.go` | `Candle` | `candles` | Hypertable |
| | `AllCandle` | `all_candles` (view) | Unified view (hot+cold) |
| | `CandleHourlySummary` | `candles_hourly_summary` | Continuous aggregate |
| `models/trade.go` | `Trade` | `trades` | Hypertable |
| `models/user.go` | `User` | `users` | Regular table |
| `models/currency.go` | `Currency` | `currencies` | Regular table |

### 2. Updated Model Files

| Old File | New File | Old Model | New Model | Old Table | New Table |
|----------|----------|-----------|-----------|-----------|-----------|
| `models/crypto_data.go` | `models/candles.go` | `CryptoData` | `Candle` | `crypto_data` | `candles` |
| `models/indicators.go` | `models/indicators.go` | `TechnicalIndicator` | `Indicator` | `crypto_indicators` | `indicators` |
| | | | `AllIndicator` | | `all_indicators` (view) |
| | | | `IndicatorLatest` | | `indicators_latest` (continuous aggregate) |
| `models/news.go` | `models/news.go` | `CryptoNews` | `News` | `crypto_news` | `news` |
| | | | `AllNews` | | `all_news` (view) |

### 3. Updated Connection File

| File | Changes |
|------|---------|
| `database/connection.go` | - Removed `AutoMigrate()` function<br>- Removed `setupTimescaleDB()` function<br>- Removed models import<br>- Added `verifyTimescaleDB()` to check extension<br>- Added `GetDatabaseInfo()` for monitoring |

---

## Breaking Changes

### Table Name Changes

**All code referencing old table names must be updated:**

```go
// OLD
db.Table("crypto_data").Where(...)
db.Table("crypto_indicators").Where(...)
db.Table("crypto_news").Where(...)

// NEW
db.Table("candles").Where(...)
db.Table("indicators").Where(...)
db.Table("news").Where(...)
```

### Model Name Changes

**All model references must be updated:**

```go
// OLD
var data models.CryptoData
var indicators []models.TechnicalIndicator
var news models.CryptoNews

// NEW
var candle models.Candle
var indicators []models.Indicator
var news models.News
```

### Column Name Changes

**Key column renames:**

- `interval_type` → `timeframe` (in candles and indicators)
- Added `exchange` column to candles
- Added `window_start`, `window_end` to replace single `time` in candles
- Renamed `value_histogram` to match SQL exactly

---

## New Features

### 1. Data Tiering Support

**Unified Views** provide transparent access to hot + cold storage:

```go
// Query hot storage only (recent data)
var candles []models.Candle
db.Where("symbol = ?", "BTC/USDT").Find(&candles)

// Query hot + cold storage (all data)
var allCandles []models.AllCandle
db.Where("symbol = ?", "BTC/USDT").Find(&allCandles)
```

**Available unified views:**
- `all_candles` - Combines hot (`candles`) + cold (`cold_storage.candles`)
- `all_indicators` - Combines hot (`indicators`) + cold (`cold_storage.indicators`)
- `all_news` - Combines hot (`news`) + cold (`cold_storage.news`)

### 2. Continuous Aggregates

**Pre-computed aggregations with automatic refresh:**

```go
// Query hourly OHLCV summary (continuous aggregate)
var summaries []models.CandleHourlySummary
db.Where("symbol = ?", "BTC/USDT").
   Order("hour DESC").
   Limit(24).
   Find(&summaries)

// Query latest indicators (continuous aggregate)
var latestIndicators []models.IndicatorLatest
db.Where("symbol = ? AND timeframe = ?", "BTC/USDT", "1h").
   Find(&latestIndicators)
```

### 3. New Models

**Trade model** for high-frequency individual trades:

```go
trade := models.Trade{
    EventTs:  time.Now(),
    Exchange: "BINANCE",
    Symbol:   "BTC/USDT",
    TradeID:  "12345",
    Price:    45000.50,
    Amount:   0.025,
    Side:     "buy",
}
db.Create(&trade)
```

**User and Currency models** for application users:

```go
// Create currency
currency := models.Currency{
    Name:   stringPtr("Bitcoin"),
    Symbol: stringPtr("BTC"),
    Type:   "crypto",
}
db.Create(&currency)

// Create user
user := models.User{
    CurrencyID: currency.ID,
    Username:   stringPtr("john_doe"),
    Email:      stringPtr("john@example.com"),
}
db.Create(&user)
```

---

## Migration Guide

### Step 1: Update Import Statements

```go
// If you had:
import "cryptoviz-backend/models"

// Keep the same, but models are now:
// - models.Candle (was CryptoData)
// - models.Indicator (was TechnicalIndicator)
// - models.News (was CryptoNews)
// - models.Trade (new)
// - models.User (new)
// - models.Currency (new)
```

### Step 2: Update Model References

**Find and replace in your codebase:**

| Find | Replace |
|------|---------|
| `models.CryptoData` | `models.Candle` |
| `models.TechnicalIndicator` | `models.Indicator` |
| `models.CryptoNews` | `models.News` |
| `.IntervalType` | `.Timeframe` |

### Step 3: Update Queries for Candles

**OLD:**
```go
var data []models.CryptoData
db.Where("symbol = ? AND interval_type = ?", symbol, interval).
   Order("time DESC").
   Find(&data)
```

**NEW:**
```go
var candles []models.Candle
db.Where("symbol = ? AND timeframe = ?", symbol, timeframe).
   Order("window_start DESC").
   Find(&candles)
```

### Step 4: Update Queries for Indicators

**OLD:**
```go
var indicators []models.TechnicalIndicator
db.Where("symbol = ? AND interval_type = ?", symbol, interval).
   Find(&indicators)
```

**NEW:**
```go
var indicators []models.Indicator
db.Where("symbol = ? AND timeframe = ?", symbol, timeframe).
   Find(&indicators)
```

### Step 5: Use Unified Views for Historical Data

**For queries spanning > 7 days, use unified views:**

```go
// This will query BOTH hot storage (last 7 days)
// AND cold storage (> 7 days) transparently
var allCandles []models.AllCandle
db.Where("symbol = ? AND timeframe = ? AND window_start >= ?",
         symbol, timeframe, thirtyDaysAgo).
   Order("window_start ASC").
   Find(&allCandles)
```

### Step 6: Remove AutoMigrate Calls

**If your code called `database.AutoMigrate()`, remove it:**

```go
// OLD - Remove this
if err := database.AutoMigrate(); err != nil {
    log.Fatal(err)
}

// NEW - Just connect
if err := database.Connect(); err != nil {
    log.Fatal(err)
}
```

---

## Repository Methods

### Candle Repository

```go
repo := models.NewCandleRepository(database.GetDB())

// Hot storage queries
candles, err := repo.GetBySymbol("BTC/USDT", "1h", 100)
candle, err := repo.GetLatest("BTC/USDT", "BINANCE")
candles, err := repo.GetByTimeRange("BTC/USDT", "1h", start, end)

// Unified views (hot + cold storage)
allCandles, err := repo.GetAllBySymbol("BTC/USDT", "1h", 1000)
allCandles, err := repo.GetAllByTimeRange("BTC/USDT", "1h", start, end)

// Continuous aggregate
summaries, err := repo.GetHourlySummary("BTC/USDT", "BINANCE", 24)

// Stats (uses database function)
stats, err := repo.GetStats("BTC/USDT", "1h")
```

### Indicator Repository

```go
repo := models.NewIndicatorRepository(database.GetDB())

// Hot storage queries
indicators, err := repo.GetBySymbolAndType("BTC/USDT", "rsi", "1h", 100)
indicator, err := repo.GetLatestByType("BTC/USDT", "macd", "1h")

// Unified views (hot + cold storage)
allIndicators, err := repo.GetAllBySymbolUnified("BTC/USDT", "1h")

// Continuous aggregate
latestIndicators, err := repo.GetLatestIndicators("BTC/USDT", "1h")
```

### News Repository

```go
repo := models.NewNewsRepository(database.GetDB())

// Hot storage queries
news, err := repo.GetAll(50)
news, err := repo.GetBySymbol("BTC/USDT", 50)
news, err := repo.GetBySource("yahoo", 50)

// Unified views (hot + cold storage)
allNews, err := repo.GetAllUnified(100)
allNews, err := repo.GetBySymbolUnified("BTC/USDT", 100)
```

### Trade Repository

```go
repo := models.NewTradeRepository(database.GetDB())

// Query methods
trades, err := repo.GetBySymbol("BTC/USDT", 1000)
trades, err := repo.GetByExchangeAndSymbol("BINANCE", "BTC/USDT", 1000)
trades, err := repo.GetByTimeRange("BTC/USDT", start, end)
trades, err := repo.GetLatestTrades(100)
```

### User Repository

```go
repo := models.NewUserRepository(database.GetDB())

user, err := repo.GetByID(uuid)
user, err := repo.GetByUsername("john_doe")
user, err := repo.GetByEmail("john@example.com")
users, err := repo.GetAll(limit, offset)
```

### Currency Repository

```go
repo := models.NewCurrencyRepository(database.GetDB())

currency, err := repo.GetBySymbol("BTC")
currencies, err := repo.GetByType("crypto")
currencies, err := repo.GetAll()
```

---

## Database Schema Summary

### Hypertables (Time-Series Data)

| Table | Partitioned By | Partition Column | Retention | Compression |
|-------|----------------|------------------|-----------|-------------|
| `trades` | `event_ts` | `exchange` | 24 hours | 7 days |
| `candles` | `window_start` | `symbol` | 2 years | 7 days |
| `indicators` | `time` | `symbol` | 1 year | 7 days |
| `news` | `time` | - | 6 months | 30 days |

### Regular Tables

| Table | Primary Key | Purpose |
|-------|-------------|---------|
| `users` | UUID | User accounts |
| `currencies` | UUID | Crypto & fiat metadata |

### Unified Views (Data Tiering)

| View | Combines | Purpose |
|------|----------|---------|
| `all_candles` | `candles` + `cold_storage.candles` | Transparent hot/cold query |
| `all_indicators` | `indicators` + `cold_storage.indicators` | Transparent hot/cold query |
| `all_news` | `news` + `cold_storage.news` | Transparent hot/cold query |

### Continuous Aggregates

| View | Refresh Policy | Purpose |
|------|----------------|---------|
| `candles_hourly_summary` | Every 1 hour | Hourly OHLCV from 1m/5m/15m candles |
| `indicators_latest` | Every 5 minutes | Latest indicator values per symbol |

---

## Testing

### Unit Tests

Update your tests to use new model names:

```go
func TestCandleRepository(t *testing.T) {
    // OLD
    data := &models.CryptoData{...}

    // NEW
    candle := &models.Candle{
        WindowStart: time.Now(),
        WindowEnd:   time.Now().Add(time.Hour),
        Exchange:    "BINANCE",
        Symbol:      "BTC/USDT",
        Timeframe:   "1h",
        Open:        45000.00,
        High:        45100.00,
        Low:         44900.00,
        Close:       45050.00,
        Volume:      1.5,
        Closed:      true,
    }
    err := repo.Create(candle)
    assert.NoError(t, err)
}
```

### Integration Tests

Test unified views for tiering:

```go
func TestUnifiedViews(t *testing.T) {
    // Create old data (will be tiered)
    oldCandle := &models.Candle{
        WindowStart: time.Now().Add(-10 * 24 * time.Hour),
        // ... rest of fields
    }

    // Query unified view
    var allCandles []models.AllCandle
    err := db.Where("symbol = ?", "BTC/USDT").Find(&allCandles).Error
    assert.NoError(t, err)

    // Should include both hot and cold data
    assert.Greater(t, len(allCandles), 0)
}
```

---

## Performance Considerations

### Query Patterns

**For recent data (< 7 days):**
- Use base tables (`candles`, `indicators`, `news`)
- Fast queries (<50ms)

**For historical data (> 7 days):**
- Use unified views (`all_candles`, `all_indicators`, `all_news`)
- Slower queries (200-500ms) but acceptable for backtesting

**For aggregated data:**
- Use continuous aggregates (`candles_hourly_summary`, `indicators_latest`)
- Pre-computed, very fast queries

### Indexing

All tables have optimized indexes created by `init.sql`:

- Composite indexes on `(exchange, symbol, time)` for trades
- Composite indexes on `(symbol, timeframe, window_start)` for candles
- Composite indexes on `(symbol, indicator_type, timeframe, time)` for indicators
- GIN index on `symbols` JSONB column for news

---

## Troubleshooting

### Error: "table crypto_data does not exist"

**Cause:** Code still references old table names
**Fix:** Update to use new table names (`candles`, `indicators`, `news`)

### Error: "column interval_type does not exist"

**Cause:** Code still references old column name
**Fix:** Update to use `timeframe` instead of `interval_type`

### Error: "column time does not exist" (in candles)

**Cause:** Candles now use `window_start` and `window_end` instead of `time`
**Fix:** Update queries to use `window_start` for time-based queries

### Models not found in GORM

**Cause:** GORM AutoMigrate trying to create tables
**Fix:** Remove all AutoMigrate calls - tables are created by `init.sql`

---

## Database Connection Changes

### New Utility Functions

```go
// Get database info for monitoring
info, err := database.GetDatabaseInfo()
// Returns:
// {
//   "timescaledb_version": "2.14.0",
//   "postgresql_version": "15.13",
//   "hypertables_count": 4,
//   "continuous_aggregates_count": 2,
//   "database_size": "156 MB"
// }
```

### Connection Flow

1. `database.Connect()` - Establishes connection
2. `verifyTimescaleDB()` - Checks TimescaleDB extension (automatic)
3. Database ready to use - NO AutoMigrate needed!

---

## Next Steps

1. **Update API handlers** to use new model names
2. **Update Kafka consumers** to write to correct tables
3. **Test unified views** for historical queries
4. **Update frontend** to use new API endpoints (if changed)
5. **Run integration tests** to verify data flow

---

## Questions?

**Database Schema:** See `database/init.sql` and `database/setup-tiering.sql`
**Data Tiering:** See `docs/DATA-TIERING-DEMO.md`
**Schema Documentation:** See `.claude/summaries/database-schema-migration.md`

**Status:** ✅ All models refactored and ready for use
