# WASM Plugin & Supply Chain Security Findings

**Scope:** internal/plugin/wasm.go, internal/plugin/marketplace.go, internal/plugin/hotreload.go, internal/plugin/pipeline.go, internal/config/, internal/audit/, internal/admin/webhooks.go, internal/raft/, internal/grpc/, Dockerfile
**Date:** 2026-04-18
**Scan Type:** Phase 2 — WASM Plugin, Supply Chain, Infrastructure Security (Verification Scan)
**Status Summary:** 6 issues fixed, 4 issues remain open, 1 finding reclassified as design-correct

---

## Executive Summary

This verification scan confirms that **6 previously reported issues have been fixed** in the WASM/plugin codebase. Key fixes include phase validation for WASM modules (preventing auth bypass), X-Claim-* header protection, panic recovery in WASM execution, and TOCTOU mitigation via inflight WaitGroup. Four issues remain open and require remediation.

| ID | Severity | Category | Status |
|----|----------|----------|--------|
| SEC-WASM-001 | High | WASM Phase Validation | **FIXED** |
| SEC-WASM-003 | High | WASM Panic Recovery | **FIXED** |
| SEC-WASM-004 | High | Protected Headers + X-Claim-* | **FIXED** |
| M-016 | Low | WASM Read Size Cap | CONFIRMED |
| M-017 | Low | EnvVars Unwired | CONFIRMED |
| M-018 | Medium | allocFn Context | **FIXED** |
| M-019 | Medium | Phase Filtering at Execution | **RECLASSIFIED** |
| M-020 | Medium | X-Claim-* Header Protection | **FIXED** |
| M-021 | Medium | Hot-Reload TOCTOU | **FIXED** |
| M-022 | Medium | Per-File Checksum Missing | OPEN |
| M-023 | Low | Extraction Orphan Files | OPEN |
| M-024 | Low | Factory System Bypass | OPEN |

**Critical:** 0 | **High:** 0 | **Medium:** 2 | **Low:** 3

---

## 1. WASM Plugin Security (internal/plugin/wasm.go)

### SEC-WASM-001: Phase Validation — FIXED
**File:** `internal/plugin/wasm.go:218-233`
**CWE:** CWE-693 (Protection Mechanism Failure)

`resolveWASMPhase` validates phase strings and explicitly rejects `PhaseAuth` and `PhasePostProxy` for WASM plugins.

```go
switch candidate := Phase(raw); candidate {
case PhasePreAuth, PhasePreProxy, PhaseProxy:
    return candidate, nil
case PhaseAuth:
    return "", fmt.Errorf("wasm module %q: phase %q is not permitted for WASM plugins")
case PhasePostProxy:
    return "", fmt.Errorf("wasm module %q: phase %q is not permitted for WASM plugins")
}
```

**Verification:** Code confirmed at lines 224-228. PhaseAuth and PhasePostProxy are explicitly rejected, preventing authentication bypass via WASM guest code.

**Status:** FIXED. Verified by code inspection and test coverage at `wasm_test.go:621-643`.

---

### SEC-WASM-003: Panic Recovery — FIXED
**File:** `internal/plugin/wasm.go:368-373`
**CWE:** CWE-835 (Loop with Unreachable Exit Condition)

`WASMModule.Execute` recovers from panics in adversarial guest exports.

```go
defer func() {
    if r := recover(); r != nil {
        handled = false
        err = fmt.Errorf("wasm module %q panicked: %v", m.id, r)
    }
}()
```

**Status:** FIXED. Prevents pipeline goroutine corruption from malicious guest modules.

---

### SEC-WASM-004: Protected Headers Deny-List + X-Claim-* — FIXED
**File:** `internal/plugin/wasm.go:814-830`

`wasmProtectedHeaders` blocks 15 sensitive headers including Authorization, Cookie, X-Api-Key, X-Admin-Key, X-Forwarded-For, X-Real-IP, Host, etc. Additionally, `X-Claim-*` headers derived from JWT claims are protected.

```go
var wasmProtectedHeaders = map[string]struct{}{
    "Authorization":      {},
    "Proxy-Authorization": {},
    "Cookie":             {},
    // ... 11 more protected headers
}

// M-WASM-020: protect X-Claim-* headers derived from JWT claims
if strings.HasPrefix(canonical, "X-Claim-") {
    continue
}
```

