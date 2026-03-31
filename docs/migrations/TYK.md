# Migration Guide: Tyk → API Cerberus

This guide helps you migrate from Tyk Gateway to API Cerberus.

## Key Differences

| Feature | Tyk | API Cerberus |
|---------|-----|--------------|
| **Configuration** | JSON files or Dashboard | Native YAML |
| **Authentication** | Multiple policies | JWT + API Key |
| **Analytics** | MongoDB/Redis | Prometheus + Tracing |
| **Clustering** | Redis + MongoDB | Built-in Raft |
| **GraphQL** | Via plugins | Native Federation |

## API Definition Migration

### Tyk API Definition
```json
{
  "name": "Example API",
  "slug": "example-api",
  "proxy": {
    "listen_path": "/example/",
    "target_url": "http://example.com",
    "strip_listen_path": true
  },
  "active": true
}
```

### API Cerberus Equivalent
```yaml
backend:
  services:
    - id: example-service
      name: Example Service
      protocol: http
      upstream:
        targets:
          - id: target-1
            address: http://example.com
            weight: 100

  routes:
    - id: example-route
      name: Example Route
      service_id: example-service
      paths:
        - /example
      strip_path: true
```

## Middleware Migration

### Authentication

**Tyk:**
```json
{
  "use_keyless": false,
  "use_standard_auth": true,
  "auth": {
    "auth_header_name": "Authorization"
  }
}
```

**API Cerberus:**
```yaml
auth:
  jwt:
    enabled: true
    secret: ${JWT_SECRET}
    issuer: apicerberus
    header: Authorization
```

### Rate Limiting

**Tyk:**
```json
{
  "rate_limit": {
    "rate": 100,
    "per": 60
  }
}
```

**API Cerberus:**
```yaml
rate_limiting:
  enabled: true
  requests_per_second: 100
  burst_size: 150
  per_user: true
  per_ip: true
```

### CORS

**Tyk:**
```json
{
  "CORS": {
    "enable": true,
    "allowed_origins": ["*"],
    "allowed_methods": ["GET", "POST"]
  }
}
```

**API Cerberus:**
```yaml
routes:
  - id: api-route
    cors:
      enabled: true
      allow_origins:
        - "*"
      allow_methods:
        - GET
        - POST
        - PUT
        - DELETE
```

## Key Migration

### Tyk
```bash
# Create key via API
curl -X POST \
  https://tyk-gateway/tyk/keys/create \
  -H "x-tyk-authorization: ${TYK_SECRET}" \
  -d '{
    "rate": 100,
    "per": 60,
    "quota_max": 1000
  }'
```

### API Cerberus
```bash
# Create user via Admin API
curl -X POST \
  http://localhost:8081/admin/api/v1/users \
  -H "Authorization: Bearer ${ADMIN_API_KEY}" \
  -d '{
    "username": "new-user",
    "credits": 1000,
    "rate_limit": 100
  }'
```

## Policy Migration

Tyk Policies → API Cerberus Users:

**Tyk Policy:**
```json
{
  "name": "Gold Plan",
  "rate": 1000,
  "per": 60,
  "quota": 10000
}
```

**API Cerberus User:**
```yaml
users:
  - id: user-123
    username: gold-user
    credits: 10000
    rate_limit: 1000
    tier: gold
```

## Analytics Migration

**Tyk:** Requires MongoDB/Redis for analytics storage.

**API Cerberus:** Uses Prometheus metrics and OpenTelemetry tracing:
```yaml
metrics:
  enabled: true
  endpoint: /metrics

tracing:
  enabled: true
  endpoint: http://jaeger:4317
```

## Step-by-Step Migration

1. **Export Tyk Configuration**
   ```bash
   curl https://tyk-dashboard/api/apis \
     -H "Authorization: ${TYK_AUTH}" > tyk-apis.json
   ```

2. **Convert to API Cerberus**
   ```bash
   ./scripts/migrate-from-tyk.sh tyk-apis.json > apicerberus.yaml
   ```

3. **Validate**
   ```bash
   apicerberus validate --config apicerberus.yaml
   ```

4. **Deploy**
   ```bash
   docker-compose up -d
   ```

## Troubleshooting

### Issue: API not accessible
**Solution:** Check `strip_path` setting. Tyk's `strip_listen_path` is equivalent.

### Issue: Authentication not working
**Solution:** API Cerberus supports JWT natively. Tyk's custom auth methods need conversion.

### Issue: Rate limiting inaccurate
**Solution:** Enable Raft clustering for distributed rate limiting consistency.

## Feature Comparison

| Feature | Tyk | API Cerberus |
|---------|-----|--------------|
| OIDC | ✅ | Via JWT |
| OAuth2 | ✅ | Via JWT |
| Basic Auth | ✅ | ❌ (JWT only) |
| HMAC | ✅ | ❌ |
| IP Whitelist | ✅ | ✅ |
| Circuit Breaker | ✅ | ✅ (Adaptive LB) |
| Cache | ✅ | ✅ |
| GraphQL | Via plugin | Native |

## Support

For migration assistance:
- GitHub Issues: https://github.com/APICerberus/APICerebrus/issues
- Documentation: https://apicerberus.com/docs
