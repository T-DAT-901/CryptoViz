# Backend-Go Architecture

## Vue d'ensemble

Le backend CryptoViz suit une architecture **Clean Architecture** avec séparation claire des responsabilités et structure conforme aux conventions Go.

## Structure du Projet (Go Standard Project Layout)

```
services/backend-go/
├── cmd/                              # Entry points
│   └── server/
│       └── main.go                   # Bootstrap application (120 lignes)
├── internal/                         # Private application code
│   ├── config/                       # Configuration
│   │   └── config.go                 # Gestion config & env vars
│   ├── controllers/                  # HTTP handlers (couche présentation)
│   │   ├── dependencies.go           # Injection de dépendances
│   │   ├── health_controller.go      # Health checks
│   │   ├── candle_controller.go      # Endpoints candles OHLCV
│   │   ├── indicator_controller.go   # Endpoints indicateurs
│   │   ├── news_controller.go        # Endpoints actualités
│   │   └── websocket_controller.go   # WebSocket handler
│   ├── routes/                       # Définition des routes
│   │   └── routes.go                 # Setup routes & middleware
│   ├── middleware/                   # Middleware HTTP
│   │   ├── cors.go                   # CORS headers
│   │   └── logger.go                 # Logging personnalisé
│   ├── dto/                          # Data Transfer Objects
│   │   └── response.go               # Structures de réponse API
│   └── kafka/                        # Kafka integration (NEW)
│       ├── config.go                 # Kafka configuration
│       ├── consumer.go               # Base consumer
│       ├── manager.go                # Consumer lifecycle management
│       ├── handlers.go               # Message handler interfaces
│       ├── utils/                    # Kafka utilities
│       │   ├── headers.go            # Header parsing
│       │   ├── deduplication.go      # Redis-based dedup
│       │   └── retry.go              # Exponential backoff
│       └── consumers/                # Topic-specific consumers
│           ├── candles.go            # Candles consumer
│           ├── trades.go             # Trades consumer
│           ├── indicators.go         # Indicators consumer
│           └── news.go               # News consumer
├── models/                           # Couche données (repository pattern)
│   ├── candles.go
│   ├── indicators.go
│   ├── news.go
│   ├── trade.go
│   ├── user.go
│   └── currency.go
├── database/                         # Couche persistance
│   └── connection.go
├── go.mod                            # Go modules
├── go.sum
├── Dockerfile                        # Container image
└── ARCHITECTURE.md                   # This file
```

**Note**: Le dossier `internal/` garantit que ce code ne peut pas être importé par d'autres projets (feature Go native).

## Principes Architecturaux

### 1. Separation of Concerns (SoC)

Chaque package a une responsabilité unique :

- **config** : Gestion de la configuration
- **controllers** : Gestion des requêtes HTTP (thin layer)
- **routes** : Configuration des routes et middleware
- **middleware** : Cross-cutting concerns (CORS, logging)
- **dto** : Structures de transfert de données
- **kafka** : Intégration Kafka (consumers, handlers, utilities)
- **models** : Logique métier et accès données
- **database** : Connexion et configuration DB

### 2. Dependency Injection

Les dépendances sont injectées via le container `Dependencies` :

```go
type Dependencies struct {
    DB            *gorm.DB
    Redis         *redis.Client
    Logger        *logrus.Logger
    CandleRepo    models.CandleRepository
    IndicatorRepo models.IndicatorRepository
    NewsRepo      models.NewsRepository
    // ...
}
```

**Avantages** :
- Facilite les tests (mocking)
- Découplage des composants
- Configuration centralisée

### 3. Repository Pattern

Abstraction de la couche d'accès aux données via des interfaces :

```go
type CandleRepository interface {
    GetBySymbol(symbol, timeframe string, limit int) ([]Candle, error)
    GetLatest(symbol, exchange string) (*Candle, error)
    GetStats(symbol, timeframe string) (map[string]interface{}, error)
    // ...
}
```

