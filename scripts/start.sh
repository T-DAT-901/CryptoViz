#!/bin/bash

# =============================================================================
# CryptoViz - Script de démarrage
# =============================================================================

set -e

# Couleurs pour les logs
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Fonction pour afficher les logs avec couleur
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
}

info() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] INFO: $1${NC}"
}

# Platform Detection (same logic as Makefile)
detect_platform() {
    UNAME_S=$(uname -s)
    UNAME_R=$(uname -r)

    if [[ "$UNAME_S" == "Darwin" ]]; then
        # Mac Docker Desktop
        OVERRIDE_FILE="-f docker-compose.mac.yml"
        PLATFORM="Mac"
    elif [[ "$UNAME_R" == *"microsoft"* ]] || [[ "$UNAME_R" == *"Microsoft"* ]]; then
        # Windows Docker Desktop via WSL2
        OVERRIDE_FILE="-f docker-compose.mac.yml"
        PLATFORM="Windows (WSL2)"
    else
        # Native Linux
        OVERRIDE_FILE=""
        PLATFORM="Linux"
    fi

    if [ -n "$OVERRIDE_FILE" ]; then
        info "Platform détectée: $PLATFORM - Utilisation de l'override Docker Desktop"
    else
        info "Platform détectée: $PLATFORM - Mode natif Linux"
    fi
}

# Vérifier que Docker est installé et en cours d'exécution
check_docker() {
    if ! command -v docker &> /dev/null; then
        error "Docker n'est pas installé. Veuillez installer Docker avant de continuer."
        exit 1
    fi

    if ! docker info &> /dev/null; then
        error "Docker n'est pas en cours d'exécution. Veuillez démarrer Docker."
        exit 1
    fi

    if ! command -v docker-compose &> /dev/null; then
        error "Docker Compose n'est pas installé. Veuillez installer Docker Compose."
        exit 1
    fi

    log "Docker et Docker Compose sont disponibles"
}

# Vérifier la configuration
check_config() {
    if [ ! -f ".env" ]; then
        warn "Fichier .env non trouvé. Copie de .env.example..."
        cp .env.example .env
        warn "Veuillez éditer le fichier .env avec vos clés API avant de continuer."
        warn "Notamment BINANCE_API_KEY et BINANCE_SECRET_KEY"
        read -p "Appuyez sur Entrée pour continuer une fois la configuration terminée..."
    fi

    log "Configuration vérifiée"
}

# Nettoyer les anciens conteneurs et volumes si nécessaire
cleanup() {
    info "Nettoyage des anciens conteneurs..."
    docker-compose -f docker-compose.yml $OVERRIDE_FILE down --remove-orphans 2>/dev/null || true

    # Optionnel: supprimer les volumes (décommentez si nécessaire)
    # docker-compose -f docker-compose.yml $OVERRIDE_FILE down -v 2>/dev/null || true

    log "Nettoyage terminé"
}

# Construire les images
build_images() {
    log "Construction des images Docker..."
    docker-compose -f docker-compose.yml $OVERRIDE_FILE build
    log "Images construites avec succès"
}

# Démarrer les services
start_services() {
    log "Démarrage des services..."

    # Démarrer d'abord l'infrastructure
    info "Démarrage de l'infrastructure (TimescaleDB, Kafka, Redis, MinIO)..."
    docker-compose -f docker-compose.yml $OVERRIDE_FILE up -d timescaledb zookeeper kafka redis minio

    # Attendre que les services soient prêts
    info "Attente de la disponibilité des services..."
    sleep 30

    # Note: minio-init and kafka-init run automatically via depends_on when their
    # dependent services start. With restart: "no", they only run once and skip on subsequent starts.

    # Démarrer les microservices
    info "Démarrage des microservices..."
    docker-compose -f docker-compose.yml $OVERRIDE_FILE up -d data-collector news-scraper

    # Démarrer le backend
    info "Démarrage du backend Go..."
    docker-compose -f docker-compose.yml $OVERRIDE_FILE up -d backend-go

    # Démarrer le scheduler d'indicateurs
    info "Démarrage du scheduler d'indicateurs..."
    docker-compose -f docker-compose.yml $OVERRIDE_FILE up -d indicators-scheduler

    # Démarrer le frontend
    info "Démarrage du frontend Vue.js..."
    docker-compose -f docker-compose.yml $OVERRIDE_FILE up -d frontend-vue

    log "Tous les services sont démarrés"
}

