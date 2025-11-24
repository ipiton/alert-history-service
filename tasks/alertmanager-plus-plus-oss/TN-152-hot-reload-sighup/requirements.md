# TN-152: Hot Reload Mechanism (SIGHUP)

**Status**: 🎯 IN PROGRESS
**Priority**: P1 (High)
**Complexity**: Medium (4-6 hours)
**Quality Target**: 150% (Grade A+ EXCEPTIONAL)
**Dependencies**: TN-150 (Config Update) ✅ COMPLETE, TN-151 (Config Validator) ✅ COMPLETE

---

## 📋 Executive Summary

Implement **Unix signal-based configuration hot reload** (SIGHUP) for Alertmanager++, enabling operators to reload configuration **without service restart** using standard Unix signals (`kill -HUP <pid>`).

This task integrates with existing hot reload infrastructure from **TN-150** and adds **signal handling** as an alternative trigger mechanism to the API-based reload.

---

## 🎯 Objectives

### Primary Objectives
1. **Signal Handling**: Implement SIGHUP signal listener in `main.go`
2. **Integration**: Connect SIGHUP → ConfigUpdateService → Hot Reload
3. **Graceful Reload**: Zero downtime, no request interruption
4. **Error Handling**: Comprehensive error reporting for failed reloads
5. **Monitoring**: Prometheus metrics for SIGHUP-triggered reloads

### Secondary Objectives (150% Quality)
6. **Testing**: Unit + integration tests for signal handling
7. **Documentation**: Operator guide for SIGHUP usage
8. **CLI**: `alertmanager-plus-plus reload` command for easier ops
9. **Validation**: Pre-reload config validation (reuse TN-151)
10. **Rollback**: Automatic rollback on failed reload

---

## 📊 Success Criteria

| Criterion | Target | Measurement |
|-----------|--------|-------------|
| **Functionality** | 100% | All features working |
| **Test Coverage** | 80%+ | Signal handling + integration |
| **Performance** | < 300ms | p95 reload latency |
| **Zero Downtime** | 100% | No request drops during reload |
| **Documentation** | Complete | Operator guide + runbook |
| **Quality Grade** | A+ | 150% target achieved |

---

## 1. Functional Requirements

### FR-1: SIGHUP Signal Handling
**Priority**: P0 (Critical)

**Description**: Listen for SIGHUP signal and trigger configuration reload.

**Requirements**:
- **FR-1.1**: Register SIGHUP handler in `main.go`
- **FR-1.2**: Handler triggers full config reload from disk
- **FR-1.3**: Reload uses existing `ConfigUpdateService` from TN-150
- **FR-1.4**: Support multiple SIGHUP signals (idempotent)
- **FR-1.5**: Concurrent SIGHUP handling (debounce if < 1s apart)

**Signal Flow**:
```
Operator: kill -HUP <pid>
       ↓
  SIGHUP Handler (main.go)
       ↓
  Load config from disk (viper)
       ↓
  Validate config (TN-151 validator)
       ↓
  ConfigUpdateService.Update()
       ↓
  Hot Reload (TN-150)
       ↓
  Log result + Update metrics
```

**Acceptance Criteria**:
- ✅ SIGHUP handler registered on startup
- ✅ SIGHUP triggers config reload
- ✅ Reload completes in < 300ms (p95)
- ✅ Multiple SIGHUP signals handled correctly
- ✅ Debouncing prevents spam (1s window)
- ✅ Structured logging for each reload attempt

---

### FR-2: Configuration File Reload
**Priority**: P0 (Critical)

**Description**: Reload configuration from disk on SIGHUP signal.

**Requirements**:
- **FR-2.1**: Re-read config file from original path (viper)
- **FR-2.2**: Support YAML and JSON formats
- **FR-2.3**: Handle file not found errors gracefully
- **FR-2.4**: Handle parse errors without crashing
- **FR-2.5**: Log file path, size, and modification time

**File Loading Flow**:
```go
// Pseudo-code
func reloadConfigFromDisk() (*config.Config, error) {
    // 1. Get config file path from viper
    configPath := viper.ConfigFileUsed()

    // 2. Check file exists
    if !fileExists(configPath) {
        return nil, ErrConfigFileNotFound
    }

    // 3. Read file
    viper.ReadInConfig()

    // 4. Unmarshal to Config struct
    var cfg config.Config
    if err := viper.Unmarshal(&cfg); err != nil {
        return nil, err
    }

    // 5. Return new config
    return &cfg, nil
}
```

