# Guide des Données en Temps Réel - CryptoViz

## Vue d'ensemble

CryptoViz propose deux méthodes pour accéder aux données en temps réel :

1. **REST API** - Pour les requêtes ponctuelles
2. **WebSocket** - Pour les flux de données continu en temps réel

## Architecture

```
Backend (Go)
├── REST API (/api/v1/*)
│   └── Requêtes HTTP → Base de données (TimescaleDB)
│
├── WebSocket (/ws/crypto)
│   └── Flux temps réel via Kafka
│       → Candles (5s, 1m, 15m, 1h, 4h, 1d)
│       → Trades
│       → Indicateurs (RSI, MACD, Bollinger, Momentum)
│       → News
│
└── Kafka (Message Broker)
    ├── crypto.aggregated.* (Candles)
    ├── crypto.raw.trades (Trades)
    ├── crypto.indicators.* (Indicateurs)
    └── crypto.news (News)
```

---

## REST API - Requêtes Ponctuelles

### Endpoints Disponibles

#### 1. **Candles (OHLCV)**

```bash
GET /api/v1/crypto/{symbol}/data
```

**Paramètres:**

- `symbol` (path) - Symbole crypto (ex: BTCUSDT)
- `interval` (query) - Intervalle (1s, 5s, 1m, 15m, 1h, 4h, 1d)
- `limit` (query) - Nombre de résultats (défaut: 100, max: 1000)
- `start` (query) - Timestamp début (optionnel)
- `end` (query) - Timestamp fin (optionnel)

**Exemple:**

```bash
curl "http://localhost:8080/api/v1/crypto/BTCUSDT/data?interval=1m&limit=50"
```

**Réponse:**

```json
{
  "success": true,
  "data": {
    "symbol": "BTCUSDT",
    "interval": "1m",
    "data": [
      {
        "time": "2025-01-20T10:30:00Z",
        "open": 45000.0,
        "high": 45100.0,
        "low": 44900.0,
        "close": 45050.0,
        "volume": 1.5
      }
    ]
  },
  "timestamp": "2025-01-20T10:35:00Z"
}
```

#### 2. **Prix Actuel**

```bash
GET /api/v1/crypto/{symbol}/latest
```

**Exemple:**

```bash
curl "http://localhost:8080/api/v1/crypto/BTCUSDT/latest"
```

**Réponse:**

```json
{
  "success": true,
  "data": {
    "symbol": "BTCUSDT",
    "price": 45050.0,
    "timestamp": "2025-01-20T10:35:42Z"
  }
}
```

#### 3. **Statistiques**

```bash
GET /api/v1/stats/{symbol}
```

**Exemple:**

```bash
curl "http://localhost:8080/api/v1/stats/BTCUSDT"
```

**Réponse:**

```json
{
  "success": true,
  "data": {
    "symbol": "BTCUSDT",
    "high_24h": 46000,
    "low_24h": 44000,
    "change_24h": 2.3,
    "volume_24h": 1250.5
  }
}
```

#### 4. **Indicateurs Techniques**

```bash
GET /api/v1/indicators/{symbol}/{type}
```

Types supportés: `rsi`, `macd`, `bollinger`, `momentum`

**Exemple:**

```bash
curl "http://localhost:8080/api/v1/indicators/BTCUSDT/rsi"
```

**Réponse:**

```json
{
  "success": true,
  "data": {
    "symbol": "BTCUSDT",
    "type": "rsi",
    "data": [
      {
        "ts": 1705757400000,
        "value": 65.3
      }
    ]
  }
}
```

#### 5. **News**

```bash
GET /api/v1/news
GET /api/v1/news/{symbol}
```

**Exemple:**

```bash
curl "http://localhost:8080/api/v1/news"
```

---

## WebSocket - Données en Temps Réel

### Utilisation Basique

#### 1. **Connecter et écouter**

