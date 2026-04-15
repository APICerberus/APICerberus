# Installation Guide

## System Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| CPU | 1 core | 2+ cores |
| Memory | 512 MB | 2 GB+ |
| Storage | 1 GB | 10 GB+ |
| OS | Linux, macOS, Windows | Linux (amd64/arm64) |

## Installation Methods

### 1. Pre-built Binary (Recommended)

Download the latest release for your platform:

**Linux (amd64):**
```bash
curl -L https://github.com/APICerberus/APICerberus/releases/latest/download/apicerberus-linux-amd64.tar.gz | tar xz
sudo mv apicerberus /usr/local/bin/
```

**Linux (arm64):**
```bash
curl -L https://github.com/APICerberus/APICerberus/releases/latest/download/apicerberus-linux-arm64.tar.gz | tar xz
sudo mv apicerberus /usr/local/bin/
```

**macOS (amd64):**
```bash
curl -L https://github.com/APICerberus/APICerberus/releases/latest/download/apicerberus-darwin-amd64.tar.gz | tar xz
sudo mv apicerberus /usr/local/bin/
```

**macOS (arm64/M1):**
```bash
curl -L https://github.com/APICerberus/APICerberus/releases/latest/download/apicerberus-darwin-arm64.tar.gz | tar xz
sudo mv apicerberus /usr/local/bin/
```

**Windows:**
Download the `.zip` from releases page and extract.

### 2. Docker

```bash
# Pull latest
docker pull ghcr.io/apicerberus/apicerberus:latest

# Run with volume mount for config
docker run -p 8080:8080 -p 9876:9876 -p 9877:9877 \
  -v $(pwd)/apicerberus.yaml:/etc/apicerberus/apicerberus.yaml \
  ghcr.io/apicerberus/apicerberus:latest

# Run with environment variables
docker run -p 8080:8080 -p 9876:9876 -p 9877:9877 \
  -e APICERBERUS_ADMIN_API_KEY="your-secret-key" \
  ghcr.io/apicerberus/apicerberus:latest
```

### 3. Kubernetes (Helm)

```bash
# Add Helm repo
helm repo add apicerberus https://charts.apicerberus.com
helm repo update

# Install
helm install apicerberus apicerberus/apicerberus \
  --set admin.apiKey="your-secret-key" \
  --namespace apicerberus \
  --create-namespace
```

See [Deployment Guide](production/DEPLOYMENT.md) for full Kubernetes documentation.

### 4. Docker Compose (Development)

```bash
# Clone repository
git clone https://github.com/APICerberus/APICerebrus.git
cd APICerebrus

# Start with Docker Compose
docker-compose up -d

# Or with PostgreSQL
docker-compose -f docker-compose.postgres.yml up -d
```

### 5. Build from Source

```bash
# Clone repository
git clone https://github.com/APICerberus/APICerebrus.git
cd APICerebrus

# Install dependencies
go mod download

# Build
make build

# Binary is at ./bin/apicerberus
./bin/apicerberus version
```

### 6. Homebrew (macOS/Linux)

```bash
brew tap apicerberus/tap
brew install apicerberus
```

## Verify Installation

```bash
# Check version
apicerberus version

# Check built-in help
apicerberus --help

# Validate config
apicerberus config validate apicerberus.yaml
```

## Post-Installation

1. Create configuration file (see [Configuration Guide](configuration.md))
2. Generate admin API key: `openssl rand -base64 32`
3. Start the gateway: `apicerberus start --config apicerberus.yaml`
4. Access dashboard at http://localhost:8080/dashboard

## Platform-Specific Notes

### Linux

- On some distributions, you may need to install `libsqlite3` if using SQLite
- For production, consider running as a systemd service

### macOS

- Apple Silicon (M1/M2) binaries are available
- Rosetta 2 can run amd64 binaries on ARM Macs

### Windows

- Use Windows-specific binary (.zip)
- Run in PowerShell or Command Prompt
- For production, consider running as a Windows Service

## Air-Gapped Installation

For environments without internet access:

1. Download binary and all assets on a connected machine
2. Transfer via USB or internal artifact store
3. Verify checksums before running

## Docker Image Tags

| Tag | Description |
|-----|-------------|
| `latest` | Most recent stable release |
| `v1.0.0` | Specific version |
| `v1.0.0-amd64` | Version + architecture |
| `main` | Latest from main branch (unstable) |

## Next Steps

- [Quick Start Guide](quick-start.md) - Get running in 5 minutes
- [Configuration Guide](configuration.md) - Customize your setup
- [Deployment Guide](production/DEPLOYMENT.md) - Production deployment