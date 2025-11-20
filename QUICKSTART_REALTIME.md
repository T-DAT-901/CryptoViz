# 🚀 Guide Rapide - Données Temps Réel

## Qu'est-ce qu'on vient d'implémenter ?

Tu as maintenant une **infrastructure complète pour les données en temps réel** :

### 📡 WebSocket Client (`services/rt.ts`)

- Connexion/Reconnection automatique
- Subscribe/Unsubscribe à des streams
- Gestion d'erreurs et recovery

### 📊 State Management (`stores/market.ts`)

- Actions pour connecter/déconnecter
- Synchronisation automatique des données
- Switching facile de symbol/timeframe

### 🎨 Composant Example (`components/RealtimeExample.vue`)

- Interface complète et stylisée
- Connecter/Déconnecter en 1 clic
- Afficher prix + candle en temps réel

### 📖 Documentation (`REALTIME_GUIDE.md`)

- Tous les endpoints REST
- Protocole WebSocket détaillé
- Exemples de code complets

---

## ⚡ Démarrage Rapide

### 1️⃣ Activer le mode temps réel

```bash
# Dans services/frontend-vue/.env.local
VITE_USE_MOCK=false
```

### 2️⃣ Démarrer le backend

```bash
make up  # Ou docker-compose up
```

### 3️⃣ Utiliser dans ton composant

```vue
<script setup lang="ts">
import { useMarketStore } from "@/stores/market";

const market = useMarketStore();

// Connecter au WebSocket
await market.connectRealtime();

// C'est tout ! Les données arrivent en temps réel 🎉
</script>

<template>
  <div v-if="market.rtConnected">
    Prix: ${{ market.tickers[market.activeSymbol]?.price }}
  </div>
</template>
```

---

## 📚 Cas d'Usage

### Afficher le prix en temps réel

```typescript
rt.subscribe("trades", "BTCUSDT");
rt.on("trade", (msg) => {
  console.log("Nouveau prix:", msg.data.price);
});
```

### Afficher les candles mises à jour

```typescript
rt.subscribe("candles", "BTCUSDT", "1m");
rt.on("candle", (msg) => {
  const { time, close, volume } = msg.data;
  console.log(`${time}: Close=$${close}, Vol=${volume}`);
});
```

### Écouter toutes les news

```typescript
rt.subscribe("news", "*");
rt.on("news", (msg) => {
  console.log("Nouvelle news:", msg.data.title);
});
```

---

## 🔌 Architecture Simple

```
Ta Vue                WebSocket               Backend
Component ──→ RTClient ──→ ws://localhost:8080 ──→ Kafka Consumers
   ↓                                                   ↓
Store                                            TimescaleDB
   ↓
Reactive data ──→ Update automatique de l'UI
```

---

## ✅ Checklist

- [x] Service WebSocket implémenté
- [x] Store Pinia enrichi
- [x] Composant example créé
- [x] Documentation complète
- [x] Pas d'erreurs TypeScript
- [x] Fallback mock data pour le dev

---

## 🎯 Backend : Endpoints Disponibles

| Type      | Action         | Exemple                                       |
| --------- | -------------- | --------------------------------------------- |
| WebSocket | Subscribe      | `rt.subscribe("candles", "BTCUSDT", "1m")`    |
| WebSocket | Unsubscribe    | `rt.unsubscribe("trades", "BTCUSDT")`         |
| REST      | Get candles    | `GET /api/v1/crypto/BTCUSDT/data?interval=1m` |
| REST      | Get price      | `GET /api/v1/crypto/BTCUSDT/latest`           |
| REST      | Get indicators | `GET /api/v1/indicators/BTCUSDT/rsi`          |
| REST      | Get news       | `GET /api/v1/news`                            |

---

## 🐛 Dépannage

### Pas de connexion WebSocket ?

```bash
# Vérifier que le backend est actif
curl http://localhost:8080/health

# Vérifier VITE_USE_MOCK dans .env.local
cat services/frontend-vue/.env.local
```

### Pas de données après la connexion ?

1. Vérifier `rt.isConnected()` → doit être `true`
2. Vérifier `rt.getSubscriptions()` → doit contenir ton abonnement
3. Vérifier les logs backend pour les erreurs Kafka

### Données obsolètes en mode mock ?

```bash
# Désactiver le mock
echo "VITE_USE_MOCK=false" > services/frontend-vue/.env.local
```

---

## 📖 Documentation Détaillée

Pour plus d'infos, consulte :

- `REALTIME_GUIDE.md` - Guide complet
- `REALTIME_IMPLEMENTATION.md` - Résumé de l'implémentation
- `services/backend-go/ARCHITECTURE.md` - Architecture backend

---

## 🚀 Prochaines Étapes Suggérées

1. **Intégrer le composant dans ton Dashboard**

   ```vue
   <RealtimeExample />
   ```

2. **Ajouter des indicateurs temps réel**

   - RSI live chart
   - MACD live chart
   - Alerts sur seuils

3. **Optimiser les performances**

   - Déduplication des messages
   - Compression des données
   - Caching local

4. **Ajouter la persistance**
   - Sauvegarder les données locales
   - Replay après reconnection

---

## 💡 Tips

- Utilise le mode mock pour tester sans backend
- Toujours unsubscribe quand tu changes de composant
- Les Wildcards (`*`) peuvent être utiles mais causent du spam
- Le buffer WebSocket est limité à 1000 messages

---

## 📞 Questions ?

Tout est documenté dans `REALTIME_GUIDE.md` !

Bon coding ! 🎉
