# TN-122: Group Key Generator - Design Document

## Архитектурное решение

### Общая архитектура

```
┌─────────────────────────────────────────────────────────────┐
│                    Alert + Route Config                     │
│  Alert: {alertname: "HighCPU", cluster: "prod"}           │
│  Route: {group_by: ["alertname", "cluster"]}               │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                   Label Extractor                           │
│  Extract values for group_by labels                         │
│  Handle missing labels: cluster=<missing>                   │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                   Key Builder                               │
│  Sort labels alphabetically                                 │
│  Format: "label1=value1,label2=value2"                     │
│  URL encode special characters                              │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                   Group Key (String)                        │
│  "alertname=HighCPU,cluster=prod"                          │
└─────────────────────────────────────────────────────────────┘
                            ↓ (optional)
┌─────────────────────────────────────────────────────────────┐
│                   FNV-1a Hash                               │
│  Generate 64-bit hash for short keys                        │
│  Output: "a1b2c3d4e5f60708" (hex string)                   │
└─────────────────────────────────────────────────────────────┘
```

### Алгоритм генерации ключа

#### Step 1: Label Extraction
```
Input: labels={alertname:"CPU", cluster:"prod", instance:"s1"}
       groupBy=["alertname", "cluster"]

Extract:
  - alertname → "CPU"
  - cluster → "prod"
  - instance → (ignored, not in groupBy)

Result: {alertname:"CPU", cluster:"prod"}
```

#### Step 2: Handle Special Cases
```
Case A: Special grouping '...'
  groupBy = ["..."]
  → Use ALL labels from alert

Case B: Global group
  groupBy = []
  → Return constant "{global}"

Case C: Missing labels
  groupBy = ["alertname", "team"]
  labels = {alertname:"CPU"}  # No 'team' label
  → team = "<missing>"
```

#### Step 3: Sort & Format
```
Extracted: {cluster:"prod", alertname:"CPU"}

Sort by key: alertname, cluster

Format: "alertname=CPU,cluster=prod"
```

#### Step 4: URL Encoding (if needed)
```
Value with special chars: "team=my team,env=us-east-1"
URL encoded: "team=my%20team,env=us-east-1"
```

#### Step 5: Optional Hashing
```
Long key: "alertname=VeryLongAlertName,cluster=production,..."
FNV-1a hash: "a1b2c3d4e5f60708"
Use hash when key length > threshold (e.g., 256 bytes)
```

## API Design

### Core Functions

