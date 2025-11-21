# =============================================================================
# CryptoViz Makefile
# =============================================================================

.PHONY: help start stop restart build clean logs status test lint format mac-start mac-start-monitoring mac-build mac-clean mac-reset-backfill debug-backfill debug-timescale check-docker-resources

# Variables
COMPOSE_FILE = docker-compose.yml
SERVICES = timescaledb zookeeper kafka kafka-ui redis minio data-collector news-scraper backend-go frontend-vue prometheus grafana node-exporter cadvisor postgres-exporter redis-exporter gatus

# Mac Detection and Override
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	MAC_OVERRIDE = -f docker-compose.mac.yml
	IS_MAC = true
	PLATFORM = Mac
else
	MAC_OVERRIDE =
	IS_MAC = false
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
ifeq ($(IS_MAC),true)
	@echo "  $(BLUE)Sur Mac, utilisez les commandes 'mac-*' pour une meilleure compatibilité:$(NC)"
	@echo "  make mac-start           # Démarrer l'application (Mac Docker Desktop)"
	@echo "  make mac-start-monitoring # Démarrer la stack de monitoring (Mac)"
	@echo "  make mac-clean           # Nettoyer Docker + stale networks (Mac)"
	@echo "  make debug-backfill      # Vérifier l'état du backfill historique"
	@echo ""
	@echo "  $(YELLOW)Commandes standards (utilisent aussi l'override Mac automatiquement):$(NC)"
endif
	@echo "  make start               # Démarrer l'application (sans monitoring)"
	@echo "  make start-monitoring    # Démarrer la stack de monitoring"
	@echo "  make logs                # Voir les logs en temps réel"
	@echo "  make stop                # Arrêter tous les services"
	@echo "  make monitor             # Afficher toutes les URLs"
	@echo ""

# Gestion des services
start: ## Démarrer l'application (infrastructure + services + app, sans monitoring)
	@echo "$(GREEN)Démarrage de CryptoViz...$(NC)"
	@./scripts/start.sh

start-infra: ## Démarrer uniquement l'infrastructure (DB, Kafka, Redis, MinIO)
	@echo "$(GREEN)Démarrage de l'infrastructure...$(NC)"
	@docker-compose up -d timescaledb zookeeper kafka redis minio
	@docker-compose up minio-init
	@docker-compose up kafka-init

start-services: ## Démarrer uniquement les microservices
	@echo "$(GREEN)Démarrage des microservices...$(NC)"
	@docker-compose up -d data-collector news-scraper

start-app: ## Démarrer uniquement l'application (backend + frontend)
	@echo "$(GREEN)Démarrage de l'application...$(NC)"
	@docker-compose up -d backend-go frontend-vue

start-monitoring: ## Démarrer uniquement la stack de monitoring
	@echo "$(GREEN)Démarrage du monitoring...$(NC)"
	@docker-compose up -d kafka-ui prometheus grafana node-exporter cadvisor postgres-exporter redis-exporter gatus

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
	@docker-compose build --no-cache

build-service: ## Construire une image spécifique (usage: make build-service SERVICE=backend-go)
	@if [ -z "$(SERVICE)" ]; then \
		echo "$(RED)Erreur: Spécifiez le service avec SERVICE=nom_du_service$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)Construction de l'image $(SERVICE)...$(NC)"
	@docker-compose build --no-cache $(SERVICE)

clean: ## Nettoyer les conteneurs, images et volumes
	@echo "$(RED)Nettoyage complet...$(NC)"
	@./scripts/stop.sh --cleanup

clean-images: ## Supprimer toutes les images Docker du projet
	@echo "$(RED)Suppression des images Docker...$(NC)"
	@docker-compose down --rmi all

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

mac-start: ## [Mac] Démarrer avec docker-compose.mac.yml (fix mount issues)
	@echo "$(BLUE)Démarrage de CryptoViz sur Mac Docker Desktop...$(NC)"
	@echo "$(YELLOW)Nettoyage des réseaux Docker obsolètes...$(NC)"
	@docker network prune -f 2>/dev/null || true
	@echo "$(GREEN)Construction des images...$(NC)"
	@docker-compose -f docker-compose.yml $(MAC_OVERRIDE) build --no-cache
	@echo "$(GREEN)Démarrage de l'infrastructure...$(NC)"
	@docker-compose -f docker-compose.yml $(MAC_OVERRIDE) up -d timescaledb zookeeper kafka redis minio
	@sleep 20
	@docker-compose -f docker-compose.yml $(MAC_OVERRIDE) up minio-init
	@docker-compose -f docker-compose.yml $(MAC_OVERRIDE) up kafka-init
	@echo "$(GREEN)Démarrage des microservices...$(NC)"
	@docker-compose -f docker-compose.yml $(MAC_OVERRIDE) up -d data-collector news-scraper
	@echo "$(GREEN)Démarrage de l'application...$(NC)"
	@docker-compose -f docker-compose.yml $(MAC_OVERRIDE) up -d backend-go frontend-vue
	@echo "$(GREEN)✓ CryptoViz démarré sur Mac$(NC)"
	@echo ""
	@echo "$(BLUE)Accès:$(NC)"
	@echo "  Frontend: http://localhost:3000"
	@echo "  Backend API: http://localhost:8080"
	@echo "  Monitoring: make mac-start-monitoring"

