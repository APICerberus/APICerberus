#!/bin/bash
# =============================================================================
# APICerebrus Installation Script (Unix/Linux/macOS)
# =============================================================================
# Usage: ./scripts/install.sh [--version v1.0.0] [--dir /opt/apicerberus]
# =============================================================================

set -euo pipefail

# Defaults
VERSION="${VERSION:-v1.0.0}"
INSTALL_DIR="${INSTALL_DIR:-/opt/apicerberus}"
DATA_DIR="${DATA_DIR:-$INSTALL_DIR/data}"
CONFIG_DIR="${CONFIG_DIR:-$INSTALL_DIR/config}"
LOG_DIR="${LOG_DIR:-$INSTALL_DIR/logs}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --version) VERSION="$2"; shift 2 ;;
        --dir) INSTALL_DIR="$2"; shift 2 ;;
        --help|-h) usage; exit 0 ;;
        *) log_error "Unknown option: $1"; exit 1 ;;
    esac
done

echo ""
echo "=========================================="
echo "  APICerebrus Installation"
echo "  Version: $VERSION"
echo "  Directory: $INSTALL_DIR"
echo "=========================================="
echo ""

# Check if running as root (needed for some operations)
is_root() { [ "$(id -u)" -eq 0 ]; }

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    # Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed."
        echo "  Install Docker: https://docs.docker.com/get-docker/"
        exit 1
    fi
    log_success "Docker found: $(docker --version)"

    # Docker Compose
    if docker compose version &> /dev/null; then
        DOCKER_COMPOSE="docker compose"
    elif command -v docker-compose &> /dev/null; then
        DOCKER_COMPOSE="docker-compose"
    else
        log_error "Docker Compose is not installed."
        exit 1
    fi
    log_success "Docker Compose found"

    # Docker daemon running
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running."
        exit 1
    fi
    log_success "Docker daemon is running"
}

# Create directories
create_directories() {
    log_info "Creating directories..."

    if is_root; then
        mkdir -p "$CONFIG_DIR" "$DATA_DIR" "$LOG_DIR"
        chmod 755 "$INSTALL_DIR"
    else
        log_warn "Not running as root. Using sudo for directory creation."
        sudo mkdir -p "$CONFIG_DIR" "$DATA_DIR" "$LOG_DIR"
        sudo chmod 755 "$INSTALL_DIR"
    fi
    log_success "Directories created"
}

# Generate secure secrets
generate_secrets() {
    log_info "Generating secure secrets..."

    if [ -z "${JWT_SECRET:-}" ]; then
        JWT_SECRET=$(openssl rand -base64 32 2>/dev/null || head -c 32 /dev/urandom | base64)
        export JWT_SECRET
    fi

    if [ -z "${ADMIN_API_KEY:-}" ]; then
        ADMIN_API_KEY=$(openssl rand -base64 32 2>/dev/null || head -c 32 /dev/urandom | base64)
        export ADMIN_API_KEY
    fi

    log_success "Secrets generated"
    echo ""
    echo "=========================================="
    echo "  IMPORTANT: Save these secrets!"
    echo "=========================================="
    echo "  JWT_SECRET=$JWT_SECRET"
    echo "  ADMIN_API_KEY=$ADMIN_API_KEY"
    echo "=========================================="
    echo "  Or restart with:"
    echo "  JWT_SECRET=$JWT_SECRET ADMIN_API_KEY=$ADMIN_API_KEY \\"
    echo "  docker compose up -d"
    echo ""
}

