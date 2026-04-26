#!/bin/bash
# =============================================================================
# APICerebrus Update Script (Unix/Linux/macOS)
# =============================================================================
# Usage: ./scripts/update.sh [--version v1.0.0]
# =============================================================================

set -euo pipefail

VERSION="${1:-v1.0.0}"

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }

# Find installation
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
    echo "Error: Cannot find APICerebrus installation"
    exit 1
fi

echo ""
echo "=========================================="
echo "  APICerebrus Update to $VERSION"
echo "=========================================="
echo ""

cd "$INSTALL_DIR"

# Load existing env
if [ -f ".env" ]; then
    set -a
    source .env
    set +a
fi

# Pull new image
log_info "Pulling new image..."
docker pull "apicerberus/apicerberus:${VERSION}"
log_success "Image pulled"

# Update compose file version
sed -i "s/image: apicerberus\/apicerberus:.*/image: apicerberus\/apicerberus:${VERSION}/" docker-compose.yml 2>/dev/null || \
    perl -i -pe "s/image: apicerberus\/apicerberus:.*/image: apicerberus\/apicerberus:${VERSION}/" docker-compose.yml

# Restart
log_info "Restarting services..."
docker compose up -d 2>/dev/null || docker-compose up -d

# Wait for health
sleep 5

# Check
if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
    log_success "APICerebrus updated and healthy!"
else
    echo "Update complete. Check logs if service is not healthy."
fi

echo ""
echo "  Run 'docker compose logs -f' to view logs"
echo ""
