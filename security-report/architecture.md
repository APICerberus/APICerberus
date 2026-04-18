# APICerebrus Security Architecture Report

**Phase 1: Recon - Architecture Map**
**Date:** 2026-04-18
**Project:** APICerebrus - Production API Gateway with Raft Clustering
**Classification:** INTERNAL

---

## 1. Tech Stack Overview

### 1.1 Backend (Go 1.26.2)

| Library | Version | Purpose | Risk Profile |
|---------|---------|---------|--------------|
| modernc.org/sqlite | v1.48.0 | SQLite database (pure Go, no CGO) | Low |
| github.com/redis/go-redis/v9 | v9.8.0 | Distributed rate limiting | Low |
| google.golang.org/grpc | v1.80.0 | gRPC server | Low |
| github.com/golang-jwt/jwt/v5 | v5.3.1 | JWT parsing/validation | Medium |
| github.com/tetratelabs/wazero | v1.11.0 | WASM runtime (sandboxed) | Medium |
| golang.org/x/crypto | v0.49.0 | Cryptographic operations | Low |
| golang.org/x/oauth2 | v0.36.0 | OAuth2/OIDC integration | Medium |
| github.com/coreos/go-oidc/v3 | v3.18.0 | OIDC provider | Medium |

### 1.2 Frontend (React 19.2.4)

| Component | Version |
|-----------|---------|
| React | 19.2.4 |
| TypeScript | 5.9.3 |
| Vite | 8.0.1 |
| Tailwind CSS | 4.2.2 |
| Zustand | 5.0.12 |

### 1.3 Infrastructure

Docker, Kubernetes, Helm, SQLite (WAL), Redis, Kafka

---

## 2. Go Packages Under internal/

| Package | Files | Purpose |
|---------|-------|---------|
| internal/admin/ | 64 | Admin REST API, OIDC provider, JWT auth, webhooks |
| internal/gateway/ | 37 | HTTP gateway, routing, load balancing, proxy |
| internal/graphql/ | 15 | GraphQL proxy, subscriptions |
| internal/federation/ | 14 | GraphQL Federation executor |
| internal/store/ | 30 | SQLite store, repositories |
| internal/plugin/ | 61 | Plugin system, auth, WASM runtime |
| internal/raft/ | 22 | Raft consensus, clustering |
| internal/portal/ | 14 | User portal server |
| internal/mcp/ | 12 | MCP server, JSON-RPC stdio/SSE |

And 15+ more packages (cli, config, grpc, ratelimit, analytics, audit, billing, certmanager, logging, metrics, tracing, loadbalancer, shutdown, migrations, version, pkg)

---

## 3. Network Entry Points

| Port | Service | Purpose |
|------|---------|---------|
| 8080 | Gateway HTTP | Main proxy traffic |
| 8443 | Gateway HTTPS | Secure proxy |
| 9876 | Admin API | Management interface |
| 9877 | Portal | User-facing portal |
| 50051 | gRPC | gRPC services |
| 12000 | Raft RPC | Cluster node communication |
| stdio | MCP Server | CLI tool integration |

---

## 4. Authentication Mechanisms

- API Key (ck_live_*/ck_test_*) via SHA256 hash
- Admin JWT (HS256, 15-min TTL, CSRF protection)
- OIDC Provider (PKCE S256, JWKS validation)
- Portal Sessions (HttpOnly, Secure, SameSite)

---

## 5. Cryptographic Libraries

- golang.org/x/crypto (bcrypt cost 12, TLS)
- github.com/golang-jwt/jwt/v5 (HS256)
- modernc.org/sqlite (CGO-free)

---

## 6. Security Controls

- API Key hashing (SHA256) - High
- Constant-time comparison - High
- Auth backoff (DoS protection) - High
- WASM sandbox (wazero 128MB) - Medium
- Credit atomic transactions - High
- TLS 1.2+ minimum - High
- Raft mTLS with TLS 1.3 - High
- bcrypt cost 12 - High
- CSRF double-submit (M-014) - High

---

## 7. Entry Point Reference

cmd/apicerberus/main.go -> cli.Run()
- start: Gateway + Admin + Portal + Raft
- mcp: JSON-RPC over stdio/SSE
- user, credit, audit, analytics, service, route, upstream, db commands

---

## 8. Trust Boundaries

HIGH-TRUST: Gateway Process, Admin API, MCP Server, Store Layer, Raft Cluster
LOW-TRUST: Client Requests, Upstream Servers, Redis, Kafka, ACME

---

Report generated: 2026-04-18