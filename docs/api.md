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

### Données Crypto

#### GET /crypto/{symbol}/data

Récupère les données OHLCV pour un symbole donné.

**Paramètres :**
- `symbol` (path) : Symbole crypto (ex: BTCUSDT)
- `interval` (query) : Intervalle de temps (1s, 5s, 1m, 15m, 1h, 4h, 1d)
- `start` (query) : Timestamp de début (optionnel)
- `end` (query) : Timestamp de fin (optionnel)
- `limit` (query) : Nombre maximum de résultats (défaut: 100, max: 1000)

**Exemple :**
```bash
GET /api/v1/crypto/BTCUSDT/data?interval=1m&limit=50
```

**Réponse :**
```json
{
  "success": true,
  "data": {
    "symbol": "BTCUSDT",
    "interval": "1m",
    "data": [
      {
        "time": "2024-01-01T12:00:00Z",
        "open": 45000.00,
        "high": 45100.00,
        "low": 44900.00,
        "close": 45050.00,
        "volume": 1.5,
