# TN-046: Implementation Tasks Checklist

## Phase 1: Setup & Structure (30 min)

- [x] Create directory structure
  - [x] `go-app/internal/infrastructure/k8s/`
  - [x] `tasks/go-migration-analysis/TN-046-k8s-secrets-client/`

- [x] Create documentation files
  - [x] `requirements.md` (480 lines) ✅
  - [x] `design.md` (850 lines) ✅
  - [x] `tasks.md` (this file)

- [ ] Verify dependencies в go.mod
  - [ ] `k8s.io/client-go` v0.28.0+
  - [ ] `k8s.io/api` v0.28.0+
  - [ ] `k8s.io/apimachinery` v0.28.0+

## Phase 2: Error Types (45 min)

- [ ] Create `go-app/internal/infrastructure/k8s/errors.go`
  - [ ] Base `K8sError` struct с Op, Message, Err fields
  - [ ] `Error()` method implementation
  - [ ] `Unwrap()` method для error wrapping support
  - [ ] `ConnectionError` type
  - [ ] `AuthError` type
  - [ ] `NotFoundError` type
  - [ ] `TimeoutError` type
  - [ ] `NewConnectionError()` constructor
  - [ ] `NewAuthError()` constructor
  - [ ] `NewNotFoundError()` constructor
  - [ ] `NewTimeoutError()` constructor
  - [ ] `wrapK8sError()` helper function
  - [ ] `isRetryableError()` helper function
  - [ ] Godoc comments для all exported types

## Phase 3: Interface & Configuration (1 hour)

- [ ] Create `go-app/internal/infrastructure/k8s/client.go` (interface part)
  - [ ] `K8sClient` interface definition
    - [ ] `ListSecrets(ctx, namespace, labelSelector)` signature
    - [ ] `GetSecret(ctx, namespace, name)` signature
    - [ ] `Health(ctx)` signature
    - [ ] `Close()` signature
  - [ ] `K8sClientConfig` struct
    - [ ] `Timeout` field (time.Duration)
    - [ ] `MaxRetries` field (int)
    - [ ] `RetryBackoff` field (time.Duration)
    - [ ] `MaxRetryBackoff` field (time.Duration)
    - [ ] `Logger` field (*slog.Logger)
  - [ ] `DefaultK8sClientConfig()` function
    - [ ] Timeout: 30s
    - [ ] MaxRetries: 3
    - [ ] RetryBackoff: 100ms
    - [ ] MaxRetryBackoff: 5s
    - [ ] Logger: slog.Default()
  - [ ] Godoc comments для interface и structs

## Phase 4: Client Implementation - Core (2 hours)

- [ ] `DefaultK8sClient` struct implementation
  - [ ] Fields:
    - [ ] `clientset` (kubernetes.Interface)
    - [ ] `config` (*K8sClientConfig)
    - [ ] `logger` (*slog.Logger)
    - [ ] `mu` (sync.RWMutex)

- [ ] `NewK8sClient()` constructor
  - [ ] Validate config (use defaults if nil)
  - [ ] Load in-cluster config (`rest.InClusterConfig()`)
  - [ ] Handle error если not in cluster
  - [ ] Apply timeout to k8sConfig
  - [ ] Create clientset (`kubernetes.NewForConfig()`)
  - [ ] Handle clientset creation error
  - [ ] Perform initial health check
  - [ ] Return error если health check fails
  - [ ] Log successful initialization
  - [ ] Godoc comments

- [ ] `retryWithBackoff()` helper method
  - [ ] Accept ctx и operation function
  - [ ] Implement retry loop (0 to MaxRetries)
  - [ ] Check context cancellation before each attempt
  - [ ] Call operation()
  - [ ] Return immediately если success
  - [ ] Check if error is retryable (`isRetryableError()`)
  - [ ] Return immediately if not retryable
  - [ ] Log retry attempt с backoff duration
  - [ ] Wait with exponential backoff
  - [ ] Check context cancellation during wait
  - [ ] Double backoff each iteration
  - [ ] Cap backoff at MaxRetryBackoff
  - [ ] Return error после max retries

## Phase 5: Client Implementation - Methods (2 hours)

- [ ] `ListSecrets()` implementation
  - [ ] Debug log entry с namespace, labelSelector
  - [ ] Create listOptions с LabelSelector
  - [ ] Set Limit to 1000 (pagination support)
  - [ ] Wrap API call в retryWithBackoff
  - [ ] Call `clientset.CoreV1().Secrets(namespace).List(ctx, listOptions)`
  - [ ] Extract secretList.Items
  - [ ] Check for pagination (secretList.Continue != "")
  - [ ] Log warning если pagination needed
  - [ ] Handle error (wrap с wrapK8sError)
  - [ ] Log error с details
  - [ ] Return nil, error on failure
  - [ ] Log success с secrets count
  - [ ] Return secrets, nil
  - [ ] Godoc comments

