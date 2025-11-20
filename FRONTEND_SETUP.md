# CryptoViz Frontend - Setup Guide

## Configuration Initiale

### 1. Installer les dépendances

```bash
cd services/frontend-vue
npm install
```

### 2. Configurer les variables d'environnement

```bash
# Copier le fichier exemple
cp .env.example .env.local
```

### 3. Éditer `.env.local` selon ton environnement

**Pour développement avec mockdata :**

```env
VITE_USE_MOCK=true
VITE_API_URL=http://localhost:8080
VITE_WS_URL=ws://localhost:8080
```

**Pour développement avec API réelle :**

```env
VITE_USE_MOCK=false
VITE_API_URL=http://localhost:8080
VITE_WS_URL=ws://localhost:8080
```

**Pour production :**

```env
VITE_USE_MOCK=false
VITE_API_URL=https://api.cryptoviz.com
VITE_WS_URL=wss://api.cryptoviz.com
```

### 4. Lancer le développement

```bash
npm run dev
```

---

## Connexion Backend ↔ Frontend

### Configuration Backend

Le backend doit écouter sur le port **8080** et exposer :

#### REST API

- **Base URL** : `http://localhost:8080/api/v1`
- **Endpoints requis** :
  - `GET /crypto/{symbol}/data?interval=1m&limit=120` → Liste des bougies (CandleDTO)
  - `GET /api/markets/tickers?symbols=BTC,ETH` → Infos des tickers
  - `GET /api/news` → Actualités crypto

#### WebSocket

- **Endpoint** : `ws://localhost:8080/ws/crypto`
- **Messages attendus** :
  ```json
  {
    "type": "price_update",
    "data": {
      "symbol": "BTCUSDT",
      "price": 67890.5,
      "change": 150.25,
      "changePercent": 0.22,
      "volume": 1000000,
      "high24h": 68000,
      "low24h": 67000
    }
  }
  ```
  ```json
  {
    "type": "candle_update",
    "data": {
      "time": "2024-04-05T17:00:00Z",
      "open": 67800,
      "high": 67950,
      "low": 67750,
      "close": 67890,
      "volume": 1000
    }
  }
  ```

### Configuration Frontend

#### Format des données (CandleDTO)

Le frontend s'attend à recevoir les bougies dans ce format :

```typescript
interface CandleDTO {
  time: string; // ISO 8601 string (ex: "2024-04-05T17:00:00Z")
  open: number; // Prix d'ouverture
  high: number; // Plus haut
  low: number; // Plus bas
  close: number; // Prix de fermeture
  volume: number; // Volume d'échange
}
```

#### CORS

Le backend doit autoriser les requêtes CORS depuis le frontend :

```
Access-Control-Allow-Origin: http://localhost:3000
Access-Control-Allow-Methods: GET, POST, OPTIONS
Access-Control-Allow-Headers: Content-Type
```

---

## Mode Mock vs API

### Mode Mock (VITE_USE_MOCK=true)

- ✅ Aucune dépendance au backend
- ✅ Données de test en local
- ✅ Parfait pour le frontend développement
- 📄 Données dans : `src/services/mocks/`

### Mode API (VITE_USE_MOCK=false)

- ✅ Données réelles du backend
- ⚠️ Backend doit être lancé sur :8080
- ✅ WebSocket pour les mises à jour temps réel

---

## Troubleshooting

### "Cannot GET /crypto/BTC/data"

- ❌ Backend n'est pas lancé ou pas sur le bon port
- ✅ Vérifier : `http://localhost:8080/crypto/BTC/data`

### WebSocket connection failed

- ❌ Backend n'accepte pas les connexions WebSocket
- ✅ Vérifier : `ws://localhost:8080/ws/crypto`

### Pas de mise à jour en temps réel

- ❌ WebSocket connecté mais pas de messages reçus
- ✅ Le backend doit envoyer les messages au format JSON attendu

### Erreur CORS

- ❌ Le backend refuse les requêtes du frontend
- ✅ Vérifier les headers CORS dans la config backend

---

## Commandes utiles

```bash
# Développement
npm run dev

# Build production
npm run build

# Preview production
npm run preview

# Linting
npm run lint

# Type checking
npm run type-check
```

---

## Architecture Frontend

```
services/frontend-vue/
├── src/
│   ├── components/       # Composants Vue
│   │   └── charts/      # Graphiques (CandleChart, RSIChart, etc.)
│   ├── services/        # Appels API et WebSocket
│   │   ├── http.ts      # Client Axios
│   │   ├── websocket.ts # Gestion temps réel
│   │   └── markets.api.ts # Endpoints API
│   ├── stores/          # État Pinia
│   │   ├── market.ts    # État des données marché
│   │   └── indicators.ts # Configuration indicateurs
│   ├── types/           # Interfaces TypeScript
│   │   └── market.d.ts  # Types CandleDTO, TickerDTO, etc.
│   └── utils/           # Utilitaires
│       └── mockTransform.ts # Transformation données mock
```

---

## Support

Pour toute question ou problème :

1. Vérifier les logs du navigateur (F12)
2. Vérifier les logs du backend
3. Tester les endpoints manuellement avec curl/Postman