**Avantages** :
- Testabilité (mock repositories)
- Changement de DB transparent
- Code métier indépendant de la persistence

### 4. Controller Pattern

Les controllers sont légers et délèguent aux repositories :

```go
func (ctrl *CandleController) GetCandleData(c *gin.Context) {
    // 1. Extraire et valider les paramètres
    symbol := c.Param("symbol")
    interval := c.DefaultQuery("interval", "1m")

    // 2. Appeler le repository
    data, err := ctrl.repo.GetBySymbol(symbol, interval, limit)

    // 3. Retourner la réponse
    c.JSON(http.StatusOK, dto.SuccessResponse(data))
}
```

**Avantages** :
- Logique de présentation isolée
- Facile à tester
- Réutilisable

## Flux de Requête

### Flux HTTP (API REST)

```
HTTP Request
    ↓
Middleware (CORS, Logger)
    ↓
Router (routes.go)
    ↓
Controller (candle_controller.go)
    ↓
Repository (models/candles.go)
    ↓
Database (GORM → TimescaleDB)
    ↓
HTTP Response (DTO)
```

### Flux Kafka (Real-time Data Ingestion) 🆕

```
data-collector (Python)
    ↓
Kafka Topics (crypto.aggregated.*, crypto.raw.trades, etc.)
    ↓
ConsumerManager (Start all consumers)
    ↓
BaseConsumer (Generic Kafka consumer with retry)
    ↓
MessageHandler (Topic-specific: Candles, Trades, Indicators, News)
    ↓
Utils (Header parsing, deduplication, validation)
    ↓
Repository (models/*.go)
    ↓
TimescaleDB (Hypertables with compression & tiering)
```

**Topics consommés** :
- `crypto.aggregated.{5s,1m,15m,1h,4h,1d}` → CandleHandler → candles table
- `crypto.raw.trades` → TradeHandler → trades table
- `crypto.indicators.{rsi,macd,bollinger,momentum}` → IndicatorHandler → indicators table
- `crypto.news` → NewsHandler → news table

**Features clés** :
- ✅ **Deduplication** : Redis-based avec TTL (évite les doublons)
- ✅ **Retry logic** : Exponential backoff pour les erreurs transitoires
- ✅ **Graceful shutdown** : Arrêt propre avec commit des offsets
- ✅ **Consumer groups** : Load balancing automatique (`backend-go-consumers`)
- ✅ **Error handling** : Skip permanent errors, retry transient errors
- ✅ **Monitoring** : Health check endpoint pour status des consumers

## Comparaison Avant/Après

### Avant (main.go monolithique)

```
main.go: 512 lignes
├── Configuration
├── Struct App
├── Connexion DB/Redis
├── Setup routes
├── 10+ handler functions
├── WebSocket handler
└── Server startup
```

**Problèmes** :
- ❌ Tout dans un fichier
- ❌ Difficile à tester
- ❌ Couplage fort
- ❌ Pas de réutilisabilité
- ❌ Ne suit pas les conventions Go

### Après (Clean Architecture + Go Standard Project Layout)

```
cmd/server/main.go: 94 lignes (bootstrap uniquement)

internal/
├── config/config.go: ~40 lignes
├── controllers/: ~500 lignes réparties en 6 fichiers
├── routes/routes.go: ~50 lignes
├── middleware/: ~60 lignes
└── dto/response.go: ~40 lignes
```

**Avantages** :
- ✅ Séparation claire des responsabilités
- ✅ Facilement testable
- ✅ Découplage
- ✅ Scalable et maintenable
- ✅ Suit les best practices Go (golang-standards/project-layout)
- ✅ Code dans `internal/` ne peut pas être importé par erreur
- ✅ Entry point clair dans `cmd/`

## Tests Unitaires

Exemple de test pour un controller :