# Démarrer le monitoring
start_monitoring() {
    log "Démarrage de la stack de monitoring..."

    info "Démarrage des services de monitoring (Grafana, Prometheus, Kafka UI, etc.)..."
    docker-compose -f docker-compose.yml $OVERRIDE_FILE up -d kafka-ui prometheus grafana node-exporter cadvisor postgres-exporter redis-exporter kafka-exporter gatus

    log "Stack de monitoring démarrée"
}

# Vérifier l'état des services
check_services() {
    log "Vérification de l'état des services..."

    # Attendre un peu pour que les services se stabilisent
    sleep 10

    # Vérifier les services
    docker-compose -f docker-compose.yml $OVERRIDE_FILE ps

    # Vérifier les logs pour des erreurs
    info "Vérification des logs pour des erreurs critiques..."

    # Vérifier TimescaleDB
    if docker-compose -f docker-compose.yml $OVERRIDE_FILE logs timescaledb | grep -q "database system is ready to accept connections"; then
        log "✓ TimescaleDB est prêt"
    else
        warn "⚠ TimescaleDB pourrait avoir des problèmes"
    fi

    # Vérifier Kafka
    if docker-compose -f docker-compose.yml $OVERRIDE_FILE logs kafka | grep -q "started (kafka.server.KafkaServer)"; then
        log "✓ Kafka est prêt"
    else
        warn "⚠ Kafka pourrait avoir des problèmes"
    fi

    # Vérifier le backend
    if docker-compose -f docker-compose.yml $OVERRIDE_FILE logs backend-go | grep -q "Server started"; then
        log "✓ Backend Go est prêt"
    else
        warn "⚠ Backend Go pourrait avoir des problèmes"
    fi
}

# Afficher les informations de connexion
show_info() {
    log "==================================================================="
    log "CryptoViz est maintenant en cours d'exécution!"
    log "==================================================================="
    log ""
    log "🌐 Frontend (Interface utilisateur): http://localhost:3000"
    log "🔧 API Backend: http://localhost:8080"
    log "📊 Base de données TimescaleDB: localhost:7432"
    log "📨 Kafka: localhost:9092"
    log "🗄️  Redis: localhost:7379"
    log "🗄️  MinIO API: http://localhost:9000"
    log "🗄️  MinIO Console: http://localhost:9001"
    log ""
    log "🔍 Monitoring:"
    log "  - Kafka UI: http://localhost:8082"
    log "  - Grafana: http://localhost:3001 (admin/admin)"
    log "  - Prometheus: http://localhost:9090"
    log "  - Gatus (Health): http://localhost:8084"
    log "  - cAdvisor (Containers): http://localhost:8083"
    log ""
    log "📋 Commandes utiles:"
    log "  - Voir les logs: docker-compose logs -f [service]"
    log "  - Arrêter: docker-compose down"
    log "  - Redémarrer un service: docker-compose restart [service]"
    log "  - Voir l'état: docker-compose ps"
    log "  - Voir toutes les URLs: make monitor"
    log ""
    log "==================================================================="
}

# Fonction principale
main() {
    log "Démarrage de CryptoViz..."

    # Détection de la plateforme
    detect_platform

    # Vérifications préliminaires
    check_docker
    check_config

    # Options de ligne de commande
    case "${1:-}" in
        --clean)
            log "Mode nettoyage activé"
            cleanup
            ;;
        --build)
            log "Mode reconstruction activé"
            build_images
            ;;
        --no-build)
            log "Mode sans reconstruction"
            ;;
        *)
            # Par défaut, utiliser les images existantes
            # Utilisez --build pour forcer la reconstruction
            log "Utilisation des images existantes (utilisez --build pour reconstruire)"
            ;;
    esac

    # Démarrer les services
    start_services

    # Démarrer le monitoring
    start_monitoring

    # Vérifier l'état
    check_services

    # Afficher les informations
    show_info

    log "Démarrage terminé avec succès!"
}

# Gestion des signaux pour un arrêt propre
trap 'error "Script interrompu"; exit 1' INT TERM

# Exécuter la fonction principale
main "$@"
