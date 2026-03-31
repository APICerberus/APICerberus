# Changelog

All notable changes to API Cerberus will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-03-31

### Added - Production Release

#### Core Gateway
- HTTP/HTTPS reverse proxy with WebSocket support
- Custom radix tree router with path parameters and wildcards
- 10 load balancing algorithms (round robin, weighted, least_conn, ip_hash, consistent_hash, adaptive, least_latency, health_weighted, random)
- Active and passive health checking
- Circuit breaker pattern with configurable thresholds
- Retry mechanism with exponential backoff
- Request/response transformation plugins
- Compression (gzip/deflate) support
- CORS handling
- IP restriction (whitelist/blacklist)
- Request size limiting
- JSON schema validation
- Bot detection

#### Authentication & Authorization
- API key authentication with SHA-256 hashing
- JWT authentication (HS256, RS256, JWKS)
- JWKS caching with automatic refresh
- Role-based access control (RBAC)
- Per-endpoint permissions with time/day restrictions
- User IP whitelisting

#### Rate Limiting
- Token bucket algorithm
- Fixed window algorithm
- Sliding window algorithm
- Leaky bucket algorithm
- Multiple scope levels (global, route, service, user, IP)

#### Multi-Tenant Management
- Embedded SQLite storage
- User management with roles
- API key management (ck_live_/ck_test_ prefixes)
- Credit-based billing system
- Atomic credit transactions
- Test key bypass for credit deduction

#### Protocol Support
- HTTP/1.1 and HTTP/2 (h2c)
- WebSocket proxy with bidirectional streaming
- gRPC proxy with HTTP/2 framing
- gRPC-Web support
- gRPC-JSON transcoding
- GraphQL query proxy
- GraphQL query depth analysis
- GraphQL complexity analysis
- GraphQL federation with schema composition
- GraphQL subscriptions

#### Clustering & HA
- Raft consensus implementation
- Distributed rate limiting
- Distributed credit balance
- Cluster health sharing
- Real-time topology visualization
- Leader election and failover

#### Observability
- Structured logging with slog
- Request/response audit logging
- Sensitive data masking
- Log retention and archival
- Real-time analytics engine
- Prometheus metrics export
- OpenTelemetry tracing
- Webhook notifications

#### Admin Interfaces
- RESTful Admin API (40+ endpoints)
- Web Dashboard (React + shadcn/ui)
  - 35+ UI components
  - Real-time charts with Recharts
  - React Flow topology visualization
  - Dark/light theme support
- User Portal with API Playground
  - CodeMirror editors
  - Request builder
  - Response viewer
- MCP Server (stdio + SSE transports)
  - 25+ tools for LLM integration
- CLI with 40+ commands

#### Operations
- TLS with ACME auto-provisioning
- Hot configuration reload (SIGHUP)
- Graceful shutdown
- Config export/import with diff
- Docker support
- Kubernetes Helm charts
- Docker Compose examples

#### Documentation
- Comprehensive architecture documentation
- OpenAPI 3.0 specification
- Migration guides (Kong, Tyk, KrakenD)
- Contributing guidelines
- Security audit report

#### CI/CD
- GitHub Actions workflows
- Automated testing (unit, integration, e2e)
- Security scanning (Trivy, gosec, govulncheck)
- Multi-arch Docker builds (amd64, arm64)
- Automated releases
- Dependabot configuration

### Performance Targets
- 50,000+ requests/second per node
- P99 latency < 1ms (no plugins)
- 99.99% availability
- < 100ms configuration propagation

## Previous Versions

See git history for changes in v0.0.1 through v0.7.0.

[1.0.0]: https://github.com/APICerberus/APICerebrus/releases/tag/v1.0.0
