# Configuration des Ports - CryptoViz

## 📋 Ports Utilisés

Pour éviter les conflits avec les services locaux existants, CryptoViz utilise des ports personnalisés pour l'accès externe aux services d'infrastructure.

### 🔌 Mapping des Ports

| Service | Port Externe | Port Interne | Description |
|---------|--------------|--------------|-------------|
| **TimescaleDB** | `7432` | `5432` | Base de données PostgreSQL/TimescaleDB |
| **Redis** | `7379` | `6379` | Cache Redis |
| **Kafka** | `9092` | `9092` | Message broker Apache Kafka |
| **Backend Go** | `8080` | `8080` | API REST et WebSocket |
| **Frontend Vue** | `3000` | `80` | Interface utilisateur |

### 🔧 Connexions Externes

#### Base de Données TimescaleDB
```bash
# Connexion via psql
psql -h localhost -p 7432 -U postgres -d cryptoviz

# Connexion via pgAdmin ou autres outils GUI
Host: localhost
Port: 7432
Database: cryptoviz
Username: postgres
Password: [voir .env]
```

#### Redis
```bash
# Connexion via redis-cli
redis-cli -h localhost -p 7379

# Test de connexion
redis-cli -h localhost -p 7379 ping
```

#### API Backend
```bash
# Health check
curl http://localhost:8080/health

# API endpoints
curl http://localhost:8080/api/v1/crypto/BTCUSDT/latest
```

#### Frontend
```bash
# Interface web
http://localhost:3000
```

### 🐳 Connexions Internes (Docker)

Les services communiquent entre eux via le réseau Docker interne avec les ports standards :

```yaml
# Exemple de configuration interne
TIMESCALE_HOST=timescaledb
TIMESCALE_PORT=5432  # Port interne standard

REDIS_HOST=redis
REDIS_PORT=6379      # Port interne standard
```

### ⚙️ Configuration des Variables d'Environnement

#### Pour les Connexions Externes (.env)
```bash
# Ports externes pour les outils de développement
TIMESCALE_PORT=7432
REDIS_PORT=7379
```

#### Pour les Services Docker (automatique)
Les services Docker utilisent automatiquement les ports internes via les noms de services.

### 🛠️ Outils de Développement

#### Connexion à la Base de Données
```bash
# Via Make (recommandé)
make db-connect

# Manuellement
docker exec -it cryptoviz-timescaledb psql -U postgres -d cryptoviz

# Depuis l'extérieur
psql -h localhost -p 7432 -U postgres -d cryptoviz
```

#### Monitoring Redis
```bash
# Via Docker
docker exec -it cryptoviz-redis redis-cli

# Depuis l'extérieur
redis-cli -h localhost -p 7379
```

### 🔍 Vérification des Ports

#### Vérifier que les ports sont libres
```bash
# Linux/macOS
netstat -tuln | grep -E ':(7432|7379|9092|8080|3000)'

# Windows
netstat -an | findstr "7432 7379 9092 8080 3000"
```

#### Tester les connexions
```bash
# TimescaleDB
telnet localhost 7432

# Redis
telnet localhost 7379

# Backend API
curl -I http://localhost:8080/health
```

### 🚨 Résolution de Problèmes

#### Port déjà utilisé
Si vous obtenez une erreur "port already in use" :

1. **Identifier le processus utilisant le port**
```bash
# Linux/macOS
lsof -i :7432
lsof -i :7379

# Windows
netstat -ano | findstr :7432
```

2. **Arrêter le service local**
```bash
# PostgreSQL
sudo systemctl stop postgresql
# ou sur macOS
brew services stop postgresql

# Redis
sudo systemctl stop redis
# ou sur macOS
brew services stop redis
```

3. **Ou modifier les ports dans docker-compose.yml**
```yaml
# Exemple pour utiliser d'autres ports
timescaledb:
  ports:
    - "5433:5432"  # Au lieu de 7432:5432

redis:
  ports:
    - "6380:6379"  # Au lieu de 7379:6379
```

### 📝 Notes Importantes

1. **Ports Externes vs Internes** : Les ports externes (7432, 7379) sont uniquement pour l'accès depuis votre machine hôte. Les services Docker communiquent entre eux via les ports internes standards.

2. **Sécurité** : Ces ports ne sont accessibles que depuis localhost. En production, utilisez des configurations de sécurité appropriées.

3. **Cohérence d'Équipe** : Tous les développeurs de l'équipe doivent utiliser les mêmes ports pour éviter les conflits de configuration.

4. **Documentation** : Mettez à jour cette documentation si vous modifiez les ports.

### 🔄 Retour aux Ports Standards

Si vous souhaitez revenir aux ports standards (5432, 6379), modifiez :

1. `docker-compose.yml` - sections ports
2. `.env.example` - variables TIMESCALE_PORT et REDIS_PORT
3. Documentation d'équipe

```bash
# Après modification
make stop
make start
