# Deployment Architecture

## Deployment Patterns

### 1. Single Node (Development)

```
┌─────────────────────────────────────────────────────────────┐
│                    Development Setup                         │
│                                                              │
│   ┌─────────────────────────────────────────────────────┐   │
│   │                 Docker Compose                       │   │
│   │                                                      │   │
│   │   ┌─────────────────────────────────────────────┐   │   │
│   │   │           API Cerberus                      │   │   │
│   │   │  ┌─────────┐  ┌─────────┐  ┌─────────┐     │   │   │
│   │   │  │  Proxy  │  │  Admin  │  │ Metrics │     │   │   │
│   │   │  │ :8080   │  │ :8081   │  │ :9090   │     │   │   │
│   │   │  └─────────┘  └─────────┘  └─────────┘     │   │   │
│   │   │                                              │   │   │
│   │   │  Storage: BadgerDB (embedded)               │   │   │
│   │   └─────────────────────────────────────────────┘   │   │
│   │                                                      │   │
│   │   Optional:                                         │   │
│   │   ┌─────────┐  ┌─────────┐  ┌─────────┐          │   │
│   │   │  Redis  │  │Prometheus│  │ Grafana │          │   │
│   │   │ (Cache) │  │(Metrics) │  │(Dashboard│         │   │
│   │   └─────────┘  └─────────┘  └─────────┘          │   │
│   │                                                      │   │
│   └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

**Use Case:** Local development, testing, small deployments

**Docker Compose:**
```yaml
version: '3.8'
services:
  apicerebrus:
    image: ghcr.io/apicerberus/apicerebrus:latest
    ports:
      - "8080:8080"
      - "8081:8081"
    volumes:
      - ./config.yaml:/etc/apicerebrus/config.yaml
      - data:/var/lib/apicerebrus
    environment:
      - APICEREBRUS_NODE_ID=node1
      - APICEREBRUS_DATA_DIR=/var/lib/apicerebrus
```

### 2. Cluster Mode (Production)

```
┌─────────────────────────────────────────────────────────────┐
│                    Production Cluster                        │
│                                                              │
│   ┌─────────────────────────────────────────────────────┐   │
│   │                 Load Balancer                        │   │
│   │              (Layer 4 / Layer 7)                     │   │
│   │         ┌───────────┬───────────┐                   │   │
│   │         ▼           ▼           ▼                   │   │
│   └─────────────────────────────────────────────────────┘   │
│                           │                                  │
│           ┌───────────────┼───────────────┐                 │
│           ▼               ▼               ▼                 │
│   ┌─────────────┐   ┌─────────────┐   ┌─────────────┐      │
│   │  Gateway 1  │◄─►│  Gateway 2  │◄─►│  Gateway 3  │      │
│   │   (Leader)  │   │ (Follower)  │   │ (Follower)  │      │
│   │  ┌───────┐  │   │  ┌───────┐  │   │  ┌───────┐  │      │
│   │  │ Proxy │  │   │  │ Proxy │  │   │  │ Proxy │  │      │
│   │  │ Admin │  │   │  │ Admin │  │   │  │ Admin │  │      │
│   │  └───────┘  │   │  └───────┘  │   │  └───────┘  │      │
│   │     │       │   │     │       │   │     │       │      │
│   │  Raft │      │   │  Raft │      │   │  Raft │      │      │
│   │  Conn │      │   │  Conn │      │   │  Conn │      │      │
│   └───────┼──────┘   └───────┼──────┘   └───────┼──────┘      │
│           └──────────────────┴──────────────────┘             │
│                                                              │
│   ┌─────────────────────────────────────────────────────┐   │
│   │              External Services                       │   │
│   │                                                      │   │
│   │   ┌─────────┐  ┌─────────┐  ┌─────────┐           │   │
│   │   │  Redis  │  │   etcd  │  │Prometheus│          │   │
│   │   │ (Cache) │  │ (Config)│  │(Metrics) │          │   │
│   │   └─────────┘  └─────────┘  └─────────┘           │   │
│   │                                                      │   │
│   └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

