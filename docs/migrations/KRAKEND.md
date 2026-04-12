# Migration Guide: KrakenD → API Cerberus

This guide helps you migrate from KrakenD to API Cerberus.

## Key Differences

| Feature | KrakenD | API Cerberus |
|---------|---------|--------------|
| **Configuration** | JSON flexible config | Structured YAML |
| **Aggregation** | Native | Via GraphQL Federation |
| **Middleware** | Go plugins | Built-in middleware |
| **Clustering** | etcd/Redis | Built-in Raft |
| **Caching** | Redis | In-memory |

## Endpoint Migration

### KrakenD Endpoint
```json
{
  "endpoints": [
    {
      "endpoint": "/users/{id}",
      "method": "GET",
      "backend": [
        {
          "url_pattern": "/users/{id}",
          "host": ["http://user-service:8080"]
        }
      ]
    }
  ]
}
```

### API Cerberus Equivalent
```yaml
backend:
  services:
    - id: user-service
      name: User Service
      protocol: http
      upstream:
        targets:
          - id: target-1
            address: http://user-service:8080
            weight: 100

  routes:
    - id: user-route
      name: User Route
      service_id: user-service
      paths:
        - /users/:id
      methods:
        - GET
```

## Response Aggregation Migration

### KrakenD Aggregation
```json
{
  "endpoints": [
    {
      "endpoint": "/aggregate",
      "backend": [
        {"url_pattern": "/users", "host": ["http://user-service:8080"]},
        {"url_pattern": "/orders", "host": ["http://order-service:8080"]}
      ]
    }
  ]
}
```

### API Cerberus GraphQL Federation
```yaml
federation:
  enabled: true
  subgraphs:
    - id: users
      url: http://user-service:8080/graphql
    - id: orders
      url: http://order-service:8080/graphql
```

Query:
```graphql
query {
  users { id name }
  orders { id total }
}
```

## Middleware Migration

### Rate Limiting

**KrakenD:**
```json
{
  "extra_config": {
    "qos/ratelimit/router": {
      "max_rate": 100,
      "client_max_rate": 10
    }
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

### Circuit Breaker

**KrakenD:**
```json
{
  "extra_config": {
    "qos/circuit-breaker": {
      "max_errors": 5,
      "timeout": "30s"
    }
  }
}
```

**API Cerberus:**
```yaml
backend:
  services:
    - id: example-service
      circuit_breaker:
        enabled: true
        threshold: 5
        timeout: 30s
        half_open_max_calls: 3
```

### JWT Validation

**KrakenD:**
```json
{
  "extra_config": {
    "auth/validator": {
      "alg": "RS256",
      "jwk_url": "https://auth.example.com/.well-known/jwks.json"
    }
  }
}
```

**API Cerberus:**
```yaml
auth:
  jwt:
    enabled: true
    secret: ${JWT_SECRET}
    issuer: auth.example.com
    algorithms:
      - RS256
```

## Backend Migration

### KrakenD Backend
```json
{
  "backend": [
    {
      "url_pattern": "/api/{id}",
      "host": ["http://service1:8080", "http://service2:8080"],
      "sd": "static",
      "method": "GET"
    }
  ]
}
```

### API Cerberus Backend
```yaml
backend:
  upstreams:
    - id: api-upstream
      name: API Upstream
      algorithm: round_robin
      targets:
        - id: target-1
          address: http://service1:8080
          weight: 100
        - id: target-2
          address: http://service2:8080
          weight: 100

  services:
    - id: api-service
      name: API Service
      upstream_id: api-upstream
```

## Response Manipulation

### KrakenD Filtering
```json
{
  "extra_config": {
    "modifier/jmespath": {
      "expr": "{id: id, name: name}"
    }
  }
}
```

### API Cerberus
Response manipulation is done via GraphQL field selection:
```graphql
query {
  user(id: "123") {
    id
    name
  }
}
```

## Step-by-Step Migration

1. **Export KrakenD Configuration**
   ```bash
   curl http://krakend:8080/__debug/config > krakend.json
   ```

2. **Convert to API Cerberus**
   ```bash
   ./scripts/migrate-from-krakend.sh krakend.json > apicerberus.yaml
   ```

3. **Validate**
   ```bash
   apicerberus validate --config apicerberus.yaml
   ```

4. **Test**
   ```bash
   docker-compose -f docker-compose.standalone.yml up
   ```

5. **Deploy**
   ```bash
   helm install apicerberus ./deployments/helm/apicerberus
   ```

## Performance Comparison

| Metric | KrakenD | API Cerberus |
|--------|---------|--------------|
| Throughput | High | High |
| Latency | Low | Low |
| Memory | Low | Medium |
| Features | Aggregation-focused | Full-featured |

## Troubleshooting

### Issue: Aggregation not working
**Solution:** API Cerberus uses GraphQL Federation. Convert REST APIs to GraphQL or use the REST aggregation feature.

### Issue: Circuit breaker not triggering
**Solution:** API Cerberus uses adaptive load balancing. Configure error rate thresholds.

### Issue: JWT validation failing
**Solution:** API Cerberus validates JWT locally. Ensure the secret/key is correctly configured.

## Feature Matrix

| Feature | KrakenD | API Cerberus |
|---------|---------|--------------|
| HTTP REST | ✅ | ✅ |
| HTTP2/gRPC | ✅ | ✅ |
| GraphQL | Limited | Native |
| Response Aggregation | ✅ | Via GraphQL |
| Backend Proxy | ✅ | ✅ |
| Load Balancing | ✅ | ✅ (Adaptive) |
| Rate Limiting | ✅ | ✅ |
| Circuit Breaker | ✅ | ✅ |
| Caching | Redis | In-memory |
| JWT | ✅ | ✅ |
| OAuth2 | Partial | Via JWT |
| GeoIP | ❌ | ✅ |
| WebSockets | ❌ | ✅ |

## Support

For migration assistance:
- GitHub Issues: https://github.com/APICerberus/APICerebrus/issues
- Documentation: https://apicerberus.com/docs
