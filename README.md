# CryptoViz - Terminal de Trading Crypto en Temps Réel

## 🏗️ Architecture

CryptoViz est une plateforme de visualisation de données crypto en temps réel, conçue comme un terminal de trading professionnel. L'architecture microservices permet une scalabilité et une maintenance optimales.

### Stack Technique

- **Frontend**: Vue.js 3 + Chart.js/D3.js
- **Backend**: Go (Gin framework)
- **Base de données**: TimescaleDB (PostgreSQL optimisé time-series)
- **Message Broker**: Apache Kafka
- **Microservices**: Python 3.11+
- **Containerisation**: Docker + Docker Compose
- **Cache**: Redis (optionnel)

## 📊 Flux de Données

```
Binance API (WebSocket) → Data Collector → Kafka → TimescaleDB
                                           ↓
Yahoo Finance → News Scraper → Kafka → Backend Go ← Frontend Vue.js
                                           ↓
Kafka → Indicators Calculator → TimescaleDB
```

## 🏢 Architecture des Services

### 1. Data Collector (Python)
- **Rôle**: Collecte des données crypto depuis Binance API
- **Technologies**: Python, WebSocket, Kafka Producer
- **Données**: Prix temps réel, volumes, orderbook
- **Intervalles**: 1s, 5s, 1min, 15min, 1h

### 2. News Scraper (Python)
- **Rôle**: Scraping des actualités crypto depuis Yahoo Finance
- **Technologies**: Python, BeautifulSoup/Scrapy, Kafka Producer
- **Fréquence**: Toutes les 5 minutes
- **Données**: Titre, contenu, sentiment, timestamp

### 3. Indicators Calculator (Python)
- **Rôle**: Calcul des indicateurs techniques
- **Technologies**: Python, TA-Lib, pandas, Kafka Consumer/Producer
- **Indicateurs**: RSI, MACD, Bollinger Bands, Momentum
- **Traitement**: Temps réel + historique

### 4. Backend Go
- **Rôle**: API REST et WebSocket pour le frontend
- **Technologies**: Go, Gin, WebSocket, TimescaleDB
- **Endpoints**:
  - `/api/v1/crypto/{symbol}/data` - Données historiques
  - `/ws/crypto` - Stream temps réel
  - `/api/v1/indicators/{symbol}` - Indicateurs techniques
  - `/api/v1/news` - Actualités

### 5. Frontend Vue.js
- **Rôle**: Interface utilisateur interactive
- **Technologies**: Vue.js 3, Chart.js, WebSocket
- **Composants**:
  - Dashboard principal avec graphiques temps réel
  - Sélecteur d'intervalles (1s, 5s, 1min, 15min, 1h)
  - Panneau d'indicateurs techniques
  - Feed d'actualités
  - Interface responsive

### 6. TimescaleDB
- **Rôle**: Stockage optimisé des données time-series
- **Partitioning**: Par symbole et intervalle de temps
- **Rétention**: Compression automatique après 7 jours
- **Indexation**: Optimisée pour les requêtes temporelles

### 7. Apache Kafka
- **Rôle**: Message broker pour streaming temps réel
- **Topics**:
  - `crypto.raw.1s` - Données brutes 1 seconde
  - `crypto.aggregated.{interval}` - Données agrégées
  - `crypto.indicators.{type}` - Indicateurs calculés
  - `crypto.news` - Actualités

## 🗂️ Structure du Projet

```
CryptoViz/
├── README.md
├── docker-compose.yml
├── .env.example
├── docs/
│   ├── api.md
│   └── deployment.md
├── services/
│   ├── backend-go/
│   │   ├── Dockerfile
│   │   ├── main.go
│   │   ├── handlers/
│   │   ├── models/
│   │   └── config/
│   ├── frontend-vue/
│   │   ├── Dockerfile
│   │   ├── package.json
│   │   ├── src/
│   │   └── public/
│   ├── data-collector/
│   │   ├── Dockerfile
│   │   ├── requirements.txt
│   │   ├── main.py
│   │   └── collectors/
│   ├── indicators-calculator/
│   │   ├── Dockerfile
│   │   ├── requirements.txt
│   │   ├── main.py
│   │   └── calculators/
│   └── news-scraper/
│       ├── Dockerfile
│       ├── requirements.txt
│       ├── main.py
│       └── scrapers/
├── database/
│   ├── init.sql
│   └── migrations/
└── kafka/
    └── topics.sh
```

