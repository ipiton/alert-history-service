# TN-122: Group Key Generator - Task Checklist

## Статус: 📋 TODO (Ready to Start)

**Started**: TBD  
**Target Completion**: 2-3 дня after start  
**Actual Completion**: TBD

**Dependencies**: TN-121 ✅ DONE

---

## Implementation Checklist

### Phase 1: Project Setup
- [ ] Create `keygen.go` in `internal/infrastructure/grouping/`
- [ ] Create `hash.go` in `internal/infrastructure/grouping/`
- [ ] Add godoc package comments
- [ ] Plan struct and interface design

### Phase 2: Core Implementation
- [ ] **keygen.go** - Group Key Generator
  - [ ] Define `GroupKey` type
  - [ ] Define `GroupKeyGenerator` struct
  - [ ] Implement `NewGroupKeyGenerator()` constructor
  - [ ] Implement `GenerateKey()` main function
  - [ ] Implement `GenerateHash()` hash function
  - [ ] Implement `generateAllLabelsKey()` for special grouping
  - [ ] Implement `generateKeyFromLabels()` for normal grouping
  - [ ] Implement `buildKey()` key builder
  - [ ] Implement `GroupKey.Parse()` key parser
  - [ ] Implement `GroupKey.String()` string method
  - [ ] Implement `GroupKey.Matches()` matcher
  - [ ] Add options: `WithHashLongKeys()`, `WithMaxKeyLength()`

- [ ] **hash.go** - FNV-1a Hashing
  - [ ] Implement `hashFNV1a()` function
  - [ ] Implement `uint64ToHex()` converter
  - [ ] Implement `HashFromKey()` convenience function
  - [ ] Add godoc comments

### Phase 3: Unit Tests
- [ ] **keygen_test.go** - Comprehensive tests
  - [ ] Test basic grouping (single label)
  - [ ] Test multiple labels grouping
  - [ ] Test label sorting (deterministic keys)
  - [ ] Test special grouping `...` (all labels)
  - [ ] Test global grouping `[]` (single group)
  - [ ] Test missing labels (`<missing>`)
  - [ ] Test empty label values
  - [ ] Test URL encoding (special characters)
  - [ ] Test key parsing (`GroupKey.Parse()`)
  - [ ] Test key matching (`GroupKey.Matches()`)
  - [ ] Test hash generation
  - [ ] Test long key hashing (optional)
  - [ ] Property-based test: determinism
  - [ ] Property-based test: same labels → same key
  - [ ] Edge case: nil labels
  - [ ] Edge case: empty labels map
  - [ ] Edge case: very long label values

- [ ] **hash_test.go** - Hash function tests
  - [ ] Test FNV-1a correctness
  - [ ] Test hash determinism
  - [ ] Test hash format (16 hex chars)
  - [ ] Test collision resistance (different inputs)

### Phase 4: Benchmarks
- [ ] **keygen_bench_test.go** - Performance tests
  - [ ] Benchmark `GenerateKey()` - simple case (2 labels)
  - [ ] Benchmark `GenerateKey()` - complex case (10 labels)
  - [ ] Benchmark `GenerateKey()` - special grouping
  - [ ] Benchmark `GenerateHash()` 
  - [ ] Benchmark `GroupKey.Parse()`
  - [ ] Memory profiling benchmarks
  - [ ] Concurrent access benchmarks

### Phase 5: Documentation
- [ ] **Godoc comments**
  - [ ] Package-level documentation
  - [ ] `GroupKey` type documentation
  - [ ] `GroupKeyGenerator` struct documentation
  - [ ] All exported functions documented
  - [ ] Examples in godoc format

- [ ] **README.md** (optional)
  - [ ] Usage examples
  - [ ] Algorithm description
  - [ ] Performance characteristics
  - [ ] Alertmanager compatibility notes

### Phase 6: Integration
- [ ] **Integration points**
  - [ ] Used by TN-123 (Group Manager)
  - [ ] Example usage in package tests
  - [ ] Integration test with Route config (TN-121)

### Phase 7: Testing and Validation
- [ ] **Manual testing**
  - [ ] Test with real alert data
  - [ ] Test with Alertmanager configs
  - [ ] Verify Alertmanager compatibility
  - [ ] Test edge cases manually

- [ ] **Performance validation**
  - [ ] Run benchmarks
  - [ ] Verify <100μs target
  - [ ] Memory profiling
  - [ ] Concurrent access testing

- [ ] **Coverage verification**
  - [ ] Run `go test -cover`
  - [ ] Verify >90% coverage
  - [ ] Add tests for uncovered branches
  - [ ] Generate coverage report

