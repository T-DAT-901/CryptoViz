#!/bin/bash

# =============================================================================
# CryptoViz - Script d'arrêt
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

# Arrêt gracieux des services
graceful_stop() {
    log "Arrêt gracieux des services CryptoViz..."

    # Arrêter le frontend en premier
    info "Arrêt du frontend..."
    docker-compose stop frontend-vue 2>/dev/null || true

    # Arrêter le backend
    info "Arrêt du backend..."
    docker-compose stop backend-go 2>/dev/null || true

    # Arrêter les microservices
    info "Arrêt des microservices..."
    docker-compose stop data-collector news-scraper indicators-scheduler 2>/dev/null || true

    # Arrêter les services de monitoring
    info "Arrêt des services de monitoring..."
    docker-compose stop kafka-ui prometheus grafana node-exporter cadvisor postgres-exporter redis-exporter gatus 2>/dev/null || true

    # Arrêter l'infrastructure
    info "Arrêt de l'infrastructure..."
    docker-compose stop kafka zookeeper redis timescaledb minio 2>/dev/null || true

    log "Arrêt gracieux terminé"
}

# Arrêt forcé
force_stop() {
    warn "Arrêt forcé des services..."
    docker-compose down --remove-orphans
    log "Arrêt forcé terminé"
}

# Nettoyage complet
cleanup_all() {
    warn "Nettoyage complet (suppression des volumes)..."
    docker-compose down -v --remove-orphans

    # Supprimer les images orphelines
    info "Suppression des images orphelines..."
    docker image prune -f 2>/dev/null || true

    # Supprimer les réseaux orphelins
    info "Suppression des réseaux orphelins..."
    docker network prune -f 2>/dev/null || true

    log "Nettoyage complet terminé"
}

# Afficher l'état des services
show_status() {
    log "État actuel des services:"
    docker-compose ps 2>/dev/null || warn "Aucun service en cours d'exécution"
}

# Fonction principale
main() {
    case "${1:-}" in
        --force)
            force_stop
            ;;
        --cleanup)
            cleanup_all
            ;;
        --status)
            show_status
            ;;
        *)
            graceful_stop
            ;;
    esac

    show_status

    log "==================================================================="
    log "CryptoViz arrêté"
    log "==================================================================="
    log ""
    log "📋 Options disponibles:"
    log "  - Redémarrer: ./scripts/start.sh"
    log "  - Arrêt forcé: ./scripts/stop.sh --force"
    log "  - Nettoyage complet: ./scripts/stop.sh --cleanup"
    log "  - Voir l'état: ./scripts/stop.sh --status"
    log ""
}

# Gestion des signaux
trap 'error "Script interrompu"; exit 1' INT TERM

# Exécuter la fonction principale
main "$@"
