# APICerebrus Cheatsheet

Common commands and patterns.

## Gateway Operations

```bash
# Start gateway
apicerberus start --config apicerberus.yaml

# Stop gateway
apicerberus stop

# Validate config
apicerberus config validate apicerberus.yaml

# Hot reload config
kill -HUP $(cat apicerberus.pid)

# Check version
apicerberus version
```

## Service & Route Management

```bash
# List services/routes/upstreams
apicerberus service list
apicerberus route list
apicerberus upstream list

# Create service
apicerberus service add --name my-api --upstream my-backend

# Create route
apicerberus route add \
  --name api \
  --service my-api \
  --paths "/api/*"

# Add upstream target
apicerberus upstream target add \
  --upstream my-backend \
  --address localhost:3000
```

## User Management

```bash
# Create user
apicerberus user create \
  --email user@example.com \
  --name "User Name"

# Create API key
apicerberus user apikey create \
  --user <user-id> \
  --name "Production" \
  --mode live

# Suspend/activate user
apicerberus user suspend <user-id>
apicerberus user activate <user-id>

# Add IP whitelist
apicerberus user ip add \
  --user <user-id> \
  --ip "10.0.0.0/8"
```

## Rate Limiting

```bash
# Check rate limit status
curl -H "X-Admin-Key: $KEY" \
  http://localhost:9876/admin/api/v1/ratelimit/status

# Per-user rate limit
apicerberus user update <user-id> --rate-limit-rps 100
```

## Credits

```bash
# Check balance
apicerberus credit balance --user <user-id>

# Top up
apicerberus credit topup \
  --user <user-id> \
  --amount 1000 \
  --reason "Monthly allocation"

# Transaction history
apicerberus credit transactions --user <user-id>
```

## Audit Logs

```bash
# Search logs
apicerberus audit search \
  --user <user-id> \
  --since 2024-01-01

# Tail logs in real-time
apicerberus audit tail --follow

# Export to CSV
apicerberus audit export --format csv --out audit.csv

# Cleanup old logs
apicerberus audit cleanup --older-than-days 30
```

## Analytics

```bash
# Overview
apicerberus analytics overview

# Top routes
apicerberus analytics requests --top 10

# Latency percentiles
apicerberus analytics latency
```

## Clustering

```bash
# Check cluster status
apicerberus cluster status

# Join cluster
apicerberus cluster join --address 192.168.1.10:12000

# Leave cluster
apicerberus cluster leave <node-id>
```

## Docker Operations

```bash
# Start
docker compose up -d

# Stop
docker compose down

# View logs
docker compose logs -f apicerberus

# Restart
docker compose restart

# Update
docker compose pull && docker compose up -d
```

## Health Checks

```bash
# Gateway health
curl http://localhost:8080/health

# Admin API health
curl -H "X-Admin-Key: $KEY" \
  http://localhost:9876/health

# Detailed status
curl -H "X-Admin-Key: $KEY" \
  http://localhost:9876/admin/api/v1/status
```

## Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `APICERBERUS_CONFIG` | Config file path | Yes |
| `APICERBERUS_JWT_SECRET` | JWT signing secret | Yes |
| `APICERBERUS_ADMIN_API_KEY` | Admin API key | Yes |
| `APICERBERUS_DATA_DIR` | Data directory | No |
| `APICERBERUS_LOG_LEVEL` | Log level (debug/info/warn/error) | No |
| `APICERBERUS_SESSION_SECRET` | Session encryption secret | No |

## Ports

| Port | Service | Description |
|------|---------|-------------|
| 8080 | Gateway HTTP | Main proxy |
| 8443 | Gateway HTTPS | TLS proxy |
| 9876 | Admin API | Management API |
| 9877 | Portal | User portal |
| 50051 | gRPC | gRPC services |
| 12000 | Raft | Cluster communication |

## File Locations

```
/opt/apicerberus/          # Default install dir
├── config/                # Configuration
│   └── apicerberus.yaml
├── data/                  # SQLite database
│   └── apicerberus.db
├── logs/                  # Log files
│   └── apicerberus.log
└── certs/                # TLS certificates
```

## Troubleshooting

```bash
# View all logs
docker compose logs -f

# Check config
cat config/apicerberus.yaml

# Verify database
sqlite3 data/apicerberus.db "PRAGMA integrity_check;"

# Reset cluster
rm -rf data/raft/*
docker compose restart
```
