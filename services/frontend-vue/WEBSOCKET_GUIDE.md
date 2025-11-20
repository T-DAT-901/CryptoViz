# WebSocket - Guide Front-End

## 🎯 Ce que tu dois demander au back-end :

### 1. **URL du WebSocket**

```
wss://votre-api.com/ws/crypto
```

### 2. **Format des messages reçus**

Exemple de ce que tu vas recevoir :

```json
{
  "type": "price_update",
  "symbol": "BTC/USDT",
  "price": 68150.5,
  "change": -120.3,
  "changePercent": -0.18,
  "volume": 1234567,
  "high24h": 69000,
  "low24h": 67800,
  "timestamp": 1695123456
}
```

### 3. **Types d'événements disponibles**

- `price_update` : Prix en temps réel
- `candle_update` : Nouvelles bougies
- `volume_update` : Volume de trading
- `heartbeat` : Keep-alive

### 4. **Authentification (si nécessaire)**

- Token d'accès ?
- Headers spéciaux ?

## 🔧 Configuration actuelle :

Le code utilise **Binance en démo** pour que tu puisses développer.

Quand le back-end sera prêt :

1. Remplace l'URL dans `websocket.ts`
2. Adapte `handleMessage()` au nouveau format
3. Teste la connexion

## 📝 Questions à poser au back-end :

1. **"Quelle est l'URL du WebSocket crypto ?"**
2. **"Quel format de données vous envoyez ?"**
3. **"Y a-t-il une authentification requise ?"**
4. **"Quelle fréquence de mise à jour ?"**
5. **"Comment gérer les reconnexions ?"**

## ✅ Ce qui est déjà prêt côté front :

- ✅ Connexion WebSocket
- ✅ Gestion des erreurs
- ✅ Reconnexion automatique
- ✅ Interface utilisateur
- ✅ Affichage temps réel

Tu n'as plus qu'à brancher sur le vrai WebSocket ! 🚀
