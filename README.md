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

2. **Configuration initiale**
```bash
make setup
# Éditer .env avec vos clés API
```

3. **Démarrage des services**
```bash
make start
```

4. **Vérification**
```bash
make status
make health

# Accéder à l'interface
open http://localhost:3000
```

## 🛠️ Commandes Make

CryptoViz utilise un Makefile complet pour simplifier la gestion du projet. Toutes les commandes sont organisées par catégorie pour une utilisation optimale.

### Aide et Information
```bash
make help          # Afficher toutes les commandes disponibles
make info          # Informations détaillées du projet
```

### 🚀 Gestion des Services

#### Démarrage
```bash
make start          # Démarrer tous les services
make start-infra    # Démarrer uniquement l'infrastructure (DB, Kafka, Redis)
make start-services # Démarrer uniquement les microservices
make start-app      # Démarrer uniquement l'application (backend + frontend)
```

#### Arrêt et Redémarrage
```bash
make stop           # Arrêter tous les services
make stop-force     # Arrêt forcé de tous les services
make restart        # Redémarrer tous les services
make restart-service SERVICE=backend-go  # Redémarrer un service spécifique
```

### 🔧 Construction et Nettoyage

```bash
make build          # Construire toutes les images Docker
make build-service SERVICE=backend-go    # Construire une image spécifique
make clean          # Nettoyer les conteneurs, images et volumes
make clean-images   # Supprimer toutes les images Docker du projet
```

### 📊 Monitoring et Logs

```bash
make logs           # Voir les logs de tous les services en temps réel
make logs-service SERVICE=backend-go     # Logs d'un service spécifique
make status         # Afficher l'état de tous les services
make health         # Vérifier la santé des services
make monitor        # Ouvrir les interfaces de monitoring
```

### 🗄️ Base de Données TimescaleDB

```bash
make db-connect     # Se connecter à la base de données
make db-backup      # Créer une sauvegarde
make db-restore BACKUP=fichier.sql      # Restaurer depuis une sauvegarde
```

### 📡 Gestion Kafka

```bash
make kafka-topics   # Lister tous les topics Kafka
make kafka-create-topic TOPIC=nom_topic # Créer un nouveau topic
make kafka-console-consumer TOPIC=crypto.raw.1s  # Écouter un topic
```

### 🧪 Développement

#### Mode Développement
```bash
make dev-backend    # Démarrer le backend en mode développement
make dev-frontend   # Démarrer le frontend en mode développement
```

#### Tests
```bash
make test           # Exécuter tous les tests
make test-backend   # Tester le backend Go
make test-python    # Tester les services Python
make test-frontend  # Tester le frontend
```

#### Linting et Formatage
```bash
make lint           # Vérifier le code avec les linters
make lint-backend   # Linter le code Go
make lint-python    # Linter le code Python
make lint-frontend  # Linter le code frontend

make format         # Formater tout le code
make format-backend # Formater le code Go
make format-python  # Formater le code Python
make format-frontend # Formater le code frontend
```

### 🔧 Utilitaires

```bash
make shell-service SERVICE=backend-go   # Ouvrir un shell dans un service
make ps             # Afficher les processus Docker
make top            # Afficher l'utilisation des ressources
make update         # Mettre à jour les dépendances
```

### 🧪 Tests API

```bash
make api-test       # Tester l'API backend
make api-crypto SYMBOL=BTCUSDT         # Tester l'endpoint crypto
```

### 🚀 Production

```bash
make prod-build     # Construire pour la production
make prod-deploy    # Déployer en production
```

### 🧹 Maintenance

```bash
make prune          # Nettoyer Docker (images, conteneurs, volumes orphelins)
make reset          # Reset complet du projet (⚠️ supprime toutes les données)
```

### Exemples d'Utilisation

#### Démarrage complet du projet
```bash
# Configuration initiale (première fois)
make setup
# Éditer le fichier .env avec vos clés API

# Démarrage
make start
make status
```

#### Développement d'un service spécifique
```bash
# Démarrer l'infrastructure
make start-infra

# Développer le backend
make dev-backend

# Dans un autre terminal, voir les logs
make logs-service SERVICE=timescaledb
```

#### Debug et monitoring
```bash
# Voir les logs en temps réel
make logs

# Vérifier la santé des services
make health

# Se connecter à la base de données
make db-connect

# Écouter les messages Kafka
make kafka-console-consumer TOPIC=crypto.raw.1s
```

#### Tests et qualité de code
```bash
# Tests complets
make test

# Vérification du code
make lint

# Formatage du code
make format
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
TIMESCALE_PORT=7432
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

## 📚 Documentation

- **[Guide de Développement](DEV.md)** - Workflows optimisés pour les développeurs et étudiants
- **[API Reference](docs/api.md)** - Documentation des endpoints
- **[Wiki](https://github.com/T-DAT-901/CryptoViz/wiki)** - Documentation complète

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/T-DAT-901/CryptoViz/issues)
- **Guide Développeur**: [DEV.md](DEV.md) - Troubleshooting et bonnes pratiques

## 📄 Licence

MIT License - voir le fichier LICENSE pour plus de détails.