```typescript
import { getRTClient } from "@/services/rt";

const rt = getRTClient();

// Établir la connexion
await rt.connect();

// S'abonner aux candles BTC 1m
rt.subscribe("candles", "BTCUSDT", "1m");

// Écouter les messages
rt.on("candle", (msg) => {
  console.log("Nouvelle candle:", msg.data);
});

// S'abonner aux trades
rt.subscribe("trades", "BTCUSDT");

rt.on("trade", (msg) => {
  console.log("Nouveau trade:", msg.data);
});
```

#### 2. **Messages WebSocket**

**Client → Serveur:**

```json
{
  "action": "subscribe",
  "type": "candles",
  "symbol": "BTCUSDT",
  "timeframe": "1m"
}
```

**Serveur → Client (Candle):**

```json
{
  "type": "candle",
  "data": {
    "symbol": "BTCUSDT",
    "timeframe": "1m",
    "time": "2025-01-20T10:35:00Z",
    "open": 45000,
    "high": 45100,
    "low": 44900,
    "close": 45050,
    "volume": 1.5
  },
  "timestamp": 1705757700000
}
```

**Serveur → Client (Trade):**

```json
{
  "type": "trade",
  "data": {
    "symbol": "BTCUSDT",
    "price": 45050,
    "quantity": 0.5,
    "side": "buy",
    "time": "2025-01-20T10:35:42Z"
  },
  "timestamp": 1705757742123
}
```

### Actions Disponibles

#### Subscribe

```typescript
rt.subscribe("candles", "BTCUSDT", "1m");
rt.subscribe("trades", "BTCUSDT");
rt.subscribe("indicators", "BTCUSDT");
rt.subscribe("news", "*"); // All news
```

#### Unsubscribe

```typescript
rt.unsubscribe("candles", "BTCUSDT", "1m");
```

#### List Subscriptions

```typescript
rt.listSubscriptions();

rt.on("subscriptions", (msg) => {
  console.log("Current subscriptions:", msg.data);
});
```

#### Ping

```typescript
rt.ping();

rt.on("pong", (msg) => {
  console.log("Server is alive!");
});
```

---

## Exemple Complet - Composant Vue

```vue
<script setup lang="ts">
import { ref, onMounted, onUnmounted } from "vue";
import { getRTClient, disconnectRT } from "@/services/rt";
import type { WSMessage } from "@/services/rt";

const rt = getRTClient();
const isConnected = ref(false);
const latestPrice = ref(0);
const latestCandle = ref(null);

onMounted(async () => {
  try {
    // Connecter au WebSocket
    await rt.connect();
    isConnected.value = true;

    // S'abonner aux candles
    rt.subscribe("candles", "BTCUSDT", "1m");

    // Écouter les candles
    rt.on("candle", (msg: WSMessage) => {
      const candle = msg.data as any;
      latestCandle.value = candle;
      latestPrice.value = candle.close;
    });

    // S'abonner aux trades
    rt.subscribe("trades", "BTCUSDT");

    // Écouter les trades
    rt.on("trade", (msg: WSMessage) => {
      const trade = msg.data as any;
      latestPrice.value = trade.price;
    });

    // Écouter les erreurs
    rt.on("error", (msg: WSMessage) => {
      console.error("WebSocket error:", msg.data);
    });
  } catch (err) {
    console.error("Failed to connect:", err);
  }
});

onUnmounted(() => {
  disconnectRT();
  isConnected.value = false;
});
</script>

<template>
  <div class="realtime-panel">
    <div class="status">
      <span
        :class="['indicator', isConnected ? 'connected' : 'disconnected']"
      ></span>
      {{ isConnected ? "Connected" : "Disconnected" }}
    </div>

    <div class="price">
      <h2>{{ latestPrice }}</h2>
      <p v-if="latestCandle">Interval: {{ latestCandle.timeframe }}</p>
    </div>
  </div>
</template>

<style scoped>
.realtime-panel {
  padding: 16px;
  border: 1px solid var(--border-primary);
  border-radius: 8px;
}

.status {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  color: var(--text-secondary);
}

.indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;

  &.connected {
    background: var(--accent-green);
    box-shadow: 0 0 10px var(--accent-green);
  }

  &.disconnected {
    background: var(--accent-red);
  }
}

.price {
  text-align: center;
}

.price h2 {
  font-size: 32px;
  color: var(--text-primary);
  margin: 0;
}
</style>
```