### Phase 8: Code Review and QA
- [ ] **Code quality**
  - [ ] Run `golangci-lint`
  - [ ] Fix all linter issues
  - [ ] Run `go vet`
  - [ ] Check for race conditions (`go test -race`)

- [ ] **Security review**
  - [ ] Review URL encoding security
  - [ ] Check for injection vulnerabilities
  - [ ] Verify key length limits (DoS protection)
  - [ ] Test with malicious input

- [ ] **Documentation review**
  - [ ] Review all godoc comments
  - [ ] Review examples
  - [ ] Spell check documentation

### Phase 9: Deployment
- [ ] **Merge and deploy**
  - [ ] Push code to `feature/TN-122-group-key-generator` branch
  - [ ] Create Pull Request
  - [ ] Address review comments
  - [ ] Merge to main branch
  - [ ] Update TN-123 to use Group Key Generator

---

## Test Coverage Goals

| Component | Target Coverage | Actual Coverage |
|-----------|----------------|-----------------|
| keygen.go | >90% | TBD |
| hash.go | >85% | TBD |
| **Total** | **>90%** | **TBD** |

---

## Performance Goals

| Metric | Target | Actual |
|--------|--------|--------|
| GenerateKey (simple) | <50μs | TBD |
| GenerateKey (complex) | <100μs | TBD |
| GenerateHash | <10μs | TBD |
| Memory per call | <1KB | TBD |
| Concurrent throughput | >10K ops/sec | TBD |

---

## Dependencies

### Internal Dependencies
- TN-121 (Grouping Configuration Parser) ✅ DONE
- `internal/core/interfaces.go` (Alert struct) ✅ EXISTS

### External Dependencies
- `hash/fnv` (standard library)
- `sort` (standard library)
- `net/url` (standard library)
- `strings` (standard library)

### Blocks
- TN-123 (Alert Group Manager) - BLOCKED until TN-122 complete

---

## Test Plan

### Unit Tests (20+ tests)

```go
// Basic tests
TestGenerateKey_SingleLabel
TestGenerateKey_MultipleLabels
TestGenerateKey_LabelSorting
TestGenerateKey_SpecialGrouping
TestGenerateKey_GlobalGrouping

// Edge cases
TestGenerateKey_MissingLabels
TestGenerateKey_EmptyValue
TestGenerateKey_SpecialCharacters
TestGenerateKey_NilLabels
TestGenerateKey_EmptyLabelsMap
TestGenerateKey_VeryLongValue

// Parsing
TestGroupKey_Parse_Valid
TestGroupKey_Parse_Special
TestGroupKey_Parse_Invalid

// Matching
TestGroupKey_Matches_Same
TestGroupKey_Matches_Different

// Hash
TestGenerateHash_Deterministic
TestHashFNV1a_Correctness
TestHashFNV1a_Collisions

// Property-based
TestProperty_Determinism
TestProperty_SameLabels_SameKey
```

### Benchmark Tests (6+ benchmarks)

```go
BenchmarkGenerateKey_Simple
BenchmarkGenerateKey_Complex
BenchmarkGenerateKey_SpecialGrouping
BenchmarkGenerateHash
BenchmarkGroupKey_Parse
BenchmarkConcurrent
```

### Integration Tests

```go
// With TN-121 (Config Parser)
TestIntegration_WithRouteConfig

// With real alert data
TestIntegration_RealAlertData

// Alertmanager compatibility
TestCompatibility_Alertmanager
```

---

## Notes

### Design Decisions
1. **FNV-1a algorithm** - Fast, good distribution, Alertmanager compatible
2. **Sorted labels** - Ensures deterministic keys
3. **URL encoding** - Handles special characters safely
4. **Optional hashing** - For very long keys (>256 bytes)
5. **`<missing>` marker** - Clear indication of missing labels

### Known Limitations
1. Maximum key length: 2048 bytes (before hashing)
2. Hash mode loses readability (trade-off for performance)
3. URL encoding adds ~10% overhead for special characters

### Future Enhancements
1. Configurable missing label marker (default: `<missing>`)
2. Custom hash algorithms (beyond FNV-1a)
3. Key compression for very large label sets
4. Cache frequently generated keys

### Alertmanager Compatibility
✅ Compatible with Alertmanager v0.23+
- Same FNV-1a algorithm
- Same key format
- Same special grouping behavior
- Can migrate without data loss

---

**Last Updated**: 2025-01-09  
**Status**: 📋 TODO (Ready to Start after TN-121 code completion)  
**Next Step**: Wait for TN-121 code to be tested and merged, then start Phase 1

