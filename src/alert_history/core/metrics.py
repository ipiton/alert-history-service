"""
Упрощенная система метрик без LegacyMetrics.

Простая система метрик для мониторинга сервиса.
"""

from typing import Any, Dict, Optional

from prometheus_client import CollectorRegistry, Counter, Gauge, Histogram

from ..logging_config import get_logger

logger = get_logger(__name__)


class SimpleMetrics:
    """
    Упрощенная система метрик.

    Заменяет LegacyMetrics более простым и понятным подходом.
    """

    def __init__(self, registry: Optional[CollectorRegistry] = None):
        """Инициализация метрик."""
        self.registry = registry or CollectorRegistry()
        self._init_metrics()

    def _init_metrics(self) -> None:
        """Инициализация всех метрик."""

        # Основные метрики webhook
        self.webhook_events_total = Counter(
            "alert_history_webhook_events_total",
            "Total webhook events processed",
            ["alertname", "status"],
            registry=self.registry,
        )

        self.webhook_errors_total = Counter(
            "alert_history_webhook_errors_total",
            "Total webhook errors",
            registry=self.registry,
        )

        # Метрики обработки
        self.request_duration_seconds = Histogram(
            "alert_history_request_duration_seconds",
            "Request duration in seconds",
            ["endpoint"],
            registry=self.registry,
        )

        # Метрики LLM
        self.llm_classifications_total = Counter(
            "alert_history_llm_classifications_total",
            "Total LLM classifications",
            ["severity", "model"],
            registry=self.registry,
        )

        self.llm_errors_total = Counter(
            "alert_history_llm_errors_total",
            "Total LLM errors",
            ["error_type"],
            registry=self.registry,
        )

        # Метрики режимов обогащения
        self.enrichment_transparent_alerts = Counter(
            "alert_history_enrichment_transparent_alerts_total",
            "Alerts processed in transparent mode",
            registry=self.registry,
        )

        self.enrichment_enriched_alerts = Counter(
            "alert_history_enrichment_enriched_alerts_total",
            "Alerts processed in enriched mode",
            registry=self.registry,
        )

        # Метрики публикации
        self.publishing_total = Counter(
            "alert_history_publishing_total",
            "Total publishing attempts",
            ["target", "status"],
            registry=self.registry,
        )

        # Метрики состояния
        self.alerts_stored = Gauge(
            "alert_history_alerts_stored_total",
            "Total alerts currently stored",
            registry=self.registry,
        )

        self.enrichment_mode = Gauge(
            "alert_history_enrichment_mode",
            "Current enrichment mode (0=transparent, 1=enriched, 2=transparent_with_recommendations)",
            registry=self.registry,
        )

    # Простые методы для обновления метрик

    def increment_webhook_events(self, alertname: str, status: str) -> None:
        """Увеличить счетчик webhook событий."""
        self.webhook_events_total.labels(alertname=alertname, status=status).inc()

    def increment_webhook_errors(self) -> None:
        """Увеличить счетчик ошибок webhook."""
        self.webhook_errors_total.inc()

    def observe_request_duration(self, endpoint: str, duration: float) -> None:
        """Записать длительность запроса."""
        self.request_duration_seconds.labels(endpoint=endpoint).observe(duration)

    def increment_llm_classifications(
        self, severity: str, model: str = "unknown"
    ) -> None:
        """Увеличить счетчик LLM классификаций."""
        self.llm_classifications_total.labels(severity=severity, model=model).inc()

    def increment_llm_errors(self, error_type: str) -> None:
        """Увеличить счетчик ошибок LLM."""
        self.llm_errors_total.labels(error_type=error_type).inc()

    def increment_transparent_alerts(self, count: int = 1) -> None:
        """Увеличить счетчик алертов в transparent режиме."""
        self.enrichment_transparent_alerts.inc(count)

    def increment_enriched_alerts(self, count: int = 1) -> None:
        """Увеличить счетчик алертов в enriched режиме."""
        self.enrichment_enriched_alerts.inc(count)

    def increment_publishing(self, target: str, status: str) -> None:
        """Увеличить счетчик публикаций."""
        self.publishing_total.labels(target=target, status=status).inc()

    def set_alerts_stored(self, count: int) -> None:
        """Установить количество сохраненных алертов."""
        self.alerts_stored.set(count)

    def set_enrichment_mode(self, mode: str) -> None:
        """Установить режим обогащения."""
        mode_map = {
            "transparent": 0,
            "enriched": 1,
            "transparent_with_recommendations": 2,
        }
        self.enrichment_mode.set(mode_map.get(mode, 0))


# Глобальный экземпляр метрик
_global_metrics: Optional[SimpleMetrics] = None


def get_metrics() -> SimpleMetrics:
    """Получить глобальный экземпляр метрик."""
    global _global_metrics

    if _global_metrics is None:
        _global_metrics = SimpleMetrics()
        logger.info("SimpleMetrics initialized")

    return _global_metrics


def set_metrics(metrics: SimpleMetrics) -> None:
    """Установить глобальный экземпляр метрик."""
    global _global_metrics
    _global_metrics = metrics
    logger.info("SimpleMetrics set globally")
