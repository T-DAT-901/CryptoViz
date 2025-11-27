# =============================================================================
# CryptoViz Makefile
# =============================================================================

.PHONY: help start stop restart build clean logs status test lint format mac-start mac-start-monitoring mac-build mac-clean mac-reset-backfill windows-start windows-start-monitoring windows-build windows-clean windows-reset-backfill debug-backfill debug-timescale check-docker-resources tiering tiering-stats

# Variables
COMPOSE_FILE = docker-compose.yml
SERVICES = timescaledb zookeeper kafka kafka-ui redis minio data-collector news-scraper backend-go indicators-scheduler frontend-vue prometheus grafana node-exporter cadvisor postgres-exporter redis-exporter gatus

# Platform Detection and Docker Desktop Override
# Detects Mac (Darwin) and Windows/WSL2 (microsoft in kernel)
UNAME_S := $(shell uname -s)
UNAME_R := $(shell uname -r)

ifeq ($(UNAME_S),Darwin)
	# Mac Docker Desktop
	DESKTOP_OVERRIDE = -f docker-compose.mac.yml
	IS_DESKTOP = true
	IS_MAC = true
	IS_WSL = false
	PLATFORM = Mac
else ifeq ($(findstring microsoft,$(UNAME_R)),microsoft)
	# Windows Docker Desktop via WSL2
	DESKTOP_OVERRIDE = -f docker-compose.mac.yml
	IS_DESKTOP = true
	IS_MAC = false
	IS_WSL = true
	PLATFORM = Windows (WSL2)
else ifeq ($(findstring Microsoft,$(UNAME_R)),Microsoft)
	# Windows Docker Desktop via WSL1 (rare)
	DESKTOP_OVERRIDE = -f docker-compose.mac.yml
	IS_DESKTOP = true
	IS_MAC = false
	IS_WSL = true
	PLATFORM = Windows (WSL)
else
	# Native Linux
	DESKTOP_OVERRIDE =
	IS_DESKTOP = false
	IS_MAC = false
	IS_WSL = false
	PLATFORM = Linux
endif