---

## Configuration

### Variables d'Environnement

```dotenv
# .env.local

# API REST
VITE_API_URL=http://localhost:8080

# WebSocket
VITE_WS_URL=ws://localhost:8080/ws/crypto

# Mode Mock (désactiver le WebSocket en dev)
VITE_USE_MOCK=false
```

---

## Bonnes Pratiques

### 1. **Gestion de la Connexion**

```typescript
// Toujours nettoyer à l'unmount
onUnmounted(() => {
  disconnectRT();
});

// Toujours attendre la connexion
try {
  await rt.connect();
} catch (err) {
  console.error("Connection failed:", err);
}
```

### 2. **Gestion des Subscriptions**

```typescript
// Unsubscribe quand on n'a plus besoin
rt.unsubscribe("candles", "BTCUSDT", "1m");

// Utiliser les wildcards pour multiple symboles
rt.subscribe("candles", "*", "1m"); // Tous les symboles
```

### 3. **Performance**

```typescript
// Déregistrer les handlers inutilisés
const unsubscribe = rt.on("candle", handler);
// Plus tard...
unsubscribe();

// Limiter le nombre de subscriptions actives
// Le WebSocket a un buffer de 1000 messages
```

### 4. **Erreur Handling**

```typescript
rt.on("error", (msg) => {
  const error = msg.data as any;
  console.error("Server error:", error.message);

  // Réconnexion automatique après 3 secondes
  setTimeout(() => {
    rt.connect();
  }, 3000);
});
```

---

## Monitoring

### Stats Endpoint

```bash
GET /ws/stats
```

**Réponse:**

```json
{
  "status": "ok",
  "data": {
    "total_connections": 42,
    "active_connections": 38,
    "messages_broadcast": 1250000,
    "messages_delivered": 1248500,
    "last_broadcast_time": 1705757742123
  }
}
```

### Health Checks

```bash
GET /health          # Service alive
GET /ready           # Service ready (DB + Redis)
```

---

## Dépannage

### WebSocket Connection Refused

```
Error: WebSocket is closed before the connection is established
```

**Solutions:**

1. Vérifier que le backend est démarré
2. Vérifier `VITE_WS_URL` dans `.env.local`
3. Vérifier les CORS (WebSocket supporte les CORS)

### Messages not being received

**Vérifications:**

1. Vérifier que la subscription est confirmée (`subscribed` ack reçu)
2. Vérifier le symbol et timeframe (case-sensitive)
3. Vérifier que Kafka est démarré (messages viennent de Kafka)

### High Latency

**Optimisations:**

1. Réduire le nombre de subscriptions
2. Utiliser des timeframes plus longs (1h au lieu de 5s)
3. Utiliser Redis pour la déduplication (déjà implémenté)

---

## Architecture Kafka (Backend)

Les données en temps réel proviennent de Kafka :

```
Data Collector (Python)
    ↓
Kafka Topics
    ├── crypto.aggregated.5s/1m/15m/1h/4h/1d → CandleHandler
    ├── crypto.raw.trades → TradeHandler
    ├── crypto.indicators.rsi/macd/bollinger/momentum → IndicatorHandler
    └── crypto.news → NewsHandler
    ↓
Database (TimescaleDB)
    ↓
WebSocket Hub
    ↓
Clients
```

**Features:**

- ✅ Déduplication Redis
- ✅ Retry avec exponential backoff
- ✅ Graceful shutdown
- ✅ Consumer groups pour load balancing

---

## Limites & Quotas

- **WebSocket Buffer:** 1000 messages max en attente
- **Max Subscriptions:** Illimité (limité par RAM)
- **Message Size:** 512 bytes max (client → serveur)
- **Read Timeout:** 60 secondes
- **Write Timeout:** 10 secondes

---

## Futures Améliorations

- [ ] Authentication/Authorization
- [ ] Rate limiting par client
- [ ] Message compression
- [ ] Historical data replay
- [ ] Alert subscriptions
- [ ] Order updates (si trading API ajoutée)