## 🚀 Démarrage Rapide

### Prérequis
- Docker 20.10+
- Docker Compose 2.0+
- 8GB RAM minimum
- 20GB espace disque

### Installation

1. **Cloner le repository**
```bash
git clone https://github.com/T-DAT-901/CryptoViz.git
cd CryptoViz
```

2. **Configuration**
```bash
cp .env.example .env
# Éditer .env avec vos clés API
```

3. **Démarrage des services**
```bash
docker-compose up -d
```

4. **Vérification**
```bash
# Vérifier que tous les services sont up
docker-compose ps

# Accéder à l'interface
open http://localhost:3000
```

## 📈 Gestion des Données

### Intervalles de Temps
- **1s**: Données brutes temps réel (rétention 24h)
- **5s**: Agrégation (rétention 7 jours)
- **1min**: Agrégation (rétention 30 jours)
- **15min**: Agrégation (rétention 6 mois)
- **1h**: Agrégation (rétention 2 ans)

### Partitioning TimescaleDB
```sql
-- Partitioning par symbole et temps
SELECT create_hypertable('crypto_data', 'time',
    partitioning_column => 'symbol',
    number_partitions => 50);

-- Compression automatique
SELECT add_compression_policy('crypto_data', INTERVAL '7 days');

-- Rétention des données
SELECT add_retention_policy('crypto_data', INTERVAL '2 years');
```

### Indicateurs Techniques Supportés

| Indicateur | Description | Paramètres |
|------------|-------------|------------|
| RSI | Relative Strength Index | Période: 14 |
| MACD | Moving Average Convergence Divergence | 12, 26, 9 |
| Bollinger Bands | Bandes de Bollinger | Période: 20, Écart: 2 |
| Momentum | Momentum | Période: 10 |

## 🔧 Configuration

### Variables d'Environnement

```bash
# API Keys
BINANCE_API_KEY=your_binance_api_key
BINANCE_SECRET_KEY=your_binance_secret_key

# Database
TIMESCALE_HOST=timescaledb
TIMESCALE_PORT=5432
TIMESCALE_DB=cryptoviz
TIMESCALE_USER=postgres
TIMESCALE_PASSWORD=password

# Kafka
KAFKA_BROKERS=kafka:9092
KAFKA_TOPICS=crypto.raw.1s,crypto.aggregated.5s,crypto.aggregated.1m

# Services
BACKEND_PORT=8080
FRONTEND_PORT=3000
```

## 📊 Monitoring

### Métriques Disponibles
- Latence des WebSockets
- Throughput Kafka
- Utilisation CPU/RAM par service
- Taille de la base de données
- Nombre de connexions actives

### Logs
```bash
# Logs en temps réel
docker-compose logs -f

# Logs d'un service spécifique
docker-compose logs -f backend-go
```

## 🔒 Sécurité

- Clés API stockées dans variables d'environnement
- Communication inter-services via réseau Docker privé
- Rate limiting sur les APIs externes
- Validation des données d'entrée

## 🚀 Déploiement Production

### Optimisations Recommandées
- Utiliser des volumes Docker persistants
- Configurer la réplication TimescaleDB
- Mettre en place un load balancer
- Activer la compression Kafka
- Configurer les alertes de monitoring

## 📝 Développement

### Ajout d'un Nouvel Indicateur
1. Créer le calculateur dans `services/indicators-calculator/calculators/`
2. Ajouter la configuration Kafka
3. Mettre à jour l'API backend
4. Ajouter le composant frontend

### Tests
```bash
# Tests unitaires
docker-compose -f docker-compose.test.yml up

# Tests d'intégration
./scripts/integration-tests.sh
```

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/T-DAT-901/CryptoViz/issues)
- **Documentation**: [Wiki](https://github.com/T-DAT-901/CryptoViz/wiki)
- **API Reference**: `/docs/api.md`

## 📄 Licence

MIT License - voir le fichier LICENSE pour plus de détails.