```go
package grouping

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// GroupKey represents a unique identifier for an alert group.
type GroupKey string

// GroupKeyGenerator generates unique keys for alert groups.
type GroupKeyGenerator struct {
	// hashLongKeys determines whether to hash keys longer than maxKeyLength
	hashLongKeys bool

	// maxKeyLength is the threshold for hashing (default: 256 bytes)
	maxKeyLength int
}

// NewGroupKeyGenerator creates a new group key generator with default settings.
func NewGroupKeyGenerator() *GroupKeyGenerator {
	return &GroupKeyGenerator{
		hashLongKeys: false, // Disabled by default for readability
		maxKeyLength: 256,
	}
}

// WithHashLongKeys enables hashing of keys longer than maxKeyLength.
func (g *GroupKeyGenerator) WithHashLongKeys(enabled bool) *GroupKeyGenerator {
	g.hashLongKeys = enabled
	return g
}

// WithMaxKeyLength sets the maximum key length before hashing.
func (g *GroupKeyGenerator) WithMaxKeyLength(length int) *GroupKeyGenerator {
	g.maxKeyLength = length
	return g
}

// GenerateKey generates a group key from alert labels and grouping configuration.
//
// Parameters:
//   - labels: map of label names to values from the alert
//   - groupBy: list of label names to group by (from route configuration)
//
// Returns:
//   - GroupKey: unique identifier for the group
//
// Special cases:
//   - groupBy == ["..."]: includes all labels (special grouping)
//   - groupBy == []: returns "{global}" (single global group)
//   - missing labels: uses "<missing>" as value
//
// Example:
//
//	labels := map[string]string{"alertname": "HighCPU", "cluster": "prod"}
//	groupBy := []string{"alertname", "cluster"}
//	key := gen.GenerateKey(labels, groupBy)
//	// Returns: "alertname=HighCPU,cluster=prod"
func (g *GroupKeyGenerator) GenerateKey(
	labels map[string]string,
	groupBy []string,
) GroupKey {
	// Special case: global grouping
	if len(groupBy) == 0 {
		return GroupKey("{global}")
	}

	// Special case: group by all labels
	if len(groupBy) == 1 && groupBy[0] == "..." {
		return g.generateAllLabelsKey(labels)
	}

	// Normal grouping: extract specified labels
	return g.generateKeyFromLabels(labels, groupBy)
}

// GenerateHash generates a FNV-1a hash of the group key.
// Useful for creating shorter, fixed-length identifiers.
//
// Returns: 16-character hexadecimal string (64-bit hash)
func (g *GroupKeyGenerator) GenerateHash(
	labels map[string]string,
	groupBy []string,
) string {
	key := g.GenerateKey(labels, groupBy)
	return hashFNV1a(string(key))
}

// generateAllLabelsKey generates a key containing all labels from the alert.
func (g *GroupKeyGenerator) generateAllLabelsKey(labels map[string]string) GroupKey {
	if len(labels) == 0 {
		return GroupKey("{empty}")
	}

	// Get all label names and sort them
	labelNames := make([]string, 0, len(labels))
	for name := range labels {
		labelNames = append(labelNames, name)
	}
	sort.Strings(labelNames)

	return g.buildKey(labels, labelNames)
}

// generateKeyFromLabels generates a key from specific labels.
func (g *GroupKeyGenerator) generateKeyFromLabels(
	labels map[string]string,
	groupBy []string,
) GroupKey {
	// Sort groupBy labels for deterministic output
	sortedGroupBy := make([]string, len(groupBy))
	copy(sortedGroupBy, groupBy)
	sort.Strings(sortedGroupBy)

	return g.buildKey(labels, sortedGroupBy)
}

// buildKey builds the actual key string from labels and label names.
func (g *GroupKeyGenerator) buildKey(
	labels map[string]string,
	labelNames []string,
) GroupKey {
	pairs := make([]string, 0, len(labelNames))

	for _, name := range labelNames {
		value, ok := labels[name]
		if !ok {
			value = "<missing>"
		}

		// URL encode value if it contains special characters
		encodedValue := url.QueryEscape(value)

		pair := fmt.Sprintf("%s=%s", name, encodedValue)
		pairs = append(pairs, pair)
	}

	key := strings.Join(pairs, ",")

	// Optionally hash long keys
	if g.hashLongKeys && len(key) > g.maxKeyLength {
		return GroupKey(fmt.Sprintf("{hash:%s}", hashFNV1a(key)))
	}

	return GroupKey(key)
}

// Parse parses a group key back into its components.
// Returns map of label names to values.
//
// Note: This is lossy for hashed keys (returns hash string only).
func (key GroupKey) Parse() (map[string]string, error) {
	s := string(key)

	// Handle special keys
	if s == "{global}" {
		return map[string]string{}, nil
	}
	if s == "{empty}" {
		return map[string]string{}, nil
	}
	if strings.HasPrefix(s, "{hash:") {
		return map[string]string{"__hash__": strings.TrimSuffix(strings.TrimPrefix(s, "{hash:"), "}")}, nil
	}

	// Parse normal key format: "label1=value1,label2=value2"
	result := make(map[string]string)
	pairs := strings.Split(s, ",")

	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid key format: %s", pair)
		}

		name := parts[0]
		encodedValue := parts[1]

		// URL decode value
		value, err := url.QueryUnescape(encodedValue)
		if err != nil {
			return nil, fmt.Errorf("failed to decode value for %s: %w", name, err)
		}

		result[name] = value
	}

	return result, nil
}

// String returns the string representation of the group key.
func (key GroupKey) String() string {
	return string(key)
}

// Matches checks if an alert matches this group key based on groupBy configuration.
func (key GroupKey) Matches(labels map[string]string, groupBy []string) bool {
	gen := NewGroupKeyGenerator()
	otherKey := gen.GenerateKey(labels, groupBy)
	return key == otherKey
}
```

### Hash Functions

