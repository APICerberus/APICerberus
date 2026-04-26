#Requires -RunAsAdministrator
# =============================================================================
# APICerebrus Update Script (Windows PowerShell)
# =============================================================================
# Usage:
#   .\scripts\update.ps1
#   .\scripts\update.ps1 -Version "v1.0.0"
# =============================================================================

param(
    [string]$Version = "v1.0.0"
)

$ErrorActionPreference = "Stop"

function Write-Info { Write-Host "[INFO] $_" -ForegroundColor Cyan }
function Write-Success { Write-Host "[OK] $_" -ForegroundColor Green }

Write-Host ""
Write-Host "==========================================" -ForegroundColor Magenta
Write-Host "  APICerebrus Update to $Version" -ForegroundColor Magenta
Write-Host "==========================================" -ForegroundColor Magenta
Write-Host ""

# Find installation
$INSTALL_DIR = $null
if (Test-Path ".\docker-compose.yml" -PathType Leaf) {
    $INSTALL_DIR = Get-Location
}
elseif (Test-Path "$env:ProgramFiles\APICerebrus\docker-compose.yml" -PathType Leaf) {
    $INSTALL_DIR = "$env:ProgramFiles\APICerebrus"
}

if (-not $INSTALL_DIR) {
    Write-Err "Cannot find APICerebrus installation."
    exit 1
}

Set-Location $INSTALL_DIR

# Load env
if (Test-Path ".env") {
    Get-Content .env | ForEach-Object {
        if ($_ -match "^(.+)=(.+)$") {
            [System.Environment]::SetEnvironmentVariable($matches[1], $matches[2])
        }
    }
}

# Pull new image
Write-Info "Pulling new image..."
docker pull "apicerberus/apicerberus:$Version"
Write-Success "Image pulled"

# Update compose file
Write-Info "Updating docker-compose.yml..."
$content = Get-Content "docker-compose.yml" -Raw
$content = $content -replace "image: apicerberus/apicerberus:.*", "image: apicerberus/apicerberus:$Version"
$content | Set-Content -Path "docker-compose.yml" -NoNewline -Encoding UTF8

# Restart
Write-Info "Restarting services..."
docker compose up -d 2>$null
if ($LASTEXITCODE -ne 0) {
    docker-compose up -d
}

Write-Info "Waiting for service..."
Start-Sleep -Seconds 5

# Check health
try {
    Invoke-WebRequest -Uri "http://localhost:8080/health" -UseBasicParsing -TimeoutSec 5 -ErrorAction Stop | Out-Null
    Write-Success "APICerebrus updated and healthy!"
}
catch {
    Write-Host "Update complete. Check logs if service is not healthy." -ForegroundColor Yellow
}

Write-Host ""
Write-Host "  Run 'docker compose logs -f' to view logs"
Write-Host ""
