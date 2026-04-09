# Dependency Audit

**Date:** 2026-04-09
**Go Version:** 1.25.0 (installed: 1.26.1)

## Core Dependencies

| Dependency | Version | Risk | Assessment |
|------------|---------|------|------------|
| `modernc.org/sqlite` | — | LOW | Pure Go SQLite driver. Complex transitive dependency tree but actively maintained. No CGO required. |
| `github.com/gorilla/websocket` | — | LOW | Industry-standard WebSocket library. Well-maintained, battle-tested. |
| `github.com/graphql-go/graphql` | v0.8.1 | MEDIUM | Stale — last updated 2022. Consider pinning to specific commit hash or evaluating alternatives. |
| `golang.org/x/net/websocket` | — | MEDIUM | Deprecated package. Use `nhooyr.io/websocket` or `gorilla/websocket` instead. |
| `golang.org/x/crypto` | — | LOW | Standard crypto extensions. Well-maintained by Go team. |
| `golang.org/x/time` | — | LOW | Rate limiting utilities. Standard library extension. |
| `github.com/hashicorp/raft` | Referenced but not used | INFO | Code references hashicorp/raft in docs but uses custom Raft implementation. |

## High-Risk: Custom Implementations

### Custom JWT Library (`internal/pkg/jwt/`)
- **Risk:** MEDIUM
- **Details:** Homegrown JWT implementation supporting HS256 and RS256
- **Concerns:** Not independently audited; potential for subtle vulnerabilities in token validation, algorithm enforcement, or claims processing
- **Recommendation:** Replace with `github.com/golang-jwt/jwt/v5` which is actively maintained and widely audited

### Custom YAML Parser (`internal/pkg/yaml/`)
- **Risk:** HIGH
- **Details:** From-scratch YAML decoder with no external dependencies
- **Concerns:** No depth limits (billion laughs vulnerability), no node count limits (memory exhaustion), not fuzz-tested
- **Recommendation:** Add hard limits for depth (max 100) and node count (max 10,000), or replace with `gopkg.in/yaml.v3`

## Transitive Dependency Concerns

| Concern | Details |
|---------|---------|
| `modernc.org/sqlite` transitive burden | Pulls in ~15 transitive dependencies. If CGo is acceptable, `github.com/mattn/go-sqlite3` has simpler dependency tree. |
| GraphQL ecosystem staleness | `graphql-go/graphql` v0.8.1 is stale; related packages in the ecosystem may also lag security patches. |
| No `govulncheck` in CI | CI workflow runs `govulncheck` but uses `@latest` — not version-pinned, potentially causing nondeterministic scans. |

## Supply Chain Assessment

| Aspect | Status |
|--------|--------|
| Dependency pinning | go.sum present — versions pinned |
| Vendor directory | Absent — relies on module proxy |
| Replace directives | None detected |
| Known typosquatting | No indicators found |
| Obscure packages | `modernc.org/*` ecosystem is well-known in Go community |

## Recommendations

1. **HIGH:** Add depth and node limits to custom YAML parser
2. **MEDIUM:** Replace `golang.org/x/net/websocket` with modern alternative
3. **MEDIUM:** Pin `govulncheck` version in CI (e.g., `govulncheck@v1.1.0`)
4. **MEDIUM:** Add `go mod tidy` and `go mod verify` to CI pipeline
5. **LOW:** Pin `graphql-go/graphql` to specific commit hash
6. **LOW:** Consider `GONOSUMCHECK` and `GONOSUMDB` policies if using private modules
7. **INFO:** Update `go.mod` to `go 1.26` to leverage latest runtime security improvements
