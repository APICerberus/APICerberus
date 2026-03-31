# System Design

## High-Level Architecture

API Cerberus follows a layered architecture with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────────────┐
│                         API Layer                               │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌───────────┐ │
│  │  Admin API  │ │  Proxy API  │ │  Health API │ │  Metrics  │ │
│  │   (REST)    │ │  (HTTP/gRPC)│ │   (HTTP)    │ │Prometheus │ │
│  └─────────────┘ └─────────────┘ └─────────────┘ └───────────┘ │
├─────────────────────────────────────────────────────────────────┤
│                      Control Plane                              │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌───────────┐ │
│  │   Config    │ │   Router    │ │ Middleware  │ │   Cache   │ │
│  │   Manager   │ │   Engine    │ │   Chain     │ │  Manager  │ │
│  └─────────────┘ └─────────────┘ └─────────────┘ └───────────┘ │
├─────────────────────────────────────────────────────────────────┤
│                       Data Plane                                │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌───────────┐ │
│  │   HTTP      │ │   gRPC      │ │  WebSocket  │ │ GraphQL   │ │
│  │  Handler    │ │   Proxy     │ │   Handler   │ │ Federation│ │
│  └─────────────┘ └─────────────┘ └─────────────┘ └───────────┘ │
├─────────────────────────────────────────────────────────────────┤
│                      Cluster Layer                              │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌───────────┐ │
│  │    Raft     │ │   Service   │ │   Load      │ │   Sync    │ │
│  │  Consensus  │ │  Discovery  │ │  Balancer   │ │  Manager  │ │
│  └─────────────┘ └─────────────┘ └─────────────┘ └───────────┘ │
├─────────────────────────────────────────────────────────────────┤
│                      Storage Layer                              │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌───────────┐ │
│  │   BadgerDB  │ │    Redis    │ │    etcd     │ │   File    │ │
│  │   (Local)   │ │   (Cache)   │ │ (External)  │ │   Store   │ │
│  └─────────────┘ └─────────────┘ └─────────────┘ └───────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

## Request Flow

### HTTP Request Lifecycle

```
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│  Client  │────►│  L4/L7   │────►│  Router  │────►│ Middleware│
│          │     │  Load    │     │ Matching │     │  Chain   │
└──────────┘     │  Balancer│     └──────────┘     └────┬─────┘
                 └──────────┘                          │
                                                       ▼
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│  Client  │◄────│  Encode  │◄────│  Backend │◄────│  Transform│
│          │     │ Response │     │ Service  │     │ Request  │
└──────────┘     └──────────┘     └──────────┘     └──────────┘
```

### WebSocket Connection Flow

```
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│  Client  │────►│  WS      │────►│  Auth    │────►│  Upgrade │
│          │     │  Handshake│    │  Check   │     │ Connection│
└──────────┘     └──────────┘     └──────────┘     └────┬─────┘
                                                        │
                        ┌────────────────────────────────┘
                        ▼
                 ┌──────────────┐
                 │ Message      │◄──────────────────────────────┐
                 │ Router       │                               │
                 └──────┬───────┘                               │
                        │                                       │
                        ▼                                       │
                 ┌──────────────┐     ┌──────────┐              │
                 │  Backend WS  │◄───►│ Backend  │──────────────┘
                 │  Connection  │     │ Service  │
                 └──────────────┘     └──────────┘
```

## Component Interactions

### Configuration Propagation

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Admin     │────►│   Config    │────►│   Raft      │
│   API       │     │   Manager   │     │   Log       │
└─────────────┘     └──────┬──────┘     └──────┬──────┘
                           │                    │
                           ▼                    ▼
                    ┌─────────────┐     ┌─────────────┐
                    │   Store     │     │   Apply     │
                    │   (Badger)  │     │   to State  │
                    └─────────────┘     └──────┬──────┘
                                               │
                    ┌──────────────────────────┼──────────┐
                    ▼                          ▼          ▼
             ┌─────────────┐           ┌─────────────┐ ┌─────────────┐
             │   Router    │           │   Cache     │ │   Rate      │
             │   Update    │           │   Warming   │ │   Limiter   │
             └─────────────┘           └─────────────┘ └─────────────┘
```

### Cluster Communication

```
┌─────────────────────────────────────────────────────────────┐
│                      Cluster Topology                        │
│                                                              │
│   ┌──────────┐              ┌──────────┐                     │
│   │  Node 1  │──────────────│  Node 2  │                     │
│   │ (Leader) │◄────────────►│(Follower)│                     │
│   └────┬─────┘   Raft RPC   └────┬─────┘                     │
│        │          │              │                           │
│        │    ┌─────┴─────┐        │                           │
│        │    │           │        │                           │
│        └────►  Node 3   ◄────────┘                           │
│             │(Follower)  │                                    │
│             └───────────┘                                    │
│                                                              │
│   ◄───── Raft Consensus (Port 7946) ─────►                   │
│   ◄───── Service Discovery (Port 7947) ──►                   │
│   ◄───── HTTP Proxy (Port 8080) ─────────►                   │
│   ◄───── HTTPS Gateway (Port 8443) ──────►                   │
│   ◄───── Admin API (Port 8081) ──────────►                   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Data Storage

