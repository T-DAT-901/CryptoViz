# Implémentation des Données Temps Réel - Résumé

## ✅ Ce qui a été implémenté

### 1. **Service WebSocket** (`services/rt.ts`)

Un client WebSocket complet avec :

- ✅ Connection/Reconnection avec exponential backoff
- ✅ Subscribe/Unsubscribe à des streams spécifiques
- ✅ Handler pattern pour différents types de messages
- ✅ Singleton instance pour une utilisation globale
- ✅ Gestion d'erreurs et graceful shutdown
- ✅ Support du mode mock (désactif en développement)

**Utilisation rapide :**

```typescript
import { getRTClient } from "@/services/rt";

const rt = getRTClient();
await rt.connect();
rt.subscribe("candles", "BTCUSDT", "1m");
rt.on("candle", (msg) => console.log(msg.data));
```

### 2. **Store Pinia Enrichi** (`stores/market.ts`)

Nouvelles actions pour gérer le WebSocket :

- `connectRealtime()` - Établir la connexion WebSocket
- `disconnectRealtime()` - Fermer la connexion
- `switchSymbol(symbol)` - Changer de symbole avec subscriptions
- `switchInterval(interval)` - Changer de timeframe avec subscriptions
- Auto-update des candles et prices en temps réel

**État global:**

```typescript
{
  rtConnected: boolean,
  activeSymbol: "BTCUSDT",
  interval: "1m",
  candles: { BTCUSDT: [...] },
  tickers: { BTCUSDT: {...} }
}
```

### 3. **Composant Example** (`components/RealtimeExample.vue`)

Un composant complet montrant :

- 🟢 Indicateur de connexion avec animation
- 🔌 Boutons Connect/Disconnect
- 📊 Sélecteurs Symbol + Timeframe
- 💰 Affichage du prix en temps réel
- 🕯️ Dernière candle avec détails
- ⚠️ Gestion des erreurs avec messages

### 4. **Documentation Complète** (`REALTIME_GUIDE.md`)

Guide complet incluant :

- Architecture Kafka → WebSocket
- Tous les endpoints REST disponibles
- Protocole WebSocket (messages client/serveur)
- Exemples de code
- Bonnes pratiques
- Dépannage

---

## 📋 Endpoints Disponibles

### REST API

| Endpoint                             | Méthode | Description           |
| ------------------------------------ | ------- | --------------------- |
| `/api/v1/crypto/{symbol}/data`       | GET     | Candles OHLCV         |
| `/api/v1/crypto/{symbol}/latest`     | GET     | Prix actuel           |
| `/api/v1/stats/{symbol}`             | GET     | Stats 24h             |
| `/api/v1/indicators/{symbol}/{type}` | GET     | Indicateur spécifique |
| `/api/v1/indicators/{symbol}`        | GET     | Tous les indicateurs  |
| `/api/v1/news`                       | GET     | Toutes les news       |
| `/api/v1/news/{symbol}`              | GET     | News par symbol       |
| `/health`                            | GET     | Health check          |
| `/ready`                             | GET     | Readiness check       |

### WebSocket

**URL:** `ws://localhost:8080/ws/crypto`

**Actions:**

- `subscribe` - S'abonner à un stream
- `unsubscribe` - Se désabonner
- `ping` - Test de connexion
- `list_subscriptions` - Lister les abonnements

**Types de données:**

- `candles` - Candles OHLCV
- `trades` - Trades individuels
- `indicators` - Indicateurs techniques
- `news` - Actualités

---

## 🔧 Configuration

### Variables d'Environnement

```dotenv
# .env.example (déjà configuré)
VITE_API_URL=http://localhost:8080
VITE_WS_URL=ws://localhost:8080/ws/crypto
VITE_USE_MOCK=true
```

### Pour activer le temps réel en dev

```bash
# 1. Créer .env.local
echo "VITE_USE_MOCK=false" > services/frontend-vue/.env.local

# 2. S'assurer que le backend est démarré
docker-compose up -d

# 3. Démarrer le frontend
cd services/frontend-vue
npm run dev
```

---

## 💡 Comment Utiliser

### Option 1 : Utiliser le composant example

