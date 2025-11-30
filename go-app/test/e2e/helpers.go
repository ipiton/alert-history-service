//go:build e2e
// +build e2e

package e2e

import (
	"context"

	"github.com/vitaliisemenov/alert-history/test/integration"
)

// Re-export integration types and functions for E2E tests

// TestInfrastructure wraps integration infrastructure with test application
type TestInfrastructure struct {
	*integration.TestInfrastructure
	TestApp *TestApplication
}

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

// SetupTestInfrastructure initializes test infrastructure + test application
func SetupTestInfrastructure(ctx context.Context) (*TestInfrastructure, error) {
	// Setup base infrastructure (PostgreSQL, Redis, Mock LLM)
	baseInfra, err := integration.SetupTestInfrastructure(ctx)
	if err != nil {
		return nil, err
	}

	// Start test application server
	testApp, err := StartTestApplication(ctx, baseInfra)
	if err != nil {
		baseInfra.Teardown(ctx)
		return nil, err
	}

	// Update BaseURL to point to test application
	baseInfra.BaseURL = testApp.Server.URL

	return &TestInfrastructure{
		TestInfrastructure: baseInfra,
		TestApp:            testApp,
	}, nil
}

// Teardown cleans up test infrastructure and application
func (ti *TestInfrastructure) Teardown(ctx context.Context) {
	if ti.TestApp != nil {
		ti.TestApp.Close()
	}
	if ti.TestInfrastructure != nil {
		ti.TestInfrastructure.Teardown(ctx)
	}
}

// NewAPITestHelper creates API test helper
func NewAPITestHelper(infra *TestInfrastructure) *APITestHelper {
	return integration.NewAPITestHelper(infra.TestInfrastructure)
}

// NewFixtures creates fixtures loader
func NewFixtures() *Fixtures {
	return integration.NewFixtures()
}
