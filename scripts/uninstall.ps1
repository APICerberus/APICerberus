#Requires -RunAsAdministrator
# =============================================================================
# APICerebrus Uninstall Script (Windows PowerShell)
# =============================================================================
# Usage:
#   .\scripts\uninstall.ps1
#   .\scripts\uninstall.ps1 -KeepData
# =============================================================================

param(
    [switch]$KeepData
)

$ErrorActionPreference = "Stop"

function Write-Info { Write-Host "[INFO] $_" -ForegroundColor Cyan }
function Write-Warn { Write-Host "[WARN] $_" -ForegroundColor Yellow }
function Write-Success { Write-Host "[OK] $_" -ForegroundColor Green }

Write-Host ""
Write-Host "==========================================" -ForegroundColor Magenta
Write-Host "  APICerebrus Uninstall" -ForegroundColor Magenta
Write-Host "==========================================" -ForegroundColor Magenta
Write-Host ""

# Find installation directory
$INSTALL_DIR = $null
if (Test-Path ".\docker-compose.yml" -PathType Leaf) {
    $INSTALL_DIR = Get-Location
}
elseif (Test-Path "$env:ProgramFiles\APICerebrus\docker-compose.yml" -PathType Leaf) {
    $INSTALL_DIR = "$env:ProgramFiles\APICerebrus"
}

if (-not $INSTALL_DIR) {
    Write-Err "Cannot find APICerebrus installation."
    Write-Host "Run this script from the installation directory."
    exit 1
}

Write-Info "Installation directory: $INSTALL_DIR"

# Stop services
Write-Info "Stopping APICerebrus..."
Set-Location $INSTALL_DIR
docker compose down 2>$null
if ($LASTEXITCODE -ne 0) {
    docker-compose down 2>$null
}
Write-Info "Services stopped"

# Remove container
Write-Info "Removing container..."
docker rm -f apicerberus-gateway 2>$null

# Remove image
$response = Read-Host "Remove APICerebrus image? [y/N]"
if ($response -eq 'y' -or $response -eq 'Y') {
    Write-Info "Removing image..."
    docker rmi "apicerberus/apicerberus:v1.0.0" 2>$null
    docker rmi "apicerberus/apicerberus:latest" 2>$null
}

# Remove data
if (-not $KeepData) {
    Write-Warn "Removing data volumes..."
    $volumes = docker volume ls -q -f "name=apicerberus" 2>$null
    foreach ($vol in $volumes) {
        docker volume rm $vol 2>$null
    }
    if (Test-Path "$INSTALL_DIR\data") {
        Remove-Item "$INSTALL_DIR\data\*" -Recurse -Force 2>$null
    }
    if (Test-Path "$INSTALL_DIR\logs") {
        Remove-Item "$INSTALL_DIR\logs\*" -Recurse -Force 2>$null
    }
    Write-Info "Data removed"
}
else {
    Write-Info "Keeping data (-KeepData specified)"
}

# Remove files
if (-not $KeepData) {
    Write-Info "Removing installation files..."
    Remove-Item "$INSTALL_DIR\docker-compose.yml" -Force 2>$null
    Remove-Item "$INSTALL_DIR\.env" -Force 2>$null
    Remove-Item "$INSTALL_DIR\config" -Recurse -Force 2>$null
    Remove-Item "$INSTALL_DIR" -Recurse -Force 2>$null
}
else {
    Write-Info "Keeping config and data"
}

Write-Host ""
Write-Host "==========================================" -ForegroundColor Green
Write-Host "  Uninstall Complete" -ForegroundColor Green
Write-Host "==========================================" -ForegroundColor Green
Write-Host ""
