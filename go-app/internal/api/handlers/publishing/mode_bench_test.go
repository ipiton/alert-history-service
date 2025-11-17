package publishing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apiservices "github.com/vitaliisemenov/alert-history/internal/api/services/publishing"
)

// BenchmarkGetPublishingMode benchmarks the overall handler performance
func BenchmarkGetPublishingMode(b *testing.B) {
	mockService := &mockModeService{
		modeInfo: &apiservices.ModeInfo{
			Mode:              "normal",
			TargetsAvailable:  true,
			EnabledTargets:    5,
			MetricsOnlyActive: false,
			TransitionCount:   12,
		},
	}
	handler := NewPublishingModeHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		w.Code = 0
		w.HeaderMap = make(http.Header)
		handler.GetPublishingMode(w, req)
	}
}

// BenchmarkGetPublishingMode_Cached benchmarks with ModeManager caching (fast path)
func BenchmarkGetPublishingMode_Cached(b *testing.B) {
	mockService := &mockModeService{
		modeInfo: &apiservices.ModeInfo{
			Mode:              "normal",
			TargetsAvailable:  true,
			EnabledTargets:    5,
			MetricsOnlyActive: false,
			TransitionCount:   12,
		},
	}
	handler := NewPublishingModeHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		w.Code = 0
		w.HeaderMap = make(http.Header)
		handler.GetPublishingMode(w, req)
	}
}

// BenchmarkGetPublishingMode_Fallback benchmarks fallback mode detection
func BenchmarkGetPublishingMode_Fallback(b *testing.B) {
	mockService := &mockModeService{
		modeInfo: &apiservices.ModeInfo{
			Mode:              "normal",
			TargetsAvailable:  true,
			EnabledTargets:    5,
			MetricsOnlyActive: false,
			// Enhanced fields omitted (fallback mode)
		},
	}
	handler := NewPublishingModeHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		w.Code = 0
		w.HeaderMap = make(http.Header)
		handler.GetPublishingMode(w, req)
	}
}

// BenchmarkGetPublishingMode_Parallel benchmarks concurrent requests
func BenchmarkGetPublishingMode_Parallel(b *testing.B) {
	mockService := &mockModeService{
		modeInfo: &apiservices.ModeInfo{
			Mode:              "normal",
			TargetsAvailable:  true,
			EnabledTargets:    5,
			MetricsOnlyActive: false,
		},
	}
	handler := NewPublishingModeHandler(mockService, nil)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
		w := httptest.NewRecorder()

		for pb.Next() {
			w.Body.Reset()
			w.Code = 0
			w.HeaderMap = make(http.Header)
			handler.GetPublishingMode(w, req)
		}
	})
}

// BenchmarkGetPublishingMode_ConditionalRequest benchmarks conditional requests (304)
func BenchmarkGetPublishingMode_ConditionalRequest(b *testing.B) {
	mockService := &mockModeService{
		modeInfo: &apiservices.ModeInfo{
			Mode:            "normal",
			EnabledTargets:  5,
			TransitionCount: 12,
		},
	}
	handler := NewPublishingModeHandler(mockService, nil)

	// First request to get ETag
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	w1 := httptest.NewRecorder()
	handler.GetPublishingMode(w1, req1)
	etag := w1.Header().Get("ETag")

	// Subsequent requests with If-None-Match
	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	req.Header.Set("If-None-Match", etag)
	w := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		w.Code = 0
		w.HeaderMap = make(http.Header)
		handler.GetPublishingMode(w, req)
	}
}

// BenchmarkGenerateETag benchmarks ETag generation
func BenchmarkGenerateETag(b *testing.B) {
	handler := NewPublishingModeHandler(nil, nil)
	modeInfo := &apiservices.ModeInfo{
		Mode:            "normal",
		EnabledTargets:  5,
		TransitionCount: 12,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = handler.generateETag(modeInfo)
	}
}

// BenchmarkJSONEncoding benchmarks JSON encoding performance
func BenchmarkJSONEncoding(b *testing.B) {
	handler := NewPublishingModeHandler(nil, nil)
	modeInfo := &apiservices.ModeInfo{
		Mode:              "normal",
		TargetsAvailable:  true,
		EnabledTargets:    5,
		MetricsOnlyActive: false,
		TransitionCount:   12,
		CurrentModeDurationSeconds: 3600.5,
		LastTransitionTime:        time.Now(),
		LastTransitionReason:      "targets_available",
	}

	w := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w.Body.Reset()
		w.HeaderMap = make(http.Header)
		handler.sendJSON(w, http.StatusOK, modeInfo)
	}
}

// BenchmarkService_GetCurrentModeInfo benchmarks service layer performance
func BenchmarkService_GetCurrentModeInfo(b *testing.B) {
	mockService := &mockModeService{
		modeInfo: &apiservices.ModeInfo{
			Mode:              "normal",
			TargetsAvailable:  true,
			EnabledTargets:    5,
			MetricsOnlyActive: false,
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = mockService.GetCurrentModeInfo(ctx)
	}
}

