"""
Webhook processor с интеграцией LLM классификации.

Обрабатывает входящие webhook события от Alertmanager и:
- Сохраняет алерты (legacy functionality)
- Автоматически классифицирует алерты (новая функциональность)
- Обновляет метрики
- Логирует операции
"""

# Standard library imports
import asyncio
import time
from typing import Any, Optional

# Local imports
from ..api.metrics import LegacyMetrics
from ..core.interfaces import Alert, AlertStatus
from ..database.sqlite_adapter import SQLiteLegacyStorage
from ..logging_config import get_alert_logger, get_performance_logger
from ..services.alert_classifier import AlertClassificationService
from ..utils.common import generate_fingerprint, parse_timestamp

alert_logger = get_alert_logger()
performance_logger = get_performance_logger()


class WebhookProcessor:
    """
    Processor для webhook событий с поддержкой LLM классификации.

    Maintains 100% backward compatibility while adding intelligent features.
    """

    def __init__(
        self,
        storage: SQLiteLegacyStorage,
        metrics: LegacyMetrics,
        classification_service: Optional[AlertClassificationService] = None,
        enable_auto_classification: bool = True,
        classification_timeout: float = 10.0,
    ):
        """Initialize webhook processor."""
        self.storage = storage
        self.metrics = metrics
        self.classification_service = classification_service
        self.enable_auto_classification = (
            enable_auto_classification and classification_service is not None
        )
        self.classification_timeout = classification_timeout

        # Processing statistics
        self.stats = {
            "total_processed": 0,
            "auto_classifications": 0,
            "classification_failures": 0,
            "storage_failures": 0,
        }

    async def process_webhook(self, webhook_data: dict[str, Any]) -> dict[str, Any]:
        """
        Обработать webhook данные от Alertmanager.

        Args:
            webhook_data: Payload от Alertmanager

        Returns:
            Результат обработки с статистикой
        """
        start_time = time.time()
        processed_count = 0
        classification_count = 0
        errors = []

        try:
            alerts_data = webhook_data.get("alerts", [])

            if not alerts_data:
                return {
                    "status": "success",
                    "processed": 0,
                    "classified": 0,
                    "errors": [],
                    "processing_time": time.time() - start_time,
                }

            # Process alerts sequentially for now (можно optimize to parallel later)
            for alert_data in alerts_data:
                try:
                    # Convert to internal Alert format
                    alert = self._convert_webhook_to_alert(alert_data)

                    # Save alert (legacy functionality - must always work)
                    await self.storage.save_alert(alert)
                    processed_count += 1

                    # Update legacy metrics
                    self.metrics.increment_alerts_received(
                        alert.alert_name, alert.status.value
                    )

                    # Automatic classification (new functionality - optional)
                    if self.enable_auto_classification:
                        classification_success = await self._try_classify_alert(alert)
                        if classification_success:
                            classification_count += 1

                    alert_logger.alert_received(
                        alert_name=alert.alert_name,
                        fingerprint=alert.fingerprint,
                        status=alert.status.value,
                        processing_time=time.time() - start_time,
                    )

                except Exception as e:
                    error_info = {
                        "alert_data": alert_data,
                        "error": str(e),
                        "error_type": type(e).__name__,
                    }
                    errors.append(error_info)

                    alert_logger.error(
                        "Failed to process individual alert",
                        error=str(e),
                        alert_data=alert_data,
                    )

                    # Increment error metrics
                    self.metrics.increment_webhook_errors()
                    self.stats["storage_failures"] += 1

            # Update processing statistics
            self.stats["total_processed"] += processed_count
            self.stats["auto_classifications"] += classification_count

            # Record processing time metric
            processing_time = time.time() - start_time
            self.metrics.observe_webhook_duration(processing_time)

            result = {
                "status": "success",
                "processed": processed_count,
                "classified": classification_count,
                "errors": errors,
                "processing_time": processing_time,
            }

            alert_logger.info(
                "Webhook processed successfully",
                processed_count=processed_count,
                classification_count=classification_count,
                error_count=len(errors),
                processing_time=processing_time,
            )

            return result

        except Exception as e:
            # Global error handling
            processing_time = time.time() - start_time

            alert_logger.error(
                "Webhook processing failed",
                error=str(e),
                processed_count=processed_count,
                processing_time=processing_time,
            )

            self.metrics.increment_webhook_errors()

            return {
                "status": "error",
                "processed": processed_count,
                "classified": classification_count,
                "errors": [{"global_error": str(e)}],
                "processing_time": processing_time,
            }

    async def _try_classify_alert(self, alert: Alert) -> bool:
        """
        Попытаться классифицировать алерт с таймаутом.

        Args:
            alert: Alert для классификации

        Returns:
            True если классификация успешна, False иначе
        """
        if not self.classification_service:
            return False

        try:
            # Classify with timeout to avoid blocking webhook processing
            classification_result = await asyncio.wait_for(
                self.classification_service.classify_alert(alert),
                timeout=self.classification_timeout,
            )

            alert_logger.alert_classified(
                alert_name=alert.alert_name,
                fingerprint=alert.fingerprint,
                severity=classification_result.severity.value,
                confidence=classification_result.confidence,
                processing_time=classification_result.processing_time,
            )

            return True

        except asyncio.TimeoutError:
            alert_logger.warning(
                "Alert classification timed out",
                alert_name=alert.alert_name,
                fingerprint=alert.fingerprint,
                timeout=self.classification_timeout,
            )
            self.stats["classification_failures"] += 1
            return False

        except Exception as e:
            alert_logger.error(
                "Alert classification failed",
                alert_name=alert.alert_name,
                fingerprint=alert.fingerprint,
                error=str(e),
            )
            self.stats["classification_failures"] += 1
            return False

    def _convert_webhook_to_alert(self, alert_data: dict[str, Any]) -> Alert:
        """Convert webhook alert data to internal Alert format."""
        # Extract required fields
        labels = alert_data.get("labels", {})
        annotations = alert_data.get("annotations", {})

        # Generate fingerprint (same logic as original)
        fingerprint = alert_data.get("fingerprint")
        if not fingerprint:
            fingerprint = generate_fingerprint(labels)

        # Parse timestamps
        starts_at = parse_timestamp(alert_data.get("startsAt"))
        ends_at = None
        if alert_data.get("endsAt"):
            ends_at = parse_timestamp(alert_data["endsAt"])

        # Determine status
        status = AlertStatus.FIRING
        if alert_data.get("status") == "resolved":
            status = AlertStatus.RESOLVED

        return Alert(
            fingerprint=fingerprint,
            alert_name=labels.get("alertname", "unknown"),
            status=status,
            labels=labels,
            annotations=annotations,
            starts_at=starts_at,
            ends_at=ends_at,
            generator_url=alert_data.get("generatorURL"),
        )

    def get_processing_stats(self) -> dict[str, Any]:
        """Получить статистику обработки webhook."""
        return self.stats.copy()

    def reset_stats(self) -> None:
        """Сбросить статистику."""
        self.stats = {
            "total_processed": 0,
            "auto_classifications": 0,
            "classification_failures": 0,
            "storage_failures": 0,
        }
