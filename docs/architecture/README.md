# API Cerberus Architecture

This directory contains comprehensive architecture documentation for API Cerberus.

## Overview

API Cerberus is a high-performance, distributed API gateway built in Go. It provides a unified entry point for microservices with features like routing, authentication, rate limiting, caching, and observability.

## Architecture Principles

1. **High Performance** - Minimal latency overhead with efficient data structures
2. **Scalability** - Horizontally scalable with Raft consensus for cluster coordination
3. **Extensibility** - Plugin-based architecture for custom middleware
4. **Resilience** - Circuit breakers, health checks, and graceful degradation
5. **Observability** - Built-in metrics, tracing, and structured logging

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              API Cerberus Cluster                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │   Gateway    │  │   Gateway    │  │   Gateway    │  │   Gateway    │   │
│  │   Node 1     │◄─┤    Node 2    │◄─┤    Node 3    │◄─┤    Node N    │   │
│  │  (Leader)    │  │  (Follower)  │  │  (Follower)  │  │  (Follower)  │   │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘   │
│         │                 │                 │                 │            │
│         └─────────────────┴─────────────────┴─────────────────┘            │
│                              Raft Consensus                                 │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Backend Services                               │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐     │
│  │ Service A│  │ Service B│  │ Service C│  │ Service D│  │ Service E│     │
│  │ REST API │  │GraphQL   │  │  gRPC    │  │ WebSocket│  │  Custom  │     │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘  └──────────┘     │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Documentation Structure

- **[System Design](system-design.md)** - High-level system design and data flow
- **[Components](components.md)** - Detailed component architecture
- **[Deployment](deployment.md)** - Deployment patterns and topology
- **[Data Flow](data-flow.md)** - Request/response lifecycle
- **[Security](security.md)** - Security architecture and threat model

## Key Metrics

| Metric | Target |
|--------|--------|
| P99 Latency | < 1ms (no plugins) |
| Throughput | > 50,000 RPS per node |
| Availability | 99.99% |
| Configuration Propagation | < 100ms |

## Technology Stack

- **Language**: Go 1.26+
- **HTTP Router**: Custom radix tree implementation
- **Consensus**: Raft (hashicorp/raft)
- **Storage**: BadgerDB (local), etcd (optional external)
- **Metrics**: Prometheus
- **Tracing**: OpenTelemetry
- **Cache**: Redis (optional)
