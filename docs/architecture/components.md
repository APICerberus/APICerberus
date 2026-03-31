# Component Architecture

## Core Components

### 1. HTTP Router

The HTTP Router is built on a custom radix tree implementation for high-performance route matching.

```
┌─────────────────────────────────────────────────────────────┐
│                      Router Component                        │
│                                                              │
│   ┌─────────────────────────────────────────────────────┐   │
│   │              Radix Tree Structure                    │   │
│   │                                                      │   │
│   │                    ┌─────┐                          │   │
│   │                    │  /  │                          │   │
│   │                    └──┬──┘                          │   │
│   │           ┌───────────┼───────────┐                 │   │
│   │           ▼           ▼           ▼                 │   │
│   │        ┌────┐      ┌────┐      ┌────┐              │   │
│   │        │api │      │ws  │      │pub │              │   │
│   │        └──┬─┘      └────┘      └────┘              │   │
│   │           │                                         │   │
│   │     ┌─────┴─────┐                                   │   │
│   │     ▼           ▼                                   │   │
   │  ┌──────┐    ┌──────┐                                │   │
│   │  │users │    │orders│                                │   │
│   │  └──┬───┘    └──┬───┘                                │   │
│   │     │           │                                    │   │
│   │   ┌─┴─┐       ┌─┴─┐                                  │   │
│   │   ▼   ▼       ▼   ▼                                  │   │
│   │ ┌───┐┌───┐  ┌───┐┌───┐                               │   │
│   │ │:id││/me│  │:id││/all│                               │   │
│   │ └───┘└───┘  └───┘└───┘                               │   │
│   │                                                      │   │
│   └─────────────────────────────────────────────────────┘   │
│                                                              │
│   Match Order:                                               │
│   1. Host-based routing                                      │
│   2. Path matching (radix tree)                              │
│   3. Method matching                                         │
│   4. Header/query matching                                   │   │
└─────────────────────────────────────────────────────────────┘
```

**Key Features:**
- O(k) path matching where k is path length
- Support for path parameters (`:id`, `:name`)
- Wildcard matching (`*filepath`)
- Method-based sub-trees
- Priority-based route selection

### 2. Middleware Chain

