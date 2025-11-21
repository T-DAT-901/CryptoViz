# CryptoViz API Documentation

## Vue d'ensemble

L'API CryptoViz fournit un accès REST et WebSocket aux données crypto en temps réel, aux indicateurs techniques et aux actualités. L'API est construite avec Go et Gin, optimisée pour les performances et la scalabilité.

## Base URL

```
http://localhost:8080/api/v1
```

## Authentification

Actuellement, l'API ne nécessite pas d'authentification. En production, il est recommandé d'implémenter JWT ou OAuth2.

## Format des Réponses

Toutes les réponses sont au format JSON avec la structure suivante :

```json
{
  "success": true,
  "data": {},
  "message": "Success",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

En cas d'erreur :

```json
{
  "success": false,
  "error": "Error message",
  "code": "ERROR_CODE",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

## Endpoints REST

### Format des Symboles

Les symboles de trading pairs utilisent le format `BASE/QUOTE` (ex: `BTC/USDT`, `ETH/BTC`). Les symboles sont passés en paramètres de requête (query parameters) car ils contiennent des slashes.

Les symboles de news utilisent des tokens simples (ex: `btc`, `eth`, `sol`).

### Données Crypto

#### GET /crypto/data

Récupère les données OHLCV pour un symbole donné.

**Paramètres :**
- `symbol` (query, required) : Symbole trading pair (ex: `BTC/USDT`, `ETH/BTC`)
- `interval` (query) : Intervalle de temps (1m, 5m, 15m, 1h, 1d) - défaut: `1m`
- `limit` (query) : Nombre maximum de résultats (défaut: 100, max: 1000)

**Exemple :**
```bash
GET /api/v1/crypto/data?symbol=BTC/USDT&interval=1m&limit=50
```

**Réponse :**
```json
{
  "success": true,
  "data": [
    {
      "window_start": "2024-01-01T12:00:00Z",
      "window_end": "2024-01-01T12:01:00Z",
      "exchange": "BINANCE",
      "symbol": "BTC/USDT",
      "timeframe": "1m",
      "open": 45000.00,
      "high": 45100.00,
      "low": 44900.00,
      "close": 45050.00,
      "volume": 1.5,
      "trade_count": 150,
      "closed": true,
      "first_trade_ts": "2024-01-01T12:00:00.123Z",
      "last_trade_ts": "2024-01-01T12:00:59.987Z",
      "duration_ms": 59864,
      "source": "collector-ws",
      "created_at": "2024-01-01T12:01:00.456Z"
    }
  ],
  "timestamp": "2024-01-01T12:01:30.123Z"
}
```

#### GET /crypto/latest

Récupère le dernier prix pour un symbole donné.

**Paramètres :**
- `symbol` (query, required) : Symbole trading pair (ex: `BTC/USDT`)
- `exchange` (query) : Exchange (défaut: `BINANCE`)

**Exemple :**
```bash
GET /api/v1/crypto/latest?symbol=BTC/USDT
```

**Réponse :**
```json
{
  "success": true,
  "data": {
    "window_start": "2024-01-01T12:00:00Z",
    "window_end": "2024-01-01T12:01:00Z",
    "exchange": "BINANCE",
    "symbol": "BTC/USDT",
    "timeframe": "1m",
    "open": 45000.00,
    "high": 45050.00,
    "low": 44980.00,
    "close": 45025.00,
    "volume": 0.75,
    "trade_count": 45,
    "closed": true
  },
  "timestamp": "2024-01-01T12:00:30.123Z"
}
```

#### GET /stats

Récupère les statistiques pour un symbole donné.

**Paramètres :**
- `symbol` (query, required) : Symbole trading pair (ex: `BTC/USDT`)
- `interval` (query) : Intervalle de temps - défaut: `1m`

**Exemple :**
```bash
GET /api/v1/stats?symbol=BTC/USDT&interval=1h
```

**Réponse :**
```json
{
  "success": true,
  "data": {
    "symbol": "BTC/USDT",
    "interval": "1h",
    "min_price": 44500.00,
    "max_price": 45500.00,
    "avg_price": 45000.00,
    "total_volume": 150.5,
    "count": 120
  },
  "timestamp": "2024-01-01T12:00:30.123Z"
}
```

### Indicateurs Techniques

#### GET /indicators/:type

Récupère les indicateurs d'un type spécifique pour un symbole donné.

**Paramètres :**
- `type` (path, required) : Type d'indicateur (`rsi`, `macd`, `bollinger`, `momentum`)
- `symbol` (query, required) : Symbole trading pair (ex: `BTC/USDT`)
- `interval` (query) : Intervalle de temps - défaut: `1m`
- `limit` (query) : Nombre maximum de résultats (défaut: 100, max: 1000)

**Exemple :**
```bash
GET /api/v1/indicators/rsi?symbol=BTC/USDT&interval=1m&limit=50
```

**Réponse :**
```json
{
  "success": true,
  "data": [
    {
      "time": "2024-01-01T12:00:00Z",
      "symbol": "BTC/USDT",
      "exchange": "BINANCE",
      "timeframe": "1m",
      "indicator_type": "rsi",
      "value": 65.5,
      "metadata": {
        "period": 14
      }
    }
  ],
  "timestamp": "2024-01-01T12:00:30.123Z"
}
```

#### GET /indicators

Récupère tous les indicateurs pour un symbole donné.

**Paramètres :**
- `symbol` (query, required) : Symbole trading pair (ex: `BTC/USDT`)
- `interval` (query) : Intervalle de temps - défaut: `1m`

**Exemple :**
```bash
GET /api/v1/indicators?symbol=BTC/USDT&interval=1m
```

**Réponse :**
```json
{
  "success": true,
  "data": {
    "rsi": [...],
    "macd": [...],
    "bollinger": [...],
    "momentum": [...]
  },
  "timestamp": "2024-01-01T12:00:30.123Z"
}
```

### Actualités

#### GET /news

Récupère toutes les actualités récentes.

**Paramètres :**
- `limit` (query) : Nombre maximum de résultats (défaut: 100)

**Exemple :**
```bash
GET /api/v1/news?limit=20
```

#### GET /news/:symbol

Récupère les actualités pour un token crypto spécifique.

**Paramètres :**
- `symbol` (path, required) : Token crypto simple (ex: `btc`, `eth`, `sol`)

**Exemple :**
```bash
GET /api/v1/news/btc
```

**Réponse :**
```json
{
  "success": true,
  "data": [
    {
      "time": "2024-01-01T12:00:00Z",
      "source": "CoinDesk",
      "url": "https://...",
      "title": "Bitcoin News Title",
      "content": "Article content...",
      "sentiment_score": 0.65,
      "symbols": ["btc", "eth"],
      "created_at": "2024-01-01T12:00:30.123Z"
    }
  ],
  "timestamp": "2024-01-01T12:00:30.123Z"
}
```

## Intervalles Supportés

Les intervalles suivants sont supportés pour toutes les requêtes d'agrégation :

- `1m` - 1 minute
- `5m` - 5 minutes
- `15m` - 15 minutes
- `1h` - 1 heure
- `1d` - 1 jour

**Note:** Les anciens intervalles (`1s`, `5s`, `4h`) ne sont plus supportés. Les requêtes avec des intervalles invalides retourneront une erreur 400 Bad Request.

**Exemple d'erreur pour intervalle invalide :**
```json
{
  "success": false,
  "error": "Invalid interval. Supported intervals: 1m, 5m, 15m, 1h, 1d",
  "example": "/api/v1/crypto/data?symbol=BTC/USDT&interval=1m"
}
```

## Codes d'Erreur

| Code | Description |
|------|-------------|
| `MISSING_PARAMETER` | Paramètre requis manquant |
| `INVALID_SYMBOL` | Format de symbole invalide (doit être BASE/QUOTE ex: BTC/USDT) |
| `INVALID_INTERVAL` | Intervalle invalide (les intervalles supportés sont: 1m, 5m, 15m, 1h, 1d) |
| `DATABASE_ERROR` | Erreur de base de données |
| `NOT_FOUND` | Ressource non trouvée |