**Use Case:** High availability, horizontal scaling

**Requirements:**
- Minimum 3 nodes for Raft consensus
- Odd number of nodes recommended
- Shared storage not required (each node has local BadgerDB)
- External Redis for caching (optional but recommended)
- External etcd for centralized config (optional)

### 3. Kubernetes Deployment

```
┌─────────────────────────────────────────────────────────────┐
│                  Kubernetes Architecture                     │
│                                                              │
│   ┌─────────────────────────────────────────────────────┐   │
│   │                  Ingress                             │   │
│   │            (nginx-ingress / traefik)                 │   │
│   └─────────────────────┬───────────────────────────────┘   │
│                         │                                    │
│   ┌─────────────────────┼───────────────────────────────┐   │
│   │              Service (apicerebrus)                   │   │
│   │              Type: LoadBalancer                      │   │
│   └─────────────────────┬───────────────────────────────┘   │
│                         │                                    │
│           ┌─────────────┼─────────────┐                     │
│           ▼             ▼             ▼                     │
│   ┌─────────────┐ ┌─────────────┐ ┌─────────────┐          │
│   │   Pod 0     │ │   Pod 1     │ │   Pod 2     │          │
│   │ (StatefulSet│ │ (StatefulSet│ │ (StatefulSet│          │
│   │   Leader)   │ │ Follower)   │ │ Follower)   │          │
│   │             │ │             │ │             │          │
│   │ ┌─────────┐ │ │ ┌─────────┐ │ │ ┌─────────┐ │          │
│   │ │ Proxy   │ │ │ │ Proxy   │ │ │ │ Proxy   │ │          │
│   │ │ Admin   │ │ │ │ Admin   │ │ │ │ Admin   │ │          │
│   │ │ Metrics │ │ │ │ Metrics │ │ │ │ Metrics │ │          │
│   │ └─────────┘ │ │ └─────────┘ │ │ └─────────┘ │          │
│   │      │      │ │      │      │ │      │      │          │
│   │ ┌────┴────┐ │ │ ┌────┴────┐ │ │ ┌────┴────┐ │          │
│   │ │ PVC     │ │ │ │ PVC     │ │ │ │ PVC     │ │          │
│   │ │(Badger) │ │ │ │(Badger) │ │ │ │(Badger) │ │          │
│   │ └─────────┘ │ │ └─────────┘ │ │ └─────────┘ │          │
│   └─────────────┘ └─────────────┘ └─────────────┘          │
│                                                              │
│   Network Policies:                                          │
│   - Port 7946: Raft (intra-cluster)                          │
│   - Port 7947: Service Discovery                             │
│   - Port 8080: HTTP/WS Proxy                                 │
│   - Port 8443: HTTPS Gateway                                 │
│   - Port 8081: Admin API                                     │
└─────────────────────────────────────────────────────────────┘
```

**Helm Chart Structure:**
```
apicerebrus/
├── Chart.yaml
├── values.yaml
├── values-production.yaml
└── templates/
    ├── deployment.yaml
    ├── statefulset.yaml
    ├── configmap.yaml
    ├── secret.yaml
    ├── service.yaml
    ├── ingress.yaml
    ├── hpa.yaml           # Horizontal Pod Autoscaler
    ├── pdb.yaml           # Pod Disruption Budget
    ├── serviceaccount.yaml
    └── networkpolicy.yaml
```

### 4. Multi-Region Deployment

