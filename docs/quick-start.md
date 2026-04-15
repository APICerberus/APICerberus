# Quick Start Guide

Get API Cerberus up and running in 5 minutes.

## 1. Download & Install

**Binary (recommended):**
```bash
# Download the latest release for your platform
curl -L https://github.com/APICerberus/APICerberus/releases/latest/download/apicerberus-linux-amd64.tar.gz | tar xz
sudo mv apicerberus /usr/local/bin/

# Verify installation
apicerberus version
```

**Docker:**
```bash
# Pull and run
docker run -p 8080:8080 -p 9876:9876 \
  -v $(pwd)/apicerberus.yaml:/etc/apicerberus/apicerberus.yaml \
  ghcr.io/apicerberus/apicerberus:latest
```

**From Source:**
```bash
git clone https://github.com/APICerberus/APICerebrus.git
cd APICerebrus
make build
./bin/apicerberus version
```

## 2. Create Configuration

Create `apicerberus.yaml`:

```yaml
# Basic configuration
gateway:
  listen: ":8080"

admin:
  api_key: "your-secret-admin-key-here"  # Generate: openssl rand -base64 32

store:
  path: "apicerberus.db"

# Enable audit logging
audit:
  enabled: true
  retention_days: 30
```

Generate a secure admin key:
```bash
openssl rand -base64 32
```

## 3. Start the Gateway

```bash
# Start with config file
apicerberus start --config apicerberus.yaml

# Or use environment variable for admin key
export APICERBERUS_ADMIN_API_KEY="your-secret-admin-key-here"
apicerberus start
```

The gateway listens on:
- **HTTP Gateway**: `:8080`
- **Admin API**: `:9876`
- **User Portal**: `:9877`

## 4. Create Your First Route

```bash
# Using Admin API
curl -X POST http://localhost:9876/admin/api/v1/routes \
  -H "X-Admin-Key: your-secret-admin-key-here" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-api",
    "service": "my-service",
    "paths": ["/api/v1/*"],
    "methods": ["GET", "POST"],
    "upstream": "http://localhost:3000"
  }'

# Create upstream target
curl -X POST http://localhost:9876/admin/api/v1/upstreams/my-service/targets \
  -H "X-Admin-Key: your-secret-admin-key-here" \
  -H "Content-Type: application/json" \
  -d '{"address": "localhost:3000", "weight": 100}'
```

## 5. Create API Key

```bash
# Create user
curl -X POST http://localhost:9876/admin/api/v1/users \
  -H "X-Admin-Key: your-secret-admin-key-here" \
  -H "Content-Type: application/json" \
  -d '{"email": "developer@example.com", "name": "Developer", "company": "Acme Inc"}'

# Create API key
curl -X POST http://localhost:9876/admin/api/v1/users/<user-id>/apikeys \
  -H "X-Admin-Key: your-secret-admin-key-here" \
  -H "Content-Type: application/json" \
  -d '{"name": "Production Key", "mode": "live"}'
```

## 6. Make Your First Request

```bash
# With API key
curl -X GET http://localhost:8080/api/v1/users \
  -H "X-API-Key: ck_live_yourkeyhere"

# Check response headers
curl -v http://localhost:8080/api/v1/users \
  -H "X-API-Key: ck_live_yourkeyhere" 2>&1 | grep -E "HTTP|X-RateLimit|RateLimit"
```

## 7. View Dashboard

Open http://localhost:8080/dashboard (or http://localhost:9877 for the user portal).

## What's Next?

- [Configuration Guide](configuration.md) - All configuration options
- [Plugin Setup](architecture/components.md#plugin-pipeline) - Enable rate limiting, auth, compression
- [Docker Deployment](production/DEPLOYMENT.md) - Production deployment with Docker/K8s
- [Admin API](api/API_NEW_FEATURES.md) - Full API reference

## Common Issues

**Port already in use:**
```bash
# Check what's using port 8080
lsof -i :8080
# Or change the port in config
gateway:
  listen: ":8081"
```

**Permission denied:**
```bash
# On Linux/macOS, you may need to make the binary executable
chmod +x apicerberus
```

**Admin key not working:**
```bash
# Ensure you're using the correct key format
# Key must be at least 32 characters
export APICERBERUS_ADMIN_API_KEY=$(openssl rand -base64 32)
```

For more help, see [Troubleshooting](TROUBLESHOOTING.md).