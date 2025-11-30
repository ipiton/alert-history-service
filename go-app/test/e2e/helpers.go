//go:build e2e
// +build e2e

package e2e

import (
	"context"

	"github.com/vitaliisemenov/alert-history/test/integration"
)

// Re-export integration types and functions for E2E tests

// TestInfrastructure wraps integration infrastructure
type TestInfrastructure = integration.TestInfrastructure

// APITestHelper wraps integration helper
type APITestHelper = integration.APITestHelper

// MockLLMServer wraps integration mock LLM
type MockLLMServer = integration.MockLLMServer

// MockLLMResponse wraps integration mock response type for E2E tests
type MockLLMResponse = integration.MockLLMResponse

// ClassificationResponse wraps integration classification response
type ClassificationResponse = integration.ClassificationResponse

// AlertmanagerWebhook wraps integration webhook type
type AlertmanagerWebhook = integration.AlertmanagerWebhook

// AlertmanagerAlert wraps integration alert type
type AlertmanagerAlert = integration.AlertmanagerAlert

// Fixtures wraps integration fixtures
type Fixtures = integration.Fixtures

// SetupTestInfrastructure initializes test infrastructure
func SetupTestInfrastructure(ctx context.Context) (*TestInfrastructure, error) {
	return integration.SetupTestInfrastructure(ctx)
}

// NewAPITestHelper creates API test helper
func NewAPITestHelper(infra *TestInfrastructure) *APITestHelper {
	return integration.NewAPITestHelper(infra)
}

// NewFixtures creates fixtures loader
func NewFixtures() *Fixtures {
	return integration.NewFixtures()
}
