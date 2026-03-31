# Migration Guide: Kong → API Cerberus

This guide helps you migrate from Kong Gateway to API Cerberus.

## Key Differences

| Feature | Kong | API Cerberus |
|---------|------|--------------|
| **Configuration** | Declarative (YAML/JSON) or DB | Native YAML with live reload |
| **Plugins** | Lua plugins | Go middleware |
| **Clustering** | PostgreSQL/Cassandra | Built-in Raft |
| **GraphQL** | Via plugins | Native + Federation |
| **Rate Limiting** | Redis-based | In-memory + Raft |

## Service Migration

### Kong Service
```yaml
# Kong
services:
- name: example-service
  url: http://example.com
  routes:
  - name: example-route
    paths:
    - /example
```

### API Cerberus Equivalent
```yaml
# API Cerberus
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

## Plugin Migration

### Rate Limiting

**Kong:**
```yaml
plugins:
- name: rate-limiting
  service: example-service
  config:
    minute: 100
    policy: redis
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

### JWT Authentication

**Kong:**
```yaml
plugins:
- name: jwt
  config:
    key_claim_name: iss
    secret_is_base64: false
```

**API Cerberus:**
```yaml
auth:
  jwt:
    enabled: true
    secret: ${JWT_SECRET}
    issuer: apicerberus
    expiry: 24h
```

### CORS

**Kong:**
```yaml
plugins:
- name: cors
  config:
    origins:
    - "*"
    methods:
    - GET
    - POST
```

**API Cerberus:**
Built into route configuration:
```yaml
routes:
  - id: cors-route
    cors:
      enabled: true
      allow_origins:
        - "*"
      allow_methods:
        - GET
        - POST
```

## Step-by-Step Migration

1. **Export Kong Configuration**
   ```bash
   kong config db_export kong-config.yaml
   ```

2. **Convert to API Cerberus Format**
   Use the migration script (see `scripts/migrate-from-kong.sh`):
   ```bash
   ./scripts/migrate-from-kong.sh kong-config.yaml > apicerberus.yaml
   ```

3. **Validate Configuration**
   ```bash
   apicerberus validate --config apicerberus.yaml
   ```

4. **Test in Staging**
   ```bash
   docker-compose -f docker-compose.standalone.yml up
   ```

5. **Deploy to Production**
   ```bash
   kubectl apply -f k8s/
   ```

## Consumer Migration

Kong Consumers → API Cerberus Users:

**Kong:**
```yaml
consumers:
- username: john-doe
  custom_id: user-123
```

**API Cerberus:**
```yaml
users:
  - id: user-123
    username: john-doe
    credits: 1000
    rate_limit: 100
```

## Upstream Migration

**Kong:**
```yaml
upstreams:
- name: example-upstream
  targets:
  - target: 10.0.0.1:8080
    weight: 100
```

**API Cerberus:**
```yaml
backend:
  upstreams:
    - id: example-upstream
      name: Example Upstream
      algorithm: round_robin
      targets:
        - id: target-1
          address: 10.0.0.1:8080
          weight: 100
```

## Troubleshooting

### Issue: Rate limits not working
**Solution:** API Cerberus uses in-memory counters. For distributed rate limiting, enable Raft clustering.

### Issue: JWT tokens rejected
**Solution:** API Cerberus expects `Authorization: Bearer <token>` header. Kong's `jwt_token_cookie_name` is not supported.

### Issue: Plugin not available
**Solution:** API Cerberus has different plugin architecture. Check the feature matrix above.

## Support

For migration assistance:
- GitHub Issues: https://github.com/APICerberus/APICerebrus/issues
- Documentation: https://apicerberus.com/docs