- [ ] `GetSecret()` implementation
  - [ ] Debug log entry с namespace, name
  - [ ] Wrap API call в retryWithBackoff
  - [ ] Call `clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})`
  - [ ] Store result в outer variable
  - [ ] Handle errors:
    - [ ] Check for NotFound (`errors.IsNotFound()`)
    - [ ] Return NewNotFoundError без retry
    - [ ] Wrap other errors с wrapK8sError
    - [ ] Log error с details
  - [ ] Log success (debug level)
  - [ ] Return secret, nil
  - [ ] Godoc comments

- [ ] `Health()` implementation
  - [ ] Create health context с 5s timeout
  - [ ] Defer cancel()
  - [ ] Call `clientset.Discovery().ServerVersion()`
  - [ ] Handle error:
    - [ ] Log warning
    - [ ] Return NewConnectionError
  - [ ] Return nil on success
  - [ ] Godoc comments

- [ ] `Close()` implementation
  - [ ] Log "Closing K8s client"
  - [ ] Lock mutex
  - [ ] Defer unlock
  - [ ] Nil out clientset reference
  - [ ] Log "K8s client closed"
  - [ ] Return nil
  - [ ] Godoc comments

## Phase 6: Unit Tests - Setup (1 hour)

- [ ] Create `go-app/internal/infrastructure/k8s/client_test.go`
  - [ ] Package declaration
  - [ ] Imports:
    - [ ] testing
    - [ ] context
    - [ ] time
    - [ ] github.com/stretchr/testify/assert
    - [ ] github.com/stretchr/testify/require
    - [ ] k8s.io/api/core/v1
    - [ ] k8s.io/apimachinery/pkg/apis/meta/v1
    - [ ] k8s.io/client-go/kubernetes/fake

  - [ ] Helper function `createFakeClient()`
    - [ ] Creates fake.Clientset с test secrets
    - [ ] Returns DefaultK8sClient с fake clientset

  - [ ] Helper function `createTestSecret()`
    - [ ] Name, namespace, labels parameters
    - [ ] Optional data parameter
    - [ ] Returns *corev1.Secret

## Phase 7: Unit Tests - Happy Path (2 hours)

- [ ] `TestNewK8sClient_WithDefaultConfig()`
  - [ ] Note: Skipped in CI (requires in-cluster)
  - [ ] Manual test

- [ ] `TestListSecrets_Success()`
  - [ ] Create fake clientset с 3 test secrets
  - [ ] All secrets have label "publishing-target=true"
  - [ ] Call ListSecrets с label selector
  - [ ] Assert no error
  - [ ] Assert length == 3
  - [ ] Assert secret names correct
  - [ ] Assert labels correct

- [ ] `TestListSecrets_EmptyResult()`
  - [ ] Create fake clientset без secrets
  - [ ] Call ListSecrets
  - [ ] Assert no error
  - [ ] Assert length == 0

- [ ] `TestListSecrets_LabelFiltering()`
  - [ ] Create fake clientset с mixed labels
  - [ ] Some secrets have publishing-target=true, some don't
  - [ ] Call ListSecrets с label selector
  - [ ] Assert only filtered secrets returned

- [ ] `TestGetSecret_Success()`
  - [ ] Create fake clientset с 1 test secret
  - [ ] Call GetSecret с correct namespace/name
  - [ ] Assert no error
  - [ ] Assert secret not nil
  - [ ] Assert secret.Name correct
  - [ ] Assert secret.Data correct

- [ ] `TestGetSecret_NotFound()`
  - [ ] Create fake clientset без secrets
  - [ ] Call GetSecret с nonexistent name
  - [ ] Assert error not nil
  - [ ] Assert secret is nil
  - [ ] Assert error type is NotFoundError
  - [ ] Use errors.As() для type checking

- [ ] `TestHealth_Success()`
  - [ ] Create fake clientset
  - [ ] Call Health()
  - [ ] Assert no error

## Phase 8: Unit Tests - Error Handling (2 hours)

- [ ] `TestListSecrets_ContextCancelled()`
  - [ ] Create fake clientset
  - [ ] Create context с immediate cancel
  - [ ] Cancel context
  - [ ] Call ListSecrets с cancelled context
  - [ ] Assert error not nil
  - [ ] Assert error type is TimeoutError

- [ ] `TestListSecrets_ContextTimeout()`
  - [ ] Create fake clientset
  - [ ] Create context с 1ms timeout
  - [ ] Call ListSecrets
  - [ ] Assert error not nil
  - [ ] Assert timeout-related error

