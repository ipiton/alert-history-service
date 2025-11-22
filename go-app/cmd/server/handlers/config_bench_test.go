package handlers

import (
	"net/http/httptest"
	"testing"

	appconfig "github.com/vitaliisemenov/alert-history/internal/config"
)

func BenchmarkConfigHandler_HandleGetConfig_JSON(b *testing.B) {
	mockService := &mockConfigService{
		source: appconfig.ConfigSourceFile,
		config: &appconfig.Config{},
	}
	handler := NewConfigHandler(mockService, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v2/config", nil)
		w := httptest.NewRecorder()
		handler.HandleGetConfig(w, req)
	}
}

func BenchmarkConfigHandler_HandleGetConfig_YAML(b *testing.B) {
	mockService := &mockConfigService{
		source: appconfig.ConfigSourceFile,
		config: &appconfig.Config{},
	}
	handler := NewConfigHandler(mockService, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v2/config?format=yaml", nil)
		w := httptest.NewRecorder()
		handler.HandleGetConfig(w, req)
	}
}

func BenchmarkConfigHandler_HandleGetConfig_Sanitized(b *testing.B) {
	mockService := &mockConfigService{
		source: appconfig.ConfigSourceFile,
		config: &appconfig.Config{},
	}
	handler := NewConfigHandler(mockService, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v2/config?sanitize=true", nil)
		w := httptest.NewRecorder()
		handler.HandleGetConfig(w, req)
	}
}

func BenchmarkConfigHandler_HandleGetConfig_Sections(b *testing.B) {
	mockService := &mockConfigService{
		source: appconfig.ConfigSourceFile,
		config: &appconfig.Config{},
	}
	handler := NewConfigHandler(mockService, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v2/config?sections=server,app", nil)
		w := httptest.NewRecorder()
		handler.HandleGetConfig(w, req)
	}
}