```go
func TestCandleController_GetCandleData(t *testing.T) {
    // Mock repository
    mockRepo := &MockCandleRepository{
        GetBySymbolFunc: func(symbol, tf string, limit int) ([]models.Candle, error) {
            return []models.Candle{{Symbol: "BTC/USDT"}}, nil
        },
    }

    // Setup controller avec mock
    deps := &Dependencies{CandleRepo: mockRepo}
    ctrl := NewCandleController(deps)

    // Test HTTP request
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Params = gin.Params{gin.Param{Key: "symbol", Value: "BTC/USDT"}}

    ctrl.GetCandleData(c)

    assert.Equal(t, 200, w.Code)
}
```

## Middleware Chain

L'ordre des middleware est important :

```go
router.Use(gin.Recovery())        // 1. Panic recovery
router.Use(middleware.CORS())     // 2. CORS headers
router.Use(middleware.Logger())   // 3. Request logging
```

## Configuration

Configuration via variables d'environnement :

```bash
# Server
PORT=8080
GIN_MODE=release

# Database
DB_HOST=timescaledb
DB_PORT=5432
DB_NAME=cryptoviz
DB_USER=postgres
DB_PASSWORD=postgres

# Redis
REDIS_HOST=redis
REDIS_PORT=6379

# Kafka
KAFKA_BROKERS=kafka:29092
```

## Routes API

### Health Checks
- `GET /health` - Service alive
- `GET /ready` - Service ready (DB + Redis)

### Candles (OHLCV)
- `GET /api/v1/crypto/:symbol/data` - Get candle data
- `GET /api/v1/crypto/:symbol/latest` - Get latest price
- `GET /api/v1/stats/:symbol` - Get statistics

### Indicateurs
- `GET /api/v1/indicators/:symbol/:type` - Get indicators by type
- `GET /api/v1/indicators/:symbol` - Get all indicators

### News
- `GET /api/v1/news` - Get all news
- `GET /api/v1/news/:symbol` - Get news by symbol

### WebSocket
- `GET /ws/crypto` - WebSocket connection

## Réponses API Standardisées

Toutes les réponses utilisent le format DTO :

```json
{
  "success": true,
  "data": { ... },
  "timestamp": "2025-01-07T12:00:00Z"
}
```

Erreur :
```json
{
  "success": false,
  "error": "Database error",
  "timestamp": "2025-01-07T12:00:00Z"
}
```

## Évolutions Futures

### Court terme
- [ ] Tests unitaires pour tous les controllers
- [ ] Tests d'intégration
- [ ] Swagger/OpenAPI documentation
- [ ] Validation des inputs (validator)

### Moyen terme
- [ ] Service layer (entre controllers et repositories)
- [ ] Rate limiting middleware
- [ ] Authentication/Authorization middleware
- [ ] Metrics (Prometheus)

### Long terme
- [ ] Event-driven architecture (CQRS)
- [ ] GraphQL API
- [ ] gRPC endpoints
- [ ] Circuit breaker pattern

## Commandes Utiles

```bash
# Build
go build -o bin/backend ./cmd/server

# Test
go test ./...

# Test avec coverage
go test -cover ./...

# Format code
go fmt ./...

# Lint
golangci-lint run

# Run
./bin/backend

# Run avec go run
go run ./cmd/server
```

## Bonnes Pratiques Appliquées

✅ **SOLID Principles**
- Single Responsibility
- Dependency Inversion
- Interface Segregation

✅ **12-Factor App**
- Configuration via env vars
- Stateless processes
- Graceful shutdown

✅ **Clean Code**
- Noms explicites
- Fonctions courtes
- Commentaires pertinents

✅ **Error Handling**
- Pas de panic en production
- Logs structurés
- Erreurs propagées correctement

## Conclusion

Cette architecture offre :
- **Maintenabilité** : Code organisé et facile à comprendre
- **Testabilité** : Chaque composant testable isolément
- **Scalabilité** : Facile d'ajouter de nouvelles features
- **Performance** : Découplage permet l'optimisation ciblée
- **Qualité** : Suit les standards de l'industrie

---

**Auteur** : CryptoViz Team
**Date** : Janvier 2025
**Version** : 2.0 (Clean Architecture)