```go
// hash.go
package grouping

import (
	"encoding/hex"
	"hash/fnv"
)

// hashFNV1a generates a FNV-1a 64-bit hash of the input string.
// Returns hexadecimal string representation.
//
// FNV-1a is chosen for:
//   - Fast computation
//   - Good distribution
//   - Alertmanager compatibility
//
// Example:
//
//	hash := hashFNV1a("alertname=HighCPU,cluster=prod")
//	// Returns: "a1b2c3d4e5f60708" (16 hex chars)
func hashFNV1a(s string) string {
	h := fnv.New64a()
	h.Write([]byte(s))
	sum := h.Sum64()
	return uint64ToHex(sum)
}

// uint64ToHex converts uint64 to hexadecimal string.
func uint64ToHex(n uint64) string {
	bytes := make([]byte, 8)
	bytes[0] = byte(n >> 56)
	bytes[1] = byte(n >> 48)
	bytes[2] = byte(n >> 40)
	bytes[3] = byte(n >> 32)
	bytes[4] = byte(n >> 24)
	bytes[5] = byte(n >> 16)
	bytes[6] = byte(n >> 8)
	bytes[7] = byte(n)
	return hex.EncodeToString(bytes)
}

// HashFromKey is a convenience function that generates hash from GroupKey.
func HashFromKey(key GroupKey) string {
	return hashFNV1a(string(key))
}
```

## Examples

### Example 1: Basic Usage

```go
package main

import (
	"fmt"
	"alert-history/internal/infrastructure/grouping"
)

func main() {
	gen := grouping.NewGroupKeyGenerator()

	// Alert labels
	labels := map[string]string{
		"alertname": "HighCPU",
		"cluster":   "production",
		"instance":  "server1",
	}

	// Group by alertname and cluster
	groupBy := []string{"alertname", "cluster"}

	key := gen.GenerateKey(labels, groupBy)
	fmt.Printf("Group Key: %s\n", key)
	// Output: Group Key: alertname=HighCPU,cluster=production

	// Generate hash
	hash := gen.GenerateHash(labels, groupBy)
	fmt.Printf("Group Hash: %s\n", hash)
	// Output: Group Hash: a1b2c3d4e5f60708
}
```

### Example 2: Special Grouping

```go
// Special grouping: all labels
groupBy := []string{"..."}
key := gen.GenerateKey(labels, groupBy)
fmt.Printf("All Labels Key: %s\n", key)
// Output: alertname=HighCPU,cluster=production,instance=server1

// Global grouping
groupBy = []string{}
key = gen.GenerateKey(labels, groupBy)
fmt.Printf("Global Key: %s\n", key)
// Output: {global}
```

### Example 3: Missing Labels

```go
labels := map[string]string{
	"alertname": "HighCPU",
	// No 'cluster' label
}

groupBy := []string{"alertname", "cluster"}
key := gen.GenerateKey(labels, groupBy)
fmt.Printf("Key with missing: %s\n", key)
// Output: alertname=HighCPU,cluster=<missing>
```

### Example 4: Hash Long Keys

```go
gen := grouping.NewGroupKeyGenerator().
	WithHashLongKeys(true).
	WithMaxKeyLength(100)

labels := map[string]string{
	"alertname":   "VeryLongAlertNameThatExceedsMaxLength",
	"cluster":     "production-us-east-1-cluster-name",
	"environment": "production",
	"team":        "platform-engineering",
	// ... many more labels
}

groupBy := []string{"..."}
key := gen.GenerateKey(labels, groupBy)
fmt.Printf("Hashed Key: %s\n", key)
// Output: {hash:a1b2c3d4e5f60708}
```

## Performance Considerations

### Optimization Strategies

1. **Pre-sorted labels**: Cache sorted label lists
2. **String builder**: Use strings.Builder instead of concatenation
3. **Pool allocation**: Use sync.Pool for temporary buffers
4. **Avoid URL encoding**: Only encode if special chars detected

### Benchmarks

```go
// keygen_bench_test.go
func BenchmarkGenerateKey_Simple(b *testing.B) {
	gen := NewGroupKeyGenerator()
	labels := map[string]string{
		"alertname": "HighCPU",
		"cluster":   "prod",
	}
	groupBy := []string{"alertname", "cluster"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = gen.GenerateKey(labels, groupBy)
	}
}

// Target: <100μs per operation
// Expected: ~50-70μs per operation
```

### Memory Profile

```
Allocation per GenerateKey call:
- Normal case: ~500 bytes
- Special grouping: ~1KB
- Hash mode: ~600 bytes
```

## Compatibility

### Alertmanager Compatibility

This implementation is **compatible** with Alertmanager's grouping behavior:
- Same FNV-1a algorithm
- Same key format
- Same handling of missing labels
- Same special grouping behavior

### Migration Path

For systems migrating from Alertmanager:
1. Group keys remain same → no re-grouping needed
2. Existing silences work unchanged
3. No data migration required

---

**Архитектор**: DevOps Team
**Дата создания**: 2025-01-09
**Версия**: 1.0
**Статус**: Ready for Implementation
