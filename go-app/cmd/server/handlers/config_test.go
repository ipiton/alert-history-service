package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appconfig "github.com/vitaliisemenov/alert-history/internal/config"
)

func TestConfigHandler_HandleGetConfig_JSON(t *testing.T) {
	mockService := &mockConfigService{
		source: appconfig.ConfigSourceFile,
		config: &appconfig.Config{},
	}
	handler := NewConfigHandler(mockService, nil)

	req := httptest.NewRequest("GET", "/api/v2/config", nil)
	w := httptest.NewRecorder()

	handler.HandleGetConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleGetConfig() status = %v, want %v", w.Code, http.StatusOK)
	}

	if !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
		t.Errorf("Content-Type = %v, want application/json", w.Header().Get("Content-Type"))
	}

	var response ConfigExportResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Response status = %v, want success", response.Status)
	}

	if response.Data == nil {
		t.Error("Response data is nil")
	}

	if response.Data.Version != "test-version-123" {
		t.Errorf("Version = %v, want test-version-123", response.Data.Version)
	}
}

func TestConfigHandler_HandleGetConfig_YAML(t *testing.T) {
	mockService := &mockConfigService{
		source: appconfig.ConfigSourceFile,
		config: &appconfig.Config{},
	}
	handler := NewConfigHandler(mockService, nil)

	req := httptest.NewRequest("GET", "/api/v2/config?format=yaml", nil)
	w := httptest.NewRecorder()

	handler.HandleGetConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleGetConfig() status = %v, want %v", w.Code, http.StatusOK)
	}

	if !strings.Contains(w.Header().Get("Content-Type"), "yaml") {
		t.Errorf("Content-Type = %v, want text/yaml", w.Header().Get("Content-Type"))
	}
}

func TestConfigHandler_HandleGetConfig_InvalidMethod(t *testing.T) {
	mockService := &mockConfigService{
		source: appconfig.ConfigSourceDefaults,
		config: &appconfig.Config{},
	}
	handler := NewConfigHandler(mockService, nil)

	req := httptest.NewRequest("POST", "/api/v2/config", nil)
	w := httptest.NewRecorder()

	handler.HandleGetConfig(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("HandleGetConfig() status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
	}

	var response ConfigExportResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Status != "error" {
		t.Errorf("Response status = %v, want error", response.Status)
	}
}

func TestConfigHandler_HandleGetConfig_InvalidFormat(t *testing.T) {
	mockService := &mockConfigService{
		source: appconfig.ConfigSourceDefaults,
		config: &appconfig.Config{},
	}
	handler := NewConfigHandler(mockService, nil)

	req := httptest.NewRequest("GET", "/api/v2/config?format=xml", nil)
	w := httptest.NewRecorder()

	handler.HandleGetConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("HandleGetConfig() status = %v, want %v", w.Code, http.StatusBadRequest)
	}

	var response ConfigExportResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Status != "error" {
		t.Errorf("Response status = %v, want error", response.Status)
	}
}

func TestConfigHandler_parseQueryParameters(t *testing.T) {
	handler := &ConfigHandler{}

	tests := []struct {
		name    string
		url     string
		wantErr bool
		check   func(*appconfig.GetConfigOptions) bool
	}{
		{
			name:    "Default parameters",
			url:     "/api/v2/config",
			wantErr: false,
			check: func(opts *appconfig.GetConfigOptions) bool {
				return opts.Format == "json" && opts.Sanitize == true
			},
		},
		{
			name:    "YAML format",
			url:     "/api/v2/config?format=yaml",
			wantErr: false,
			check: func(opts *appconfig.GetConfigOptions) bool {
				return opts.Format == "yaml"
			},
		},
		{
			name:    "Unsanitized",
			url:     "/api/v2/config?sanitize=false",
			wantErr: false,
			check: func(opts *appconfig.GetConfigOptions) bool {
				return opts.Sanitize == false
			},
		},
		{
			name:    "Section filtering",
			url:     "/api/v2/config?sections=server,database",
			wantErr: false,
			check: func(opts *appconfig.GetConfigOptions) bool {
				return len(opts.Sections) == 2 &&
					opts.Sections[0] == "server" &&
					opts.Sections[1] == "database"
			},
		},
		{
			name:    "Invalid format",
			url:     "/api/v2/config?format=xml",
			wantErr: true,
			check:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			opts, err := handler.parseQueryParameters(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseQueryParameters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil && !tt.check(&opts) {
				t.Error("parseQueryParameters() returned unexpected options")
			}
		})
	}
}
