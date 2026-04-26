#!/bin/bash
# =============================================================================
# APICerebrus Uninstall Script (Unix/Linux/macOS)
# =============================================================================
# Usage: ./scripts/uninstall.sh [--keep-data]
# =============================================================================

set -euo pipefail

KEEP_DATA=false

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --keep-data) KEEP_DATA=true; shift ;;
        --help|-h) echo "Usage: $0 [--keep-data]"; exit 0 ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

# Find installation directory
find_installdir() {
    if [ -f "$(pwd)/docker-compose.yml" ] && grep -q "apicerberus" "$(pwd)/docker-compose.yml" 2>/dev/null; then
        echo "$(pwd)"
    elif [ -f "/opt/apicerberus/docker-compose.yml" ]; then
        echo "/opt/apicerberus"
    else
        echo ""
    fi
}

INSTALL_DIR=$(find_installdir)

if [ -z "$INSTALL_DIR" ]; then
    echo "Error: Cannot find APICerebrus installation."
    echo "Run this script from the installation directory, or check /opt/apicerberus"
    exit 1
fi

echo ""
echo "=========================================="
echo "  APICerebrus Uninstall"
echo "  Directory: $INSTALL_DIR"
echo "=========================================="
echo ""

# Stop services
log_info "Stopping APICerebrus..."
cd "$INSTALL_DIR"
docker compose down 2>/dev/null || docker-compose down 2>/dev/null
log_info "Services stopped"

# Remove container
log_info "Removing container..."
docker rm -f apicerberus-gateway 2>/dev/null || true

# Remove image (optional)
read -p "Remove APICerebrus image? [y/N] " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    log_info "Removing image..."
    docker rmi "apicerberus/apicerberus:v1.0.0" 2>/dev/null || true
    docker rmi "apicerberus/apicerberus:latest" 2>/dev/null || true
fi

# Remove data
if [ "$KEEP_DATA" = false ]; then
    log_warn "Removing data volumes..."
    docker volume rm "$(docker volume ls -q -f name=apicerberus 2>/dev/null | head -1)" 2>/dev/null || true
    rm -rf "$INSTALL_DIR/data"/* 2>/dev/null || true
    rm -rf "$INSTALL_DIR/logs"/* 2>/dev/null || true
    log_info "Data removed"
else
    log_info "Keeping data (--keep-data specified)"
fi

# Remove files
if [ "$KEEP_DATA" = false ]; then
    log_info "Removing installation files..."
    rm -f "$INSTALL_DIR/docker-compose.yml"
    rm -f "$INSTALL_DIR/.env"
    rm -rf "$INSTALL_DIR/config"
    rm -rf "$INSTALL_DIR"
else
    log_info "Keeping config and data"
fi

echo ""
echo "=========================================="
echo "  Uninstall Complete"
echo "=========================================="
echo ""
