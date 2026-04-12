# Security Architecture

## Threat Model

### STRIDE Analysis

| Threat | Component | Mitigation |
|--------|-----------|------------|
| **Spoofing** | Identity | JWT validation, API key authentication, mTLS |
| **Tampering** | Data integrity | TLS 1.3, request signing, checksum validation |
| **Repudiation** | Logging | Immutable audit logs, request IDs, timestamping |
| **Information Disclosure** | Confidentiality | Encryption in transit, field-level encryption |
| **Denial of Service** | Availability | Rate limiting, circuit breakers, resource quotas |
| **Elevation of Privilege** | Authorization | RBAC, scope-based permissions, token binding |

## Security Layers

```
┌─────────────────────────────────────────────────────────────┐
│                   Security Architecture                      │
│                                                              │
│   ┌─────────────────────────────────────────────────────┐   │
│   │   Layer 5: Application Security                      │   │
│   │   • Input validation                                 │   │
│   │   • Output encoding                                  │   │
│   │   • CSRF protection                                  │   │
│   └─────────────────────────────────────────────────────┘   │
│   ┌─────────────────────────────────────────────────────┐   │
│   │   Layer 4: Authentication & Authorization            │   │
│   │   • JWT/OAuth2 verification                          │   │
│   │   • API key validation                               │   │
│   │   • RBAC enforcement                                 │   │
│   └─────────────────────────────────────────────────────┘   │
│   ┌─────────────────────────────────────────────────────┐   │
│   │   Layer 3: Transport Security                        │   │
│   │   • TLS 1.3                                          │   │
│   │   • Certificate pinning                              │   │
│   │   • HSTS headers                                     │   │
│   └─────────────────────────────────────────────────────┘   │
│   ┌─────────────────────────────────────────────────────┐   │
│   │   Layer 2: Network Security                          │   │
│   │   • IP allowlisting                                  │   │
│   │   • DDoS protection                                  │   │
│   │   • Network segmentation                             │   │
│   └─────────────────────────────────────────────────────┘   │
│   ┌─────────────────────────────────────────────────────┐   │
│   │   Layer 1: Infrastructure Security                   │   │
│   │   • Container isolation                              │   │
│   │   • Secrets management                               │   │
│   │   • Security scanning                                │   │
│   └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Authentication Mechanisms

### JWT Token Flow

```
┌─────────────────────────────────────────────────────────────┐
│                     JWT Authentication                       │
│                                                              │
│   ┌──────────┐      ┌──────────┐      ┌──────────┐        │
│   │  Client  │      │  Gateway │      │  JWKS    │        │
│   │          │      │          │      │  Server  │        │
│   └────┬─────┘      └────┬─────┘      └────┬─────┘        │
│        │                 │                 │               │
│        │── Request + ───►│                 │               │
│        │   JWT Token     │                 │               │
│        │                 │                 │               │
│        │                 │── Fetch Key ──►│               │
│        │                 │   (if needed)   │               │
│        │                 │◄── Public Key ──┤               │
│        │                 │                 │               │
│        │                 │── Verify ──────┐│               │
│        │                 │   Signature    ││               │
│        │                 │   Expiration   ││               │
│        │                 │   Claims       ││               │
│        │                 │◄───────────────┘│               │
│        │                 │                 │               │
│        │◄── Response ────┤                 │               │
│        │   (or 401)      │                 │               │
│        │                 │                 │               │
└─────────────────────────────────────────────────────────────┘
```

### API Key Authentication

```
┌─────────────────────────────────────────────────────────────┐
│                    API Key Authentication                    │
│                                                              │
│   Request Headers:                                           │
│   X-API-Key: ak_live_xxxxxxxxxxxx                           │
│   X-API-Key-Secret: (optional, for HMAC)                    │
│                                                              │
│   ┌─────────────────────────────────────────────────────┐   │
│   │              Key Validation Process                  │   │
│   │                                                      │   │
│   │   1. Extract API Key from header                     │   │
│   │   2. Parse key prefix (live/test)                    │   │
│   │   3. Hash key (SHA-256)                              │   │
│   │   4. Lookup in BadgerDB                              │   │
│   │   5. Verify key status (active/revoked)              │   │
│   │   6. Check rate limits for key                       │   │
│   │   7. Enrich context with key metadata                │   │
│   │   8. Log access attempt                              │   │
│   │                                                      │   │
│   └─────────────────────────────────────────────────────┘   │
│                                                              │
│   Key Metadata:                                              │
│   - ID, Name, Organization                                   │
│   - Scopes (read, write, admin)                              │
│   - Rate limit tier                                          │
│   - IP restrictions                                          │
│   - Created/expires timestamps                               │
└─────────────────────────────────────────────────────────────┘
```

## Authorization Model

### RBAC Structure

```
┌─────────────────────────────────────────────────────────────┐
│                   RBAC Implementation                        │
│                                                              │
│   ┌──────────┐     ┌──────────┐     ┌──────────┐          │
│   │   User   │────►│   Role   │────►│ Permission│          │
│   │          │     │          │     │          │          │
│   └──────────┘     └──────────┘     └──────────┘          │
│        │                │                │                 │
│        │                │                │                 │
│   ┌────┴────┐      ┌────┴────┐      ┌────┴────┐          │
│   │  Teams  │      │  Groups │      │  Scopes │          │
│   │         │      │         │      │         │          │
│   └─────────┘      └─────────┘      └─────────┘          │
│                                                              │
│   Default Roles:                                             │
│   • superadmin - Full access                                 │
│   • admin - Manage services, routes, users                   │
│   • operator - View metrics, restart services                │
│   • viewer - Read-only access                                │
│   • service - Service-to-service communication               │
│                                                              │
│   Permission Format: `{resource}:{action}:{scope}`          │
│   Examples:                                                  │
│   - `services:read:*`                                        │
│   - `routes:write:production`                                │
│   - `cluster:manage:leader`                                  │
└─────────────────────────────────────────────────────────────┘
```

## TLS Configuration

### Certificate Management

```
┌─────────────────────────────────────────────────────────────┐
│                  TLS Certificate Flow                        │
│                                                              │
│   ┌─────────────────────────────────────────────────────┐   │
│   │              Certificate Sources                     │   │
│   │                                                      │   │
│   │   ┌──────────┐    ┌──────────┐    ┌──────────┐     │   │
│   │   │  Static  │    │   ACME   │    │   mTLS   │     │   │
│   │   │  Files   │    │Let's Encrypt│  │  CA      │     │   │
│   │   └────┬─────┘    └────┬─────┘    └────┬─────┘     │   │
│   │        └─────────────────┴─────────────────┘         │   │
│   │                      │                               │   │
│   │                      ▼                               │   │
│   │              ┌───────────────┐                       │   │
│   │              │  TLS Manager  │                       │   │
│   │              └───────┬───────┘                       │   │
│   │                      │                               │   │
│   │         ┌────────────┼────────────┐                  │   │
│   │         ▼            ▼            ▼                  │   │
│   │    ┌────────┐   ┌────────┐   ┌────────┐             │   │
│   │    │ Public │   │ Private│   │   CA   │             │   │
│   │    │  Cert  │   │  Key   │   │  Cert  │             │   │
│   │    └────────┘   └────────┘   └────────┘             │   │
│   │                                                      │   │
│   └─────────────────────────────────────────────────────┘   │
│                                                              │
│   Auto-Renewal:                                              │
│   • Check expiry daily                                       │
│   • Renew at 30 days before expiry                           │
│   • Hot reload without restart                               │
│   • Store in encrypted format                                │
└─────────────────────────────────────────────────────────────┘
```

### TLS Handshake

```
┌─────────────────────────────────────────────────────────────┐
│                     TLS 1.3 Handshake                        │
│                                                              │
│   Client                           Server                   │
│     │                                │                      │
│     │── ClientHello ────────────────►│                      │
│     │   + Key Share                  │                      │
│     │   + Supported Groups           │                      │
│     │   + Signature Algorithms       │                      │
│     │                                │                      │
│     │◄── ServerHello ─────────────────┤                      │
│     │   + Key Share                  │                      │
│     │   + Certificate                │                      │
│     │   + {EncryptedExtensions}      │                      │
│     │                                │                      │
│     │── {Finished} ─────────────────►│                      │
│     │   + HTTP Request               │                      │
│     │                                │                      │
│     │◄── {Finished, HTTP Response} ───┤                      │
│     │                                │                      │
│     │                                │                      │
│     │  [Application Data]            │                      │
│     │  (0-RTT capable for resumption)│                      │
│                                                              │
│   Cipher Suites (Priority Order):                            │
│   • TLS_AES_256_GCM_SHA384                                  │
│   • TLS_CHACHA20_POLY1305_SHA256                            │
│   • TLS_AES_128_GCM_SHA256                                  │
└─────────────────────────────────────────────────────────────┘
```

## Secrets Management

```
┌─────────────────────────────────────────────────────────────┐
│                  Secrets Architecture                        │
│                                                              │
│   ┌─────────────────────────────────────────────────────┐   │
│   │              Secret Sources                          │   │
│   │                                                      │   │
│   │   ┌──────────┐   ┌──────────┐   ┌──────────┐       │   │
│   │   │ Env Vars │   │  Vault   │   │  Files   │       │   │
│   │   │ (dev)    │   │(production)│  │(mounted) │       │   │
│   │   └────┬─────┘   └────┬─────┘   └────┬─────┘       │   │
│   │        └───────────────┼───────────────┘             │   │
│   │                        │                             │   │
│   │                        ▼                             │   │
│   │              ┌─────────────────┐                     │   │
│   │              │ Secrets Manager │                     │   │
│   │              └────────┬────────┘                     │   │
│   │                       │                              │   │
│   │                       ▼                              │   │
│   │              ┌─────────────────┐                     │   │
│   │              │  Memory Cache   │                     │   │
│   │              │  (never disk)   │                     │   │
│   │              └─────────────────┘                     │   │
│   │                                                      │   │
│   └─────────────────────────────────────────────────────┘   │
│                                                              │
│   Secrets:                                                   │
│   • JWT signing keys                                         │
│   • API key hashes (argon2)                                  │
│   • TLS private keys                                         │
│   • Database credentials                                     │
│   • External service tokens                                  │
│                                                              │
│   Never Log:                                                 │
│   • Raw API keys                                             │
│   • JWT tokens (full)                                        │
│   • Passwords                                                │
│   • Private keys                                             │
└─────────────────────────────────────────────────────────────┘
```

## Security Headers

| Header | Value | Purpose |
|--------|-------|---------|
| Strict-Transport-Security | `max-age=31536000; includeSubDomains` | Enforce HTTPS |
| X-Content-Type-Options | `nosniff` | Prevent MIME sniffing |
| X-Frame-Options | `DENY` | Prevent clickjacking |
| X-XSS-Protection | `1; mode=block` | XSS protection (legacy) |
| Content-Security-Policy | `default-src 'self'` | CSP restrictions |
| Referrer-Policy | `strict-origin-when-cross-origin` | Referrer control |
| Permissions-Policy | `geolocation=(), microphone=()` | Feature policy |

## Audit Logging

```
┌─────────────────────────────────────────────────────────────┐
│                    Audit Events                              │
│                                                              │
│   Events Logged:                                             │
│   • Authentication attempts (success/failure)               │
│   • Authorization failures                                   │
│   • Configuration changes                                    │
│   • Cluster membership changes                               │
│   • Certificate operations                                   │
│   • Rate limit violations                                    │
│                                                              │
│   ┌─────────────────────────────────────────────────────┐   │
│   │              Audit Log Format                        │   │
│   │                                                      │   │
│   │   {                                                  │   │
│   │     "timestamp": "2025-01-15T10:30:00Z",             │   │
│   │     "event_type": "AUTHENTICATION",                  │   │
│   │     "event_outcome": "SUCCESS",                      │   │
│   │     "actor": {                                       │   │
│   │       "id": "user_123",                              │   │
│   │       "ip": "192.168.1.100",                         │   │
│   │       "user_agent": "Mozilla/5.0..."                 │   │
│   │     },                                               │   │
│   │     "resource": {                                    │   │
│   │       "type": "service",                             │   │
│   │       "id": "svc_456",                               │   │
│   │       "action": "UPDATE"                             │   │
│   │     },                                               │   │
│   │     "request_id": "req_abc123"                       │   │
│   │   }                                                  │   │
│   │                                                      │   │
│   └─────────────────────────────────────────────────────┘   │
│                                                              │
│   Retention:                                                 │
│   • Hot storage: 7 days (in BadgerDB)                       │
│   • Warm storage: 90 days (S3/GCS)                          │
│   • Cold storage: 7 years (compliance)                       │
│                                                              │
│   Protection:                                                │
│   • Immutable writes                                         │
│   • Cryptographic checksums                                  │
│   • Tamper-evident logging                                   │
└─────────────────────────────────────────────────────────────┘
```

## Vulnerability Scanning

```
┌─────────────────────────────────────────────────────────────┐
│              Security Scanning Pipeline                      │
│                                                              │
│   ┌─────────────────────────────────────────────────────┐   │
│   │              CI/CD Security Checks                   │   │
│   │                                                      │   │
│   │   Code ──► ┌────────┐ ──► ┌────────┐ ──► ┌────────┐│   │
│   │            │ gosec  │     │govulncheck│   │  trivy ││   │
│   │            │(SAST)  │     │(vulns)   │   │(images)││   │
│   │            └────────┘     └────────┘     └────────┘│   │
│   │                                                      │   │
│   │   Checks:                                            │   │
│   │   • Hardcoded secrets                                │   │
│   │   • SQL injection patterns                           │   │
│   │   • Unsafe deserializations                          │   │
│   │   • Weak cryptography                                │   │
│   │   • OS vulnerabilities (containers)                  │   │
│   │                                                      │   │
│   └─────────────────────────────────────────────────────┘   │
│                                                              │
│   ┌─────────────────────────────────────────────────────┐   │
│   │              Runtime Security                        │   │
│   │                                                      │   │
│   │   • Syscall filtering (seccomp)                      │   │
│   │   • AppArmor/SELinux profiles                        │   │
│   │   • Read-only root filesystem                        │   │
│   │   • Non-root user execution                          │   │
│   │   • Resource limits (cgroups)                        │   │
│   │                                                      │   │
│   └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```