# Create docker-compose.yml
create_compose_file() {
    log_info "Creating docker-compose.yml..."

    cat > "$INSTALL_DIR/docker-compose.yml" << 'EOF'
name: apicerberus

services:
  apicerberus:
    image: apicerberus/apicerberus:${VERSION:-v1.0.0}
    container_name: apicerberus-gateway
    restart: unless-stopped
    ports:
      - "8080:8080"   # HTTP Gateway
      - "8443:8443"   # HTTPS Gateway
      - "9876:9876"   # Admin API
      - "9877:9877"   # Portal
      - "50051:50051" # gRPC
    volumes:
      - ./config:/config:ro
      - apicerberus-data:/data
      - apicerberus-logs:/logs
    environment:
      - APICERBERUS_CONFIG=/config/apicerberus.yaml
      - APICERBERUS_DATA_DIR=/data
      - APICERBERUS_LOG_DIR=/logs
      - APICERBERUS_LOG_LEVEL=info
      # Security: Set these environment variables!
      - APICERBERUS_JWT_SECRET=${JWT_SECRET:?JWT_SECRET is required}
      - APICERBERUS_ADMIN_API_KEY=${ADMIN_API_KEY:?ADMIN_API_KEY is required}
    healthcheck:
      test: ["CMD", "/app/apicerberus", "health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 30s
    networks:
      - apicerberus

volumes:
  apicerberus-data:
  apicerberus-logs:

networks:
  apicerberus:
    driver: bridge
EOF
    log_success "docker-compose.yml created"
}

# Create default config
create_config() {
    log_info "Creating default configuration..."

    if [ ! -f "$CONFIG_DIR/apicerberus.yaml" ]; then
        cat > "$CONFIG_DIR/apicerberus.yaml" << 'EOF'
# APICerebrus Configuration
# See: apicerberus.yaml.example in project root for full reference

gateway:
  listen: ":8080"
  http_port: 8080
  https_port: 8443
  read_timeout: 30s
  write_timeout: 30s
  max_body_bytes: 10485760

admin:
  listen: ":9876"
  api_key: "${ADMIN_API_KEY}"  # Set via environment variable

portal:
  listen: ":9877"

store:
  driver: "sqlite"
  path: "/data/apicerberus.db"
  busy_timeout: 5s
  journal_mode: "WAL"
  foreign_keys: true

ratelimit:
  enabled: true
  strategy: "token_bucket"
  global_limit: 10000
  per_user_limit: 1000
  window: 60s

audit:
  enabled: true
  retention_days: 30
  buffer_size: 10000
  batch_size: 100
  flush_interval: 1s

cluster:
  enabled: false
  bind: ":12000"
  node_id: "${HOSTNAME:-node1}"

logging:
  level: "info"
  format: "json"

# TLS/ACME Configuration (optional)
# tls:
#   enabled: true
#   acme:
#     enabled: true
#     email: "admin@example.com"
#     domains:
#       - "api.example.com"
EOF
        log_success "Default config created at $CONFIG_DIR/apicerberus.yaml"
    else
        log_warn "Config already exists at $CONFIG_DIR/apicerberus.yaml"
    fi
}

# Pull image
pull_image() {
    log_info "Pulling APICerebrus image (${VERSION})..."
    docker pull "apicerberus/apicerberus:${VERSION}" || {
        log_error "Failed to pull image. Trying latest..."
        docker pull apicerberus/apicerberus:latest
    }
    log_success "Image pulled"
}

# Start services
start_services() {
    log_info "Starting APICerebrus..."
    cd "$INSTALL_DIR"

    # Export secrets for docker compose
    export JWT_SECRET ADMIN_API_KEY VERSION

    if docker compose up -d 2>/dev/null || docker-compose up -d 2>/dev/null; then
        log_success "APICerebrus started"
    else
        log_error "Failed to start APICerebrus"
        exit 1
    fi

    # Wait for health check
    log_info "Waiting for service to be healthy..."
    sleep 5

    # Check health
    if curl -sf http://localhost:8080/health > /dev/null 2>&1; then
        log_success "APICerebrus is healthy!"
    else
        log_warn "Service may still be starting. Check with: docker compose logs apicerberus"
    fi
}

# Print status
print_status() {
    echo ""
    echo "=========================================="
    echo "  Installation Complete!"
    echo "=========================================="
    echo ""
    echo "  URLs:"
    echo "    Gateway:     http://localhost:8080"
    echo "    Admin API:   http://localhost:9876"
    echo "    Portal:      http://localhost:9877"
    echo "    Dashboard:   http://localhost:9876/dashboard"
    echo ""
    echo "  Commands:"
    echo "    View logs:     cd $INSTALL_DIR && docker compose logs -f"
    echo "    Stop:          cd $INSTALL_DIR && docker compose down"
    echo "    Restart:       cd $INSTALL_DIR && docker compose restart"
    echo "    Update:        cd $INSTALL_DIR && docker compose pull && docker compose up -d"
    echo "    Uninstall:     ./scripts/uninstall.sh"
    echo ""
    echo "  Config:        $CONFIG_DIR/apicerberus.yaml"
    echo "  Data:          $DATA_DIR"
    echo "  Logs:          $LOG_DIR"
    echo ""
}

# Main
main() {
    check_prerequisites
    create_directories
    generate_secrets
    create_compose_file
    create_config
    pull_image
    start_services
    print_status
}

main
