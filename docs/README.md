# API Cerberus Documentation

Welcome to the API Cerberus documentation. This guide covers all aspects of the API gateway including installation, configuration, architecture, and production deployment.

## Getting Started

- [Quick Start Guide](quick-start.md) - Get up and running in 5 minutes
- [Installation](installation.md) - Installation methods for all platforms
- [Configuration Guide](configuration.md) - Complete configuration reference

## Core Concepts

- [Architecture Overview](architecture/ARCHITECTURE.md) - System design and component overview
- [Plugin Pipeline](architecture/components.md) - PRE_AUTH → AUTH → PRE_PROXY → PROXY → POST_PROXY phases
- [Request Flow](architecture/data-flow.md) - How requests flow through the gateway
- [Security Model](architecture/security.md) - Authentication, authorization, and audit logging

## Deployment

- [Deployment Guide](production/DEPLOYMENT.md) - Docker, Kubernetes, and bare metal
- [Docker Swarm](production/DOCKER_SWARM_ARCHITECTURE.md) - Multi-node cluster setup
- [Scaling](production/SCALING.md) - Horizontal and vertical scaling strategies
- [Monitoring](production/MONITORING.md) - Metrics, alerting, and observability
- [Security Hardening](production/SECURITY_HARDENING.md) - Production security best practices
- [Troubleshooting](production/TROUBLESHOOTING.md) - Common issues and solutions

## Configuration Reference

- [Rate Limiting](RATE_LIMITING.md) - Token bucket, sliding window, and leaky bucket algorithms
- [Redis Rate Limiting](REDIS_RATE_LIMITING.md) - Distributed rate limiting with Redis
- [Audit Logging](architecture/components.md#audit-logging) - Request/response logging and PII masking
- [Kafka Streaming](KAFKA_AUDIT_STREAMING.md) - SIEM integration via Kafka
- [OpenTelemetry Tracing](TRACING.md) - Distributed tracing setup
- [ACME/Let's Encrypt](ACME_RAFT_SYNC.md) - Automatic TLS certificate management
- [WASM Plugins](WASM_PLUGINS.md) - Custom plugin development

## API Reference

- [Admin API](api/API_NEW_FEATURES.md) - Complete Admin API documentation
- [OpenAPI Spec](api/openapi.yaml) - Machine-readable API specification

## Migration Guides

- [From Kong](migrations/KONG.md)
- [From Tyk](migrations/TYK.md)
- [From KrakenD](migrations/KRAKEND.md)

## Architecture Decisions

- [Architecture Decisions](ARCHITECTURE_DECISIONS.md) - Design rationale and trade-offs
- [Security Audit](SECURITY_AUDIT.md) - Third-party security assessment

## Contributing

- [Contributing Guide](CONTRIBUTING.md) - Development setup and contribution guidelines

## Help & Support

- [Troubleshooting](TROUBLESHOOTING.md) - Common issues and solutions
- [Runbook](production/RUNBOOK.md) - Operational procedures for production

## Key Features

| Feature | Description |
|---------|-------------|
| **Multi-Database** | SQLite (default) or PostgreSQL support |
| **Plugin Pipeline** | 5-phase extensible plugin system with 20+ built-in plugins |
| **Rate Limiting** | Local and Redis-backed (token bucket, sliding window, leaky bucket) |
| **Billing/Credits** | Atomic credit transactions, per-route costs, test key bypass |
| **Audit Logging** | GZIP compressed, field masking, Kafka export, FTS search |
| **GraphQL Federation** | Apollo-compatible schema composition and query planning |
| **Raft Clustering** | Multi-region support with automatic mTLS certificate sync |
| **OIDC Provider** | Built-in OIDC Authorization Server (discovery, JWKS, auth code flow) |
| **WASM Plugins** | Sandboxed custom logic in any pipeline phase |
| **ACME TLS** | Automatic Let's Encrypt certificate provisioning and renewal |

## Quick Configuration Example

```yaml
gateway:
  listen: ":8080"
  read_timeout: 30s
  write_timeout: 30s

store:
  driver: "sqlite"  # or "postgres"
  path: "apicerberus.db"

ratelimit:
  enabled: true
  strategy: "token_bucket"
  global_limit: 1000
  per_user_limit: 100

audit:
  enabled: true
  retention_days: 30

cluster:
  enabled: true
  bind: ":12000"
  mtls:
    auto_generate: true
```

## System Requirements

- **Minimum**: 1 CPU, 512MB RAM
- **Recommended**: 2+ CPUs, 2GB+ RAM
- **Storage**: SQLite works well for up to ~10K requests/sec; PostgreSQL recommended for higher throughput
- **Supported Platforms**: Linux (amd64/arm64), macOS (amd64/arm64), Windows (amd64)

## Version Information

Current version: **v1.0.0**
- Go coverage: 80.4%
- Total tests: 5938
- Frontend tests: 314