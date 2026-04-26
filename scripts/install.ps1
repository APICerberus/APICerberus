#Requires -RunAsAdministrator
# =============================================================================
# APICerebrus Installation Script (Windows PowerShell)
# =============================================================================
# Usage:
#   .\scripts\install.ps1
#   .\scripts\install.ps1 -Version "v1.0.0" -InstallDir "D:\APICerebrus"
# =============================================================================

param(
    [string]$Version = "v1.0.0",
    [string]$InstallDir = "$env:ProgramFiles\APICerebrus",
    [switch]$SkipDockerCheck
)

$ErrorActionPreference = "Stop"

# Colors
function Write-Info { Write-Host "[INFO] $_" -ForegroundColor Cyan }
function Write-Success { Write-Host "[OK] $_" -ForegroundColor Green }
function Write-Warn { Write-Host "[WARN] $_" -ForegroundColor Yellow }
function Write-Err { Write-Host "[ERROR] $_" -ForegroundColor Red }

Write-Host ""
Write-Host "==========================================" -ForegroundColor Magenta
Write-Host "  APICerebrus Installation" -ForegroundColor Magenta
Write-Host "  Version: $Version" -ForegroundColor Magenta
Write-Host "  Directory: $InstallDir" -ForegroundColor Magenta
Write-Host "==========================================" -ForegroundColor Magenta
Write-Host ""

# Check prerequisites
function Test-Prerequisites {
    Write-Info "Checking prerequisites..."

    if (-not $SkipDockerCheck) {
        # Check Docker
        try {
            $dockerVersion = docker --version 2>$null
            if (-not $dockerVersion) { throw "Docker not found" }
            Write-Success "Docker found: $dockerVersion"
        }
        catch {
            Write-Err "Docker is not installed."
            Write-Host "  Install Docker Desktop: https://docker.com/get-started"
            exit 1
        }

        # Check Docker Compose
        try {
            docker compose version 2>$null | Out-Null
            $script:DOCKER_COMPOSE = "docker compose"
            Write-Success "Docker Compose v2 found"
        }
        catch {
            try {
                docker-compose --version 2>$null | Out-Null
                $script:DOCKER_COMPOSE = "docker-compose"
                Write-Success "Docker Compose found"
            }
            catch {
                Write-Err "Docker Compose is not installed."
                exit 1
            }
        }

        # Check Docker daemon
        try {
            docker info 2>$null | Out-Null
            Write-Success "Docker daemon is running"
        }
        catch {
            Write-Err "Docker daemon is not running. Please start Docker Desktop."
            exit 1
        }
    }
}

# Create directories
function New-InstallationDirectories {
    Write-Info "Creating directories..."

    $configDir = "$InstallDir\config"
    $dataDir = "$InstallDir\data"
    $logDir = "$InstallDir\logs"

    New-Item -ItemType Directory -Force -Path $configDir | Out-Null
    New-Item -ItemType Directory -Force -Path $dataDir | Out-Null
    New-Item -ItemType Directory -Force -Path $logDir | Out-Null

    Write-Success "Directories created at $InstallDir"
    return $configDir, $dataDir, $logDir
}