- [ ] `TestGetSecret_ContextCancelled()`
  - [ ] Similar to ListSecrets
  - [ ] Verify context cancellation handling

- [ ] `TestRetryLogic_TransientError()`
  - [ ] Mock operation that fails 2 times, succeeds 3rd
  - [ ] Call retryWithBackoff
  - [ ] Assert operation succeeds
  - [ ] Verify 3 attempts made

- [ ] `TestRetryLogic_PermanentError()`
  - [ ] Mock operation that returns NotFound
  - [ ] Call retryWithBackoff
  - [ ] Assert immediate failure (no retry)
  - [ ] Verify only 1 attempt made

- [ ] `TestRetryLogic_ExhaustedRetries()`
  - [ ] Mock operation that always fails с retryable error
  - [ ] Call retryWithBackoff
  - [ ] Assert error after max retries
  - [ ] Verify MaxRetries+1 attempts made

- [ ] `TestRetryBackoff_ExponentialIncrease()`
  - [ ] Mock operation timing
  - [ ] Verify backoff increases: 100ms, 200ms, 400ms
  - [ ] Verify backoff capped at MaxRetryBackoff

## Phase 9: Unit Tests - Edge Cases (1.5 hours)

- [ ] `TestConcurrentAccess()`
  - [ ] Create fake clientset
  - [ ] Launch 10 goroutines calling ListSecrets
  - [ ] Launch 10 goroutines calling GetSecret
  - [ ] Wait for all to complete
  - [ ] Run с go test -race
  - [ ] Assert no race conditions

- [ ] `TestListSecrets_EmptyNamespace()`
  - [ ] Call ListSecrets с empty namespace string
  - [ ] Assert appropriate error или default behavior

- [ ] `TestListSecrets_EmptyLabelSelector()`
  - [ ] Call ListSecrets с empty label selector
  - [ ] Assert returns all secrets (no filtering)

- [ ] `TestGetSecret_EmptyName()`
  - [ ] Call GetSecret с empty name
  - [ ] Assert appropriate error

- [ ] `TestClose_MultipleCalls()`
  - [ ] Create client
  - [ ] Call Close() twice
  - [ ] Assert no panic
  - [ ] Assert both calls succeed

- [ ] `TestClose_AfterOperations()`
  - [ ] Create client
  - [ ] Perform operations (ListSecrets, GetSecret)
  - [ ] Call Close()
  - [ ] Attempt operation after close
  - [ ] Assert appropriate error

## Phase 10: Benchmarks (1 hour)

- [ ] `BenchmarkListSecrets_10Secrets()`
  - [ ] Create fake clientset с 10 secrets
  - [ ] Benchmark ListSecrets operation
  - [ ] Target: < 500ms per op (но fake будет faster)
  - [ ] Report allocations

- [ ] `BenchmarkListSecrets_100Secrets()`
  - [ ] Create fake clientset с 100 secrets
  - [ ] Benchmark ListSecrets operation
  - [ ] Report allocations

- [ ] `BenchmarkGetSecret()`
  - [ ] Create fake clientset с 1 secret
  - [ ] Benchmark GetSecret operation
  - [ ] Target: < 200ms per op
  - [ ] Report allocations

- [ ] `BenchmarkHealth()`
  - [ ] Create fake clientset
  - [ ] Benchmark Health operation
  - [ ] Target: < 100ms per op
  - [ ] Report allocations

## Phase 11: Error Tests (1 hour)

- [ ] Create `errors_test.go`
  - [ ] `TestK8sError_Error()`
    - [ ] Create K8sError с и без underlying error
    - [ ] Assert error message format correct

  - [ ] `TestK8sError_Unwrap()`
    - [ ] Create K8sError с underlying error
    - [ ] Call Unwrap()
    - [ ] Assert underlying error returned

  - [ ] `TestConnectionError_Type()`
    - [ ] Create ConnectionError
    - [ ] Assert type assertion works
    - [ ] Use errors.As() для verification

  - [ ] Similar tests для AuthError, NotFoundError, TimeoutError

  - [ ] `TestWrapK8sError_Unauthorized()`
    - [ ] Create k8s Unauthorized error
    - [ ] Call wrapK8sError
    - [ ] Assert returns AuthError

  - [ ] `TestWrapK8sError_NotFound()`
    - [ ] Create k8s NotFound error
    - [ ] Call wrapK8sError
    - [ ] Assert returns NotFoundError

  - [ ] `TestWrapK8sError_Timeout()`
    - [ ] Create k8s Timeout error
    - [ ] Call wrapK8sError
    - [ ] Assert returns TimeoutError

  - [ ] `TestIsRetryableError_Transient()`
    - [ ] Test timeout error → retryable
    - [ ] Test server error → retryable
    - [ ] Test rate limit → retryable

  - [ ] `TestIsRetryableError_Permanent()`
    - [ ] Test unauthorized → not retryable
    - [ ] Test forbidden → not retryable
    - [ ] Test not found → not retryable
    - [ ] Test invalid → not retryable

