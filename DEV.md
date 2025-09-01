# 🎓 Guide de Développement CryptoViz

> **Guide pratique pour les étudiants** - Workflows optimisés, bonnes pratiques et troubleshooting

## 📋 Table des Matières

- [🚀 Premier Jour - Setup Initial](#-premier-jour---setup-initial)
- [💻 Développement Quotidien](#-développement-quotidien)
- [🔧 Workflows par Scénario](#-workflows-par-scénario)
- [🐛 Debugging et Troubleshooting](#-debugging-et-troubleshooting)
- [👥 Travail en Équipe](#-travail-en-équipe)
- [⚡ Optimisations et Bonnes Pratiques](#-optimisations-et-bonnes-pratiques)
- [🆘 Erreurs Communes et Solutions](#-erreurs-communes-et-solutions)
- [📚 Ressources et Liens Utiles](#-ressources-et-liens-utiles)

---

## 🚀 Premier Jour - Setup Initial

### Étape 1 : Cloner et Configurer

```bash
# Cloner le projet
git clone https://github.com/T-DAT-901/CryptoViz.git
cd CryptoViz

# Configuration initiale (crée le fichier .env)
make setup
```

### Étape 2 : Configurer les Clés API

Éditez le fichier `.env` créé :

```bash
# Ouvrir avec votre éditeur préféré
code .env
# ou
nano .env
```

**⚠️ Important :** Demandez les clés API à votre équipe ou créez un compte Binance (testnet pour le développement).

### Étape 3 : Premier Démarrage Complet

```bash
# Démarrage complet (prend 5-10 minutes la première fois)
make start

# Vérifier que tout fonctionne
make status
make health
```

**✅ Succès :** Vous devriez voir tous les services "Up" et "healthy".

---

## 💻 Développement Quotidien

### 🌅 Routine du Matin (Démarrage Rapide)

```bash
# 1. Vérifier l'état des services
make status

# 2. Si l'infrastructure tourne déjà, parfait !
# Sinon, démarrer uniquement l'infrastructure
make start-infra

# 3. Démarrer votre service en mode développement
make dev-backend    # Pour le backend Go
# ou
make dev-frontend   # Pour le frontend Vue.js
```

**💡 Astuce :** `make dev-*` lance les services en mode hot-reload sans Docker, c'est beaucoup plus rapide !

### 🌙 Routine du Soir

```bash
# Arrêter tous les services (garde les données)
make stop
```

### 🔄 Pendant le Développement

```bash
# Voir les logs en temps réel
make logs

# Logs d'un service spécifique
make logs-service SERVICE=backend-go

# Redémarrer un service après modification
make restart-service SERVICE=data-collector

# Tester l'API
make api-test
```

---

## 🔧 Workflows par Scénario

### Scénario 1 : "Je développe le Backend Go"

```bash
# 1. Démarrer l'infrastructure
make start-infra

# 2. Mode développement backend (hot reload)
make dev-backend

# 3. Dans un autre terminal, voir les logs de la DB
make logs-service SERVICE=timescaledb

# 4. Tester l'API
make api-test
curl http://localhost:8080/health
```

### Scénario 2 : "Je développe le Frontend Vue.js"

```bash
# 1. Démarrer l'infrastructure + backend
make start-infra
make restart-service SERVICE=backend-go

# 2. Mode développement frontend
make dev-frontend

# 3. Ouvrir http://localhost:3000 dans le navigateur
```

### Scénario 3 : "Je développe un Microservice Python"

```bash
# 1. Démarrer l'infrastructure
make start-infra

# 2. Modifier le code dans services/data-collector/

# 3. Redémarrer le service
make restart-service SERVICE=data-collector

# 4. Voir les logs
make logs-service SERVICE=data-collector

# 5. Tester avec Kafka
make kafka-console-consumer TOPIC=crypto.raw.1s
```

### Scénario 4 : "Je teste une nouvelle fonctionnalité"

```bash
# 1. Build et test complet
make build
make test

# 2. Démarrage propre
make clean
make start

# 3. Tests d'intégration
make api-test
```

---

## 🐛 Debugging et Troubleshooting

### 🔍 Diagnostic Rapide

```bash
# Voir l'état de tous les services
make status

# Vérifier la santé des services
make health

# Voir les processus Docker
make ps

# Utilisation des ressources
make top
```

### 📋 Checklist de Debug

1. **Service ne démarre pas ?**
   ```bash
   make logs-service SERVICE=nom_du_service
   ```

2. **Erreur de connexion à la DB ?**
   ```bash
   make db-connect
   # Si ça marche, le problème est ailleurs
   ```

3. **Kafka ne fonctionne pas ?**
   ```bash
   make kafka-topics
   # Doit lister les topics
   ```

4. **API ne répond pas ?**
   ```bash
   make api-test
   curl http://localhost:8080/health
   ```

### 🔧 Outils de Debug Avancés

```bash
# Ouvrir un shell dans un service
make shell-service SERVICE=backend-go

# Se connecter à la base de données
make db-connect

# Écouter les messages Kafka
make kafka-console-consumer TOPIC=crypto.raw.1s

# Voir les métriques système
make monitor
```

---

## 👥 Travail en Équipe

### 🔄 Synchronisation avec l'Équipe

```bash
# Récupérer les dernières modifications
git pull origin main

# Mettre à jour les dépendances
make update

# Rebuild si nécessaire (nouvelles dépendances)
make build
```

### 📝 Bonnes Pratiques Git

```bash
# Avant de commencer à coder
git pull origin main
make status  # Vérifier que tout fonctionne

# Pendant le développement
git add .
git commit -m "feat: ajout endpoint crypto data"

# Avant de push
make test    # S'assurer que les tests passent
git push origin feature/crypto-endpoint
```

### 🚫 À Éviter

- ❌ Committer le fichier `.env` (contient les clés API)
- ❌ Faire `make start` à chaque modification
- ❌ Laisser tous les services tourner en permanence
- ❌ Ignorer les logs d'erreur

---

## ⚡ Optimisations et Bonnes Pratiques

### 🚀 Accélérer le Développement

1. **Utilisez les modes dev** : `make dev-backend` au lieu de Docker
2. **Démarrage sélectif** : `make start-infra` puis services individuels
3. **Logs ciblés** : `make logs-service SERVICE=...` au lieu de `make logs`
4. **Cache Docker** : Évitez `make clean` sauf si nécessaire

### 💾 Économiser les Ressources

```bash
# Arrêter les services non utilisés
make stop

# Nettoyer l'espace disque (attention : supprime les données)
make prune

# Voir l'utilisation des ressources
make top
```

### 🎯 Workflow Optimal par Rôle

**Backend Developer :**
```bash
make start-infra → make dev-backend → make api-test
```

**Frontend Developer :**
```bash
make start-infra → make restart-service SERVICE=backend-go → make dev-frontend
```

**DevOps/Full-Stack :**
```bash
make start → make test → make monitor
```

---

## 🆘 Erreurs Communes et Solutions

### ❌ "Service failed to build"

**Problème :** Erreur de compilation Docker

**Solutions :**
```bash
# 1. Voir les logs détaillés
make logs-service SERVICE=nom_du_service

# 2. Rebuild propre
make clean
make build-service SERVICE=nom_du_service

# 3. Vérifier les dépendances
make update
```

### ❌ "Port already in use"

**Problème :** Un service utilise déjà le port

**Solutions :**
```bash
# 1. Voir ce qui utilise le port
make ps

# 2. Arrêter tous les services
make stop

# 3. Redémarrer proprement
make start
```

### ❌ "Cannot connect to database"

**Problème :** TimescaleDB non accessible

**Solutions :**
```bash
# 1. Vérifier que la DB tourne
make status

# 2. Tester la connexion
make db-connect

# 3. Redémarrer la DB
make restart-service SERVICE=timescaledb
```

### ❌ "Kafka consumer not receiving messages"

**Problème :** Messages Kafka non reçus

**Solutions :**
```bash
# 1. Vérifier les topics
make kafka-topics

# 2. Tester manuellement
make kafka-console-consumer TOPIC=crypto.raw.1s

# 3. Redémarrer Kafka
make restart-service SERVICE=kafka
```

### ❌ "Frontend shows blank page"

**Problème :** Interface Vue.js ne charge pas

**Solutions :**
```bash
# 1. Vérifier les logs frontend
make logs-service SERVICE=frontend-vue

# 2. Tester le backend
make api-test

# 3. Mode dev pour debug
make dev-frontend
```

---

## 📚 Ressources et Liens Utiles

### 📖 Documentation Technique

- **[README Principal](README.md)** - Architecture et overview
- **[API Documentation](docs/api.md)** - Endpoints et schemas
- **[Docker Compose](docker-compose.yml)** - Configuration des services

### 🌐 URLs de Développement

- **Frontend** : http://localhost:3000
- **Backend API** : http://localhost:8080
- **Health Check** : http://localhost:8080/health
- **TimescaleDB** : localhost:5432 (user: postgres)
- **Kafka** : localhost:9092
- **Redis** : localhost:6379

### 🛠️ Outils Recommandés

**Éditeurs :**
- VS Code avec extensions Go, Vue.js, Docker
- IntelliJ IDEA / PyCharm
- Vim/Neovim pour les puristes

**Clients API :**
- Postman / Insomnia
- curl (ligne de commande)
- Thunder Client (VS Code)

**Monitoring :**
- Docker Desktop
- Portainer (interface Docker)
- pgAdmin (PostgreSQL/TimescaleDB)

### 📱 Extensions VS Code Utiles

```bash
# Extensions recommandées
code --install-extension ms-vscode.go
code --install-extension Vue.volar
code --install-extension ms-azuretools.vscode-docker
code --install-extension ms-python.python
```

### 🔗 Liens Externes

- **[Binance API Docs](https://binance-docs.github.io/apidocs/)**
- **[Vue.js 3 Guide](https://vuejs.org/guide/)**
- **[Go Documentation](https://golang.org/doc/)**
- **[TimescaleDB Docs](https://docs.timescale.com/)**
- **[Apache Kafka Docs](https://kafka.apache.org/documentation/)**

---

## 🎯 Résumé des Commandes Essentielles

### 🚀 Démarrage
```bash
make setup          # Configuration initiale
make start-infra    # Infrastructure seulement
make dev-backend    # Mode développement backend
make dev-frontend   # Mode développement frontend
```

### 🔍 Monitoring
```bash
make status         # État des services
make logs           # Logs de tous les services
make health         # Vérification santé
make api-test       # Test de l'API
```

### 🔧 Maintenance
```bash
make restart        # Redémarrage rapide
make clean          # Nettoyage complet
make update         # Mise à jour dépendances
make test           # Tests complets
```

---

**💡 Conseil Final :** Gardez ce guide ouvert pendant vos premières semaines de développement. N'hésitez pas à expérimenter avec les commandes - le projet est conçu pour être robuste !

**🆘 Besoin d'aide ?** Consultez la section [Erreurs Communes](#-erreurs-communes-et-solutions) ou demandez à votre équipe.

---

*Dernière mise à jour : Septembre 2025*