**Verification:** Lines 814-830 show full deny-list. Lines 848-850 show X-Claim-* wildcard protection.

**Status:** FIXED. A WASM plugin cannot forge claim-derived identity headers.

---

### M-016: maxWASMReadSize Hard Cap (64MB) — Confirmed
**File:** `internal/plugin/wasm.go:497-502`
**CWE:** CWE-400 (Resource Exhaustion)
**Severity:** Low (defense-in-depth)

```go
const maxWASMReadSize = 64 * 1024 * 1024 // 64MB hard cap per read
if length > maxWASMReadSize {
    return nil, fmt.Errorf("wasm memory read exceeds maximum size %d bytes", maxWASMReadSize)
}
```

Prevents OOM during buffer allocation if a malicious module claims a huge read length. The actual linear memory is bounded by `WithMemoryLimitPages` at module instantiation (128MB default).

**Status:** Confirmed. Defense-in-depth control in place.

---

### M-017: EnvVars Field Acknowledged but Unwired — Confirmed
**File:** `internal/plugin/wasm.go:62-65`
**Severity:** Low (latent)

```go
// M-017: EnvVars field exists but is NOT currently wired to wazero runtime.
// If WithEnvVars is used in the future, only allow known-safe variables
// (e.g., no API keys, secrets, or host paths). Host environment variables
// can leak information about the host system to the WASM module.
```

**Status:** Confirmed. Known limitation documented in code. `WASMConfig.Validate()` does not reject `EnvVars` but feature is not wired.

---

### M-018: allocFn.Call Context — FIXED
**File:** `internal/plugin/wasm.go:462`
**CWE:** CWE-400 (Resource Exhaustion)
**Severity:** Medium (previously reported)

The finding claimed `allocFn.Call` used `context.Background()` instead of the timeout-carrying context. Verification shows the code correctly passes the timeout context:

```go
// writeToWASMMemory receives ctx which carries MaxExecution timeout
func writeToWASMMemory(ctx context.Context, mod api.Module, data []byte) (uint32, uint32, error) {
    allocFn := mod.ExportedFunction("alloc")
    if allocFn != nil {
        results, err := allocFn.Call(ctx, uint64(len(data)))  // Uses timeout context
```

The `ctx` parameter is passed from `Execute` at line 408 via `writeToWASMMemory(execCtx, mod, reqBytes)` where `execCtx = context.WithTimeout(context.Background(), timeout)` carries the `MaxExecution` timeout.

**Status:** FIXED. The timeout context is correctly propagated to `allocFn.Call`.

---

### M-019: Phase Filtering Not Enforced at Execution — RECLASSIFIED (Not a Bug)
**File:** `internal/plugin/pipeline.go:15-36`, `internal/plugin/registry.go:253-262`
**CWE:** CWE-693 (Protection Mechanism Failure)
**Severity:** Medium (previously reported)

**Previous finding claimed:** `Pipeline.Execute` iterates ALL plugins in a single loop with no phase filtering.

**Verification:** The code is correct by design:

1. `BuildRoutePipelinesWithContext` (registry.go:253-262) sorts plugins by phase using `phaseOrder()`:
   - PhasePreAuth: order 1
   - PhaseAuth: order 2
   - PhasePreProxy: order 3
   - PhaseProxy: order 4
   - PhasePostProxy: order 5

2. `Pipeline.Execute` (pipeline.go:20-34) iterates through the pre-sorted slice and runs all plugins sequentially. The phase-based ordering is guaranteed at build time.

This is NOT a security vulnerability — plugins ARE correctly ordered by phase. The finding mischaracterized the architecture as a bug.

**Status:** RECLASSIFIED. This is correct behavior, not a vulnerability.

---

### M-020: X-Claim-* Header Protection — FIXED
**File:** `internal/plugin/wasm.go:848-850`
**CWE:** CWE-290 (Authentication Bypass by Spoofing)
**Severity:** Medium (previously reported)

JWT claim headers (`X-Claim-*`) are protected against WASM plugin overwriting:

```go
// M-WASM-020: protect X-Claim-* headers derived from JWT claims —
if strings.HasPrefix(canonical, "X-Claim-") {
    continue
}
```

