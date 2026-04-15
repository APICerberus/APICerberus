# Rate Limiting

API Cerberus provides multiple rate limiting strategies for local and distributed deployments.

## Strategies

### Token Bucket (Default)

The token bucket algorithm allows bursts while maintaining an average rate.

```yaml
ratelimit:
  enabled: true
  strategy: "token_bucket"
  global_limit: 1000    # Global requests per window
  per_user_limit: 100  # Per-user requests per window
  window: 60s          # Time window
```

**Characteristics:**
- Allows short bursts up to bucket capacity
- Smooth rate limiting over time
- Memory efficient

### Sliding Window

The sliding window algorithm provides more precise rate limiting.

```yaml
ratelimit:
  strategy: "sliding_window"
  window: 60s
  bucket_size: 100
```

**Characteristics:**
- Smoother rate limiting than fixed window
- Lower burst tolerance
- More memory usage

### Leaky Bucket

The leaky bucket algorithm enforces a constant output rate.

```yaml
ratelimit:
  strategy: "leaky_bucket"
  rate: 50           # Requests per second
  bucket_size: 100  # Max queue size
```

**Characteristics:**
- Constant rate enforcement
- Excess requests are dropped
- Fair queuing

## Distributed Rate Limiting (Redis)

For multi-instance deployments, use Redis-backed rate limiting:

```yaml
ratelimit:
  enabled: true
  strategy: "token_bucket"
  redis:
    enabled: true
    host: "localhost:6379"
    password: ""
    db: 0
    pool_size: 10
    min_idle_conns: 5
    connect_timeout: 5s
    read_timeout: 3s
    write_timeout: 3s
```

## Per-Route Rate Limits

Configure rate limits per route:

```bash
# Create route with custom rate limit
curl -X POST http://localhost:9876/admin/api/v1/routes \
  -H "X-Admin-Key: your-key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "limited-route",
    "paths": ["/api/v1/expensive"],
    "ratelimit": {
      "limit": 10,
      "window": 60
    }
  }'
```

## Per-User Rate Limits

Set rate limits per user:

```bash
# Set user-specific rate limit
curl -X PUT http://localhost:9876/admin/api/v1/users/<user-id>/ratelimit \
  -H "X-Admin-Key: your-key" \
  -H "Content-Type: application/json" \
  -d '{"limit": 50, "window": 60}'
```

## Response Headers

When rate limiting is enabled, responses include:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1640000000
```

## Rate Limit Exceeded Response

When a client exceeds their rate limit:

```http
HTTP/1.1 429 Too Many Requests
Content-Type: application/json

{
  "error": "rate_limit_exceeded",
  "message": "Rate limit exceeded. Try again in 30 seconds.",
  "retry_after": 30
}
```

## Configuration Reference

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | bool | `true` | Enable/disable rate limiting |
| `strategy` | string | `token_bucket` | Algorithm: token_bucket, sliding_window, leaky_bucket |
| `global_limit` | int | 1000 | Global rate limit |
| `per_user_limit` | int | 100 | Per-user rate limit |
| `window` | duration | 60s | Time window |
| `redis.enabled` | bool | `false` | Enable Redis backend |
| `redis.host` | string | `localhost:6379` | Redis address |

## Algorithm Comparison

| Algorithm | Burst Handling | Memory | Precision |
|-----------|---------------|--------|-----------|
| Token Bucket | Good | Low | Medium |
| Sliding Window | Medium | Medium | High |
| Leaky Bucket | Poor | Low | High |

## Best Practices

1. **Set reasonable defaults** - Start conservative and adjust
2. **Use Redis for clusters** - Ensures consistent limits across instances
3. **Monitor closely** - Watch for legitimate users hitting limits
4. **Communicate limits** - Document rate limits in API documentation
5. **Implement retry logic** - Clients should handle 429 responses gracefully