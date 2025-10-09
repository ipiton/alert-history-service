"""
Legacy metrics adapter maintaining backward compatibility.

Provides same Prometheus metrics as original alert_history_service.py
while supporting multi-pod deployment without metric duplication.
"""

# Standard library imports
from typing import Optional

# Third-party imports
from prometheus_client import CollectorRegistry, Counter, Gauge, Histogram


class LegacyMetrics:
    """
    Legacy metrics collector maintaining backward compatibility.

    Provides exact same metrics as original service while supporting
    multi-pod deployment with proper aggregation.
    """

    def __init__(self, registry: Optional[CollectorRegistry] = None):
        """Initialize legacy metrics."""
        self.registry = registry or CollectorRegistry()

        # Legacy metrics (exact same names as original)
        self._init_legacy_metrics()

        # New metrics for LLM and publishing features
        self._init_new_metrics()

    def _init_legacy_metrics(self) -> None:
        """Initialize legacy metrics with same names as original."""

        # Webhook events counter (main metric from original)
        self.webhook_events_total = Counter(
            "alert_history_webhook_events_total",
            "Total number of webhook events processed",
            ["alertname", "status"],
            registry=self.registry,
        )

        # Request latency histogram
        self.request_latency_seconds = Histogram(
            "alert_history_request_latency_seconds",
            "Request latency in seconds",
            ["endpoint"],
            registry=self.registry,
        )

        # Webhook errors counter
        self.webhook_errors_total = Counter(
            "alert_history_webhook_errors_total",
            "Total number of webhook errors",
            registry=self.registry,
        )

        # Database operations
        self.database_operations_total = Counter(
            "alert_history_database_operations_total",
            "Total database operations",
            ["operation", "status"],
            registry=self.registry,
        )

        # Current alerts in database
        self.alerts_stored_total = Gauge(
            "alert_history_alerts_stored_total",
            "Total alerts currently stored",
            registry=self.registry,
        )

        # Database size
        self.database_size_bytes = Gauge(
            "alert_history_database_size_bytes",
            "Database size in bytes",
            registry=self.registry,
        )

    def _init_new_metrics(self) -> None:
        """Initialize new metrics for LLM and publishing features."""

        # LLM Classification metrics
        self.classification_total = Counter(
            "alert_history_classification_total",
            "Total alerts classified by LLM",
            ["severity", "model"],
            registry=self.registry,
        )

        self.classification_duration_seconds = Histogram(
            "alert_history_classification_duration_seconds",
            "LLM classification duration in seconds",
            ["model"],
            registry=self.registry,
        )

        self.classification_errors_total = Counter(
            "alert_history_classification_errors_total",
            "LLM classification errors",
            ["error_type"],
            registry=self.registry,
        )

        self.classification_confidence = Histogram(
            "alert_history_classification_confidence",
            "LLM classification confidence scores",
            ["severity"],
            registry=self.registry,
        )

        # Publishing metrics
        self.publishing_total = Counter(
            "alert_history_publishing_total",
            "Total alert publishing attempts",
            ["target", "status"],
            registry=self.registry,
        )

        self.publishing_duration_seconds = Histogram(
            "alert_history_publishing_duration_seconds",
            "Alert publishing duration in seconds",
            ["target"],
            registry=self.registry,
        )

        self.publishing_targets_discovered = Gauge(
            "alert_history_publishing_targets_discovered",
            "Number of discovered publishing targets",
            registry=self.registry,
        )

        self.publishing_queue_size = Gauge(
            "alert_history_publishing_queue_size",
            "Current publishing queue size",
            registry=self.registry,
        )

        # Enrichment mode metrics
        self.enrichment_mode_switches = Counter(
            "alert_history_enrichment_mode_switches_total",
            "Total enrichment mode switches",
            ["from_mode", "to_mode"],
            registry=self.registry,
        )

        self.enrichment_mode_status = Gauge(
            "alert_history_enrichment_mode_status",
            "Current enrichment mode (0=transparent, 1=enriched, 2=transparent_with_recommendations)",
            registry=self.registry,
        )

        self.enrichment_mode_requests = Counter(
            "alert_history_enrichment_mode_requests_total",
            "Total enrichment mode API requests",
            ["method", "mode"],
            registry=self.registry,
        )

        self.enrichment_transparent_alerts = Counter(
            "alert_history_enrichment_transparent_alerts_total",
            "Alerts processed in transparent mode (no LLM)",
            registry=self.registry,
        )

        self.enrichment_enriched_alerts = Counter(
            "alert_history_enrichment_enriched_alerts_total",
            "Alerts processed in enriched mode (with LLM)",
            registry=self.registry,
        )

        # Cache metrics
        self.cache_hits_total = Counter(
            "alert_history_cache_hits_total",
            "Cache hits",
            ["cache_type"],
            registry=self.registry,
        )

        self.cache_misses_total = Counter(
            "alert_history_cache_misses_total",
            "Cache misses",
            ["cache_type"],
            registry=self.registry,
        )

        # Health metrics
        self.service_started_time = Gauge(
            "alert_history_service_started_time",
            "Unix timestamp when service started",
            registry=self.registry,
        )

        self.active_connections = Gauge(
            "alert_history_active_connections",
            "Number of active connections",
            registry=self.registry,
        )

    # Legacy metric methods (exact same interface as original)

    def increment_alerts_received(self, alertname: str, status: str) -> None:
        """Increment alerts received counter (legacy compatibility)."""
        self.webhook_events_total.labels(alertname=alertname, status=status).inc()

    def increment_webhook_errors(self) -> None:
        """Increment webhook errors counter (legacy compatibility)."""
        self.webhook_errors_total.inc()

    def observe_webhook_duration(self, duration: float) -> None:
        """Observe webhook processing duration (legacy compatibility)."""
        self.request_latency_seconds.labels(endpoint="webhook").observe(duration)

    def observe_history_duration(self, duration: float) -> None:
        """Observe history request duration (legacy compatibility)."""
        self.request_latency_seconds.labels(endpoint="history").observe(duration)

    def observe_report_duration(self, duration: float) -> None:
        """Observe report request duration (legacy compatibility)."""
        self.request_latency_seconds.labels(endpoint="report").observe(duration)

    def set_alerts_stored(self, count: int) -> None:
        """Set current alerts stored count (legacy compatibility)."""
        self.alerts_stored_total.set(count)

    def set_database_size(self, size_bytes: int) -> None:
        """Set database size (legacy compatibility)."""
        self.database_size_bytes.set(size_bytes)

    # New metric methods for LLM and publishing

    def increment_classification(self, severity: str, model: str = "unknown") -> None:
        """Increment classification counter."""
        self.classification_total.labels(severity=severity, model=model).inc()

    def observe_classification_duration(
        self, duration: float, model: str = "unknown"
    ) -> None:
        """Observe classification duration."""
        self.classification_duration_seconds.labels(model=model).observe(duration)

    def increment_classification_error(self, error_type: str) -> None:
        """Increment classification error counter."""
        self.classification_errors_total.labels(error_type=error_type).inc()

    def set_enrichment_mode(self, mode: str) -> None:
        """Set enrichment mode status."""
        mode_map = {
            "transparent": 0,
            "enriched": 1,
            "transparent_with_recommendations": 2,
        }
        self.enrichment_mode_status.set(mode_map.get(mode, 0))

    def increment_classifications(
        self, severity: str, cached: bool = False, model: str = "unknown"
    ) -> None:
        """Increment classification counter (alias for legacy compatibility)."""
        self.increment_classification(severity=severity, model=model)
        if cached:
            self.increment_cache_hit("classification")
        else:
            self.increment_cache_miss("classification")

    def increment_classification_errors(self) -> None:
        """Increment classification errors (alias for legacy compatibility)."""
        self.increment_classification_error("general")

    def observe_classification_confidence(
        self, confidence: float, severity: str
    ) -> None:
        """Observe classification confidence score."""
        self.classification_confidence.labels(severity=severity).observe(confidence)

    def increment_publishing(self, target: str, success: bool) -> None:
        """Increment publishing counter."""
        status = "success" if success else "failure"
        self.publishing_total.labels(target=target, status=status).inc()

    def observe_publishing_duration(self, duration: float, target: str) -> None:
        """Observe publishing duration."""
        self.publishing_duration_seconds.labels(target=target).observe(duration)

    def set_publishing_targets_count(self, count: int) -> None:
        """Set number of discovered publishing targets."""
        self.publishing_targets_discovered.set(count)

    def set_publishing_queue_size(self, size: int) -> None:
        """Set publishing queue size."""
        self.publishing_queue_size.set(size)

    def increment_cache_hit(self, cache_type: str) -> None:
        """Increment cache hit counter."""
        self.cache_hits_total.labels(cache_type=cache_type).inc()

    def increment_cache_miss(self, cache_type: str) -> None:
        """Increment cache miss counter."""
        self.cache_misses_total.labels(cache_type=cache_type).inc()

    def set_service_start_time(self, timestamp: float) -> None:
        """Set service start time."""
        self.service_started_time.set(timestamp)

    def set_active_connections(self, count: int) -> None:
        """Set active connections count."""
        self.active_connections.set(count)

    def set_gauge(
        self, name: str, value: float, labels: Optional[dict[str, str]] = None
    ) -> None:
        """Set gauge metric (generic interface)."""
        # For now, we'll map to existing gauges or create a generic one
        # This is a fallback implementation
        if name == "alerts_stored":
            self.set_alerts_stored(int(value))
        elif name == "database_size":
            self.set_database_size(int(value))
        elif name == "active_connections":
            self.set_active_connections(int(value))
        else:
            # For unknown metrics, we could create a dynamic gauge
            # but for now, just log and ignore
            pass

    def increment_counter(
        self, name: str, value: float = 1.0, labels: Optional[dict[str, str]] = None
    ) -> None:
        """Increment counter metric (generic interface)."""
        # Map to existing counters or create fallback
        if name == "classification":
            severity = labels.get("severity", "unknown") if labels else "unknown"
            model = labels.get("model", "unknown") if labels else "unknown"
            self.increment_classification(severity, model)
        elif name == "webhook_events":
            alertname = labels.get("alertname", "unknown") if labels else "unknown"
            status = labels.get("status", "unknown") if labels else "unknown"
            self.increment_alerts_received(alertname, status)
        elif name == "webhook_errors":
            self.increment_webhook_errors()
        else:
            # For unknown counters, just log and ignore for now
            pass

    def observe_histogram(
        self, name: str, value: float, labels: Optional[dict[str, str]] = None
    ) -> None:
        """Observe histogram metric (generic interface)."""
        # Map to existing histograms
        if name == "classification_duration":
            model = labels.get("model", "unknown") if labels else "unknown"
            self.observe_classification_duration(value, model)
        elif name == "webhook_duration":
            self.observe_webhook_duration(value)
        elif name == "request_latency":
            endpoint = labels.get("endpoint", "webhook") if labels else "webhook"
            if endpoint == "webhook":
                self.observe_webhook_duration(value)
            elif endpoint == "history":
                self.observe_history_duration(value)
            elif endpoint == "report":
                self.observe_report_duration(value)
        else:
            # For unknown histograms, just log and ignore for now
            pass


