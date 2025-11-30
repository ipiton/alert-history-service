//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnrichmentAPI_GetMode tests GET /enrichment/mode
func TestEnrichmentAPI_GetMode(t *testing.T) {
	t.Skip("TODO: Implement in full Phase 2 - requires running server")

	ctx := context.Background()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)

	// TODO: GET /enrichment/mode
	// TODO: Assert returns current mode (transparent/enriched)
	_ = helper
}

// TestEnrichmentAPI_SetMode tests POST /enrichment/mode
func TestEnrichmentAPI_SetMode(t *testing.T) {
	t.Skip("TODO: Implement in full Phase 2")

	ctx := context.Background()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)

	// TODO: POST /enrichment/mode {mode: "enriched"}
	// TODO: Assert mode changed
	// TODO: Verify persisted in Redis

	// TODO: GET /enrichment/mode
	// TODO: Assert mode is "enriched"
	_ = helper
}

// TestEnrichmentAPI_ModeImpactsProcessing tests mode affects alert processing
func TestEnrichmentAPI_ModeImpactsProcessing(t *testing.T) {
	t.Skip("TODO: Implement in full Phase 2")

	// TODO: Set mode to "transparent"
	// TODO: Send webhook
	// TODO: Assert no LLM classification

	// TODO: Set mode to "enriched"
	// TODO: Send webhook
	// TODO: Assert LLM classification performed
}