# Couleurs pour l'affichage
GREEN = \033[0;32m
YELLOW = \033[1;33m
RED = \033[0;31m
BLUE = \033[0;34m
NC = \033[0m # No Color

# Aide par défaut
help: ## Afficher cette aide
	@echo "$(GREEN)CryptoViz - Commandes disponibles ($(PLATFORM)):$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-25s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(GREEN)Exemples d'utilisation:$(NC)"
ifeq ($(IS_DESKTOP),true)
	@echo "  $(BLUE)✓ Auto-détection Docker Desktop activée ($(PLATFORM))$(NC)"
	@echo ""
endif
	@echo "  $(GREEN)Commandes standards (auto-détection de plateforme):$(NC)"
	@echo "  make start               # Démarrer l'application complète (avec monitoring)"
	@echo "  make build               # Construire les images"
	@echo "  make stop                # Arrêter tous les services"
	@echo "  make logs                # Voir les logs en temps réel"
	@echo "  make monitor             # Afficher toutes les URLs d'accès"
	@echo ""
ifeq ($(IS_MAC),true)
	@echo "  $(YELLOW)Commandes Mac spécifiques (optionnelles):$(NC)"
	@echo "  make mac-start           # Démarrage complet (Mac)"
	@echo "  make mac-clean           # Nettoyage complet (Mac)"
	@echo "  make debug-backfill      # Vérifier backfill historique"
	@echo ""
else ifeq ($(IS_WSL),true)
	@echo "  $(YELLOW)Commandes Windows/WSL2 spécifiques (optionnelles):$(NC)"
	@echo "  make windows-start       # Démarrage complet (WSL2)"
	@echo "  make windows-clean       # Nettoyage complet (WSL2)"
	@echo "  make debug-backfill      # Vérifier backfill historique"
	@echo ""
endif

# Gestion des services
start: ## Démarrer l'application complète (infrastructure + services + app + monitoring)
	@echo "$(GREEN)Démarrage de CryptoViz...$(NC)"
	@./scripts/start.sh

start-infra: ## Démarrer uniquement l'infrastructure (DB, Kafka, Redis, MinIO)
	@echo "$(GREEN)Démarrage de l'infrastructure...$(NC)"
	@docker-compose -f docker-compose.yml $(DESKTOP_OVERRIDE) up -d timescaledb zookeeper kafka redis minio
	@docker-compose -f docker-compose.yml $(DESKTOP_OVERRIDE) up minio-init
	@docker-compose -f docker-compose.yml $(DESKTOP_OVERRIDE) up kafka-init

start-services: ## Démarrer uniquement les microservices
	@echo "$(GREEN)Démarrage des microservices...$(NC)"
	@docker-compose -f docker-compose.yml $(DESKTOP_OVERRIDE) up -d data-collector news-scraper

start-app: ## Démarrer uniquement l'application (backend + frontend)
	@echo "$(GREEN)Démarrage de l'application...$(NC)"
	@docker-compose -f docker-compose.yml $(DESKTOP_OVERRIDE) up -d backend-go indicators-scheduler frontend-vue

start-monitoring: ## Démarrer uniquement la stack de monitoring
	@echo "$(GREEN)Démarrage du monitoring...$(NC)"
	@docker-compose -f docker-compose.yml $(DESKTOP_OVERRIDE) up -d kafka-ui prometheus grafana node-exporter cadvisor postgres-exporter redis-exporter gatus

stop: ## Arrêter tous les services
	@echo "$(YELLOW)Arrêt de CryptoViz...$(NC)"
	@./scripts/stop.sh

stop-force: ## Arrêt forcé de tous les services
	@echo "$(RED)Arrêt forcé de CryptoViz...$(NC)"
	@./scripts/stop.sh --force

restart: stop start ## Redémarrer tous les services

restart-service: ## Redémarrer un service spécifique (usage: make restart-service SERVICE=backend-go)
	@if [ -z "$(SERVICE)" ]; then \
		echo "$(RED)Erreur: Spécifiez le service avec SERVICE=nom_du_service$(NC)"; \
		exit 1; \
	fi
	@echo "$(YELLOW)Redémarrage du service $(SERVICE)...$(NC)"
	@docker-compose restart $(SERVICE)

# Construction et nettoyage
build: ## Construire toutes les images Docker
	@echo "$(GREEN)Construction des images Docker...$(NC)"
	@docker-compose -f docker-compose.yml $(DESKTOP_OVERRIDE) build

build-service: ## Construire une image spécifique (usage: make build-service SERVICE=backend-go)
	@if [ -z "$(SERVICE)" ]; then \
		echo "$(RED)Erreur: Spécifiez le service avec SERVICE=nom_du_service$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)Construction de l'image $(SERVICE)...$(NC)"
	@docker-compose -f docker-compose.yml $(DESKTOP_OVERRIDE) build $(SERVICE)

clean: ## Nettoyer les conteneurs, images et volumes
	@echo "$(RED)Nettoyage complet...$(NC)"
	@./scripts/stop.sh --cleanup
	@echo "$(RED)Suppression explicite du volume de base de données...$(NC)"
	@docker volume rm cryptoviz_timescaledb_data 2>/dev/null || true
	@echo "$(RED)Suppression des images Docker...$(NC)"
	@docker-compose -f docker-compose.yml $(DESKTOP_OVERRIDE) down --rmi all
	@echo "$(RED)Suppression du cache de build Docker...$(NC)"
	@docker builder prune -af

# Monitoring et logs
logs: ## Voir les logs de tous les services en temps réel
	@docker-compose logs -f

logs-service: ## Voir les logs d'un service spécifique (usage: make logs-service SERVICE=backend-go)
	@if [ -z "$(SERVICE)" ]; then \
		echo "$(RED)Erreur: Spécifiez le service avec SERVICE=nom_du_service$(NC)"; \
		exit 1; \
	fi
	@docker-compose logs -f $(SERVICE)

status: ## Afficher l'état de tous les services
	@echo "$(GREEN)État des services CryptoViz:$(NC)"
	@docker-compose ps

health: ## Vérifier la santé des services
	@echo "$(GREEN)Vérification de la santé des services...$(NC)"
	@docker-compose ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}"

# Base de données
db-migrate: ## Exécuter les migrations de base de données (idempotent)
	@echo "$(GREEN)Running database migrations...$(NC)"
	@./scripts/run-migrations.sh

db-connect: ## Se connecter à la base de données TimescaleDB
	@docker-compose exec timescaledb psql -U postgres -d cryptoviz

db-backup: ## Créer une sauvegarde de la base de données
	@echo "$(GREEN)Création d'une sauvegarde de la base de données...$(NC)"
	@mkdir -p backups
	@docker-compose exec timescaledb pg_dump -U postgres cryptoviz > backups/cryptoviz_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "$(GREEN)Sauvegarde créée dans le dossier backups/$(NC)"

db-restore: ## Restaurer la base de données (usage: make db-restore BACKUP=fichier.sql)
	@if [ -z "$(BACKUP)" ]; then \
		echo "$(RED)Erreur: Spécifiez le fichier avec BACKUP=fichier.sql$(NC)"; \
		exit 1; \
	fi
	@echo "$(YELLOW)Restauration de la base de données depuis $(BACKUP)...$(NC)"
	@docker-compose exec -T timescaledb psql -U postgres -d cryptoviz < $(BACKUP)

db-verify-tiering: ## Vérifier si le tiering (cold storage) est configuré
	@echo "$(GREEN)Vérification de la configuration du tiering...$(NC)"
	@docker-compose exec -T timescaledb psql -U postgres -d cryptoviz -c "\
		SELECT EXISTS ( \
			SELECT FROM information_schema.tables \
			WHERE table_name = 'all_candles' \
		) AS all_candles_exists, \
		EXISTS ( \
			SELECT FROM information_schema.schemata \
			WHERE schema_name = 'cold_storage' \
		) AS cold_storage_exists;" || echo "$(RED)Base de données non accessible$(NC)"

db-setup-tiering: ## Configurer le cold storage et les vues unifiées
	@echo "$(GREEN)Configuration du cold storage et du tiering...$(NC)"
	@docker-compose exec -T timescaledb psql -U postgres -d cryptoviz < database/setup-tiering.sql
	@echo "$(GREEN)Configuration du tiering terminée!$(NC)"
	@echo "$(YELLOW)Vérification de l'installation...$(NC)"
	@make db-verify-tiering

db-reset: ## Réinitialiser la base de données complètement (SUPPRIME TOUTES LES DONNÉES)
	@echo "$(RED)ATTENTION: Cette commande va SUPPRIMER TOUTES LES DONNÉES DE LA BASE!$(NC)"
	@read -p "Tapez 'yes' pour confirmer: " confirm; \
	if [ "$$confirm" = "yes" ]; then \
		echo "$(YELLOW)Arrêt de timescaledb...$(NC)"; \
		docker-compose stop timescaledb; \
		echo "$(RED)Suppression du volume de la base de données...$(NC)"; \
		docker volume rm cryptoviz_timescaledb_data 2>/dev/null || true; \
		echo "$(GREEN)Redémarrage de timescaledb (exécutera tous les scripts d'init)...$(NC)"; \
		docker-compose up -d timescaledb; \
		sleep 10; \
		echo "$(GREEN)Réinitialisation terminée! Tous les scripts ont été exécutés.$(NC)"; \
		make db-verify-tiering; \
	else \
		echo "$(YELLOW)Réinitialisation annulée$(NC)"; \
	fi

tiering: ## Exécuter manuellement le tiering (hot → cold storage)
	@echo "$(GREEN)Exécution du tiering manuel (hot → cold storage)...$(NC)"
	@docker-compose exec -T timescaledb psql -U postgres -d cryptoviz \
		-c "SELECT tier_old_candles();" \
		-c "SELECT tier_old_indicators();" \
		-c "SELECT tier_old_news();"
	@echo "$(GREEN)Tiering terminé!$(NC)"
	@echo "$(YELLOW)Vérification des résultats...$(NC)"
	@make tiering-stats

tiering-stats: ## Afficher les statistiques de tiering (hot/cold distribution)
	@echo "$(GREEN)Statistiques de tiering (hot vs cold storage):$(NC)"
	@docker-compose exec -T timescaledb psql -U postgres -d cryptoviz \
		-c "SELECT * FROM get_tiering_stats();"

# Kafka
kafka-topics: ## Lister les topics Kafka
	@docker-compose exec kafka kafka-topics --list --bootstrap-server localhost:9092

kafka-create-topic: ## Créer un topic Kafka (usage: make kafka-create-topic TOPIC=nom_du_topic)
	@if [ -z "$(TOPIC)" ]; then \
		echo "$(RED)Erreur: Spécifiez le topic avec TOPIC=nom_du_topic$(NC)"; \
		exit 1; \
	fi
	@docker-compose exec kafka kafka-topics --create --topic $(TOPIC) --bootstrap-server localhost:9092 --partitions 3 --replication-factor 1

kafka-console-consumer: ## Écouter un topic Kafka (usage: make kafka-console-consumer TOPIC=crypto.raw.trades)
	@if [ -z "$(TOPIC)" ]; then \
		echo "$(RED)Erreur: Spécifiez le topic avec TOPIC=nom_du_topic$(NC)"; \
		exit 1; \
	fi
	@docker-compose exec kafka kafka-console-consumer --topic $(TOPIC) --bootstrap-server localhost:9092 --from-beginning

# MinIO
minio-console: ## Afficher les informations de connexion MinIO
	@echo "$(GREEN)MinIO Console Access:$(NC)"
	@echo "  API: http://localhost:9000"
	@echo "  Console: http://localhost:9001"
	@echo "  Username: minioadmin"
	@echo "  Password: minioadmin"

# Monitoring
grafana-console: ## Afficher les informations de connexion Grafana
	@echo "$(GREEN)Grafana Dashboard Access:$(NC)"
	@echo "  URL: http://localhost:3001"
	@echo "  Username: admin"
	@echo "  Password: admin"
	@echo "  $(YELLOW)Note: Changez le mot de passe à la première connexion$(NC)"

prometheus-ui: ## Afficher les informations de Prometheus
	@echo "$(GREEN)Prometheus Access:$(NC)"
	@echo "  URL: http://localhost:9090"
	@echo "  Targets: http://localhost:9090/targets"
	@echo "  Graph: http://localhost:9090/graph"
	@echo "  Metrics: http://localhost:9090/metrics"

# =============================================================================
# Mac Docker Desktop - Commandes spécifiques
# =============================================================================

mac-start: ## [Mac/Windows] Démarrer avec override Docker Desktop (fix mount issues)
	@echo "$(BLUE)Démarrage de CryptoViz sur $(PLATFORM) Docker Desktop...$(NC)"
	@echo "$(YELLOW)Nettoyage des réseaux Docker obsolètes...$(NC)"
	@docker network prune -f 2>/dev/null || true
	@echo "$(GREEN)Construction des images...$(NC)"
	@docker-compose -f docker-compose.yml $(DESKTOP_OVERRIDE) build
	@echo "$(GREEN)Démarrage de l'infrastructure...$(NC)"
	@docker-compose -f docker-compose.yml $(DESKTOP_OVERRIDE) up -d timescaledb zookeeper kafka redis minio
	@sleep 20
	@docker-compose -f docker-compose.yml $(DESKTOP_OVERRIDE) up minio-init
	@docker-compose -f docker-compose.yml $(DESKTOP_OVERRIDE) up kafka-init
	@echo "$(GREEN)Démarrage des microservices...$(NC)"
	@docker-compose -f docker-compose.yml $(DESKTOP_OVERRIDE) up -d data-collector news-scraper
	@echo "$(GREEN)Démarrage de l'application...$(NC)"
	@docker-compose -f docker-compose.yml $(DESKTOP_OVERRIDE) up -d backend-go indicators-scheduler frontend-vue
	@echo "$(GREEN)✓ CryptoViz démarré sur $(PLATFORM)$(NC)"
	@echo ""
	@echo "$(BLUE)Accès:$(NC)"
	@echo "  Frontend: http://localhost:3000"
	@echo "  Backend API: http://localhost:8080"
ifeq ($(IS_MAC),true)
	@echo "  Monitoring: make mac-start-monitoring"
else ifeq ($(IS_WSL),true)
	@echo "  Monitoring: make windows-start-monitoring"
endif

mac-start-monitoring: ## [Mac/Windows] Démarrer la stack de monitoring (Docker Desktop)
	@echo "$(BLUE)Démarrage du monitoring sur $(PLATFORM)...$(NC)"
	@docker-compose -f docker-compose.yml $(DESKTOP_OVERRIDE) up -d kafka-ui prometheus grafana node-exporter cadvisor postgres-exporter redis-exporter gatus
	@echo "$(GREEN)✓ Monitoring démarré$(NC)"
	@echo ""
	@echo "$(BLUE)Accès monitoring:$(NC)"
	@echo "  Grafana: http://localhost:3001 (admin/admin)"
	@echo "  Prometheus: http://localhost:9090"
	@echo "  Kafka UI: http://localhost:8082"

mac-build: ## [Mac/Windows] Construire les images avec override Docker Desktop
	@echo "$(BLUE)Construction des images pour $(PLATFORM)...$(NC)"
	@docker-compose -f docker-compose.yml $(DESKTOP_OVERRIDE) build

mac-clean: ## [Mac/Windows] Nettoyer Docker + stale networks/volumes (Docker Desktop)
	@echo "$(BLUE)Nettoyage Docker pour $(PLATFORM)...$(NC)"
	@docker-compose -f docker-compose.yml $(DESKTOP_OVERRIDE) down -v --remove-orphans
	@echo "$(YELLOW)Nettoyage des réseaux obsolètes...$(NC)"
	@docker network prune -f
	@echo "$(YELLOW)Nettoyage des volumes orphelins...$(NC)"
	@docker volume prune -f
	@echo "$(YELLOW)Nettoyage des images...$(NC)"
	@docker image prune -f
	@echo "$(GREEN)✓ Nettoyage $(PLATFORM) terminé$(NC)"

# Windows/WSL2 aliases (same as Mac commands, just different names for clarity)
windows-start: mac-start ## [Windows/WSL2] Alias for mac-start (same Docker Desktop fixes)
windows-start-monitoring: mac-start-monitoring ## [Windows/WSL2] Alias for mac-start-monitoring
windows-build: mac-build ## [Windows/WSL2] Alias for mac-build
windows-clean: mac-clean ## [Windows/WSL2] Alias for mac-clean

# =============================================================================
# Debugging - Diagnostic des problèmes multi-machines
# =============================================================================

debug-backfill: ## Vérifier l'état du backfill historique (database-backed)
	@echo "$(GREEN)=== Vérification du Backfill ===$(NC)"
	@echo ""
	@echo "$(YELLOW)1. État de la table backfill_progress:$(NC)"
	@docker exec cryptoviz-timescaledb psql -U postgres -d cryptoviz -c "\
		SELECT symbol, timeframe, status, \
		       backfill_start_ts::date as start_date, \
		       current_position_ts::date as current_date, \
		       total_candles_fetched, total_batches_processed \
		FROM backfill_progress \
		ORDER BY symbol, timeframe \
		LIMIT 20;" 2>/dev/null || echo "$(RED)Erreur: TimescaleDB non accessible ou table non créée$(NC)"
	@echo ""
	@echo "$(YELLOW)2. Statistiques par statut:$(NC)"
	@docker exec cryptoviz-timescaledb psql -U postgres -d cryptoviz -c "\
		SELECT status, COUNT(*) as count, \
		       SUM(total_candles_fetched) as total_candles \
		FROM backfill_progress \
		GROUP BY status \
		ORDER BY status;" 2>/dev/null || echo "$(RED)Erreur: TimescaleDB non accessible$(NC)"
	@echo ""
	@echo "$(YELLOW)3. Variable ENABLE_BACKFILL dans .env:$(NC)"
	@grep -i "ENABLE_BACKFILL" .env || echo "$(RED)Variable non trouvée dans .env$(NC)"
	@echo ""
	@echo "$(YELLOW)4. Logs du data-collector (dernières 30 lignes):$(NC)"
	@docker logs cryptoviz-data-collector 2>&1 | grep -i "backfill" | tail -30 || echo "$(RED)Container non trouvé ou pas de logs backfill$(NC)"
	@echo ""
	@echo "$(YELLOW)5. État du container data-collector:$(NC)"
	@docker ps -a | grep data-collector || echo "$(RED)Container non trouvé$(NC)"

debug-timescale: ## Vérifier les données historiques dans TimescaleDB
	@echo "$(GREEN)=== Vérification des Données Historiques ===$(NC)"
	@echo ""
	@echo "$(YELLOW)1. Comptage par intervalle:$(NC)"
	@docker exec cryptoviz-timescaledb psql -U postgres -d cryptoviz -c "\
		SELECT interval, COUNT(*) as count, \
		       MIN(open_time) as first_candle, \
		       MAX(open_time) as last_candle \
		FROM ohlcv_1m \
		GROUP BY interval \
		ORDER BY interval;" 2>/dev/null || echo "$(RED)Erreur: TimescaleDB non accessible$(NC)"
	@echo ""
	@echo "$(YELLOW)2. Symboles collectés:$(NC)"
	@docker exec cryptoviz-timescaledb psql -U postgres -d cryptoviz -c "\
		SELECT symbol, COUNT(*) as candles \
		FROM ohlcv_1m \
		WHERE interval = '1m' \
		GROUP BY symbol \
		ORDER BY candles DESC \
		LIMIT 10;" 2>/dev/null || echo "$(RED)Erreur: TimescaleDB non accessible$(NC)"
	@echo ""
	@echo "$(YELLOW)3. Données récentes (dernière heure):$(NC)"
	@docker exec cryptoviz-timescaledb psql -U postgres -d cryptoviz -c "\
		SELECT COUNT(*) as recent_candles \
		FROM ohlcv_1m \
		WHERE open_time > NOW() - INTERVAL '1 hour';" 2>/dev/null || echo "$(RED)Erreur: TimescaleDB non accessible$(NC)"

check-docker-resources: ## Nettoyer les ressources Docker obsolètes
	@echo "$(GREEN)=== Nettoyage des Ressources Docker Obsolètes ===$(NC)"
	@echo ""
	@echo "$(YELLOW)Réseaux Docker:$(NC)"
	@docker network ls
	@echo ""
	@echo "$(YELLOW)Nettoyage des réseaux non utilisés...$(NC)"
	@docker network prune -f
	@echo ""
	@echo "$(YELLOW)Volumes Docker:$(NC)"
	@docker volume ls
	@echo ""
	@echo "$(YELLOW)Voulez-vous nettoyer les volumes non utilisés? (peut supprimer des données)$(NC)"
	@read -p "Tapez 'yes' pour confirmer: " confirm; \
	if [ "$$confirm" = "yes" ]; then \
		docker volume prune -f; \
		echo "$(GREEN)✓ Volumes nettoyés$(NC)"; \
	else \
		echo "$(YELLOW)Nettoyage des volumes annulé$(NC)"; \
	fi

# Développement
dev-backend: ## Démarrer le backend en mode développement
	@echo "$(GREEN)Démarrage du backend en mode développement...$(NC)"
	@if [ -f .env.local ]; then \
		echo "$(YELLOW)Utilisation de .env.local pour le développement local$(NC)"; \
		set -a && . ./.env.local && set +a && cd services/backend-go && go run cmd/server/main.go; \
	else \
		echo "$(RED)Fichier .env.local non trouvé. Exécutez 'make setup' d'abord.$(NC)"; \
		exit 1; \
	fi

dev-frontend: ## Démarrer le frontend en mode développement
	@echo "$(GREEN)Démarrage du frontend en mode développement...$(NC)"
	@cd services/frontend-vue && npm run dev

# Tests
test: ## Exécuter tous les tests
	@echo "$(GREEN)Exécution des tests...$(NC)"
	@make test-backend
	@make test-python

test-backend: ## Tester le backend Go
	@echo "$(GREEN)Tests du backend Go...$(NC)"
	@cd services/backend-go && go test ./...

test-python: ## Tester les services Python
	@echo "$(GREEN)Tests des services Python...$(NC)"
	@docker-compose exec data-collector python -m pytest tests/ || true
	@docker-compose exec news-scraper python -m pytest tests/ || true

test-frontend: ## Tester le frontend
	@echo "$(GREEN)Tests du frontend...$(NC)"
	@cd services/frontend-vue && npm test || true

# Linting et formatage
lint: ## Vérifier le code avec les linters
	@echo "$(GREEN)Vérification du code...$(NC)"
	@make lint-backend
	@make lint-python
	@make lint-frontend

lint-backend: ## Linter le code Go
	@echo "$(GREEN)Linting du backend Go...$(NC)"
	@cd services/backend-go && golangci-lint run || true

lint-python: ## Linter le code Python
	@echo "$(GREEN)Linting des services Python...$(NC)"
	@docker-compose exec data-collector flake8 . || true
	@docker-compose exec news-scraper flake8 . || true

lint-frontend: ## Linter le code frontend
	@echo "$(GREEN)Linting du frontend...$(NC)"
	@cd services/frontend-vue && npm run lint || true

format: ## Formater le code
	@echo "$(GREEN)Formatage du code...$(NC)"
	@make format-backend
	@make format-python
	@make format-frontend

format-backend: ## Formater le code Go
	@echo "$(GREEN)Formatage du backend Go...$(NC)"
	@cd services/backend-go && go fmt ./...

format-python: ## Formater le code Python
	@echo "$(GREEN)Formatage des services Python...$(NC)"
	@docker-compose exec data-collector black . || true
	@docker-compose exec news-scraper black . || true

format-frontend: ## Formater le code frontend
	@echo "$(GREEN)Formatage du frontend...$(NC)"
	@cd services/frontend-vue && npm run format || true

# Configuration
setup: ## Configuration initiale du projet
	@echo "$(GREEN)Configuration initiale de CryptoViz...$(NC)"
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "$(YELLOW)Fichier .env créé. Veuillez le configurer avec vos clés API.$(NC)"; \
	fi
	@if [ ! -f .env.local ]; then \
		cp .env.local.example .env.local; \
		echo "$(YELLOW)Fichier .env.local créé pour le développement local.$(NC)"; \
	fi
	@chmod +x scripts/*.sh
	@echo "$(GREEN)Configuration terminée!$(NC)"

update: ## Mettre à jour les dépendances
	@echo "$(GREEN)Mise à jour des dépendances...$(NC)"
	@cd services/backend-go && go mod tidy
	@cd services/frontend-vue && npm update || true

# Monitoring
monitor: ## Ouvrir les interfaces de monitoring
	@echo "$(GREEN)Ouverture des interfaces de monitoring...$(NC)"
	@echo ""
	@echo "$(GREEN)=== Application ===$(NC)"
	@echo "Frontend: http://localhost:3000 (Docker) ou http://localhost:5173 (dev)"
	@echo "Backend API: http://localhost:8080"
	@echo ""
	@echo "$(GREEN)=== Infrastructure ===$(NC)"
	@echo "TimescaleDB: localhost:7432"
	@echo "Kafka UI: http://localhost:8082"
	@echo "Kafka: localhost:9092"
	@echo "Redis: localhost:7379"
	@echo "MinIO API: http://localhost:9000"
	@echo "MinIO Console: http://localhost:9001"
	@echo ""
	@echo "$(GREEN)=== Monitoring Stack ===$(NC)"
	@echo "Grafana: http://localhost:3001 (admin/admin)"
	@echo "Prometheus: http://localhost:9090"
	@echo "Gatus (Health): http://localhost:8084"
	@echo "cAdvisor (Containers): http://localhost:8083"
	@echo "Node Exporter (System): http://localhost:9100/metrics"
	@echo "Postgres Exporter (DB): http://localhost:9187/metrics"
	@echo "Redis Exporter (Cache): http://localhost:9121/metrics"

# Production
prod-build: ## Construire pour la production
	@echo "$(GREEN)Construction pour la production...$(NC)"
	@docker-compose -f docker-compose.yml build --no-cache

prod-deploy: ## Déployer en production
	@echo "$(GREEN)Déploiement en production...$(NC)"
	@docker-compose -f docker-compose.yml up -d

# Utilitaires
shell-service: ## Ouvrir un shell dans un service (usage: make shell-service SERVICE=backend-go)
	@if [ -z "$(SERVICE)" ]; then \
		echo "$(RED)Erreur: Spécifiez le service avec SERVICE=nom_du_service$(NC)"; \
		exit 1; \
	fi
	@docker-compose exec $(SERVICE) /bin/bash || docker-compose exec $(SERVICE) /bin/sh

ps: ## Afficher les processus Docker
	@docker-compose ps

top: ## Afficher l'utilisation des ressources
	@docker-compose top

# API Testing
api-test: ## Tester l'API backend
	@echo "$(GREEN)Test de l'API backend...$(NC)"
	@curl -s http://localhost:8080/health | jq . || echo "Backend non disponible"

api-crypto: ## Tester l'endpoint crypto (usage: make api-crypto SYMBOL=BTC/USDT)
	@if [ -z "$(SYMBOL)" ]; then \
		echo "$(RED)Erreur: Spécifiez le symbole avec SYMBOL=BTC/USDT$(NC)"; \
		exit 1; \
	fi
	@curl -s "http://localhost:8080/api/v1/crypto/latest?symbol=$(SYMBOL)" | jq . || echo "Endpoint non disponible"

# Maintenance
prune: ## Nettoyer Docker (images, conteneurs, volumes orphelins)
	@echo "$(YELLOW)Nettoyage Docker...$(NC)"
	@docker system prune -f
	@docker volume prune -f

reset: ## Reset complet du projet (ATTENTION: supprime toutes les données)
	@echo "$(RED)ATTENTION: Cette commande va supprimer toutes les données!$(NC)"
	@read -p "Êtes-vous sûr? [y/N] " -n 1 -r; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		echo ""; \
		echo "$(RED)Reset en cours...$(NC)"; \
		docker-compose down -v --remove-orphans; \
		docker system prune -af; \
		echo "$(GREEN)Reset terminé$(NC)"; \
	else \
		echo ""; \
		echo "$(YELLOW)Reset annulé$(NC)"; \
	fi

# Informations
info: ## Afficher les informations du projet
	@echo "$(GREEN)=== CryptoViz - Informations du Projet ===$(NC)"
	@echo "Plateforme détectée: $(PLATFORM)"
ifeq ($(IS_DESKTOP),true)
	@echo "$(BLUE)Docker Desktop Override: docker-compose.mac.yml (actif)$(NC)"
endif
	@echo "Version Docker: $(shell docker --version)"
	@echo "Version Docker Compose: $(shell docker-compose --version)"
	@echo "Services configurés: $(SERVICES)"
	@echo "Fichier compose: $(COMPOSE_FILE)"
	@echo ""
ifeq ($(IS_DESKTOP),true)
	@echo "$(BLUE)=== Docker Desktop Auto-détecté ===$(NC)"
	@echo "✓ Toutes les commandes standards (make start, make build, etc.)"
	@echo "  utilisent automatiquement les overrides Docker Desktop"
	@echo ""
	@echo "Commandes spécifiques disponibles:"
ifeq ($(IS_MAC),true)
	@echo "  - make mac-start, make mac-start-monitoring, make mac-clean"
else ifeq ($(IS_WSL),true)
	@echo "  - make windows-start, make windows-start-monitoring, make windows-clean"
endif
	@echo ""
	@echo "Limitations connues:"
	@echo "  - node-exporter: métriques système réduites (pas de rslave)"
	@echo "  - cAdvisor: mode non-privilégié (moins de métriques)"
	@echo "  Documentation: TROUBLESHOOTING-DESKTOP.md"
	@echo ""
endif
	@echo "$(GREEN)URLs d'accès:$(NC)"
	@echo "  Frontend: http://localhost:3000 (Docker) ou http://localhost:5173 (dev)"
	@echo "  Backend API: http://localhost:8080"
	@echo "  Health Check: http://localhost:8080/health"
	@echo ""
	@echo "$(GREEN)Debugging:$(NC)"
	@echo "  make debug-backfill          - Vérifier l'état du backfill"
	@echo "  make debug-timescale         - Vérifier les données historiques"
	@echo "  make check-docker-resources  - Nettoyer les ressources obsolètes"
	@echo ""

# Par défaut, afficher l'aide
.DEFAULT_GOAL := help
