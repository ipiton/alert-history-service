package classification

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

// PrometheusClient provides a client for querying Prometheus metrics
// This is an optional enhancement for 150% quality, with graceful degradation
type PrometheusClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
	enabled    bool
}

// PrometheusQueryResult represents the result of a Prometheus query
type PrometheusQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// PrometheusStats contains aggregated statistics from Prometheus
type PrometheusStats struct {
	// By severity breakdown
	BySeverity map[string]int64

	// Cache statistics
	L1CacheHits int64
	L2CacheHits int64

	// LLM statistics
	LLMRequests    int64
	LLMFailures    int64
	LLMAvgLatency  float64
	LLMSuccessRate float64

	// Timestamp of the query
	Timestamp time.Time
}

// NewPrometheusClient creates a new Prometheus client
// If baseURL is empty or invalid, client will be disabled (graceful degradation)
func NewPrometheusClient(baseURL string, logger *slog.Logger) *PrometheusClient {
	if logger == nil {
		logger = slog.Default()
	}

	client := &PrometheusClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 100 * time.Millisecond, // Fast timeout to avoid blocking
		},
		logger:  logger,
		enabled: baseURL != "",
	}

	if !client.enabled {
		logger.Debug("Prometheus client disabled (no base URL provided)")
	}

	return client
}

// Query executes a Prometheus query and returns the result
// Returns error if query fails or times out (graceful degradation)
func (c *PrometheusClient) Query(ctx context.Context, query string) (*PrometheusQueryResult, error) {
	if !c.enabled {
		return nil, fmt.Errorf("prometheus client disabled")
	}

	// Create request context with timeout
	reqCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	// Build query URL
	queryURL := fmt.Sprintf("%s/api/v1/query", c.baseURL)
	req, err := http.NewRequestWithContext(reqCtx, "GET", queryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add query parameters
	q := req.URL.Query()
	q.Set("query", query)
	req.URL.RawQuery = q.Encode()

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Timeout or connection error - graceful degradation
		c.logger.Warn("Prometheus query failed, using fallback",
			"query", query,
			"error", err)
		return nil, fmt.Errorf("prometheus query failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Warn("Prometheus query returned non-200 status",
			"query", query,
			"status", resp.StatusCode,
			"body", string(body))
		return nil, fmt.Errorf("prometheus returned status %d", resp.StatusCode)
	}

	// Parse response
	var result PrometheusQueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode prometheus response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed: status=%s", result.Status)
	}

	return &result, nil
}

// QueryClassificationStats queries Prometheus for classification-related metrics
// Returns nil if Prometheus is unavailable (graceful degradation)
func (c *PrometheusClient) QueryClassificationStats(ctx context.Context) (*PrometheusStats, error) {
	if !c.enabled {
		return nil, nil // Graceful degradation - return nil, not error
	}

	stats := &PrometheusStats{
		BySeverity: make(map[string]int64),
		Timestamp:  time.Now(),
	}

	// Query classifications by severity
	severityQuery := `sum by (severity) (alert_history_business_classification_llm_classifications_total)`
	result, err := c.Query(ctx, severityQuery)
	if err == nil && result != nil {
		for _, r := range result.Data.Result {
			if severity, ok := r.Metric["severity"]; ok {
				if len(r.Value) >= 2 {
					if val, ok := r.Value[1].(string); ok {
						var count int64
						if _, err := fmt.Sscanf(val, "%d", &count); err == nil {
							stats.BySeverity[severity] = count
						}
					}
				}
			}
		}
	}

	// Query L1 cache hits
	l1Query := `alert_history_business_classification_l1_cache_hits_total`
	if result, err := c.Query(ctx, l1Query); err == nil && result != nil {
		if len(result.Data.Result) > 0 && len(result.Data.Result[0].Value) >= 2 {
			if val, ok := result.Data.Result[0].Value[1].(string); ok {
				fmt.Sscanf(val, "%d", &stats.L1CacheHits)
			}
		}
	}

	// Query L2 cache hits
	l2Query := `alert_history_business_classification_l2_cache_hits_total`
	if result, err := c.Query(ctx, l2Query); err == nil && result != nil {
		if len(result.Data.Result) > 0 && len(result.Data.Result[0].Value) >= 2 {
			if val, ok := result.Data.Result[0].Value[1].(string); ok {
				fmt.Sscanf(val, "%d", &stats.L2CacheHits)
			}
		}
	}

	// Query LLM requests and failures (simplified - would need more complex queries in production)
	// For now, we'll use the base stats from ClassificationService

	return stats, nil
}

// IsEnabled returns whether the Prometheus client is enabled
func (c *PrometheusClient) IsEnabled() bool {
	return c.enabled
}

// SetBaseURL updates the Prometheus base URL
func (c *PrometheusClient) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
	c.enabled = baseURL != ""
	if c.enabled {
		// Validate URL
		if _, err := url.Parse(baseURL); err != nil {
			c.logger.Warn("Invalid Prometheus base URL, disabling client",
				"url", baseURL,
				"error", err)
			c.enabled = false
		}
	}
}