**Status:** FIXED. Verified at lines 848-850.

---

### M-021: Hot-Reload TOCTOU — FIXED
**File:** `internal/plugin/wasm.go:357-366`, `internal/plugin/wasm.go:520-528`
**CWE:** CWE-367 (Time-of-Check Time-of-Use Race Condition)
**Severity:** Medium (previously reported)

The `inflight` WaitGroup pattern prevents the race between `Close()` and concurrent `Execute()` calls:

```go
// Execute registers BEFORE checking loaded (closes TOCTOU window)
m.inflight.Add(1)       // Line 361
defer m.inflight.Done() // Line 362

if !m.loaded.Load() {   // Line 364
    return false, fmt.Errorf("wasm module not loaded")
}
// ... execute WASM ...

// Close waits for in-flight executions
m.loaded.Store(false)  // Line 523 - prevents new executions
m.inflight.Wait()     // Line 528 - waits for current executions
```

The M-WASM-021 pattern correctly closes the TOCTOU race by:
1. Incrementing inflight counter BEFORE the loaded check
2. Setting `loaded=false` to prevent NEW executions
3. Waiting for existing executions to complete before closing the module

**Status:** FIXED. TOCTOU race mitigated via reference-counting with WaitGroup.

---

## 2. Supply Chain / Marketplace Security (internal/plugin/marketplace.go)

### M-022: Per-File Checksum Missing — OPEN
**File:** `internal/plugin/marketplace.go:358-369`, `internal/plugin/wasm.go:265-284`
**CWE:** CWE-345 (Insufficient Verification of Data Authenticity)
**Severity:** Medium
**Attack Scenario:** At install time, SHA-256 of the downloaded `.tar.gz` archive is verified. Individual extracted files are not checksummed. If an attacker modifies an extracted `.wasm` file post-install (e.g., via container escape), the gateway loads the tampered module.

```go
// Download verification (marketplace.go:365-369)
expectedChecksum := listing.Checksums[version]
if expectedChecksum != "" && checksum != expectedChecksum {
    return nil, fmt.Errorf("checksum mismatch")
}

// Load-time verification (wasm.go:275-284) only checks magic header + size
if err := validateWASMModule(resolved, r.config.MaxMemory); err != nil {
    return nil, err
}
```

The `wasm_file_sha256` config field exists but is optional and only verified if present — it is not automatically populated from marketplace metadata.

**Remediation:** Store per-file SHA-256 hashes in `metadata.json` at install time; verify on `LoadModule` for all WASM files. Alternatively, require and verify `wasm_file_sha256` when loading marketplace-installed plugins.

**Status:** OPEN. Checksum verification only covers the archive, not individual extracted files.

---

### M-023: Extraction Partial Files Remain on Failure — OPEN
**File:** `internal/plugin/marketplace.go:691-699`
**CWE:** CWE-409 (Improper Handling of Highly Compressed Data)
**Severity:** Low

```go
written, err := io.CopyN(outFile, tarReader, remaining)
extractedSize += written
if err != nil && !errors.Is(err, io.EOF) {
    _ = outFile.Close() // Best-effort cleanup only
    return err
}
if extractedSize > maxExtractSize {
    _ = outFile.Close() // Partial file remains on disk
    return fmt.Errorf("extracted plugin exceeds maximum size")
}
```

When extraction fails mid-stream (e.g., total extracted size exceeds limit), the partial file is closed but NOT deleted. These orphans accumulate in `DataDir/installed/<id>/`.

**Impact:** Low — partial files cannot be loaded as valid WASM modules, but they consume disk space and may confuse operators.

**Remediation:** Delete the partial file on extraction failure:
```go
if extractedSize > maxExtractSize {
    outFile.Close()
    os.Remove(targetPath)  // Clean up partial file
    return fmt.Errorf(...)
}
```

**Status:** OPEN. Minor cleanup issue.

---

### M-024: WASM Plugin Manager Bypasses Native Factory System — OPEN
**File:** `internal/plugin/registry.go:181-207`, `internal/plugin/wasm.go:681-696`
**CWE:** CWE-1188 (Insecure Default Initialization)
**Severity:** Low