## Phase 12: Integration & Validation (1.5 hours)

- [ ] Run all tests
  - [ ] `cd go-app && go test ./internal/infrastructure/k8s/... -v`
  - [ ] Assert all tests pass
  - [ ] Check test output для warnings

- [ ] Run tests с race detector
  - [ ] `go test ./internal/infrastructure/k8s/... -race`
  - [ ] Assert no race conditions detected

- [ ] Run benchmarks
  - [ ] `go test ./internal/infrastructure/k8s/... -bench=. -benchmem`
  - [ ] Review allocation counts
  - [ ] Review operation times

- [ ] Check test coverage
  - [ ] `go test ./internal/infrastructure/k8s/... -coverprofile=coverage.out`
  - [ ] `go tool cover -func=coverage.out`
  - [ ] Assert coverage >= 80%
  - [ ] Review uncovered lines

- [ ] Run linter
  - [ ] `cd go-app && golangci-lint run ./internal/infrastructure/k8s/...`
  - [ ] Assert zero warnings
  - [ ] Fix any issues

- [ ] Build verification
  - [ ] `cd go-app && go build ./...`
  - [ ] Assert successful build
  - [ ] No compilation errors

## Phase 13: Documentation (1 hour)

- [ ] Add package documentation
  - [ ] Package comment в client.go
  - [ ] Describe purpose (K8s client wrapper for Publishing System)
  - [ ] Usage example
  - [ ] Reference TN-046, TN-047

- [ ] Verify Godoc comments
  - [ ] All exported types documented
  - [ ] All exported functions documented
  - [ ] All methods documented
  - [ ] Examples где appropriate

- [ ] Create usage example file (optional)
  - [ ] `go-app/internal/infrastructure/k8s/example_test.go`
  - [ ] Example of creating client
  - [ ] Example of listing secrets
  - [ ] Example of error handling

- [ ] Update main README (if needed)
  - [ ] Mention K8s integration
  - [ ] Link to TN-046 documentation

## Phase 14: Final Validation & Commit (30 min)

- [ ] Final checklist review
  - [ ] All tasks completed
  - [ ] All tests passing
  - [ ] Coverage >= 80%
  - [ ] Zero linter warnings
  - [ ] Documentation complete

- [ ] Git operations
  - [ ] Review changed files: `git status`
  - [ ] Add files: `git add go-app/internal/infrastructure/k8s/`
  - [ ] Add docs: `git add tasks/go-migration-analysis/TN-046-k8s-secrets-client/`
  - [ ] Commit: `git commit -m "feat(k8s): TN-046 implement K8s secrets client (80%+ coverage, 25+ tests)"`
  - [ ] Verify commit: `git log -1 --stat`

- [ ] Quality metrics
  - [ ] Lines of code: ~250 LOC (client.go) + ~80 (errors.go)
  - [ ] Test code: ~350+ LOC (client_test.go) + ~200 (errors_test.go)
  - [ ] Test count: 25+ tests
  - [ ] Benchmarks: 4
  - [ ] Coverage: 80%+
  - [ ] Grade: A (90-95 points target)

## Summary Statistics

### Estimated Time: 16 hours (2 days)
- Phase 1: 0.5h
- Phase 2: 0.75h
- Phase 3: 1h
- Phase 4: 2h
- Phase 5: 2h
- Phase 6: 1h
- Phase 7: 2h
- Phase 8: 2h
- Phase 9: 1.5h
- Phase 10: 1h
- Phase 11: 1h
- Phase 12: 1.5h
- Phase 13: 1h
- Phase 14: 0.5h

### Deliverables:
- **Production Code**: ~330 LOC
  - client.go: ~250 lines
  - errors.go: ~80 lines

- **Test Code**: ~550+ LOC
  - client_test.go: ~350 lines
  - errors_test.go: ~200 lines

- **Documentation**: ~2,030 lines
  - requirements.md: 480 lines ✅
  - design.md: 850 lines ✅
  - tasks.md: 700 lines ✅

- **Tests**: 25+ unit tests, 4 benchmarks
- **Coverage Target**: 80%+
- **Quality Grade**: A (90-95 points)

### Dependencies Unblocked After Completion:
- ✅ TN-047: Target Discovery Manager (READY)
- ✅ TN-050: RBAC Documentation (READY)
- ✅ TN-048, TN-049: Refresh & Health (blocked by TN-047)

---

**Status**: IN PROGRESS
**Started**: 2025-11-07
**Target Completion**: 2025-11-09 (2 days)
**Actual Completion**: TBD
