//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClassificationAPI_WithCacheHit tests classification with cache hit
func TestClassificationAPI_WithCacheHit(t *testing.T) {
	t.Skip("TODO: Implement in full Phase 2 - requires running server")

	ctx := context.Background()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)

	// Pre-seed cache
	// TODO: helper.SeedCache(ctx, cacheKey, classification)

	// TODO: POST /classification/classify
	// TODO: Assert 0 LLM calls (cache hit)
	// TODO: Assert classification result
}

// TestClassificationAPI_WithCacheMiss tests classification with LLM call
func TestClassificationAPI_WithCacheMiss(t *testing.T) {
	t.Skip("TODO: Implement in full Phase 2")

	ctx := context.Background()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)

	// Configure mock LLM
	helper.MockLLM.SetDefaultResponses()

	// TODO: POST /classification/classify with new alert
	// TODO: Assert LLM called
	// TODO: Assert classification result stored in cache
}

// TestClassificationAPI_ForceBypassCache tests force=true bypasses cache
func TestClassificationAPI_ForceBypassCache(t *testing.T) {
	t.Skip("TODO: Implement in full Phase 2")

	ctx := context.Background()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)

	// Seed cache
	// Configure mock LLM
	helper.MockLLM.SetDefaultResponses()

	initialCount := helper.MockLLM.GetRequestCount()

	// TODO: POST /classification/classify?force=true
	// TODO: Assert LLM called despite cache
	finalCount := helper.MockLLM.GetRequestCount()
	assert.Greater(t, finalCount, initialCount, "LLM should be called with force=true")
}

// TestClassificationAPI_Timeout tests LLM timeout fallback
func TestClassificationAPI_Timeout(t *testing.T) {
	t.Skip("TODO: Implement in full Phase 2")

	ctx := context.Background()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)

	// Configure high latency
	helper.MockLLM.SetLatency(10 * time.Second)

	// TODO: POST /classification/classify
	// TODO: Assert fallback to rule-based classification
	// TODO: Assert response within timeout
}

// TestClassificationAPI_Stats tests GET /classification/stats
func TestClassificationAPI_Stats(t *testing.T) {
	t.Skip("TODO: Implement in full Phase 2")

	// TODO: Perform several classifications
	// TODO: GET /classification/stats
	// TODO: Assert cache hit rate, total classifications
}