### Local Storage Schema (BadgerDB)

```
Key Prefix          │ Description                    │ Value Format
────────────────────┼────────────────────────────────┼─────────────────
svc/               │ Service definitions            │ JSON
route/             │ Route configurations           │ JSON
user/              │ User credentials               │ JSON (hashed)
cluster/nodes/     │ Cluster node metadata          │ JSON
cluster/raft/      │ Raft state                     │ Binary
config/global/     │ Global configuration           │ JSON
analytics/         │ Aggregated metrics             │ JSON
ssl/certs/         │ TLS certificates               │ PEM
```

### In-Memory Structures

```go
// Radix Tree for Route Matching
type Router struct {
    tree        *radix.Tree          // Path -> Route
    methods     map[string]*radix.Tree // HTTP method trees
    hosts       map[string]*radix.Tree // Host-based routing
    plugins     []Plugin              // Global plugins
}

// Connection Pool
type BackendPool struct {
    services    map[string]*Service
    connections chan *Connection
    healthChecks map[string]*HealthCheck
}

// Rate Limiter State
type RateLimiter struct {
    buckets     map[string]*TokenBucket
    counters    map[string]*SlidingWindow
    configs     map[string]*RateLimitConfig
}
```

## Scalability Patterns

### Horizontal Scaling

```
                    ┌──────────────┐
                    │   Global     │
                    │   LB/DNS     │
                    └──────┬───────┘
                           │
           ┌───────────────┼───────────────┐
           ▼               ▼               ▼
    ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
    │   Region 1   │ │   Region 2   │ │   Region 3   │
    │ ┌──────────┐ │ │ ┌──────────┐ │ │ ┌──────────┐ │
    │ │ Gateway  │ │ │ │ Gateway  │ │ │ │ Gateway  │ │
    │ │ Cluster  │ │ │ │ Cluster  │ │ │ │ Cluster  │ │
    │ └──────────┘ │ │ └──────────┘ │ │ └──────────┘ │
    └──────────────┘ └──────────────┘ └──────────────┘
```

### Caching Strategy

```
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│  Request │────►│  Cache   │────►│   Hit?   │─Yes►│  Return  │
│          │     │   Key    │     │          │     │  Cached  │
└──────────┘     └──────────┘     └────┬─────┘     └──────────┘
                                       │No
                                       ▼
                                ┌──────────┐
                                │ Forward  │
                                │ Request  │
                                └────┬─────┘
                                     │
                                     ▼
┌──────────┐     ┌──────────┐     ┌──────────┐
│  Update  │────►│  Store   │────►│ Response │
│  Cache   │     │  Cache   │     │  Client  │
└──────────┘     └──────────┘     └──────────┘
```

## Fault Tolerance

### Circuit Breaker Pattern

```
┌─────────────────────────────────────────────────────────────┐
│                    Circuit Breaker States                    │
│                                                              │
│   ┌─────────┐      Failure      ┌─────────┐                  │
│   │  CLOSED │◄──────────────────│  OPEN   │                  │
│   │ (Normal)│                   │ (Block) │                  │
│   └────┬────┘                   └────┬────┘                  │
│        │                            │                        │
│        │ Success                    │ Timeout                │
│        ▼                            ▼                        │
│   ┌─────────┐      Half-Open      ┌─────────┐                │
│   │  HALF   │◄───────────────────│  Retry  │                │
│   │  OPEN   │                     │  Count  │                │
│   └─────────┘                     └─────────┘                │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Health Check Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Health Check System                      │
│                                                              │
│   ┌───────────────┐                                         │
│   │ Health Monitor│                                         │
│   └───────┬───────┘                                         │
│           │                                                  │
│     ┌─────┴─────┬─────────────┬─────────────┐               │
│     ▼           ▼             ▼             ▼               │
│ ┌────────┐ ┌────────┐   ┌────────┐   ┌────────┐            │
│ │Active  │ │Passive │   │  TCP   │   │  HTTP  │            │
│ │Checks  │ │Checks  │   │ Checks │   │ Checks │            │
│ └────┬───┘ └────┬───┘   └───┬────┘   └───┬────┘            │
│      │          │           │            │                  │
│      ▼          ▼           ▼            ▼                  │
│ ┌──────────────────────────────────────────────────────┐   │
│ │              Backend Service Status                   │   │
│ │  [Healthy] ──► [Degraded] ──► [Unhealthy] ──► [Out]  │   │
│ └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```