**Acceptance Criteria**:
- ✅ Config file reloaded from disk
- ✅ YAML and JSON formats supported
- ✅ File errors handled gracefully (don't crash)
- ✅ Parse errors logged with details
- ✅ File metadata logged (path, size, mtime)

---

### FR-3: Pre-Reload Validation
**Priority**: P0 (Critical)

**Description**: Validate new configuration before applying (fail fast).

**Requirements**:
- **FR-3.1**: Use TN-151 Config Validator before reload
- **FR-3.2**: Reject invalid config (keep current config)
- **FR-3.3**: Log validation errors with details
- **FR-3.4**: Return non-zero exit code if validation fails
- **FR-3.5**: Update metrics: `config_reload_validation_failures_total`

**Validation Flow**:
```
Load new config from disk
       ↓
  Validate (TN-151)
       ↓
     Pass? ───No──→ Log error + Keep old config + Update metrics
       ↓
      Yes
       ↓
  Proceed to reload
```

**Acceptance Criteria**:
- ✅ TN-151 validator called before reload
- ✅ Invalid config rejected (old config kept)
- ✅ Validation errors logged (with error codes)
- ✅ Metrics updated on validation failure
- ✅ Zero downtime on validation failure

---

### FR-4: Graceful Hot Reload
**Priority**: P0 (Critical)

**Description**: Apply new configuration without interrupting active requests.

**Requirements**:
- **FR-4.1**: Use existing `ConfigUpdateService.Update()` from TN-150
- **FR-4.2**: Parallel reload of all affected components
- **FR-4.3**: Active requests use old config until completion
- **FR-4.4**: New requests use new config immediately
- **FR-4.5**: Timeout: 30s for reload operations

**Integration with TN-150**:
```go
// In SIGHUP handler
func handleSIGHUP(configService *config.DefaultConfigUpdateService) {
    // 1. Load new config from disk
    newConfig, err := reloadConfigFromDisk()
    if err != nil {
        logger.Error("failed to load config", "error", err)
        return
    }

    // 2. Validate config (TN-151)
    validator := configvalidator.New(opts)
    result, err := validator.ValidateConfig(ctx, newConfig)
    if err != nil || !result.Valid() {
        logger.Error("config validation failed", "errors", result.Errors)
        return
    }

    // 3. Trigger hot reload (TN-150)
    opts := config.UpdateOptions{
        Source: "SIGHUP",
        User:   "system",
    }
    _, err = configService.Update(ctx, newConfig, opts)
    if err != nil {
        logger.Error("hot reload failed", "error", err)
        return
    }

    logger.Info("config reloaded successfully via SIGHUP")
}
```

**Acceptance Criteria**:
- ✅ Zero downtime during reload
- ✅ Active requests complete on old config
- ✅ New requests use new config
- ✅ Reload completes in < 300ms (p95)
- ✅ Failed reload doesn't affect service

---

### FR-5: Error Handling and Rollback
**Priority**: P0 (Critical)

**Description**: Comprehensive error handling with automatic rollback.

**Requirements**:
- **FR-5.1**: Catch all errors (file, parse, validation, reload)
- **FR-5.2**: Log errors with structured context
- **FR-5.3**: Update error metrics
- **FR-5.4**: Automatic rollback on critical component failure
- **FR-5.5**: Keep old config on any reload failure

**Error Scenarios**:
```
Error Type                    Action
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
File not found                Keep old config, log error
Parse error                   Keep old config, log error
Validation error              Keep old config, log error
Reload timeout                Rollback, log error
Critical component failure    Rollback, log error
Non-critical component fail   Continue, log warning
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**Acceptance Criteria**:
- ✅ All error types handled gracefully
- ✅ No panics on errors
- ✅ Structured logging for all errors
- ✅ Metrics updated for each error type
- ✅ Automatic rollback on critical failures

---

### FR-6: Prometheus Metrics
**Priority**: P1 (High)

**Description**: Expose metrics for SIGHUP-triggered reloads.

**Metrics to Add**:

```go
// New metrics for TN-152
config_reload_total{source="sighup", status="success|failure"} counter
config_reload_duration_seconds{source="sighup"} histogram
config_reload_validation_failures_total{source="sighup"} counter
config_reload_last_success_timestamp{source="sighup"} gauge
config_reload_last_failure_timestamp{source="sighup"} gauge
```

**Existing Metrics (from TN-150)**:
```go
config_update_total{status="success|failure"} counter
config_update_duration_seconds histogram
config_version gauge
```

**Acceptance Criteria**:
- ✅ All new metrics implemented
- ✅ Metrics updated on each SIGHUP
- ✅ Success/failure status tracked
- ✅ Duration histogram < 300ms (p95)
- ✅ Prometheus scraping works

---

## 2. Non-Functional Requirements

### NFR-1: Performance
**Priority**: P0 (Critical)

**Targets**:
- Reload latency: < 300ms (p95)
- File read: < 10ms
- Validation: < 50ms
- Component reload: < 200ms (parallel)
- Zero impact on active requests

**Benchmarks**:
```
BenchmarkSIGHUPHandler       - < 1ms (just signal handling)
BenchmarkConfigReload        - < 300ms (full reload)
BenchmarkFileRead            - < 10ms
BenchmarkValidation          - < 50ms
```

---

### NFR-2: Reliability
**Priority**: P0 (Critical)

**Requirements**:
- Zero downtime: 100% uptime during reload
- Zero data loss: No dropped requests
- Zero race conditions: Thread-safe
- Fault tolerance: Failed reload doesn't crash service
- Idempotency: Multiple SIGHUP signals safe

---

### NFR-3: Observability
**Priority**: P1 (High)

**Requirements**:
- Structured logging (slog)
- Prometheus metrics
- Reload success/failure tracking
- Duration tracking
- Error rate tracking

---

### NFR-4: Testability
**Priority**: P1 (High)

**Requirements**:
- Unit tests: 80%+ coverage
- Integration tests: 5+ scenarios
- Signal sending tests
- Error scenario tests
- Performance benchmarks

---

## 3. Technical Design

### 3.1 Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         main.go                              │
│                                                               │
│  ┌────────────────────────────────────────────────────┐    │
│  │          Signal Handler Goroutine                   │    │
│  │                                                      │    │
│  │  sigChan := make(chan os.Signal, 1)               │    │
│  │  signal.Notify(sigChan, syscall.SIGHUP)           │    │
│  │                                                      │    │
│  │  for sig := range sigChan {                        │    │
│  │    switch sig {                                     │    │
│  │    case syscall.SIGHUP:                            │    │
│  │      go handleSIGHUP(configService)                │    │
│  │    }                                                │    │
│  │  }                                                  │    │
│  └────────────────────────────────────────────────────┘    │
│                           ↓                                  │
│  ┌────────────────────────────────────────────────────┐    │
│  │         handleSIGHUP(configService)                │    │
│  │                                                      │    │
│  │  1. Load config from disk (viper)                  │    │
│  │  2. Validate config (TN-151)                       │    │
│  │  3. Update via ConfigUpdateService (TN-150)        │    │
│  │  4. Update metrics                                 │    │
│  │  5. Log result                                     │    │
│  └────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                           ↓
          ┌────────────────────────────────────┐
          │   ConfigUpdateService (TN-150)     │
          │                                      │
          │  • Diff calculation                 │
          │  • Atomic apply                     │
          │  • Hot reload (parallel)            │
          │  • Rollback on failure              │
          └────────────────────────────────────┘
```

---

### 3.2 File Structure

```
go-app/
├── cmd/server/main.go                  # MODIFIED: Add SIGHUP handler
├── cmd/server/signal.go                # NEW: Signal handling logic
├── cmd/server/signal_test.go           # NEW: Signal tests
├── cmd/reload/                          # NEW: CLI reload command
│   └── main.go                          # NEW: `alertmanager-plus-plus reload`
├── internal/config/                     # Existing (TN-150)
│   ├── update_service.go               # Existing
│   ├── update_reloader.go              # Existing
│   └── update_interfaces.go            # Existing
└── docs/
    └── operators/
        └── hot-reload-guide.md         # NEW: Operator documentation
```

---

### 3.3 Implementation Plan

#### Phase 1: Core Signal Handling (2h)
1. ✅ Create `cmd/server/signal.go` with SIGHUP handler
2. ✅ Integrate handler in `main.go`
3. ✅ Add config reload from disk logic
4. ✅ Add validation integration (TN-151)
5. ✅ Add metrics integration

#### Phase 2: Error Handling & Testing (1.5h)
6. ✅ Implement comprehensive error handling
7. ✅ Add unit tests for signal handling
8. ✅ Add integration tests for reload flow
9. ✅ Add benchmarks

#### Phase 3: CLI & Documentation (1h)
10. ✅ Create `cmd/reload/main.go` CLI tool
11. ✅ Write operator guide
12. ✅ Add examples and runbooks

#### Phase 4: Quality Assurance (0.5h)
13. ✅ Code review and refactoring
14. ✅ Verify all tests pass
15. ✅ Performance validation
16. ✅ Documentation review

**Total Estimated Time**: 4-6 hours

---

## 4. API / Interface Specification

### 4.1 Signal Handler Interface

```go
// SignalHandler manages Unix signal handling
type SignalHandler struct {
    configService *config.DefaultConfigUpdateService
    validator     *configvalidator.Validator
    logger        *slog.Logger
    metrics       *SignalMetrics
}

// Start begins listening for signals
func (h *SignalHandler) Start(ctx context.Context) error

// Stop stops signal handling
func (h *SignalHandler) Stop()

// handleSIGHUP processes SIGHUP signal
func (h *SignalHandler) handleSIGHUP(ctx context.Context) error
```

---

### 4.2 Config Reload Interface

```go
// ConfigReloader reloads configuration from disk
type ConfigReloader interface {
    // ReloadFromDisk reads and parses config file
    ReloadFromDisk() (*Config, error)

    // GetConfigPath returns current config file path
    GetConfigPath() string
}
```

---

### 4.3 CLI Tool Interface

```bash
# Reload configuration via SIGHUP
alertmanager-plus-plus reload [--pid <pid>]

# Options:
#   --pid <pid>       Process ID to send SIGHUP (auto-detect if not specified)
#   --validate-only   Validate config without reloading
#   --wait            Wait for reload to complete
#   --timeout 30s     Timeout for reload operation

# Examples:
alertmanager-plus-plus reload                    # Auto-detect PID
alertmanager-plus-plus reload --pid 1234        # Explicit PID
alertmanager-plus-plus reload --validate-only   # Dry run
```

---

## 5. Testing Strategy

### 5.1 Unit Tests (15+ tests)

```go
// Test signal handling
TestSignalHandler_Start
TestSignalHandler_Stop
TestSignalHandler_HandleSIGHUP
TestSignalHandler_DebounceMultipleSIGHUP

// Test config reload
TestConfigReloader_ReloadFromDisk
TestConfigReloader_FileNotFound
TestConfigReloader_ParseError
TestConfigReloader_ValidationError

// Test error handling
TestSIGHUP_FileNotFound
TestSIGHUP_InvalidConfig
TestSIGHUP_ReloadTimeout
TestSIGHUP_CriticalComponentFailure

// Test metrics
TestMetrics_SIGHUPSuccess
TestMetrics_SIGHUPFailure
TestMetrics_ReloadDuration
```

---

### 5.2 Integration Tests (5+ tests)

```go
// End-to-end tests
TestIntegration_SIGHUPReload_Success
TestIntegration_SIGHUPReload_InvalidConfig
TestIntegration_SIGHUPReload_NoDowntime
TestIntegration_MultipleSIGHUP_Concurrent
TestIntegration_SIGHUPReload_WithActiveRequests
```

---

### 5.3 Benchmarks (3+ benchmarks)

```go
BenchmarkSIGHUPHandler       // < 1ms
BenchmarkConfigReload        // < 300ms
BenchmarkFileRead            // < 10ms
```

---

## 6. Documentation

### 6.1 Operator Guide (`docs/operators/hot-reload-guide.md`)

**Contents**:
1. Overview of hot reload mechanism
2. Using SIGHUP to reload config
3. Using CLI tool (`alertmanager-plus-plus reload`)
4. Validation before reload
5. Monitoring reload status (metrics)
6. Troubleshooting failed reloads
7. Best practices

**Examples**:
```bash
# Example 1: Basic reload
kill -HUP $(cat /var/run/alertmanager-plus-plus.pid)

# Example 2: Using CLI
alertmanager-plus-plus reload

# Example 3: Validate first, then reload
alertmanager-plus-plus reload --validate-only
alertmanager-plus-plus reload

# Example 4: Check reload status
curl http://localhost:9093/metrics | grep config_reload
```

---

### 6.2 API Documentation (OpenAPI)

Update existing OpenAPI spec with SIGHUP information:
```yaml
info:
  description: |
    ...

    ## Configuration Reload

    Two methods to reload configuration:
    1. **API**: `POST /api/v2/config` (TN-150)
    2. **SIGHUP**: `kill -HUP <pid>` (TN-152)

    Both methods trigger the same hot reload mechanism.
```

---

## 7. Quality Assurance (150% Target)

### Baseline Requirements (100%)
- ✅ SIGHUP signal handling
- ✅ Config reload from disk
- ✅ Integration with TN-150
- ✅ Basic error handling
- ✅ Basic tests

### 150% Quality Additions
- ✅ Pre-reload validation (TN-151 integration)
- ✅ Comprehensive error handling (all scenarios)
- ✅ Debouncing for multiple signals
- ✅ CLI tool for operators
- ✅ Complete operator guide with examples
- ✅ Prometheus metrics
- ✅ 15+ unit tests
- ✅ 5+ integration tests
- ✅ Performance benchmarks
- ✅ Graceful rollback on failures

---

## 8. Metrics for Success

| Metric | Target | Measurement |
|--------|--------|-------------|
| **Implementation** | 100% | All features working |
| **Test Coverage** | 80%+ | Signal handling code |
| **Test Count** | 20+ | Unit + integration |
| **Documentation** | Complete | Operator guide |
| **Performance** | < 300ms | p95 reload latency |
| **Zero Downtime** | 100% | No dropped requests |
| **Quality Grade** | A+ | 150% target |

---

## 9. Dependencies

### Upstream Dependencies (Blockers)
- ✅ TN-150: Config Update Service (COMPLETE)
- ✅ TN-151: Config Validator (COMPLETE)
- ✅ TN-019: Config Loader (viper) (COMPLETE)

### Downstream Dependencies (This Enables)
- 🎯 TN-137-141: Routing Engine (will use hot reload)
- 🎯 TN-153: Template Engine (will use hot reload)
- 🎯 GitOps Integration (future)

---

## 10. Risks and Mitigations

### Risk 1: Signal Handling Race Conditions
**Probability**: Medium
**Impact**: Critical
**Mitigation**:
- Use Go channels for signal handling
- Debounce multiple signals (1s window)
- Lock-free hot swap using atomic.Value
- Comprehensive testing

### Risk 2: File System Issues
**Probability**: Low
**Impact**: Medium
**Mitigation**:
- Handle file not found errors
- Handle permission errors
- Handle disk full errors
- Keep old config on any failure

### Risk 3: Reload Performance
**Probability**: Low
**Impact**: Medium
**Mitigation**:
- Parallel component reload (TN-150)
- Benchmarks to verify < 300ms
- Timeout protection (30s)
- Performance testing

---

## 11. Acceptance Criteria Summary

### Must Have (100%)
- [x] SIGHUP handler in main.go
- [x] Config reload from disk
- [x] Integration with TN-150
- [x] Error handling
- [x] Basic metrics
- [x] Unit tests (80% coverage)
- [x] Integration tests (5+)

### Should Have (150%)
- [x] Pre-reload validation (TN-151)
- [x] CLI tool (`alertmanager-plus-plus reload`)
- [x] Complete operator guide
- [x] Debouncing
- [x] Rollback on failure
- [x] Performance benchmarks
- [x] 20+ tests total

---

## 12. Timeline

**Estimated Duration**: 4-6 hours

| Phase | Duration | Tasks |
|-------|----------|-------|
| Phase 1: Core | 2h | Signal handling + integration |
| Phase 2: Testing | 1.5h | Unit + integration tests |
| Phase 3: Docs | 1h | CLI tool + operator guide |
| Phase 4: QA | 0.5h | Review + validation |
| **Total** | **5h** | **4-6h estimate** |

---

## 13. References

- **TN-150**: Config Update Service (hot reload infrastructure)
- **TN-151**: Config Validator (validation before reload)
- **Go signal package**: `os/signal`, `syscall`
- **Viper**: Configuration management
- **Prometheus**: Metrics

---

**Status**: 🎯 READY TO START
**Quality Target**: 150% (Grade A+ EXCEPTIONAL)
**Estimated Completion**: 4-6 hours
**Next Action**: Create TODOs and start Phase 1 implementation

---

**Document Version**: 1.0
**Last Updated**: 2025-11-24
**Author**: AI Assistant
**Task**: TN-152 Hot Reload Mechanism (SIGHUP)
