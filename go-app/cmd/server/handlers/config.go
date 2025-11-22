package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	appconfig "github.com/vitaliisemenov/alert-history/internal/config"
	"gopkg.in/yaml.v3"
)

// ConfigHandler handles configuration export requests
type ConfigHandler struct {
	configService appconfig.ConfigService
	logger        *slog.Logger
	metrics       *ConfigExportMetrics
}

// NewConfigHandler creates a new ConfigHandler
func NewConfigHandler(
	configService appconfig.ConfigService,
	logger *slog.Logger,
) *ConfigHandler {
	if logger == nil {
		logger = slog.Default()
	}

	return &ConfigHandler{
		configService: configService,
		logger:        logger,
		metrics:       NewConfigExportMetrics(),
	}
}

// HandleGetConfig handles GET /api/v2/config requests
func (h *ConfigHandler) HandleGetConfig(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	ctx := r.Context()

	// Log incoming request
	h.logger.Info("Config export request received",
		"method", r.Method,
		"path", r.URL.Path,
		"query", r.URL.RawQuery,
		"remote_addr", r.RemoteAddr,
	)

	// Validate HTTP method
	if r.Method != http.MethodGet {
		h.logger.Warn("Invalid HTTP method for config export", "method", r.Method)
		if h.metrics != nil {
			h.metrics.RecordError("validation")
		}
		h.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Parse query parameters
	opts, err := h.parseQueryParameters(r)
	if err != nil {
		h.logger.Warn("Failed to parse query parameters", "error", err)
		if h.metrics != nil {
			h.metrics.RecordError("validation")
		}
		h.respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid query parameters: %v", err))
		return
	}

	// Check authorization for unsanitized config
	if !opts.Sanitize {
		// TODO: Add admin check when auth middleware is available
		// For now, allow unsanitized config (will be secured in production)
		h.logger.Warn("Unsanitized config requested", "remote_addr", r.RemoteAddr)
	}

	// Get config from service
	configResp, err := h.configService.GetConfig(ctx, opts)
	if err != nil {
		h.logger.Error("Failed to export configuration", "error", err)
		if h.metrics != nil {
			h.metrics.RecordError("service")
			h.metrics.RecordRequest(opts.Format, opts.Sanitize, "error", time.Since(startTime), 0)
		}
		h.respondError(w, http.StatusInternalServerError, "failed to export configuration")
		return
	}

	// Serialize response based on format
	var responseBody []byte
	var contentType string

	switch strings.ToLower(opts.Format) {
	case "yaml":
		// Convert ConfigResponse to YAML
		yamlData := map[string]interface{}{
			"version":         configResp.Version,
			"source":          string(configResp.Source),
			"loaded_at":       configResp.LoadedAt.Format(time.RFC3339),
			"config_file_path": configResp.ConfigFilePath,
			"config":          configResp.Config,
		}

		responseBody, err = yaml.Marshal(yamlData)
		if err != nil {
			h.logger.Error("Failed to serialize YAML", "error", err)
			if h.metrics != nil {
				h.metrics.RecordError("serialization")
				h.metrics.RecordRequest(opts.Format, opts.Sanitize, "error", time.Since(startTime), 0)
			}
			h.respondError(w, http.StatusInternalServerError, "failed to serialize configuration")
			return
		}
		contentType = "text/yaml"

	case "json", "":
		// Default to JSON
		response := ConfigExportResponse{
			Status: "success",
			Data: &ConfigData{
				Version:        configResp.Version,
				Source:         string(configResp.Source),
				LoadedAt:       configResp.LoadedAt.Format(time.RFC3339),
				ConfigFilePath: configResp.ConfigFilePath,
				Config:         configResp.Config,
			},
		}

		responseBody, err = json.Marshal(response)
		if err != nil {
			h.logger.Error("Failed to serialize JSON", "error", err)
			if h.metrics != nil {
				h.metrics.RecordError("serialization")
				h.metrics.RecordRequest(opts.Format, opts.Sanitize, "error", time.Since(startTime), 0)
			}
			h.respondError(w, http.StatusInternalServerError, "failed to serialize configuration")
			return
		}
		contentType = "application/json"

	default:
		h.logger.Warn("Invalid format requested", "format", opts.Format)
		if h.metrics != nil {
			h.metrics.RecordError("validation")
			h.metrics.RecordRequest(opts.Format, opts.Sanitize, "error", time.Since(startTime), 0)
		}
		h.respondError(w, http.StatusBadRequest, fmt.Sprintf("invalid format: %s (supported: json, yaml)", opts.Format))
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Config-Version", configResp.Version)
	w.Header().Set("X-Config-Source", string(configResp.Source))

	// Write response
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(responseBody); err != nil {
		h.logger.Error("Failed to write response", "error", err)
		return
	}

	// Record metrics and log success
	duration := time.Since(startTime)
	responseSize := len(responseBody)
	if h.metrics != nil {
		h.metrics.RecordRequest(opts.Format, opts.Sanitize, "success", duration, responseSize)
	}

	h.logger.Info("Config exported successfully",
		"format", opts.Format,
		"sanitized", opts.Sanitize,
		"sections", opts.Sections,
		"duration_ms", duration.Milliseconds(),
		"response_size_bytes", responseSize,
		"version", configResp.Version,
	)
}

// parseQueryParameters parses query parameters into GetConfigOptions
func (h *ConfigHandler) parseQueryParameters(r *http.Request) (appconfig.GetConfigOptions, error) {
	opts := appconfig.GetConfigOptions{
		Format:   "json", // Default format
		Sanitize: true,   // Default: sanitize secrets
		Sections: nil,    // Default: all sections
	}

	query := r.URL.Query()

	// Parse format
	if format := query.Get("format"); format != "" {
		format = strings.ToLower(format)
		if format != "json" && format != "yaml" {
			return opts, fmt.Errorf("invalid format: %s (supported: json, yaml)", format)
		}
		opts.Format = format
	}

	// Parse sanitize flag
	if sanitize := query.Get("sanitize"); sanitize != "" {
		opts.Sanitize = sanitize != "false"
	}

	// Parse sections filter
	if sections := query.Get("sections"); sections != "" {
		sectionList := strings.Split(sections, ",")
		opts.Sections = make([]string, 0, len(sectionList))
		for _, s := range sectionList {
			s = strings.TrimSpace(s)
			if s != "" {
				opts.Sections = append(opts.Sections, s)
			}
		}
	}

	return opts, nil
}

// respondError writes an error response
func (h *ConfigHandler) respondError(w http.ResponseWriter, statusCode int, message string) {
	response := ConfigExportResponse{
		Status: "error",
		Error:  message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode error response", "error", err)
	}
}