mac-start-monitoring: ## [Mac] Démarrer la stack de monitoring sur Mac
	@echo "$(BLUE)Démarrage du monitoring sur Mac...$(NC)"
	@docker-compose -f docker-compose.yml $(MAC_OVERRIDE) up -d kafka-ui prometheus grafana node-exporter cadvisor postgres-exporter redis-exporter gatus
	@echo "$(GREEN)✓ Monitoring démarré$(NC)"
	@echo ""
	@echo "$(BLUE)Accès monitoring:$(NC)"
	@echo "  Grafana: http://localhost:3001 (admin/admin)"
	@echo "  Prometheus: http://localhost:9090"
	@echo "  Kafka UI: http://localhost:8082"

mac-build: ## [Mac] Construire les images avec override Mac
	@echo "$(BLUE)Construction des images pour Mac...$(NC)"
	@docker-compose -f docker-compose.yml $(MAC_OVERRIDE) build --no-cache

mac-clean: ## [Mac] Nettoyer Docker + stale networks/volumes (Mac)
	@echo "$(BLUE)Nettoyage Docker pour Mac...$(NC)"
	@docker-compose -f docker-compose.yml $(MAC_OVERRIDE) down -v --remove-orphans
	@echo "$(YELLOW)Nettoyage des réseaux obsolètes...$(NC)"
	@docker network prune -f
	@echo "$(YELLOW)Nettoyage des volumes orphelins...$(NC)"
	@docker volume prune -f
	@echo "$(YELLOW)Nettoyage des images...$(NC)"
	@docker image prune -f
	@echo "$(GREEN)✓ Nettoyage Mac terminé$(NC)"

mac-reset-backfill: ## [Mac] Supprimer l'état du backfill et redémarrer
	@echo "$(YELLOW)Suppression de l'état du backfill...$(NC)"
	@rm -f services/data-collector/backfill_state.json
	@echo "$(GREEN)✓ État du backfill supprimé$(NC)"
	@echo "$(BLUE)Au prochain démarrage, le backfill sera réexécuté$(NC)"
	@echo "Voulez-vous redémarrer le data-collector maintenant? (Ctrl+C pour annuler)"
	@read -p "Appuyez sur Entrée pour continuer..."
	@docker-compose -f docker-compose.yml $(MAC_OVERRIDE) restart data-collector
	@echo "$(GREEN)✓ Data-collector redémarré$(NC)"

# =============================================================================
# Debugging - Diagnostic des problèmes multi-machines
# =============================================================================

debug-backfill: ## Vérifier l'état du backfill historique
	@echo "$(GREEN)=== Vérification du Backfill ===$(NC)"
	@echo ""
	@echo "$(YELLOW)1. État du fichier backfill_state.json:$(NC)"
	@if [ -f services/data-collector/backfill_state.json ]; then \
		echo "$(GREEN)✓ Fichier trouvé$(NC)"; \
		ls -lh services/data-collector/backfill_state.json; \
		echo ""; \
		echo "$(YELLOW)Contenu (premiers symboles):$(NC)"; \
		head -20 services/data-collector/backfill_state.json 2>/dev/null || cat services/data-collector/backfill_state.json; \
	else \
		echo "$(RED)✗ Fichier NON trouvé - Le backfill n'a jamais été exécuté$(NC)"; \
	fi
	@echo ""
	@echo "$(YELLOW)2. Variable ENABLE_BACKFILL dans .env:$(NC)"
	@grep -i "ENABLE_BACKFILL" .env || echo "$(RED)Variable non trouvée dans .env$(NC)"
	@echo ""
	@echo "$(YELLOW)3. Logs du data-collector (dernières 30 lignes):$(NC)"
	@docker logs cryptoviz-data-collector 2>&1 | grep -i "backfill" | tail -30 || echo "$(RED)Container non trouvé ou pas de logs backfill$(NC)"
	@echo ""
	@echo "$(YELLOW)4. État du container data-collector:$(NC)"
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
ifeq ($(IS_MAC),true)
	@echo "$(BLUE)Override Mac: docker-compose.mac.yml (actif)$(NC)"
endif
	@echo "Version Docker: $(shell docker --version)"
	@echo "Version Docker Compose: $(shell docker-compose --version)"
	@echo "Services configurés: $(SERVICES)"
	@echo "Fichier compose: $(COMPOSE_FILE)"
	@echo ""
ifeq ($(IS_MAC),true)
	@echo "$(BLUE)=== Informations Mac ===$(NC)"
	@echo "Utilisez 'make mac-start' pour un démarrage optimisé Mac"
	@echo "Les commandes standards utilisent automatiquement l'override Mac"
	@echo "Limitations connues:"
	@echo "  - node-exporter: métriques système réduites (pas de rslave)"
	@echo "  - cAdvisor: mode non-privilégié (moins de métriques)"
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
