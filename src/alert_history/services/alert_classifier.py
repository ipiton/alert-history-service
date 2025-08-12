"""
Alert Classifier Service для интеллектуальной классификации алертов.

Интегрирует LLM classification с кэшированием и storage:
- Умная классификация через LLM Proxy
- Кэширование результатов для производительности
- Сохранение классификаций в базу данных
- Метрики и мониторинг всех операций
- Fallback стратегии при сбоях LLM
"""

# Standard library imports
import asyncio
import time
from typing import Any, Optional

# Local imports
from ..core.base_classes import BaseClassificationService
from ..core.interfaces import (
    Alert,
    AlertSeverity,
    ClassificationResult,
    IAlertStorage,
    ICache,
    ILLMClient,
    IMetricsCollector,
)
from ..logging_config import get_alert_logger, get_performance_logger
from ..utils.common import normalize_severity
from ..utils.decorators import measure_time

alert_logger = get_alert_logger()
performance_logger = get_performance_logger()


class AlertClassificationService(BaseClassificationService):
    """
    Высокоуровневый сервис классификации алертов.

    Объединяет LLM classification, caching, storage и monitoring
    в единый сервис с SOLID архитектурой.
    """

    def __init__(
        self,
        llm_client: ILLMClient,
        storage: IAlertStorage,
        metrics: Optional[IMetricsCollector] = None,
        cache: Optional[ICache] = None,
        cache_ttl: int = 3600,
        enable_fallback: bool = True,
    ):
        """Initialize alert classification service."""
        super().__init__(llm_client, storage, metrics, cache)

        self.cache_ttl = cache_ttl
        self.enable_fallback = enable_fallback

        # Classification statistics
        self._classification_stats = {
            "total_requests": 0,
            "cache_hits": 0,
            "llm_requests": 0,
            "fallback_used": 0,
            "errors": 0,
        }

    async def _do_initialize(self) -> None:
        """Initialize LLM client and connections."""
        # Initialize LLM client if it supports async context
        if hasattr(self.llm_client, "__aenter__"):
            await self.llm_client.__aenter__()

        alert_logger.info(
            "Alert Classification Service initialized",
            cache_enabled=self.cache is not None,
            cache_ttl=self.cache_ttl,
            fallback_enabled=self.enable_fallback,
        )

    async def _do_shutdown(self) -> None:
        """Cleanup LLM client connections."""
        if hasattr(self.llm_client, "__aexit__"):
            await self.llm_client.__aexit__(None, None, None)

        alert_logger.info(
            "Alert Classification Service shutdown", **self._classification_stats
        )

    @measure_time()
    async def classify_alert(
        self,
        alert: Alert,
        context: Optional[dict[str, Any]] = None,
        force_refresh: bool = False,
    ) -> ClassificationResult:
        """
        Классифицировать алерт с умным кэшированием.

        Args:
            alert: Alert для классификации
            context: Дополнительный контекст
            force_refresh: Принудительно обновить кэш

        Returns:
            ClassificationResult с результатом классификации
        """
        start_time = time.time()
        self._classification_stats["total_requests"] += 1

        try:
            # 1. Попробовать получить из кэша (если не force_refresh)
            if not force_refresh:
                cached_result = await self._get_from_cache(alert)
                if cached_result:
                    self._classification_stats["cache_hits"] += 1
                    self._record_metric("classification_cache_hits", 1)

                    alert_logger.info(
                        "Alert classification from cache",
                        alert_name=alert.alert_name,
                        fingerprint=alert.fingerprint,
                        severity=cached_result.severity.value,
                        confidence=cached_result.confidence,
                        source="cache",
                    )

                    return cached_result

            # 2. Классификация через LLM
            result = await self._classify_with_llm(alert, context)

            # 3. Сохранить в кэш и storage
            await self._save_classification_result(alert, result)

            # 4. Обновить метрики
            self._update_classification_metrics(result, time.time() - start_time)

            alert_logger.alert_classified(
                alert_name=alert.alert_name,
                fingerprint=alert.fingerprint,
                severity=result.severity.value,
                confidence=result.confidence,
                processing_time=result.processing_time,
            )

            return result

        except Exception as e:
            self._classification_stats["errors"] += 1
            self._record_metric(
                "classification_errors", 1, {"error_type": type(e).__name__}
            )

            # Fallback стратегия
            if self.enable_fallback:
                fallback_result = await self._create_fallback_result(alert, str(e))
                self._classification_stats["fallback_used"] += 1

                alert_logger.error(
                    "Classification failed, using fallback",
                    alert_name=alert.alert_name,
                    fingerprint=alert.fingerprint,
                    error=str(e),
                    fallback_severity=fallback_result.severity.value,
                )

                return fallback_result

            # Если fallback отключен - пробрасываем ошибку
            alert_logger.exception(
                "Alert classification failed",
                alert_name=alert.alert_name,
                fingerprint=alert.fingerprint,
                error=str(e),
            )
            raise

    async def get_classification_history(
        self, fingerprint: str
    ) -> Optional[ClassificationResult]:
        """
        Получить сохраненную классификацию по fingerprint.

        Args:
            fingerprint: Fingerprint алерта

        Returns:
            ClassificationResult если найден, иначе None
        """
        try:
            # Сначала проверяем кэш
            cache_key = self._generate_classification_cache_key_by_fingerprint(
                fingerprint
            )
            if self.cache:
                cached_result = await self.cache.get(cache_key)
                if cached_result:
                    return cached_result

            # Если в кэше нет - ищем в storage
            if hasattr(self.storage, "get_classification"):
                return await self.storage.get_classification(fingerprint)

            return None

        except Exception as e:
            alert_logger.error(
                "Failed to get classification history",
                fingerprint=fingerprint,
                error=str(e),
            )
            return None

    async def get_classification_stats(self) -> dict[str, Any]:
        """Получить статистику классификации."""
        stats = self._classification_stats.copy()

        # Добавляем вычисляемые метрики
        total_requests = stats["total_requests"]
        if total_requests > 0:
            stats["cache_hit_rate"] = stats["cache_hits"] / total_requests
            stats["llm_usage_rate"] = stats["llm_requests"] / total_requests
            stats["fallback_rate"] = stats["fallback_used"] / total_requests
            stats["error_rate"] = stats["errors"] / total_requests

        return stats

    async def bulk_classify_alerts(
        self,
        alerts: list[Alert],
        context: Optional[dict[str, Any]] = None,
        max_parallel: int = 5,
    ) -> list[ClassificationResult]:
        """
        Массовая классификация алертов с контролем параллелизма.

        Args:
            alerts: Список алертов для классификации
            context: Общий контекст для всех алертов
            max_parallel: Максимальное количество параллельных запросов

        Returns:
            Список результатов классификации в том же порядке
        """
        if not alerts:
            return []

        alert_logger.info(
            "Starting bulk classification",
            count=len(alerts),
            max_parallel=max_parallel,
        )

        # Создаем семафор для ограничения параллелизма
        semaphore = asyncio.Semaphore(max_parallel)

        async def classify_with_semaphore(alert: Alert) -> ClassificationResult:
            async with semaphore:
                return await self.classify_alert(alert, context)

        try:
            # Запускаем классификацию параллельно
            results = await asyncio.gather(
                *[classify_with_semaphore(alert) for alert in alerts],
                return_exceptions=True,
            )

            # Обрабатываем результаты и исключения
            processed_results = []
            errors = 0

            for i, result in enumerate(results):
                if isinstance(result, Exception):
                    alert_logger.error(
                        "Bulk classification error",
                        alert_index=i,
                        alert_fingerprint=alerts[i].fingerprint,
                        error=str(result),
                    )
                    # Создаем fallback результат
                    fallback_result = self._create_fallback_result(alerts[i])
                    processed_results.append(fallback_result)
                    errors += 1
                else:
                    processed_results.append(result)

            alert_logger.info(
                "Bulk classification completed",
                total=len(alerts),
                errors=errors,
                success_rate=(len(alerts) - errors) / len(alerts),
            )

            return processed_results

        except Exception as e:
            alert_logger.error("Bulk classification failed", error=str(e))
            # Возвращаем fallback результаты для всех алертов
            return [self._create_fallback_result(alert) for alert in alerts]

    async def refresh_classification(
        self,
        fingerprint: str,
        context: Optional[dict[str, Any]] = None,
    ) -> ClassificationResult:
        """
        Принудительно обновить классификацию алерта.

        Args:
            fingerprint: Отпечаток алерта
            context: Контекст для классификации

        Returns:
            Новый результат классификации
        """
        # Очищаем кэш для данного fingerprint
        if self.cache:
            cache_key = f"classification:{fingerprint}"
            await self.cache.delete(cache_key)

        # Получаем алерт из storage
        alert = await self.storage.get_alert(fingerprint)
        if not alert:
            raise ValueError(f"Alert not found: {fingerprint}")

        # Выполняем новую классификацию
        return await self.classify_alert(alert, context, force_refresh=True)

    async def _classify_with_llm(
        self, alert: Alert, context: Optional[dict[str, Any]]
    ) -> ClassificationResult:
        """Выполнить классификацию через LLM."""
        self._classification_stats["llm_requests"] += 1

        # Подготовить контекст для LLM
        enhanced_context = await self._prepare_llm_context(alert, context)

        # Классификация через LLM
        result = await self.llm_client.classify_alert(alert, enhanced_context)

        # Валидация и нормализация результата
        validated_result = await self._validate_classification_result(alert, result)

        performance_logger.llm_request(
            model=getattr(self.llm_client, "model", "unknown"),
            operation="classification",
            success=True,
            processing_time=result.processing_time,
            tokens_used=(
                result.metadata.get("llm_usage", {}).get("total_tokens")
                if result.metadata
                else None
            ),
        )

        return validated_result

    async def _prepare_llm_context(
        self, alert: Alert, context: Optional[dict[str, Any]]
    ) -> dict[str, Any]:
        """Подготовить расширенный контекст для LLM."""
        enhanced_context = {
            "timestamp": time.time(),
            "service_version": "1.0.0",
        }

        # Добавляем базовый контекст
        if context:
            enhanced_context.update(context)

        # Анализ паттернов алерта (если доступна история)
        try:
            alert_patterns = await self._analyze_alert_patterns(alert)
            if alert_patterns:
                enhanced_context["patterns"] = alert_patterns
        except Exception as e:
            alert_logger.warning(
                "Failed to analyze alert patterns",
                alert_name=alert.alert_name,
                error=str(e),
            )

        return enhanced_context

    async def _analyze_alert_patterns(self, alert: Alert) -> Optional[dict[str, Any]]:
        """Анализ паттернов алерта на основе истории."""
        try:
            # Получаем историю похожих алертов
            similar_alerts = await self.storage.get_alerts(
                filters={"alertname": alert.alert_name}, limit=50, offset=0
            )

            if len(similar_alerts) < 2:
                return None

            # Анализируем паттерны
            patterns = {
                "frequency": len(similar_alerts),
                "recent_count_24h": len(
                    [
                        a
                        for a in similar_alerts
                        if a.starts_at
                        and (time.time() - a.starts_at.timestamp()) < 86400
                    ]
                ),
                "firing_ratio": len(
                    [a for a in similar_alerts if a.status.value == "firing"]
                )
                / len(similar_alerts),
                "common_labels": self._find_common_labels(similar_alerts),
                "severity_distribution": self._analyze_severity_distribution(
                    similar_alerts
                ),
            }

            return patterns

        except Exception as e:
            alert_logger.warning(f"Pattern analysis failed: {e}")
            return None

    def _find_common_labels(self, alerts: list) -> dict[str, str]:
        """Найти общие лейблы в алертах."""
        if not alerts:
            return {}

        # Получаем пересечение лейблов
        common_labels = set(alerts[0].labels.keys())
        for alert in alerts[1:]:
            common_labels &= set(alert.labels.keys())

        # Возвращаем общие лейблы с их значениями
        result = {}
        for label in common_labels:
            values = {alert.labels.get(label) for alert in alerts}
            if len(values) == 1:  # Если значение одинаковое у всех
                result[label] = list(values)[0]

        return result

    def _analyze_severity_distribution(self, alerts: list) -> dict[str, int]:
        """Анализ распределения severity в алертах."""
        distribution = {}

        for alert in alerts:
            severity = alert.labels.get("severity", "unknown")
            normalized_severity = normalize_severity(severity)
            distribution[normalized_severity] = (
                distribution.get(normalized_severity, 0) + 1
            )

        return distribution

    async def _validate_classification_result(
        self, alert: Alert, result: ClassificationResult
    ) -> ClassificationResult:
        """Валидация и нормализация результата классификации."""

        # Проверяем confidence в допустимых пределах
        confidence = max(0.0, min(1.0, result.confidence))

        # Проверяем что reasoning не пустой
        reasoning = (
            result.reasoning.strip() if result.reasoning else "No reasoning provided"
        )

        # Ограничиваем количество рекомендаций
        recommendations = result.recommendations[:5] if result.recommendations else []

        # Добавляем метаданные о валидации
        metadata = result.metadata or {}
        metadata.update(
            {
                "validated": True,
                "validation_timestamp": time.time(),
                "original_confidence": result.confidence,
            }
        )

        return ClassificationResult(
            severity=result.severity,
            confidence=confidence,
            reasoning=reasoning,
            recommendations=recommendations,
            processing_time=result.processing_time,
            metadata=metadata,
        )

    async def _save_classification_result(
        self, alert: Alert, result: ClassificationResult
    ) -> None:
        """Сохранить результат классификации в кэш и storage."""
        try:
            # Сохраняем в кэш
            if self.cache:
                cache_key = self._generate_classification_cache_key(alert)
                await self.cache.set(cache_key, result, self.cache_ttl)

            # Сохраняем в persistent storage
            if hasattr(self.storage, "save_classification"):
                await self.storage.save_classification(alert.fingerprint, result)

        except Exception as e:
            alert_logger.error(
                "Failed to save classification result",
                alert_name=alert.alert_name,
                fingerprint=alert.fingerprint,
                error=str(e),
            )
            # Не пробрасываем ошибку - классификация успешна

    async def _get_from_cache(self, alert: Alert) -> Optional[ClassificationResult]:
        """Получить классификацию из кэша."""
        if not self.cache:
            return None

        cache_key = self._generate_classification_cache_key(alert)
        return await self.cache.get(cache_key)

    async def _create_fallback_result(
        self, alert: Alert, error_message: str
    ) -> ClassificationResult:
        """Создать fallback результат при ошибке LLM."""

        # Простая эвристика на основе лейблов
        severity = AlertSeverity.WARNING  # По умолчанию
        confidence = 0.1  # Низкая уверенность

        # Анализ severity из лейблов
        if "severity" in alert.labels:
            label_severity = normalize_severity(alert.labels["severity"])
            if label_severity in ["critical", "warning", "info", "noise"]:
                severity = AlertSeverity(label_severity)
                confidence = 0.3  # Немного выше для лейблов

        # Анализ критичности по имени алерта
        critical_keywords = ["down", "dead", "failed", "error", "critical", "outage"]
        if any(keyword in alert.alert_name.lower() for keyword in critical_keywords):
            severity = AlertSeverity.CRITICAL
            confidence = 0.4

        reasoning = (
            f"Fallback classification due to LLM error: {error_message}. "
            f"Based on alert labels and name analysis."
        )

        recommendations = [
            "Review LLM service connectivity",
            "Check alert labeling for better classification",
            "Consider manual review of this alert",
        ]

        return ClassificationResult(
            severity=severity,
            confidence=confidence,
            reasoning=reasoning,
            recommendations=recommendations,
            processing_time=0.0,
            metadata={
                "fallback": True,
                "error": error_message,
                "method": "label_analysis",
            },
        )

    def _update_classification_metrics(
        self, result: ClassificationResult, total_time: float
    ) -> None:
        """Обновить метрики классификации."""
        if not self.metrics:
            return

        # Основные метрики
        self.metrics.increment_counter(
            "alert_classification_total", {"severity": result.severity.value}
        )

        self.metrics.observe_histogram(
            "alert_classification_duration_seconds", total_time
        )

        self.metrics.observe_histogram(
            "alert_classification_confidence",
            result.confidence,
            {"severity": result.severity.value},
        )

        # Метрики производительности
        if result.processing_time > 0:
            self.metrics.observe_histogram(
                "llm_classification_duration_seconds", result.processing_time
            )

    def _generate_classification_cache_key_by_fingerprint(
        self, fingerprint: str
    ) -> str:
        """Генерация ключа кэша по fingerprint."""
        return f"classification:{fingerprint}"