class MultiPodMetricsCompatibility:
    """
    Compatibility layer for multi-pod metrics.

    Ensures metrics are properly aggregated across pods
    and legacy dashboard queries continue to work.
    """

    def __init__(self, metrics: LegacyMetrics):
        """Initialize compatibility layer."""
        self.metrics = metrics

    def get_legacy_metric_help(self) -> dict:
        """Get help text for legacy metrics compatibility."""
        return {
            "alert_history_webhook_events_total": {
                "help": "Total number of webhook events processed (aggregated across all pods)",
                "type": "counter",
                "aggregation": "sum",
            },
            "alert_history_request_latency_seconds": {
                "help": "Request latency in seconds (histogram across all pods)",
                "type": "histogram",
                "aggregation": "histogram_quantile",
            },
            "alert_history_webhook_errors_total": {
                "help": "Total number of webhook errors (aggregated across all pods)",
                "type": "counter",
                "aggregation": "sum",
            },
            "alert_history_alerts_stored_total": {
                "help": "Total alerts currently stored (shared database across pods)",
                "type": "gauge",
                "aggregation": "max",  # Same for all pods since shared DB
            },
            "alert_history_database_size_bytes": {
                "help": "Database size in bytes (shared database across pods)",
                "type": "gauge",
                "aggregation": "max",  # Same for all pods since shared DB
            },
        }

    def get_prometheus_aggregation_rules(self) -> str:
        """
        Get Prometheus recording rules for proper multi-pod aggregation.

        These rules ensure backward compatibility with existing dashboards
        while supporting horizontal scaling.
        """
        return """
# Multi-pod aggregation rules for Alert History Service
groups:
  - name: alert_history_aggregation
    interval: 30s
    rules:
      # Legacy compatibility aggregations
      - record: alert_history:webhook_events_rate
        expr: sum(rate(alert_history_webhook_events_total[5m])) by (alertname, status)

      - record: alert_history:request_latency_p95
        expr: histogram_quantile(0.95, sum(rate(alert_history_request_latency_seconds_bucket[5m])) by (le, endpoint))

      - record: alert_history:request_latency_p50
        expr: histogram_quantile(0.50, sum(rate(alert_history_request_latency_seconds_bucket[5m])) by (le, endpoint))

      - record: alert_history:total_errors_rate
        expr: sum(rate(alert_history_webhook_errors_total[5m]))

      - record: alert_history:top_noisy_alerts_24h
        expr: topk(10, sum by (alertname) (increase(alert_history_webhook_events_total[24h])))

      # New LLM and publishing metrics
      - record: alert_history:classification_rate
        expr: sum(rate(alert_history_classification_total[5m])) by (severity)

      - record: alert_history:classification_success_rate
        expr: |
          sum(rate(alert_history_classification_total[5m])) /
          (sum(rate(alert_history_classification_total[5m])) + sum(rate(alert_history_classification_errors_total[5m])))

      - record: alert_history:publishing_success_rate
        expr: |
          sum(rate(alert_history_publishing_total{status="success"}[5m])) by (target) /
          sum(rate(alert_history_publishing_total[5m])) by (target)

      # Enrichment mode metrics
      - record: alert_history:enrichment_mode_current
        expr: max(alert_history_enrichment_mode_status)

      - record: alert_history:enrichment_transparent_rate
        expr: sum(rate(alert_history_enrichment_transparent_alerts_total[5m]))

      - record: alert_history:enrichment_enriched_rate
        expr: sum(rate(alert_history_enrichment_enriched_alerts_total[5m]))

      - record: alert_history:enrichment_mode_switch_rate
        expr: sum(rate(alert_history_enrichment_mode_switches_total[5m]))

      - record: alert_history:enrichment_efficiency
        expr: |
          sum(rate(alert_history_enrichment_enriched_alerts_total[5m])) /
          (sum(rate(alert_history_enrichment_transparent_alerts_total[5m])) + sum(rate(alert_history_enrichment_enriched_alerts_total[5m])))

      # Health and performance metrics
      - record: alert_history:active_pods
        expr: count(up{job="alert-history"} == 1)

      - record: alert_history:pod_load_balance
        expr: |
          max(sum(rate(alert_history_webhook_events_total[5m])) by (pod)) -
          min(sum(rate(alert_history_webhook_events_total[5m])) by (pod))
"""


# Global metrics instance for dependency injection
_global_metrics: Optional[LegacyMetrics] = None


def get_metrics() -> LegacyMetrics:
    """
    Get global metrics instance for dependency injection.

    Returns:
        LegacyMetrics: Global metrics instance
    """
    global _global_metrics

    if _global_metrics is None:
        _global_metrics = LegacyMetrics()

    return _global_metrics


def set_global_metrics(metrics: LegacyMetrics) -> None:
    """
    Set global metrics instance.

    Args:
        metrics: Metrics instance to set globally
    """
    global _global_metrics
    _global_metrics = metrics
