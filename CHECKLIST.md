# CryptoViz - Project Development Tasks

> **Last Updated:** 2025-11-20
> **Status:** Backend Complete, Frontend Integration In Progress
> **Version:** 0.6.0
> **Services:** backend-go ✅ | data-collector ✅ | frontend-vue 🟡 | news-scraper 🟡 | indicators-calculator 🟡

## 📋 Table of Contents

- [Recent Major Updates](#recent-major-updates)
- [Project Overview](#project-overview)
- [Core Requirements](#core-requirements)
- [Current Implementation Status](#current-implementation-status)
- [Task Checklist](#task-checklist)
- [Technical Debt](#technical-debt)
- [Future Enhancements](#future-enhancements)
- [Summary](#summary)

---

## 📋 Recent Major Updates

### 2025-11-20: Interval Migration & Kafka Reset ✅ 🆕
- **Objective**: Migrate aggregation intervals from `5s, 1m, 15m, 1h, 4h, 1d` to `1m, 5m, 15m, 1h, 1d`
- **Changes Across Entire Stack**:
  - **Configuration**: Updated .env with new intervals and interval-specific hot/cold retention policies
  - **Kafka Topics**: Deleted old topics (5s, 4h), created new topics (5m, 1d) with proper retention
  - **Data Collector**: Updated Python aggregator to use new timeframes `['1m', '5m', '15m', '1h', '1d']`
  - **Backend-Go**: Updated Kafka consumers, created ValidateInterval middleware
  - **Database**: Updated schema, cleanup functions, and hot/cold tiering with interval-specific retention
  - **Frontend**: Created IntervalSelector component and intervals constants
  - **Documentation**: Updated API docs with new supported intervals
  - **WebSocket Test**: Updated test-websocket.html buttons and functions
- **Kafka Full Reset**: All aggregated topics deleted and recreated with fresh 7-day backfill data
- **Hot/Cold Storage**: Implemented .env-based retention per interval
  - 1m: 7 days hot + 30 days cold = 37 total
  - 5m: 14 days hot + 90 days cold = 104 total
  - 15m: 30 days hot + 180 days cold = 210 total
  - 1h: 90 days hot + 730 days cold = 820 total
  - 1d: stays hot indefinitely (100 years)
- **Result**: Clean data pipeline with optimized retention policies, all services operational with new intervals

### 2025-11-20: API Symbol Routing Fix ✅
- **Issue Resolved**: Trading pair symbols (BTC/USDT) with slashes conflicting with URL routing
- **Solution Implemented**: Query parameters for symbol filtering (REST best practice)
- **Refactored** 5 REST API endpoints from path to query parameters
  - `/crypto/:symbol/data` → `/crypto/data?symbol=BTC/USDT`
  - `/crypto/:symbol/latest` → `/crypto/latest?symbol=BTC/USDT`
  - `/stats/:symbol` → `/stats?symbol=BTC/USDT`
  - `/indicators/:symbol/:type` → `/indicators/:type?symbol=BTC/USDT`
  - `/indicators/:symbol` → `/indicators?symbol=BTC/USDT`
- **Created** ValidateSymbolQuery middleware for symbol validation
- **Updated** 3 controller methods in CandleController, 2 in IndicatorController
- **Updated** API documentation with new query parameter format
- **Result**: Clean URLs, no encoding needed, BTC/USDT format preserved throughout system

### 2025-11-20: WebSocket Real-time Streaming Complete ✅
- **Implemented** complete WebSocket infrastructure with Hub/Client architecture
- **Created** internal/websocket/ package (hub.go, client.go)
- **Added** subscription filtering by type/symbol/timeframe with wildcard support
- **Integrated** Kafka consumers → WebSocket broadcast pipeline
- **Implemented** Read/Write pumps with ping/pong, connection management
- **Created** interactive HTML test client (test-websocket.html)
- **Connected** TradeHandler & CandleHandler to WebSocket Hub
- **Result**: Real-time data streaming from Kafka → backend-go → WebSocket clients (~630 trades/min)

### 2025-11-20: Kafka Library Migration (kafka-go → confluent-kafka-go) ✅ 🆕
- **Issue Identified**: segmentio/kafka-go FetchMessage() blocking indefinitely with consumer groups
  - Consumer offset stuck at `-`, 115K+ messages in Kafka but not consumed
  - Multiple fixes attempted (offset management, select removal, consumer group deletion)
- **Migrated** from kafka-go to confluent-kafka-go v2.3.0
  - Completely rewrote consumer.go using Poll() instead of FetchMessage()
  - Updated Dockerfile with librdkafka support (CGO_ENABLED=1, -tags dynamic)
  - Added librdkafka-dev build dependencies and librdkafka runtime
  - Resolved glibc/musl compatibility (Alpine Linux)
- **Result**: Consumers working perfectly, 0 lag, processing ~630 trades/minute

### 2025-01-07: Kafka Integration ✅
- **Implemented** complete Kafka consumer integration for real-time data ingestion
- **Created** internal/kafka/ package with 11 new files
- **Added** 4 topic-specific consumers: Candles, Trades, Indicators, News
- **Integrated** segmentio/kafka-go (later migrated to confluent-kafka-go)
- **Implemented** Redis-based deduplication, exponential backoff retry, graceful shutdown
- **Connected** data-collector (Python) → Kafka → backend-go → TimescaleDB pipeline
- **Result**: Real-time data flowing from Kafka to database, foundation for WebSocket streaming

### 2025-01-07: Go Standard Project Layout ✅
- **Restructured** to follow [golang-standards/project-layout](https://github.com/golang-standards/project-layout)
- **Created** `cmd/server/` for application entry point
- **Moved** all application code to `internal/` (prevents external imports)
- **Updated** all import paths to use `internal/` prefix
- **Updated** Dockerfile to build from `cmd/server`
- **Result**: Industry-standard Go project structure

### 2025-01-07: Clean Architecture Refactoring ✅
- **Refactored** backend-go from monolithic (512 lines main.go) to Clean Architecture
- **Created** modular structure: config/, dto/, middleware/, controllers/, routes/
- **Reduced** main.go to 94 lines (bootstrap only)
- **Implemented** Repository Pattern, Dependency Injection, Controller Pattern
- **Updated** all models: Candle, Indicator, News, Trade, User, Currency
- **Added** comprehensive [ARCHITECTURE.md](../services/backend-go/ARCHITECTURE.md) documentation
- **Result**: Production-ready, maintainable, testable codebase

### 2025-01-07: Database Schema Enhancement ✅
- **Implemented** TimescaleDB continuous aggregates (replacing materialized views)
- **Added** data tiering with hot/cold storage (SSD + S3/MinIO)
- **Created** unified views for transparent querying across storage tiers
- **Configured** automatic compression (70-90% savings) and retention policies
- **Added** 3 new models: Trade, User, Currency
- **Updated** [PRESENTATION-BACKEND-DB.md](../docs/PRESENTATION-BACKEND-DB.md) for demos

---

## Project Overview

**Purpose:** CryptoViz is a real-time cryptocurrency trading terminal with live data streaming, technical indicators, news aggregation, and interactive charts.

**System Architecture:**
```
┌──────────────────┐     ┌─────────────┐     ┌──────────────┐
│ Data Collector   │────▶│   Kafka     │────▶│  Backend-Go  │
│ (Binance WS)     │     │  (Topics)   │     │  (API + WS)  │
└──────────────────┘     └─────────────┘     └───────┬──────┘
                                                      │
┌──────────────────┐                                  │
│ News Scraper     │────────────────────────────────▶ │
│ (RSS Feeds)      │                                  │
└──────────────────┘                                  │
                                                      ▼
┌──────────────────┐                         ┌───────────────┐
│ Indicators Calc  │───────────────────────▶ │ TimescaleDB   │
│ (Python)         │                         │ + Redis       │
└──────────────────┘                         └───────────────┘
                                                      │
                                                      │ REST/WS
                                                      ▼
                                              ┌───────────────┐
                                              │ Frontend-Vue  │
                                              │ (Vue 3 + TS)  │
                                              └───────────────┘
```

**Tech Stack:**

**Backend (backend-go):**
- **Language:** Go 1.23
- **Framework:** Gin (REST API)
- **ORM:** GORM
- **WebSocket:** gorilla/websocket v1.5.1
- **Kafka:** confluent-kafka-go v2.3.0

**Data Pipeline:**
- **Database:** TimescaleDB (time-series PostgreSQL)
- **Cache:** Redis
- **Message Broker:** Apache Kafka
- **Data Collector:** Python (Binance WebSocket)
- **News Scraper:** Python (RSS feeds)
- **Indicators:** Python (TA-Lib calculations)

**Frontend (frontend-vue):**
- **Framework:** Vue 3 + TypeScript
- **UI Library:** Naive UI
- **Charts:** Chart.js + D3.js
- **State:** Pinia
- **WebSocket:** Socket.io-client
- **Build:** Vite

**Infrastructure:**
- **Containerization:** Docker + Docker Compose
- **Reverse Proxy:** Nginx
- **Monitoring:** Grafana + Prometheus (planned)

---

## Core Requirements

### ✅ Must Have
- [x] **REST API** - HTTP endpoints for data access
- [x] **WebSocket Support** - Real-time bidirectional communication
- [x] **Database Integration** - TimescaleDB for time-series data
- [x] **Real-time Data Streaming** - Push live updates to clients ✅ NEW
- [x] **Producer/Consumer Paradigm** - Kafka integration (confluent-kafka-go) ✅ NEW
- [x] **Temporal Dimension** - Time-range queries and historical data
- [x] **Fast & Competitive** - Optimized for performance (~630 trades/min, 3-4ms latency)
- [x] **Clean Architecture** - Maintainable, testable codebase
- [x] **Go Best Practices** - Standard project layout

---

## Current Implementation Status

### ✅ Completed Features

#### 1. Project Structure & Configuration (Updated 2025-01-07)
- [x] Go module initialization
- [x] Environment variable configuration
- [x] Docker multi-stage build (updated for cmd/server)
- [x] Docker Compose integration
- [x] **Go Standard Project Layout** (cmd/, internal/)
- [x] Configuration management package (internal/config)

#### 2. Database Layer (Enhanced 2025-01-07)
- [x] GORM integration
- [x] TimescaleDB connection with pool configuration
- [x] **Database-first approach** using init.sql (AutoMigrate removed)
- [x] **Hypertable setup** with time + space partitioning (4 hypertables)
- [x] **Continuous aggregates** for performance optimization
- [x] **Data tiering** (hot/cold storage) with unified views
- [x] **Compression and retention** policies
- [x] Health check functionality

#### 3. Data Models & Repositories (Refactored 2025-01-07)
- [x] **6 Complete Models** with Repository Pattern
  - [x] Candle (OHLCV data, 7 repository methods)
  - [x] Indicator (Technical indicators, continuous aggregates)
  - [x] News (With sentiment, JSONB arrays)
  - [x] Trade (High-frequency data)
  - [x] User (UUID primary key)
  - [x] Currency (Crypto & fiat metadata)

#### 4. REST API (Clean Architecture 2025-01-07)
- [x] Gin router with modular routing (internal/routes)
- [x] CORS middleware (internal/middleware)
- [x] Custom logging middleware
- [x] Standardized API response DTOs (internal/dto)
- [x] **Controller pattern** (6 controllers in internal/controllers)
- [x] **Dependency injection** container
- [x] All 9 REST endpoints implemented

#### 5. WebSocket (Complete Overhaul 2025-11-20) ✅
- [x] **WebSocket Hub** (internal/websocket/hub.go)
  - Central connection manager with broadcast channel (1000 buffer)
  - Thread-safe client map with RWMutex
  - Hub statistics (connections, messages broadcast/delivered)
  - Helper methods: Broadcast(), BroadcastTrade(), BroadcastCandle()
- [x] **WebSocket Client** (internal/websocket/client.go)
  - Individual connection handling with Read/Write pumps
  - Subscription system with type/symbol/timeframe filtering
  - Wildcard support (`*` for all symbols)
  - Message buffer (256 per client)
  - Ping/pong mechanism (Read: 60s, Write: 10s, Ping: 54s)
- [x] **WebSocket Controller** (internal/controllers/websocket_controller.go)
  - Handle() - WebSocket upgrade and client registration
  - Stats() - Hub statistics endpoint
- [x] **Kafka Integration**
  - TradeHandler broadcasts to WebSocket after DB insertion
  - CandleHandler broadcasts closed candles only
  - BroadcastFunc callback pattern
- [x] **Endpoints**
  - `ws://localhost:8080/ws/crypto` - WebSocket endpoint
  - `GET /ws/stats` - Hub statistics
- [x] **Test Client** (test-websocket.html)
  - Interactive browser-based WebSocket test client
  - Real-time trade/candle display
  - Subscription management UI
  - Live statistics dashboard

#### 6. Infrastructure
- [x] Redis client initialization
- [x] Graceful shutdown with proper cleanup
- [x] Structured logging (JSON format)
- [x] Clean bootstrap (94 lines in cmd/server/main.go)

---

## Task Checklist

### Phase 1: Core Infrastructure ✅ COMPLETED

- [x] Setup Go project structure
- [x] Configure dependencies
- [x] Dockerfile multi-stage build
- [x] Database integration (GORM + TimescaleDB)
- [x] Data models with Repository Pattern
- [x] REST API foundation
- [x] **Clean Architecture refactoring**
- [x] **Go Standard Project Layout**

---

### Phase 2: Kafka Integration ✅ COMPLETED (with Library Migration)

**Priority:** 🔴 **HIGHEST** - Required for real-time functionality

#### 2.1 Kafka Consumer Setup ✅
- [x] Install Kafka client library (~~`github.com/segmentio/kafka-go`~~ → `confluent-kafka-go/v2 v2.3.0`)
- [x] Create Kafka consumer manager (internal/kafka/manager.go)
- [x] Consumer connection configuration (internal/kafka/config.go)
- [x] Error handling and reconnection logic (internal/kafka/utils/retry.go)

#### 2.2 Topic Consumers ✅
- [x] Candles Consumer (crypto.aggregated.* → candles table)
- [x] Trades Consumer (crypto.raw.trades → trades table)
- [x] Technical Indicators Consumer (crypto.indicators.* → indicators table)
- [x] News Consumer (crypto.news → news table)

#### 2.3 Integration ✅
- [x] Start consumers on app startup (cmd/server/main.go)
- [x] Message routing to handlers (BaseConsumer + MessageHandler interface)
- [x] Message deduplication (Redis-based with TTL)
- [x] Graceful shutdown with offset commits
- [x] Build verified (21MB binary)

#### 2.4 Library Migration (2025-11-20) ✅
- [x] **Issue Resolved**: kafka-go FetchMessage() blocking with consumer groups
  - Symptom: Consumer offset stuck at `-`, no messages consumed despite 115K+ in topics
  - Root cause: kafka-go consumer group blocking behavior
  - Attempted fixes: offset management, select removal, consumer deletion (all failed)
- [x] **Migration**: Switched to confluent-kafka-go
  - Complete rewrite of consumer.go using Poll() instead of FetchMessage()
  - Updated go.mod: added confluent-kafka-go/v2 v2.3.0
  - Updated Dockerfile: CGO_ENABLED=1, -tags dynamic, librdkafka dependencies
  - Fixed Alpine Linux compatibility (glibc → musl with dynamic linking)
- [x] **Generic Message Handling**: Changed from kafka-go.Message to interface{}
  - Created HandlerMessage struct for handler compatibility
  - Updated all handlers (trades, candles, indicators, news) to use interface{}
  - Preserved header parsing functionality with utils.MessageHeader

**Implementation Details:**
- ✅ 11 new files created in internal/kafka/
- ✅ confluent-kafka-go library (requires CGO + librdkafka)
- ✅ Consumer groups for load balancing (working perfectly)
- ✅ Exponential backoff retry logic
- ✅ Header parsing and validation
- ✅ Integration with existing repositories
- ✅ Updated ARCHITECTURE.md with Kafka flows

**Performance Metrics:**
- ✅ **Processing Rate**: ~630 trades/minute
- ✅ **Insert Latency**: 3-4ms per record
- ✅ **Consumer Lag**: 0 (all caught up)
- ✅ **Database**: 69K+ trades, 21K+ candles

**Actual Effort:** ~1 day initial + ~3 hours migration (completed 2025-11-20)

---

### Phase 3: WebSocket Real-time Streaming ✅ COMPLETED

**Priority:** 🔴 **HIGH**

#### 3.1 Connection Management ✅
- [x] WebSocket Hub with connection pool (internal/websocket/hub.go)
- [x] Client registration/unregistration system
- [x] Session management with goroutines (ReadPump/WritePump)
- [x] Subscription tracking per client with flexible filtering
  - [x] Subscribe by type (trade, candle)
  - [x] Subscribe by symbol (BTC/USDT or `*` wildcard)
  - [x] Subscribe by timeframe (1m, 5m, 15m, 1h, 1d for candles)
  - [x] List subscriptions action
- [x] Connection statistics (active connections, total, messages)

#### 3.2 Message Broadcasting ✅
- [x] Central broadcast manager in Hub
- [x] Targeted broadcast by symbol with subscription filtering
- [x] Message buffering (1000 in Hub, 256 per client)
- [x] Non-blocking broadcast with channel select
- [x] Automatic client disconnection on buffer full
- [x] Message types: trade, candle, ack, error, pong

#### 3.3 Integration with Kafka ✅
- [x] Forward crypto trades to WebSocket clients (TradeHandler)
- [x] Forward candles to WebSocket clients (CandleHandler - closed only)
- [x] Forward news to WebSocket clients (NewsHandler) ✅ 2025-11-20
- [x] BroadcastFunc callback pattern
- [x] SetBroadcast() methods on handlers
- [x] Connected in main.go during startup
- [ ] Forward indicators to WebSocket clients (pending)

#### 3.4 Protocol & Testing ✅
- [x] JSON message protocol defined
- [x] Client message handling (action: subscribe, unsubscribe, ping, list_subscriptions)
- [x] Server message format (type, data, timestamp)
- [x] Ping/pong mechanism (60s read timeout, 10s write timeout, 54s ping period)
- [x] Interactive HTML test client (test-websocket.html)
  - Real-time message display
  - Subscription management UI
  - Live statistics (trades, candles, latest price)
  - Connection status indicator
  - Quick subscription buttons

#### 3.5 Endpoints ✅
- [x] `ws://localhost:8080/ws/crypto` - WebSocket streaming endpoint
- [x] `GET /ws/stats` - Hub statistics (JSON)

**Implementation Files:**
- ✅ internal/websocket/hub.go (~250 lines)
- ✅ internal/websocket/client.go (~350 lines)
- ✅ internal/controllers/websocket_controller.go (updated)
- ✅ internal/controllers/dependencies.go (added WSHub)
- ✅ internal/kafka/consumers/trades.go (added broadcast)
- ✅ internal/kafka/consumers/candles.go (added broadcast)
- ✅ cmd/server/main.go (Hub initialization)
- ✅ test-websocket.html (~300 lines)

**Actual Effort:** ~4 hours (completed 2025-11-20)
**Depends on:** Phase 2 (Kafka Integration) ✅

---

### Phase 3.5: Frontend-Vue Integration ✅ COMPLETE (2025-11-20)

**Priority:** 🔴 **HIGHEST** - User interface for the platform

#### 3.5.1 WebSocket Integration ✅
- [x] Frontend WebSocket client (websocket.ts, rt.ts)
- [x] Mock Binance connection for development
- [x] **Connected to backend-go WebSocket** (`ws://localhost:8080/ws/crypto`)
- [x] Updated message handlers for backend format
- [x] Updated subscription types (trades → trade, candles → candle)
- [x] Fixed symbol format (BTCUSDT → BTC/USDT)
- [ ] Test real-time trade streaming (requires running services)
- [ ] Test real-time candle updates (requires running services)

#### 3.5.2 REST API Integration ✅
- [x] HTTP client setup (http.ts, axios)
- [x] API service modules (crypto.api.ts, markets.api.ts, indicators.api.ts)
- [x] **Connected to backend-go REST API** (`http://localhost:8080/api/v1`)
- [x] Updated all endpoints to use query parameters (symbol, interval)
- [x] Implemented historical data fetching (fetchCandles)
- [x] Implemented indicators API calls (fetchRSI, fetchMACD, fetchBollinger, fetchMomentum)
- [x] Updated stats endpoint to use query parameters
- [x] **Data transformation for backend format** (window_start → time mapping)
- [x] **Fixed interval naming** (24h → 1d across all components)
- [ ] Error handling and retry logic (basic implementation exists)
- [ ] Loading states and caching (basic implementation exists)

#### 3.5.3 UI Components ✅
- [x] **Core Components** (10 components)
  - [x] CryptoPricePanel - Real-time price display
  - [x] NewsFeed - News articles display
  - [x] IndicatorsPanel - Technical indicators
  - [x] CommunitySentiment - Sentiment analysis
  - [x] DashboardHeader - Navigation
  - [x] IntervalSelector - Timeframe selection (uses 1m, 5m, 15m, 1h, 1d)
  - [x] ViewModeToggle - Chart view modes
- [x] Chart.js integration for candlestick charts
- [x] D3.js for advanced visualizations
- [x] Pinia stores (market.ts, indicators.ts)
- [x] **Connected to real data sources** (VITE_USE_MOCK=false)
- [ ] Polish UI/UX (future enhancement)
- [ ] Responsive design testing (future enhancement)

#### 3.5.4 Build & Deployment ✅
- [x] Vite build configuration
- [x] Dockerfile with Nginx
- [x] TypeScript configuration
- [x] **Environment variables (.env integration)**
- [x] **Docker Compose integration with build args**
- [ ] Build and test in production mode (requires deployment)
- [x] **CORS configuration with backend** (already configured in backend-go)

**Current Status:**
- ✅ Project structure and dependencies complete
- ✅ UI components and layouts built
- ✅ WebSocket client connected to backend-go
- ✅ **Integrated with backend-go REST API endpoints**
- ✅ **Connected to real data sources (mock mode disabled)**
- ✅ All API services updated for backend-go format
- ✅ Symbol format unified (BTC/USDT)
- ✅ Environment variables configured

**Changes Made (2025-11-20):**
1. **Environment Configuration**
   - Added VITE_API_URL, VITE_WS_URL, VITE_USE_MOCK to root .env
   - Updated docker-compose.yml to pass environment variables as build args

2. **REST API Integration**
   - [markets.api.ts](services/frontend-vue/src/services/markets.api.ts): Updated fetchCandles to use query params (/api/v1/crypto/data?symbol=BTC/USDT)
   - [markets.api.ts](services/frontend-vue/src/services/markets.api.ts): Updated fetchIndicators to use query params (/api/v1/indicators/{type}?symbol=BTC/USDT)
   - [indicators.api.ts](services/frontend-vue/src/services/indicators.api.ts): Updated all indicator functions (RSI, MACD, Bollinger, Momentum)
   - [crypto.api.ts](services/frontend-vue/src/services/crypto.api.ts): Updated stats endpoint to use query params

3. **WebSocket Integration**
   - [market.ts](services/frontend-vue/src/stores/market.ts): Updated subscription types (trades → trade, candles → candle)
   - [market.ts](services/frontend-vue/src/stores/market.ts): Changed default symbol from BTCUSDT to BTC/USDT
   - [rt.ts](services/frontend-vue/src/services/rt.ts): Already compatible with backend-go format

4. **Chart Data Fixes (2025-11-20 Session)** 🆕
   - [markets.api.ts](services/frontend-vue/src/services/markets.api.ts): Added data transformation for REST API (lines 36-45)
     - Maps backend `window_start` field → frontend `time` field
     - Strips extra backend fields (window_end, exchange, trade_count, etc.)
     - Returns clean CandleDTO format for Chart.js
   - [market.ts](services/frontend-vue/src/stores/market.ts): Added WebSocket candle transformation (lines 53-65)
     - Same `window_start` → `time` mapping for real-time updates
     - Ensures consistency between historical and live data
   - [CandleChart.vue](services/frontend-vue/src/components/charts/CandleChart.vue): Fixed time format (line 240)
     - Changed from string `candle.time` to numeric `new Date(candle.time).getTime()`
     - chartjs-chart-financial library requires numeric timestamps, not strings
     - Line charts were more lenient, but candlestick charts are stricter
   - **Interval Naming (24h → 1d):**
     - [TradingChart.vue](services/frontend-vue/src/components/charts/TradingChart.vue): Updated timeframes array (line 113)
     - [TradingChart.vue](services/frontend-vue/src/components/charts/TradingChart.vue): Updated limit calculation (line 168)
     - [TradingChart.vue](services/frontend-vue/src/components/charts/TradingChart.vue): Updated TypeScript types (line 598)
     - [indicators.ts](services/frontend-vue/src/stores/indicators.ts): Updated state and action types (lines 9, 36)
   - **Result**: Both line and candlestick charts now working with backend-go data

5. **Configuration**
   - Updated [docker-compose.yml](docker-compose.yml:293-320) frontend service with environment variables
   - Created environment variable section in root [.env](.env:29-38)

**Implementation Files:**
- ✅ [services/frontend-vue/src/services/](services/frontend-vue/src/services/) (API clients updated)
- ✅ [services/frontend-vue/src/components/](services/frontend-vue/src/components/) (10 components)
- ✅ [services/frontend-vue/src/stores/market.ts](services/frontend-vue/src/stores/market.ts) (Pinia state updated)
- ✅ [services/frontend-vue/src/constants/intervals.ts](services/frontend-vue/src/constants/intervals.ts) (1m, 5m, 15m, 1h, 1d)
- ✅ [services/frontend-vue/Dockerfile](services/frontend-vue/Dockerfile)
- ✅ [docker-compose.yml](docker-compose.yml) (frontend service configured)
- ✅ [.env](.env) (VITE_* variables added)

**Testing Instructions:**
To test the frontend integration:
```bash
# Option 1: Run frontend in development mode
cd services/frontend-vue
npm install
npm run dev
# Access at http://localhost:5173

# Option 2: Run with Docker Compose (production mode)
docker-compose --profile ui up -d frontend-vue
# Access at http://localhost:3000
```

**Actual Effort:** 1-2 hours (configuration and API alignment)
**Depends on:** Phase 3 (WebSocket Streaming) ✅

---

### Phase 4: News Scraper Integration ✅ COMPLETE

**Priority:** 🟠 **HIGH** - News feed functionality + scalability demonstration

#### 4.1 Current Implementation ✅
- [x] RSS feed scraping (CoinDesk, Bitcoin.com)
- [x] Crypto keyword detection (BTC, ETH, XRP, ADA, SOL, DOGE, BNB, LTC)
- [x] Article storage (articles.json)
- [x] Basic UI (ui.py)
- [x] Sentiment analysis placeholder (sentiment.py)

#### 4.2 Architecture Redesign (POC for Scalability) ✅
- [x] **ARCHITECTURE.md created** - Scalable multi-source design
- [x] **Base Source Interface** (abstract class for all sources)
- [x] **Article Data Model** (dataclass with Kafka serialization)
- [x] **Source Manager** (orchestrates multiple sources)
- [x] **Config System** (YAML-based source configuration)
- [x] **Directory Structure** (core/, sources/, models/)

**New Architecture Benefits:**
- ✅ **Easy to add sources**: Just create new file + 3 lines config
- ✅ **POC stubs**: Twitter, Reddit, Bloomberg (demonstrate scalability)
- ✅ **Priority system**: Control source execution order
- ✅ **Enable/disable sources**: Via config.yaml
- ✅ **Independent testing**: Mock any source

#### 4.3 Core Components ✅
- [x] **Kafka Producer** (confluent-kafka)
  - Message format matching NewsHandler
  - Callback-based delivery reports
  - Async producer with batching (16KB, 100ms linger)
- [x] **VADER Sentiment Analysis** (vaderSentiment)
  - Score calculation (-1 to +1)
  - Integration with Article model
- [x] **Crypto Detector** (refactored from scraper.py)
  - Multi-symbol detection
  - Configurable keywords from config.yaml
- [x] **Logging** (logging.basicConfig for simplicity)

#### 4.4 Source Implementations ✅
- [x] **RSSSource** (refactored from scraper.py)
  - CoinDesk RSS feeds (4 feeds, priority 1)
  - CoinTelegraph RSS feeds (3 feeds, priority 2)
  - Async fetching with aiohttp
  - Date filtering (days_back configuration)
- [x] **TwitterSource (POC stub)**
  - Demonstrates Twitter API integration
  - enabled=false by default
  - Shows how to add social media
- [x] **RedditSource (POC stub)**
  - Demonstrates Reddit API integration
  - enabled=false by default
- [x] **BloombergSource (POC stub)**
  - Demonstrates financial news scraping
  - enabled=false by default

#### 4.5 Kafka Integration ✅
- [x] Connect to `crypto.news` topic
- [x] Message serialization (JSON via to_kafka_message())
- [x] Deduplication (based on URL)
- [x] Batch publishing with flush
- [x] Error recovery and delivery callbacks

#### 4.6 Testing & Documentation ✅
- [x] **Integration test with Kafka → backend-go** ✅
  - news-scraper publishing to crypto.news topic
  - backend-go NewsHandler consuming messages
  - 243 articles stored in TimescaleDB
  - REST API endpoints verified (GET /api/v1/news, /api/v1/news/{symbol})
- [x] **Docker integration** ✅
  - Built and deployed with docker-compose
  - Environment variable configuration working
  - Kafka topic auto-created and registered in .env
- [x] **End-to-end pipeline verified** ✅
  - CoinDesk (priority 1) + CoinTelegraph (priority 2) active
  - 76 articles/cycle with symbol detection
  - Sentiment analysis working (-0.90 to 0.0 range)
  - 60s polling interval
- [ ] Unit tests for each source (future enhancement)
- [ ] Performance metrics dashboard (future enhancement)

**Current Status:**
- ✅ **Architecture designed** (ARCHITECTURE.md created)
- ✅ **Refactored to modular structure** (core/, sources/, models/)
- ✅ **Kafka producer implemented** (confluent-kafka with batching)
- ✅ **VADER sentiment implemented** (vaderSentiment integration)
- ✅ **CoinDesk configured as priority 1** (4 RSS feeds enabled)
- ✅ **Integration testing COMPLETE** (Docker + Kafka + backend-go + TimescaleDB + REST API)

**Implementation Files:**
```
services/news-scraper/
├── ARCHITECTURE.md ✅ (scalable multi-source design doc)
├── app.py ✅ (orchestrator with source manager, env var support)
├── config.yaml ✅ (CoinDesk + CoinTelegraph enabled, POC stubs disabled)
├── requirements.txt ✅ (cleaned dependencies: aiohttp, feedparser, confluent-kafka, vaderSentiment)
├── Dockerfile ✅ (fixed CMD to run app.py)
├── core/
│   ├── kafka_producer.py ✅ (confluent-kafka with batching, delivery callbacks)
│   ├── sentiment.py ✅ (VADER sentiment analysis -1 to +1)
│   └── crypto_detector.py ✅ (configurable symbol detection)
├── sources/
│   ├── base_source.py ✅ (abstract base class, plugin architecture)
│   ├── rss_source.py ✅ (async RSS with aiohttp, date filtering)
│   ├── twitter_source.py ✅ (POC stub, demonstrates social media integration)
│   ├── reddit_source.py ✅ (POC stub, demonstrates PRAW integration)
│   └── bloomberg_source.py ✅ (POC stub, demonstrates web scraping)
└── models/
    └── article.py ✅ (dataclass with to_kafka_message() matching NewsHandler format)

Configuration:
├── .env ✅ (added crypto.news to KAFKA_TOPICS)
└── docker-compose.yml ✅ (news-scraper service configured with KAFKA_BROKERS env var)
```

**Test Results:**
- ✅ 243 articles stored in TimescaleDB
- ✅ 76 articles published per 60s cycle
- ✅ Symbol detection working (btc, eth, ltc, etc.)
- ✅ Sentiment analysis working (-0.90 to 0.0 range)
- ✅ REST API: GET /api/v1/news and /api/v1/news/{symbol}
- ✅ Pipeline: news-scraper → Kafka → backend-go → TimescaleDB → API

**Actual Effort:** 1 day (2025-11-20)
**Depends on:** Phase 2 (Kafka Integration) ✅, Phase 1 (Backend-Go Core) ✅
**Demonstrates:** Scalability, modularity, production-ready patterns, Abstract Factory pattern

---

### Phase 5: API Enhancement 🟡 PARTIAL

**Priority:** 🟡 **MEDIUM** - Backend improvements

#### 5.1 Symbol Routing & Validation ✅ COMPLETE (2025-11-20)
- [x] **Symbol validator middleware** (middleware/symbol_validator.go)
  - ValidateSymbolQuery() for trading pair symbols (BASE/QUOTE format)
  - Validates query parameter: `?symbol=BTC/USDT`
  - Normalizes symbols to uppercase
  - Stores validated symbol in context
  - Clear error messages with examples
- [x] **API routes refactored** (routes.go)
  - Changed from path parameters (`/crypto/:symbol/data`) to query parameters (`/crypto/data?symbol=BTC/USDT`)
  - Affected 5 endpoints: crypto/data, crypto/latest, stats, indicators/:type, indicators
  - News endpoint uses simple tokens (btc, eth) - no slash issue
- [x] **Controllers updated**
  - CandleController: 3 methods (GetCandleData, GetLatestPrice, GetStats)
  - IndicatorController: 2 methods (GetByType, GetAll)
  - All read normalized symbol from context
- [x] **API documentation updated** (docs/api.md)
  - Added symbol format explanation section
  - Updated all 5 endpoint examples with query parameters
  - Added error codes section

**Solution:** Query parameters (Option 2) chosen for REST best practices
- ✅ No URL encoding needed for slashes
- ✅ Preserves BTC/USDT format throughout system (DB, Kafka, WebSocket)
- ✅ Industry standard for filtering/selection
- ✅ Zero data migration required

#### 5.2 Additional Enhancements ⏳
- [ ] Standardized error codes
- [ ] Pagination (cursor-based)
- [ ] Advanced filtering & sorting
- [ ] Rate limiting per endpoint

**Estimated Effort:** 2-3 days remaining

---

### Phase 6: Monitoring & Observability ❌ TODO

**Priority:** 🟡 **MEDIUM** - Production monitoring

#### 6.1 Grafana Dashboard
- [ ] **Grafana setup** (Docker Compose)
- [ ] TimescaleDB data source configuration
- [ ] Trading metrics dashboard (trades/min, volume, prices)
- [ ] System health dashboard (CPU, memory, connections)
- [ ] Kafka consumer lag monitoring
- [ ] WebSocket connection metrics
- [ ] Alert rules configuration

#### 6.2 Prometheus Integration
- [ ] Prometheus server setup
- [ ] Backend-go metrics endpoint (`/metrics`)
- [ ] Custom metrics (requests, latency, errors)
- [ ] Kafka consumer metrics
- [ ] WebSocket metrics

#### 6.3 Logging
- [ ] Centralized logging (ELK stack or Loki)
- [ ] Log aggregation from all services
- [ ] Log parsing and indexing

**Estimated Effort:** 4-5 days

---

### Phase 7: Testing & Quality ❌ TODO

**Priority:** 🟠 **HIGH** - Code quality and reliability

- [ ] Unit tests for repositories
- [ ] Unit tests for controllers
- [ ] Unit tests for frontend components
- [ ] Integration tests (backend-go ↔ Kafka ↔ DB)
- [ ] E2E tests (frontend ↔ backend)
- [ ] Load testing (Apache JMeter / k6)
- [ ] WebSocket stress testing

**Target Coverage:** 70%+
**Estimated Effort:** 5-7 days

---

### Phase 8: Production Readiness ❌ TODO

**Priority:** 🔴 **HIGH** - Security and scalability

- [ ] Authentication & Authorization (JWT)
- [ ] Rate limiting (backend API)
- [ ] Redis caching implementation
- [ ] API security (HTTPS, headers)
- [ ] WebSocket authentication
- [ ] Input sanitization
- [ ] CORS configuration
- [ ] Secrets management
- [ ] Horizontal scaling setup

**Estimated Effort:** 6-8 days

---

### Phase 9: Documentation 🟡 PARTIAL (60%)

**Priority:** 🟡 **MEDIUM** - Project documentation

#### 9.1 API Documentation ⏳
- [ ] Complete docs/api.md
- [ ] Swagger/OpenAPI specification
- [ ] WebSocket protocol documentation
- [ ] Frontend integration guide

#### 9.2 Code Documentation ✅
- [x] Comprehensive comments in controllers
- [x] Package-level documentation

#### 9.3 Architecture Documentation 🟡
- [x] Complete ARCHITECTURE.md guide
- [x] Component descriptions
- [x] Data flow diagrams
- [x] Before/after comparison
- [ ] System architecture diagrams (mermaid)
- [ ] Deployment guide
- [ ] Docker Compose documentation

#### 9.4 User Documentation ⏳
- [ ] Installation guide
- [ ] Configuration guide
- [ ] User manual
- [ ] FAQ

**Estimated Effort:** 2-3 days remaining

---

## Technical Debt

### Current Issues

2. **⚠️ Redis Not Used for Caching** 🟠 HIGH
   - Performance not optimized
   - Priority: High
   - Effort: 2-3 days

4. **⚠️ No Tests** 🟠 HIGH
   - Code quality uncertain
   - Priority: High
   - Effort: 5-7 days

5. **⚠️ No Authentication** 🟠 MEDIUM-HIGH
   - API publicly accessible
   - Priority: Medium-High (production)
   - Effort: 3-4 days

### ✅ Resolved Issues (2025-11-20)

1. **✅ RESOLVED: WebSocket Only Echoes Messages** 🆕 (2025-11-20)
   - Was: Real-time streaming not functional, no Kafka → WebSocket bridge
   - Now: Complete WebSocket Hub with Kafka integration
   - Solution: Implemented internal/websocket/ package with Hub/Client architecture
   - Result: Live data streaming to clients (~630 trades/min)

2. **✅ RESOLVED: kafka-go Consumer Blocking** 🆕 (2025-11-20)
   - Was: FetchMessage() blocking indefinitely, consumer offset stuck at `-`
   - Now: confluent-kafka-go Poll() working perfectly, 0 lag
   - Solution: Migrated to confluent-kafka-go with Poll(), updated Dockerfile for CGO
   - Result: Consumers processing 630 trades/min with 3-4ms latency

3. **✅ RESOLVED: No Kafka Integration** (2025-01-07)
   - Was: No real-time data ingestion, blocker for WebSocket
   - Now: Complete Kafka consumer integration operational
   - Solution: Implemented internal/kafka/ package with 4 consumers
   - Result: Data flowing from data-collector → Kafka → backend-go → TimescaleDB

4. **✅ RESOLVED: Monolithic main.go**
   - Was: 512 lines, everything in one file
   - Now: 120 lines, bootstrap only
   - Solution: Clean Architecture + Go Standard Layout

5. **✅ RESOLVED: Not following Go conventions**
   - Was: Flat structure, no cmd/ or internal/
   - Now: Follows golang-standards/project-layout
   - Solution: Restructured to cmd/ and internal/

---

## Future Enhancements

- GraphQL API
- gRPC Support
- Advanced Analytics
- Alerting System
- Multi-tenancy
- Data Export (CSV, PDF)
- Machine Learning Integration

---

## Summary

### Completion Status

| Phase | Status | Progress | Priority |
|-------|--------|----------|----------|
| **Phase 1: Backend-Go Core** | **✅ Complete** | **100%** | ✅ |
| **Phase 1.5: Clean Architecture** | **✅ Complete** | **100%** | ✅ |
| **Phase 1.6: Go Standard Layout** | **✅ Complete** | **100%** | ✅ |
| **Phase 2: Kafka Integration** | **✅ Complete** | **100%** | ✅ |
| **Phase 2.5: Kafka Library Migration** | **✅ Complete** | **100%** | ✅ |
| **Phase 3: WebSocket Streaming** | **✅ Complete** | **100%** | ✅ |
| **Phase 3.5: Frontend-Vue Integration** | **✅ Complete** | **95%** | 🔴 **HIGHEST** |
| **Phase 4: News Scraper Integration** | **✅ Complete** | **100%** | 🟠 |
| **Phase 5: API Enhancement** | **🟡 In Progress** | **70%** | 🟡 |
| Phase 6: Monitoring & Observability | ❌ Not Started | 0% | 🟡 |
| Phase 7: Testing & Quality | ❌ Not Started | 0% | 🟠 |
| Phase 8: Production Readiness | ❌ Not Started | 10% | 🔴 |
| Phase 9: Documentation | 🟡 Partial | 60% | 🟡 |
| **Overall Progress** | **🟢 Backend + Frontend Complete** | **~75%** | - |

### Immediate Next Steps (Priority Order)

1. **🔴 HIGHEST: Test Frontend Integration** (Phase 3.5 - Testing)
   - [ ] Run frontend in development mode (`npm run dev`)
   - [ ] Test WebSocket connection to backend-go
   - [ ] Test REST API endpoints
   - [ ] Verify real-time trade/candle streaming
   - [ ] Test with Docker Compose (`--profile ui`)

2. **🟠 HIGH: News Scraper → Kafka** (Phase 4)
   - [ ] Add Kafka producer to news-scraper
   - [ ] Publish to `crypto.news` topic
   - [ ] Implement sentiment analysis
   - [ ] Verify NewsHandler receives data

3. **🟡 MEDIUM: Grafana Setup** (Phase 6)
   - [ ] Add Grafana to Docker Compose
   - [ ] Create trading metrics dashboard
   - [ ] Add Prometheus metrics endpoint

4. **🟠 HIGH: Testing** (Phase 7)
   - [ ] Unit tests for backend-go
   - [ ] Integration tests
   - [ ] E2E tests with frontend

5. **🔴 HIGH: Production Security** (Phase 8)
   - [ ] JWT authentication
   - [ ] Rate limiting
   - [ ] HTTPS configuration

### Timeline Estimate

- **Backend MVP:** ✅ **ACHIEVED** (Kafka + WebSocket + Real-time streaming)
- **Frontend Integration:** 🔴 **IN PROGRESS** (2-3 days remaining)
- **Full MVP (with UI):** 1 week (frontend + news scraper + basic monitoring)
- **Production-Ready:** 3-4 weeks (tests + security + monitoring + polish)

---

## WebSocket Testing Instructions

### Quick Test (Browser)
1. Open `test-websocket.html` in browser
2. Click "Connect"
3. Click "Subscribe All Trades" to see live trades
4. Click "Subscribe 5m Candles (All)" to see live candles

### CLI Test (wscat)
```bash
# Install wscat
npm install -g wscat

# Connect
wscat -c ws://localhost:8080/ws/crypto

# Subscribe to BTC trades
> {"action":"subscribe","type":"trade","symbol":"BTC/USDT"}

# Subscribe to all 5m candles
> {"action":"subscribe","type":"candle","symbol":"*","timeframe":"5m"}
```

### Verify Performance
```bash
# Check WebSocket stats
curl http://localhost:8080/ws/stats | jq .

# Check recent database activity
docker exec cryptoviz-timescaledb psql -U postgres -d cryptoviz \
  -c "SELECT COUNT(*) FROM trades WHERE created_at > NOW() - INTERVAL '1 minute';"
```

---

## Recent Updates

### 2025-11-20: Interval Migration & Kafka Reset ✅
- **Interval Migration Complete** (Across Entire Stack)
  - ✅ Migrated from `5s, 1m, 15m, 1h, 4h, 1d` to `1m, 5m, 15m, 1h, 1d`
  - ✅ Removed obsolete intervals: 5s, 4h
  - ✅ Added new intervals: 5m, 1d
  - ✅ Updated 15+ files across configuration, Kafka, data-collector, backend-go, database, frontend
- **Kafka Topics Reset**
  - ✅ Deleted all 5 old aggregated topics (with stale data)
  - ✅ Recreated topics with proper retention policies via docker-compose kafka-init
  - ✅ Fresh 7-day backfill completed: 100K+ 1m candles, 20K+ 5m candles
- **Backend-Go Enhancements**
  - ✅ Created ValidateInterval middleware (rejects old intervals with 400 Bad Request)
  - ✅ Applied middleware to 4 routes (/crypto/data, /stats, /indicators/:type, /indicators)
  - ✅ Updated Kafka consumers to subscribe to all 5 new topics
- **Database Schema Updates**
  - ✅ Updated cleanup_old_data() function with interval-specific retention
  - ✅ Implemented hot/cold tiering via .env configuration per interval
  - ✅ Automatic deletion of obsolete interval data (1s, 5s, 4h)
- **Frontend Components**
  - ✅ Created IntervalSelector.vue component with all 5 intervals
  - ✅ Created intervals.ts constants file (SUPPORTED_INTERVALS, validation helpers)
- **Documentation & Testing**
  - ✅ Updated API docs (docs/api.md) with new supported intervals
  - ✅ Updated test-websocket.html (buttons, functions, placeholders)
  - ✅ Tested all REST endpoints with new intervals (all passing)
  - ✅ Verified WebSocket candle streaming for all intervals

### 2025-11-20: API Symbol Routing Fix + Phase 5 Progress ✅
- **Phase 5.1 Completed: Symbol Routing & Validation** (60% → 70%)
  - ✅ Created ValidateSymbolQuery middleware for trading pair symbols
  - ✅ Refactored 5 REST endpoints from path to query parameters
  - ✅ Updated CandleController (3 methods) and IndicatorController (2 methods)
  - ✅ Updated API documentation (docs/api.md)
  - ✅ Solution: Query parameters (Option 2) for REST best practices
    - No URL encoding needed, BTC/USDT format preserved
    - Industry standard for filtering/selection
    - Zero data migration required
  - ✅ Tested all endpoints successfully
- **Overall Progress:** 62% → 68%

### 2025-11-20: News Scraper Integration COMPLETE + WebSocket Broadcasting ✅
- **Phase 4 Completed** (90% → 100%)
  - ✅ Scalable architecture with Abstract Factory pattern
  - ✅ Core components: Kafka producer, VADER sentiment, crypto detector
  - ✅ Source implementations: RSS + POC stubs (Twitter, Reddit, Bloomberg)
  - ✅ End-to-end pipeline verified:
    - news-scraper → Kafka (crypto.news) → backend-go NewsHandler → TimescaleDB → REST API
  - ✅ Docker integration complete with environment variable configuration
  - ✅ 243 articles stored from CoinDesk + CoinTelegraph
  - ✅ REST API endpoints working: GET /api/v1/news, /api/v1/news/{symbol}
- **Phase 3.3 Enhanced: News WebSocket Broadcasting** ✅
  - Added `broadcast` field to NewsHandler
  - Implemented `SetBroadcast()` method
  - Broadcasting news to WebSocket clients after DB insertion
  - Broadcasting to each symbol mentioned in news article
  - Connected to WebSocket Hub in main.go
  - Pattern consistent with TradeHandler and CandleHandler
- **Configuration Updated**
  - Added `crypto.news` to `.env` KAFKA_TOPICS
  - Added environment variable support in config.yaml
  - Fixed Dockerfile CMD to run app.py
- **Overall Progress:** 58% → 62%

### 2025-11-20: Project Restructure & Frontend Integration
- **Renamed** from "Backend-Go Microservice" to "CryptoViz Project" tasks
- **Added** Phase 3.5: Frontend-Vue Integration (🔴 HIGHEST PRIORITY)
  - Vue 3 + TypeScript frontend exists with 10+ components
  - WebSocket client ready (currently using Binance mock)
  - Needs connection to backend-go WebSocket and REST API
- **Added** Phase 4: News Scraper Integration
  - Basic scraper working (RSS feeds → JSON)
  - Needs Kafka producer integration
  - Needs sentiment analysis implementation
- **Added** Phase 6: Monitoring & Observability (Grafana + Prometheus)
- **Renumbered** all subsequent phases (Testing → Phase 7, Production → Phase 8, Docs → Phase 9)
- **Updated** completion table: Overall progress 72% → 58% (accounting for frontend work)
- **Updated** immediate next steps to prioritize frontend integration

---

**Last Reviewed:** 2025-11-20
**Last Updated:** 2025-11-20 (Phase 3.5 Frontend Integration Complete)
**Next Review:** After frontend integration testing
**Focus:** Testing frontend integration with backend-go (E2E validation)