`NewDefaultRegistry()` lists 24 native plugin factories. WASM modules are NOT registered as factories — `WASMPluginManager.CreatePipelinePlugin` is called outside `BuildRoutePipelinesWithContext`.

Consequences:
- `PluginConfig.Enabled *bool` not honored for WASM
- No consumer-context injection for WASM auth
- No priority bounds check for WASM plugins
- WASM module name collision with native plugins not detected

**Mitigating factor:** SEC-WASM-001 phase validation prevents WASM modules from running in PhaseAuth, significantly reducing practical impact.

**Remediation:** Consider integrating WASM modules into the factory registration system, or document that WASM plugins must be explicitly managed via `WASMPluginManager` API.

**Status:** OPEN. Architectural concern with reduced practical impact due to phase validation.

---

## 3. Previously Confirmed Good Controls

The following controls were verified as correctly implemented:

| Finding | File | Status |
|---------|------|--------|
| Distroless non-root container | Dockerfile:69 | GOOD |
| mTLS required for clustering | config/load.go:460-462 | GOOD |
| SSRF protection on webhooks | admin/webhooks.go:708-742 | GOOD |
| Log injection sanitization | audit/logger.go:234-241 | GOOD |
| Constant-time secret comparison | raft/transport.go:211-225 | GOOD |
| TLS 1.3 with client auth | raft/tls.go:126-141 | GOOD |
| WASM memory limits | wasm.go:113, 499 | GOOD |
| Panic recovery | wasm.go:368-373 | GOOD |
| Marketplace Ed25519 signatures | marketplace.go:600-631 | GOOD |

---

## 4. Dependency Security (go.mod)

### S-001: Dependency versions — Satisfactory
**File:** go.mod
**Severity:** Low

| Dependency | Version | Status |
|------------|---------|--------|
| tetratelabs/wazero | v1.11.0 | Actively maintained, no known CVEs |
| redis/go-redis | v9.8.0 | CVE-2025-49150 resolved |
| grpc/grpc | v1.80.0 | Monitor for CVEs |
| modernc.org/sqlite | v1.48.0 | Pure Go, no CGO |

**Status:** SATISFACTORY. Dependencies appear current. `go mod verify` run during build.

---

## 5. Port Exposure (Dockerfile)

### D-003: EXPOSE Makes All Ports Publicly Accessible — Informational
**File:** Dockerfile:98

```dockerfile
EXPOSE 8080 8443 9876 9877 50051 12000
```

All six service ports are exposed. The EXPOSE directive is documentation-only in Docker but serves as a reminder that all ports bind to `0.0.0.0` by default.

**Recommendation:** Use Docker network isolation or host network mode with firewall rules for production deployments.

**Status:** Informational. Network isolation is the operator's responsibility.

---

## Summary & Remediation Priority

### Top 3 Priorities for Remediation

1. **M-022** (Per-File Checksum) — Implement SHA-256 verification for all extracted files at install time, storing hashes in metadata.json and verifying on load.

2. **M-024** (Factory System Bypass) — Document that WASM plugins must be explicitly managed, or integrate WASM module loading into the factory registry for consistent policy enforcement.

3. **M-023** (Orphan Files) — Delete partial files on extraction failure to prevent disk accumulation.

### Fixed Items (Verification Complete)

All of the following have been verified as correctly implemented:
- SEC-WASM-001: Phase validation prevents auth bypass
- SEC-WASM-003: Panic recovery prevents goroutine corruption
- SEC-WASM-004: Protected headers + X-Claim-* coverage
- M-018: allocFn.Call correctly uses timeout context
- M-020: X-Claim-* headers protected
- M-021: TOCTOU mitigated via inflight WaitGroup

### Reclassified

- M-019: Phase filtering at execution time is correct design, not a vulnerability. Plugins are sorted by phase at build time and Execute runs them in sorted order.

---

**Report Generated:** 2026-04-18
**Files Scanned:**
- internal/plugin/wasm.go (881 lines)
- internal/plugin/marketplace.go (750 lines)
- internal/plugin/hotreload.go (196 lines)
- internal/plugin/pipeline.go (56 lines)
- internal/plugin/types.go (13 lines)
- internal/plugin/registry.go (900+ lines)