# Generate secrets
function New-Secrets {
    Write-Info "Generating secure secrets..."

    if (-not $env:JWT_SECRET) {
        $bytes = New-Object byte[] 32
        [Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
        $script:JWT_SECRET = [Convert]::ToBase64String($bytes)
    }

    if (-not $env:ADMIN_API_KEY) {
        $bytes = New-Object byte[] 32
        [Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
        $script:ADMIN_API_KEY = [Convert]::ToBase64String($bytes)
    }

    # Persist to file for next run
    $envFile = "$InstallDir\.env"
    @"
JWT_SECRET=$JWT_SECRET
ADMIN_API_KEY=$ADMIN_API_KEY
VERSION=$Version
"@ | Set-Content -Path $envFile -NoNewline

    Write-Success "Secrets generated"
    Write-Host ""
    Write-Host "==========================================" -ForegroundColor Yellow
    Write-Host "  IMPORTANT: Save these secrets!" -ForegroundColor Yellow
    Write-Host "==========================================" -ForegroundColor Yellow
    Write-Host "  JWT_SECRET=$JWT_SECRET"
    Write-Host "  ADMIN_API_KEY=$ADMIN_API_KEY"
    Write-Host "==========================================" -ForegroundColor Yellow
    Write-Host ""

    return $envFile
}

# Create docker-compose.yml
function New-DockerCompose {
    param([string]$ConfigDir)

    Write-Info "Creating docker-compose.yml..."

    $composeContent = @"
name: apicerberus

services:
  apicerberus:
    image: apicerberus/apicerberus:${Version}
    container_name: apicerberus-gateway
    restart: unless-stopped
    ports:
      - "8080:8080"
      - "8443:8443"
      - "9876:9876"
      - "9877:9877"
      - "50051:50051"
    volumes:
      - ./config:/config:ro
      - apicerberus-data:C:/data
      - apicerberus-logs:C:/logs
    environment:
      - APICERBERUS_CONFIG=/config/apicerberus.yaml
      - APICERBERUS_DATA_DIR=C:/data
      - APICERBERUS_LOG_DIR=C:/logs
      - APICERBERUS_LOG_LEVEL=info
      - APICERBERUS_JWT_SECRET=${env:JWT_SECRET}
      - APICERBERUS_ADMIN_API_KEY=${env:ADMIN_API_KEY}
    healthcheck:
      test: ["CMD", "C:/apicerberus.exe", "health"]
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
"@

    $composeContent | Set-Content -Path "$InstallDir\docker-compose.yml" -NoNewline -Encoding UTF8
    Write-Success "docker-compose.yml created"
}

# Create default config
function New-DefaultConfig {
    param([string]$ConfigDir)

    Write-Info "Creating default configuration..."

    $configPath = "$ConfigDir\apicerberus.yaml"
    if (-not (Test-Path $configPath)) {
        $configContent = @"
# APICerebrus Configuration

gateway:
  listen: ":8080"
  http_port: 8080
  https_port: 8443
  read_timeout: 30s
  write_timeout: 30s
  max_body_bytes: 10485760

admin:
  listen: ":9876"

portal:
  listen: ":9877"

store:
  driver: "sqlite"
  path: "C:/data/apicerberus.db"
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

cluster:
  enabled: false
  bind: ":12000"
  node_id: "node1"

logging:
  level: "info"
  format: "json"
"@
        $configContent | Set-Content -Path $configPath -NoNewline -Encoding UTF8
        Write-Success "Default config created at $configPath"
    }
    else {
        Write-Warn "Config already exists at $configPath"
    }
}

# Pull image
function Start-ImagePull {
    Write-Info "Pulling APICerebrus image..."
    docker pull "apicerberus/apicerberus:$Version" 2>$null
    if ($LASTEXITCODE -ne 0) {
        Write-Warn "Failed to pull $Version, trying latest..."
        docker pull apicerberus/apicerberus:latest
    }
    Write-Success "Image pulled"
}

# Start services
function Start-APICerebrus {
    Write-Info "Starting APICerebrus..."
    Set-Location $InstallDir

    & docker compose up -d 2>$null
    if ($LASTEXITCODE -ne 0) {
        & docker-compose up -d 2>$null
    }

    if ($LASTEXITCODE -eq 0) {
        Write-Success "APICerebrus started"
    }
    else {
        Write-Err "Failed to start APICerebrus"
        exit 1
    }

    Write-Info "Waiting for service to be healthy..."
    Start-Sleep -Seconds 5

    try {
        Invoke-WebRequest -Uri "http://localhost:8080/health" -UseBasicParsing -TimeoutSec 5 -ErrorAction Stop | Out-Null
        Write-Success "APICerebrus is healthy!"
    }
    catch {
        Write-Warn "Service may still be starting. Check with: docker compose logs apicerberus"
    }
}

# Print status
function Show-InstallComplete {
    Write-Host ""
    Write-Host "==========================================" -ForegroundColor Green
    Write-Host "  Installation Complete!" -ForegroundColor Green
    Write-Host "==========================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "  URLs:"
    Write-Host "    Gateway:     http://localhost:8080"
    Write-Host "    Admin API:   http://localhost:9876"
    Write-Host "    Portal:      http://localhost:9877"
    Write-Host "    Dashboard:   http://localhost:9876/dashboard"
    Write-Host ""
    Write-Host "  Commands:"
    Write-Host "    View logs:     cd $InstallDir; docker compose logs -f"
    Write-Host "    Stop:          cd $InstallDir; docker compose down"
    Write-Host "    Restart:       cd $InstallDir; docker compose restart"
    Write-Host "    Update:        cd $InstallDir; docker compose pull; docker compose up -d"
    Write-Host "    Uninstall:     .\scripts\uninstall.ps1"
    Write-Host ""
    Write-Host "  Config:        $InstallDir\config\apicerberus.yaml"
    Write-Host "  Data:          $InstallDir\data"
    Write-Host "  Logs:          $InstallDir\logs"
    Write-Host ""
}

# Main
function Main {
    Test-Prerequisites
    $configDir, $dataDir, $logDir = New-InstallationDirectories
    $envFile = New-Secrets
    New-DockerCompose -ConfigDir $configDir
    New-DefaultConfig -ConfigDir $configDir
    Start-ImagePull
    Start-APICerebrus
    Show-InstallComplete
}

Main
