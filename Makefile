# =============================================================================
# CryptoViz Makefile
# =============================================================================

.PHONY: help start stop restart build clean logs status test lint format

# Variables
COMPOSE_FILE = docker-compose.yml
SERVICES = timescaledb kafka redis data-collector news-scraper indicators-calculator backend-go frontend-vue

# Couleurs pour l'affichage
GREEN = \033[0;32m
YELLOW = \033[1;33m
RED = \033[0;31m
NC = \033[0m # No Color

# Aide par défaut
help: ## Afficher cette aide
	@echo "$(GREEN)CryptoViz - Commandes disponibles:$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(GREEN)Exemples d'utilisation:$(NC)"
	@echo "  make start          # Démarrer tous les services"
	@echo "  make logs           # Voir les logs en temps réel"
	@echo "  make stop           # Arrêter tous les services"
	@echo "  make restart        # Redémarrer tous les services"
	@echo ""

# Gestion des services
start: ## Démarrer tous les services
	@echo "$(GREEN)Démarrage de CryptoViz...$(NC)"
	@./scripts/start.sh

start-infra: ## Démarrer uniquement l'infrastructure (DB, Kafka, Redis)
	@echo "$(GREEN)Démarrage de l'infrastructure...$(NC)"
	@docker-compose up -d timescaledb zookeeper kafka redis
	@docker-compose up kafka-init

start-services: ## Démarrer uniquement les microservices
	@echo "$(GREEN)Démarrage des microservices...$(NC)"
	@docker-compose up -d data-collector news-scraper indicators-calculator

start-app: ## Démarrer uniquement l'application (backend + frontend)
	@echo "$(GREEN)Démarrage de l'application...$(NC)"
	@docker-compose up -d backend-go frontend-vue

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

kafka-console-consumer: ## Écouter un topic Kafka (usage: make kafka-console-consumer TOPIC=crypto.raw.1s)
	@if [ -z "$(TOPIC)" ]; then \
		echo "$(RED)Erreur: Spécifiez le topic avec TOPIC=nom_du_topic$(NC)"; \
		exit 1; \
	fi
	@docker-compose exec kafka kafka-console-consumer --topic $(TOPIC) --bootstrap-server localhost:9092 --from-beginning

# Développement
dev-backend: ## Démarrer le backend en mode développement
	@echo "$(GREEN)Démarrage du backend en mode développement...$(NC)"
	@cd services/backend-go && go run main.go

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
	@docker-compose exec indicators-calculator python -m pytest tests/ || true
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
	@docker-compose exec indicators-calculator flake8 . || true
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
	@docker-compose exec indicators-calculator black . || true
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
	@chmod +x scripts/*.sh
	@echo "$(GREEN)Configuration terminée!$(NC)"

update: ## Mettre à jour les dépendances
	@echo "$(GREEN)Mise à jour des dépendances...$(NC)"
	@cd services/backend-go && go mod tidy
	@cd services/frontend-vue && npm update || true

# Monitoring
monitor: ## Ouvrir les interfaces de monitoring
	@echo "$(GREEN)Ouverture des interfaces de monitoring...$(NC)"
	@echo "Frontend: http://localhost:3000"
	@echo "Backend API: http://localhost:8080"
	@echo "TimescaleDB: localhost:7432"
	@echo "Kafka: localhost:9092"
	@echo "Redis: localhost:7379"

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

api-crypto: ## Tester l'endpoint crypto (usage: make api-crypto SYMBOL=BTCUSDT)
	@if [ -z "$(SYMBOL)" ]; then \
		echo "$(RED)Erreur: Spécifiez le symbole avec SYMBOL=BTCUSDT$(NC)"; \
		exit 1; \
	fi
	@curl -s "http://localhost:8080/api/v1/crypto/$(SYMBOL)/latest" | jq . || echo "Endpoint non disponible"

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
	@echo "Version Docker: $(shell docker --version)"
	@echo "Version Docker Compose: $(shell docker-compose --version)"
	@echo "Services configurés: $(SERVICES)"
	@echo "Fichier compose: $(COMPOSE_FILE)"
	@echo ""
	@echo "$(GREEN)URLs d'accès:$(NC)"
	@echo "  Frontend: http://localhost:3000"
	@echo "  Backend API: http://localhost:8080"
	@echo "  Health Check: http://localhost:8080/health"
	@echo ""

# Par défaut, afficher l'aide
.DEFAULT_GOAL := help
