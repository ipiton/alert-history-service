package performance

import (
	"context"
	"log/slog"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Profiler provides application profiling capabilities
type Profiler struct {
	logger           *slog.Logger
	enableCPUProfile bool
	enableMemProfile bool
	cpuProfileFile   string
	memProfileFile   string

	// Metrics
	goroutines     prometheus.Gauge
	memoryAlloc    prometheus.Gauge
	memorySys      prometheus.Gauge
	gcPauseTotal   prometheus.Counter
	gcCount        prometheus.Counter
}

// NewProfiler creates a new profiler
func NewProfiler(logger *slog.Logger) *Profiler {
	if logger == nil {
		logger = slog.Default()
	}

	return &Profiler{
		logger:           logger,
		enableCPUProfile: false, // Enable via config
		enableMemProfile: false,  // Enable via config
		goroutines: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "alert_history",
			Subsystem: "performance",
			Name:      "goroutines",
			Help:      "Number of goroutines",
		}),
		memoryAlloc: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "alert_history",
			Subsystem: "performance",
			Name:      "memory_alloc_bytes",
			Help:      "Bytes allocated and not yet freed",
		}),
		memorySys: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: "alert_history",
			Subsystem: "performance",
			Name:      "memory_sys_bytes",
			Help:      "Bytes obtained from system",
		}),
		gcPauseTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "alert_history",
			Subsystem: "performance",
			Name:      "gc_pause_total_nanoseconds",
			Help:      "Total GC pause time in nanoseconds",
		}),
		gcCount: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: "alert_history",
			Subsystem: "performance",
			Name:      "gc_count_total",
			Help:      "Total number of GC cycles",
		}),
	}
}

// StartCPUProfile starts CPU profiling
func (p *Profiler) StartCPUProfile(ctx context.Context) error {
	if !p.enableCPUProfile {
		return nil
	}

	// CPU profiling would be started here
	// For now, just log
	p.logger.Info("CPU profiling enabled")
	return nil
}

// StopCPUProfile stops CPU profiling
func (p *Profiler) StopCPUProfile() error {
	if !p.enableCPUProfile {
		return nil
	}

	pprof.StopCPUProfile()
	p.logger.Info("CPU profiling stopped")
	return nil
}

// WriteMemProfile writes memory profile
func (p *Profiler) WriteMemProfile() error {
	if !p.enableMemProfile {
		return nil
	}

	// Memory profiling would be written here
	// For now, just update metrics
	p.UpdateMetrics()
	return nil
}

// UpdateMetrics updates Prometheus metrics with current runtime stats
func (p *Profiler) UpdateMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	p.goroutines.Set(float64(runtime.NumGoroutine()))
	p.memoryAlloc.Set(float64(m.Alloc))
	p.memorySys.Set(float64(m.Sys))
	p.gcPauseTotal.Add(float64(m.PauseTotalNs))
	p.gcCount.Add(float64(m.NumGC))
}

// StartMetricsCollection starts periodic metrics collection
func (p *Profiler) StartMetricsCollection(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.UpdateMetrics()
		case <-ctx.Done():
			return
		}
	}
}

// GetRuntimeStats returns current runtime statistics
func (p *Profiler) GetRuntimeStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"goroutines":      runtime.NumGoroutine(),
		"memory_alloc_mb": float64(m.Alloc) / 1024 / 1024,
		"memory_sys_mb":   float64(m.Sys) / 1024 / 1024,
		"gc_count":        m.NumGC,
		"gc_pause_ms":     float64(m.PauseTotalNs) / 1000000,
	}
}