```
┌─────────────────────────────────────────────────────────────┐
│                   Middleware Pipeline                        │
│                                                              │
│   Request ──► ┌─────────┐ ──► ┌─────────┐ ──► ┌─────────┐  │
│               │  Auth   │     │  Rate   │     │  Cache  │  │
│               │Middleware│    │ Limiter │     │  Check  │  │
│               └─────────┘     └─────────┘     └────┬────┘  │
│                                                    │       │
│   Response ◄── ┌─────────┐ ◄── ┌─────────┐ ◄──────┘       │
│                │  Log    │     │Transform│                 │
│                │Response │     │Response │                 │
│                └─────────┘     └─────────┘                 │
│                                                              │
│   ┌─────────────────────────────────────────────────────┐   │
│   │              Middleware Interface                    │   │
│   │                                                      │   │
│   │   type Middleware interface {                        │   │
│   │       Name() string                                  │   │
│   │       Execute(ctx *Context, next Handler) error      │   │
│   │       Priority() int                                 │   │
│   │   }                                                  │   │
│   │                                                      │   │
│   └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

**Built-in Middleware:**
| Middleware | Priority | Description |
|------------|----------|-------------|
| RequestID | 100 | Adds unique request identifier |
| Logger | 200 | Request/response logging |
| Auth | 300 | Authentication verification |
| RateLimiter | 400 | Rate limiting enforcement |
| Cache | 500 | Response caching |
| Transform | 600 | Request/response transformation |
| CircuitBreaker | 700 | Circuit breaker pattern |

### 3. Load Balancer

```
┌─────────────────────────────────────────────────────────────┐
│                     Load Balancer                            │
│                                                              │
│   ┌─────────────┐     ┌─────────────────────────────────┐   │
│   │   Request   │────►│         Algorithm Selector       │   │
│   └─────────────┘     └───────────────┬─────────────────┘   │
│                                       │                      │
│              ┌────────────────────────┼────────────────┐    │
│              ▼                        ▼                ▼    │
│        ┌─────────┐              ┌─────────┐      ┌────────┐ │
│        │ Round   │              │ Least   │      │ Weighted│ │
│        │ Robin   │              │ Conn    │      │ Round   │ │
│        └────┬────┘              └────┬────┘      │ Robin   │ │
│             │                        │          └────┬───┘ │
│             └────────────────────────┴───────────────┘      │
│                                       │                      │
│                                       ▼                      │
│                              ┌─────────────────┐            │
│                              │  Health Filter  │            │
│                              └────────┬────────┘            │
│                                       │                      │
│              ┌────────────────────────┼────────────────┐    │
│              ▼                        ▼                ▼    │
│        ┌──────────┐           ┌──────────┐      ┌─────────┐ │
│        │ Backend 1│           │ Backend 2│      │Backend 3│ │
│        │ Healthy  │           │ Healthy  │      │ Healthy │ │
│        └──────────┘           └──────────┘      └─────────┘ │
└─────────────────────────────────────────────────────────────┘
```

**Algorithms:**
- **Round Robin** - Sequential distribution
- **Weighted Round Robin** - Based on backend weights
- **Least Connections** - Routes to backend with fewest active connections
- **IP Hash** - Consistent hashing by client IP
- **Latency-based** - Routes to lowest latency backend

### 4. Authentication System

```
┌─────────────────────────────────────────────────────────────┐
│                   Authentication Flow                        │
│                                                              │
│   ┌─────────────────────────────────────────────────────┐   │
│   │              Auth Middleware                         │   │
│   │                                                      │   │
│   │   ┌─────────┐    ┌─────────┐    ┌─────────┐        │   │
│   │   │ Extract │───►│ Validate│───►│ Enrich  │        │   │
│   │   │ Token   │    │ Token   │    │ Context │        │   │
│   │   └─────────┘    └────┬────┘    └─────────┘        │   │
│   │                       │                            │   │
│   │              ┌────────┴────────┐                   │   │
│   │              ▼                 ▼                   │   │
│   │        ┌─────────┐      ┌──────────┐              │   │
│   │        │  JWT    │      │ API Key  │              │   │
│   │        │ Verify  │      │ Verify   │              │   │
│   │        └─────────┘      └──────────┘              │   │
│   │                                                      │   │
│   └─────────────────────────────────────────────────────┘   │
│                                                              │
│   Token Sources:                                             │
│   - Authorization header (Bearer)                            │
│   - X-API-Key header                                         │
│   - Cookie                                                   │
│   - Query parameter                                          │
└─────────────────────────────────────────────────────────────┘
```

### 5. Rate Limiter

```
┌─────────────────────────────────────────────────────────────┐
│                    Rate Limiter                              │
│                                                              │
│   ┌─────────────────────────────────────────────────────┐   │
│   │           Rate Limit Strategies                      │   │
│   │                                                      │   │
│   │   Token Bucket          Sliding Window              │   │
│   │   ┌─────────┐           ┌───────────────┐          │   │
│   │   │  ○○○○○  │           │ ┌─┬─┬─┬─┬─┐   │          │   │
│   │   │  ○○○○○  │           │ │█│█│░│░│░│   │          │   │
│   │   │  ○○○○○  │           │ └─┴─┴─┴─┴─┘   │          │   │
│   │   │  Bucket │           │  t-4..t       │          │   │
│   │   └─────────┘           └───────────────┘          │   │
│   │                                                      │   │
│   │   Fixed Window          Leaky Bucket                │   │
│   │   ┌───────────┐         ┌─────────┐                 │   │
│   │   │ ████████░ │         │  ═══╦═  │                 │   │
│   │   │  Window 1 │         │   Queue │                 │   │
│   │   │ ████████░ │         │  ═════  │                 │   │
│   │   │  Window 2 │         └─────────┘                 │   │
│   │   └───────────┘                                     │   │
│   │                                                      │   │
│   └─────────────────────────────────────────────────────┘   │
│                                                              │
│   Scope Levels:                                              │
│   - Global (all requests)                                    │
│   - Per-route                                                │
│   - Per-service                                              │
│   - Per-user/API key                                         │
│   - Per-IP address                                           │
└─────────────────────────────────────────────────────────────┘
```

### 6. WebSocket Handler

```
┌─────────────────────────────────────────────────────────────┐
│                   WebSocket Manager                          │
│                                                              │
│   ┌─────────────────────────────────────────────────────┐   │
│   │              Connection Lifecycle                    │   │
│   │                                                      │   │
│   │   Client          Gateway           Backend         │   │
│   │     │               │                 │             │   │
│   │     │── Upgrade ───►│                 │             │   │
│   │     │◄── 101 ───────┤                 │             │   │
│   │     │               │                 │             │   │
│   │     │◄── Message ──►│◄── Forward ───►│             │   │
│   │     │               │◄── Response ───┤             │   │
│   │     │◄── Response ──┤                 │             │   │
│   │     │               │                 │             │   │
│   │     │── Close ─────►│◄── Propagate ─►│             │   │
│   │     │               │                 │             │   │
│   │                                                      │   │
│   └─────────────────────────────────────────────────────┘   │
│                                                              │
│   Features:                                                  │
│   - Bidirectional message routing                            │
│   - Connection pooling                                       │
│   - Message buffering                                        │
│   - Heartbeat/ping-pong                                      │
│   - Graceful close handling                                  │
└─────────────────────────────────────────────────────────────┘
```

### 7. gRPC Proxy

```
┌─────────────────────────────────────────────────────────────┐
│                      gRPC Proxy                              │
│                                                              │
│   ┌─────────────────────────────────────────────────────┐   │
│   │           gRPC Request Flow                          │   │
│   │                                                      │   │
│   │   ┌──────────┐    ┌──────────┐    ┌──────────┐     │   │
│   │   │  HTTP/2  │───►│  gRPC    │───►│ Service  │     │   │
│   │   │  Request │    │ Handler  │    │ Registry │     │   │
│   │   └──────────┘    └────┬─────┘    └────┬─────┘     │   │
│   │                        │               │            │   │
│   │                   ┌────┴────┐     ┌────┴────┐       │   │
│   │                   ▼         ▼     ▼         ▼       │   │
│   │              ┌────────┐ ┌────────┐ ┌────────┐      │   │
│   │              │Unary   │ │Stream  │ │Reflection│    │   │
│   │              │RPC     │ │RPC     │ │Service   │    │   │
│   │              └────────┘ └────────┘ └────────┘      │   │
│   │                                                      │   │
│   └─────────────────────────────────────────────────────┘   │
│                                                              │
│   Protocol Translation:                                      │
│   - HTTP/1.1 ↔ HTTP/2                                        │
│   - REST ↔ gRPC (via grpc-gateway)                          │
│   - WebSocket ↔ gRPC streaming                               │
└─────────────────────────────────────────────────────────────┘
```

### 8. Cluster Manager (Raft)

```
┌─────────────────────────────────────────────────────────────┐
│                   Raft Consensus                             │
│                                                              │
│   ┌─────────────────────────────────────────────────────┐   │
│   │              Node States                             │   │
│   │                                                      │   │
│   │        ┌─────────────────────────────────────┐      │   │
│   │        ▼                                     │      │   │
│   │   ┌─────────┐      Timeout         ┌─────────┐     │   │
│   │   │Follower │ ───────────────────► │Candidate│     │   │
│   │   └────┬────┘                      └────┬────┘     │   │
│   │        ▲                                │          │   │
│   │        │         Majority Vote          │          │   │
│   │        └────────────────────────────────┘          │   │
│   │        │                                          │   │
│   │        │ Heartbeat/AppendEntries                   │   │
│   │        ▼                                          │   │
│   │   ┌─────────┐                                     │   │
│   │   │ Leader  │                                     │   │
│   │   └─────────┘                                     │   │
│   │                                                      │   │
│   └─────────────────────────────────────────────────────┘   │
│                                                              │
│   Log Replication:                                           │
│   1. Client submits command                                  │
│   2. Leader appends to log                                   │
│   3. Leader broadcasts AppendEntries                         │
│   4. Followers append to log                                 │
│   5. Leader commits on majority                              │
│   6. Leader applies to state machine                         │
│   7. Leader responds to client                               │
└─────────────────────────────────────────────────────────────┘
```

### 9. Configuration Manager

```
┌─────────────────────────────────────────────────────────────┐
│                 Configuration Manager                        │
│                                                              │
│   ┌─────────────────────────────────────────────────────┐   │
│   │              Config Sources                          │   │
│   │                                                      │   │
│   │   ┌──────────┐   ┌──────────┐   ┌──────────┐       │   │
│   │   │   File   │   │   API    │   │  Consul  │       │   │
│   │   │  (YAML)  │   │  (REST)  │   │   (KV)   │       │   │
│   │   └────┬─────┘   └────┬─────┘   └────┬─────┘       │   │
│   │        └──────────────┼──────────────┘              │   │
│   │                       ▼                            │   │
│   │              ┌─────────────────┐                   │   │
│   │              │  Config Manager │                   │   │
│   │              │   (Hot Reload)  │                   │   │
│   │              └────────┬────────┘                   │   │
│   │                       │                            │   │
│   │         ┌─────────────┼─────────────┐              │   │
│   │         ▼             ▼             ▼              │   │
│   │    ┌─────────┐   ┌─────────┐   ┌─────────┐        │   │
│   │    │ Validate│   │  Apply  │   │  Notify │        │   │
│   │    │  Config │   │  Config │   │ Listeners│       │   │
│   │    └─────────┘   └─────────┘   └─────────┘        │   │
│   │                                                      │   │
│   └─────────────────────────────────────────────────────┘   │
│                                                              │
│   Hot Reload:                                                │
│   - File watcher for YAML configs                            │
│   - Event-driven updates from API                            │
│   - Raft-synchronized cluster state                          │
│   - Graceful transition (no dropped connections)             │
└─────────────────────────────────────────────────────────────┘
```

### 10. Metrics & Observability

```
┌─────────────────────────────────────────────────────────────┐
│                  Observability Stack                         │
│                                                              │
│   ┌─────────────────────────────────────────────────────┐   │
│   │              Metrics Collection                      │   │
│   │                                                      │   │
│   │   Request Metrics          System Metrics           │   │
│   │   ───────────────          ──────────────           │   │
│   │   • Total requests         • CPU usage              │   │
│   │   • Request latency        • Memory usage           │   │
│   │   • Response codes         • Goroutines             │   │
│   │   • Error rate             • GC pauses              │   │
│   │   • Throughput             • Open connections       │   │
│   │                                                      │   │
│   └─────────────────────────────────────────────────────┘   │
│                                                              │
│   ┌─────────────────────────────────────────────────────┐   │
│   │              Tracing                                 │   │
│   │                                                      │   │
│   │   [Client]──►[Gateway]──►[Auth]──►[Cache]──►[Backend]│  │
│   │      │          │         │        │         │      │   │
│   │      └──────────┴─────────┴────────┴─────────┘      │   │
│   │                 Distributed Trace                   │   │
│   │                                                      │   │
│   │   OpenTelemetry / W3C Trace Context                  │   │
│   └─────────────────────────────────────────────────────┘   │
│                                                              │
│   Exporters:                                                 │
│   - Prometheus (metrics endpoint)                            │
│   - OpenTelemetry Collector                                  │
│   - Jaeger / Zipkin (tracing)                                │
│   - Elasticsearch (logs)                                     │
└─────────────────────────────────────────────────────────────┘
```

## Component Relationships

```
┌─────────────────────────────────────────────────────────────┐
│              Component Dependency Graph                      │
│                                                              │
│                    ┌─────────────┐                          │
│                    │   Config    │                          │
│                    │   Manager   │                          │
│                    └──────┬──────┘                          │
│                           │                                  │
│              ┌────────────┼────────────┐                    │
│              ▼            ▼            ▼                    │
│        ┌─────────┐  ┌─────────┐  ┌─────────┐               │
│        │ Router  │  │  Auth   │  │  Rate   │               │
│        │         │  │ Manager │  │ Limiter │               │
│        └────┬────┘  └────┬────┘  └────┬────┘               │
│             │            │            │                      │
│             └────────────┼────────────┘                      │
│                          ▼                                  │
│                   ┌─────────────┐                           │
│                   │ Middleware  │                           │
│                   │   Chain     │                           │
│                   └──────┬──────┘                           │
│                          │                                   │
│        ┌─────────────────┼─────────────────┐                │
│        ▼                 ▼                 ▼                │
│   ┌─────────┐      ┌─────────┐      ┌─────────┐            │
│   │ HTTP    │      │ WebSocket│     │  gRPC   │            │
│   │ Handler │      │ Handler  │     │ Handler │            │
│   └────┬────┘      └────┬────┘     └────┬────┘            │
│        │                │                │                  │
│        └────────────────┼────────────────┘                  │
│                         ▼                                   │
│                  ┌─────────────┐                            │
│                  │   Backend   │                            │
│                  │   Pool      │                            │
│                  └─────────────┘                            │
│                                                              │
│   Supporting: Cluster Manager, Metrics, Health Monitor      │
└─────────────────────────────────────────────────────────────┘
```