```vue
<template>
  <RealtimeExample />
</template>

<script setup>
import RealtimeExample from "@/components/RealtimeExample.vue";
</script>
```

### Option 2 : Intégrer dans vos composants

```vue
<script setup lang="ts">
import { useMarketStore } from "@/stores/market";

const market = useMarketStore();

onMounted(async () => {
  await market.connectRealtime();
});

onUnmounted(() => {
  market.disconnectRealtime();
});
</script>

<template>
  <div v-if="market.rtConnected">
    Latest price: {{ market.tickers[market.activeSymbol]?.price }}
  </div>
</template>
```

### Option 3 : Appel direct du client

```typescript
import { getRTClient } from "@/services/rt";

const rt = getRTClient();

// Directement depuis n'importe où
await rt.connect();
rt.subscribe("trades", "BTCUSDT");
rt.on("trade", handler);
```

---

## 🎯 Architecture

```
┌─────────────────────────────────┐
│   Frontend Vue 3                │
│  ├─ Components                  │
│  │  └─ RealtimeExample.vue     │
│  ├─ Stores                      │
│  │  └─ market.ts (enrichi)      │
│  └─ Services                    │
│     └─ rt.ts (WebSocket)        │
└─────────────────────────────────┘
         ↓ ws://localhost:8080/ws/crypto
┌─────────────────────────────────┐
│   Backend Go                    │
│  ├─ WebSocket Hub               │
│  │  └─ ws/client.go             │
│  │  └─ ws/hub.go                │
│  ├─ Kafka Consumers             │
│  │  └─ Candles, Trades, etc.   │
│  └─ REST Controllers            │
│     └─ candle, indicator, news  │
└─────────────────────────────────┘
         ↓ Kafka topics
┌─────────────────────────────────┐
│   Message Broker (Kafka)        │
│  ├─ crypto.aggregated.*         │
│  ├─ crypto.raw.trades           │
│  ├─ crypto.indicators.*         │
│  └─ crypto.news                 │
└─────────────────────────────────┘
         ↓ Consume
┌─────────────────────────────────┐
│   Database (TimescaleDB)        │
│  ├─ candles                     │
│  ├─ trades                      │
│  ├─ indicators                  │
│  └─ news                        │
└─────────────────────────────────┘
```

---

## ✨ Fonctionnalités Clés

### Reconnection Automatique

- Exponential backoff (3s, 6s, 12s, 24s, 48s)
- Max 5 tentatives
- Reset du compteur après connexion réussie

### Subscription Management

- Wildcards (ex: `*` pour tous les symboles)
- Filtering côté serveur (évite le spam réseau)
- Facile à changer sans reconnecter

### Performance

- Buffer WebSocket limité (1000 messages)
- Déduplication Redis côté backend
- Compression des timeframes longs
- Limite de 500 candles en mémoire frontend

### Gestion d'Erreurs

- Messages d'erreur clairs
- Reconnection auto
- Graceful shutdown
- Validation des messages

---

## 🚀 Prochaines Étapes

1. **Importer le composant dans Dashboard**

   ```vue
   <RealtimeExample />
   ```

2. **Tester avec le backend**

   ```bash
   make up
   ```

3. **Configurer les environnements**

   - Dev: `VITE_USE_MOCK=false`
   - Prod: Vrai URL WebSocket

4. **Ajouter des indicateurs temps réel**
   - RSI live
   - MACD live
   - Alerts sur prix

---

## 📚 Fichiers Modifiés

| Fichier                          | Changement                           |
| -------------------------------- | ------------------------------------ |
| `services/rt.ts`                 | ✨ Nouveau service WebSocket complet |
| `stores/market.ts`               | 🔄 Actions temps réel ajoutées       |
| `components/RealtimeExample.vue` | ✨ Nouveau composant example         |
| `REALTIME_GUIDE.md`              | ✨ Documentation complète            |
| `.env.example`                   | (déjà configuré)                     |

---

## 💬 Support

Pour des questions :

1. Consulter `REALTIME_GUIDE.md`
2. Voir les exemples dans `RealtimeExample.vue`
3. Vérifier les logs de la console du navigateur
4. S'assurer que le backend et Kafka sont actifs
