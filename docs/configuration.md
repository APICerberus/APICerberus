# Configuration Guide

Complete configuration reference for API Cerberus.

## Configuration File Structure

```yaml
# api Cerberus Configuration
# All options have sensible defaults

gateway:
  listen: ":8080"              # Gateway HTTP listen address
  https_listen: ":8443"        # Gateway HTTPS listen address (TLS)
  read_timeout: 30s            # Read timeout
  write_timeout: 30s           # Write timeout
  idle_timeout: 60s            # Idle connection timeout
  max_body_bytes: 10485760     # Max request body size (10MB)
  trusted_proxies: []          # CIDR ranges for X-Forwarded-For trust
  enable_http2: true           # Enable HTTP/2 support

admin:
  api_key: ""                  # Admin API key (required, min 32 chars)
  listen: ":9876"             # Admin API listen address

portal:
  enabled: true
  listen: ":9877"             # User portal listen address

store:
  driver: "sqlite"             # "sqlite" or "postgres"
  path: "apicerberus.db"      # SQLite database path
  busy_timeout: 5s             # SQLite busy timeout
  journal_mode: "WAL"          # SQLite journal mode
  max_open_conns: 25           # Max open connections
  foreign_keys: true           # Enable foreign key constraints
  synchronous: "NORMAL"        # SQLite synchronous mode
  # PostgreSQL options (when driver: "postgres")
  postgres:
    host: "localhost"
    port: 5432
    user: "apicerberus"
    password: ""
    database: "apicerberus"
    ssl_mode: "disable"
    max_conns: 25

ratelimit:
  enabled: true
  strategy: "token_bucket"     # "token_bucket", "sliding_window", "leaky_bucket"
  redis:
    enabled: false             # Enable Redis-backed distributed rate limiting
    host: "localhost:6379"
    password: ""
    db: 0

billing:
  enabled: true
  test_key_bypass: true        # Allow ck_test_* keys to bypass billing

audit:
  enabled: true
  retention_days: 30            # How long to keep audit logs
  buffer_size: 10000            # Async buffer size
  batch_size: 100               # Batch write size
  flush_interval: 1s            # Flush interval
  compress: true                # GZIP compress archived logs
  kafka:
    enabled: false
    brokers: []
    topic: "audit"
    async: true

cluster:
  enabled: false
  bind: ":12000"               # Raft bind address
  join: []                     # Addresses to join on startup
 mtls:
    auto_generate: true         # Auto-generate CA and node certs
    auto_cert_dir: "./certs"    # Directory for auto-generated certs

tracing:
  enabled: false
  service_name: "apicerberus"
  exporter: "otlp"              # "jaeger", "zipkin", "otlp", "stdout"
  endpoint: "http://localhost:4317"
  sampling_rate: 0.1

plugins:
  enabled:
    - correlation_id
    - ip_restriction
    - bot_detection
    - api_key
    - jwt
    - rate_limit
    - request_validation
    - cors
    - circuit_breaker
    - retry
    - timeout
    - compression
    - response_transform