```
┌─────────────────────────────────────────────────────────────┐
│                   Multi-Region Topology                      │
│                                                              │
│   ┌─────────────────────────────────────────────────────┐   │
│   │                  Global DNS                         │   │
│   │           (Geo-based routing)                       │   │
│   └─────────────────────┬───────────────────────────────┘   │
│           ┌─────────────┼─────────────┐                     │
│           ▼             ▼             ▼                     │
│   ┌─────────────┐ ┌─────────────┐ ┌─────────────┐          │
│   │  Region US  │ │  Region EU  │ │  Region AP  │          │
│   │             │ │             │ │             │          │
│   │ ┌─────────┐ │ │ ┌─────────┐ │ │ ┌─────────┐ │          │
│   │ │Cluster A│ │ │ │Cluster B│ │ │ │Cluster C│ │          │
│   │ │ 3 nodes │ │ │ │ 3 nodes │ │ │ │ 3 nodes │ │          │
│   │ └────┬────┘ │ │ └────┬────┘ │ │ └────┬────┘ │          │
│   │      │      │ │      │      │ │      │      │          │
│   │ ┌────┴────┐ │ │ ┌────┴────┐ │ │ ┌────┴────┐ │          │
│   │ │ Services│ │ │ │ Services│ │ │ │ Services│ │          │
│   │ │ (Local) │ │ │ │ (Local) │ │ │ │ (Local) │ │          │
│   │ └─────────┘ │ │ └─────────┘ │ │ └─────────┘ │          │
│   └─────────────┘ └─────────────┘ └─────────────┘          │
│                                                              │
│   Federation:                                                │
│   - Cross-cluster GraphQL federation                         │
│   - Global rate limiting (via Redis Cluster)                 │
│   - Centralized metrics aggregation                          │
└─────────────────────────────────────────────────────────────┘
```

## Port Reference

| Port | Protocol | Description | External Access |
|------|----------|-------------|-----------------|
| 8080 | TCP | HTTP/WS Proxy | Yes |
| 8443 | TCP | HTTPS Gateway | Yes |
| 8081 | TCP | Admin API | Restricted |
| 9090 | TCP | Prometheus Metrics | Internal |
| 7946 | TCP | Raft Consensus | Internal only |
| 7947 | TCP | Service Discovery | Internal only |

## Resource Requirements

### Minimum (Development)
- CPU: 0.5 cores
- Memory: 512MB
- Storage: 10GB

### Recommended (Production per node)
- CPU: 2 cores
- Memory: 4GB
- Storage: 50GB SSD
- Network: 1Gbps

### High Traffic (Production per node)
- CPU: 4+ cores
- Memory: 8GB+
- Storage: 100GB NVMe SSD
- Network: 10Gbps

## Security Considerations

### Network Segmentation
```
┌─────────────────────────────────────────────────────────────┐
│                   Network Zones                              │
│                                                              │
│   ┌─────────────────────────────────────────────────────┐   │
│   │   DMZ (Public)                                      │   │
│   │   - Load Balancer                                   │   │
│   │   - WAF/CDN                                         │   │
│   └─────────────────────────┬───────────────────────────┘   │
│                             │                                │
│   ┌─────────────────────────┼───────────────────────────┐   │
│   │   Application Zone                                    │   │
│   │   - API Cerberus nodes                                │   │
│   │   - Internal load balancer                            │   │
│   └─────────────────────────┬───────────────────────────┘   │
│                             │                                │
│   ┌─────────────────────────┼───────────────────────────┐   │
│   │   Data Zone                                           │   │
│   │   - Redis cluster                                     │   │
│   │   - etcd cluster                                      │   │
│   │   - Backend services                                  │   │
│   └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### TLS Configuration
- **External**: Let's Encrypt or custom certificates
- **Internal**: mTLS between cluster nodes
- **Backend**: Optional TLS to backend services

## Backup and Recovery

### Backup Strategy
```
┌─────────────────────────────────────────────────────────────┐
│                   Backup Components                          │
│                                                              │
│   1. Configuration (etcd/BadgerDB)                          │
│      - Daily snapshots                                      │
│      - Retention: 30 days                                   │
│                                                              │
│   2. TLS Certificates                                        │
│      - Encrypted backup to S3/GCS                           │
│      - Automatic renewal                                    │
│                                                              │
│   3. Analytics Data                                          │
│      - Continuous replication                               │
│      - Time-series database (Prometheus remote storage)     │
└─────────────────────────────────────────────────────────────┘
```

### Disaster Recovery
- RPO: < 5 minutes (Raft log replication)
- RTO: < 2 minutes (automated failover)
- Cross-region replication for multi-region setups
