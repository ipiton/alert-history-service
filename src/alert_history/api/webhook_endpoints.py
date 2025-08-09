"""
Webhook API endpoints для приема алертов от Alertmanager.

Endpoints:
- POST /webhook - legacy webhook (backward compatibility)
- POST /webhook/proxy - intelligent proxy webhook (Phase 3)
"""

# Standard library imports
import asyncio
from datetime import datetime
from typing import Any, Dict, List, Optional

# Third-party imports
from fastapi import APIRouter, BackgroundTasks, Depends, HTTPException, Request
from fastapi.responses import JSONResponse
from pydantic import BaseModel, Field

# Local imports
from ..api.metrics import LegacyMetrics
from ..core.interfaces import (
    Alert,
    AlertSeverity,
    AlertStatus,
    EnrichedAlert,
    PublishingTarget,
)
from ..logging_config import get_logger
from ..services.alert_classifier import AlertClassificationService
from ..services.alert_formatter import AlertFormatter
from ..services.alert_publisher import AlertPublisher
from ..services.filter_engine import AlertFilterEngine, FilterAction
from ..services.target_discovery import DynamicTargetManager
from ..services.webhook_processor import WebhookProcessor

logger = get_logger(__name__)

# Create router
webhook_router = APIRouter(prefix="/webhook", tags=["Webhook"])


# Pydantic models
class WebhookAlertRequest(BaseModel):
    """Request model для webhook alerts."""

    alerts: List[Dict[str, Any]]
    receiver: str = "default"
    status: str = "firing"
    externalURL: Optional[str] = None
    version: str = "4"
    groupKey: Optional[str] = None
    truncatedAlerts: Optional[int] = 0


class ProxyWebhookResponse(BaseModel):
    """Response model для proxy webhook."""

    message: str
    processed_alerts: int
    published_alerts: int
    filtered_alerts: int
    classification_results: Optional[Dict[str, Any]] = None
    publishing_results: Optional[Dict[str, Any]] = None
    metrics_only_mode: bool = False
    processing_time_ms: int


# Dependency injection functions
async def get_target_manager() -> DynamicTargetManager:
    """Get target manager instance."""
    # Import here to avoid circular imports
    from ..core.app_state import app_state

    if not hasattr(app_state, "target_manager"):
        from ..services.target_discovery import (
            TargetDiscoveryConfig,
            DynamicTargetManager,
        )

        config = TargetDiscoveryConfig(
            enabled=True,
            secret_labels=["alert-history.io/target=true"],
            secret_namespaces=["default", "monitoring"],
        )
        app_state.target_manager = DynamicTargetManager(config)

    return app_state.target_manager


async def get_alert_publisher() -> AlertPublisher:
    """Get alert publisher instance."""
    from ..core.app_state import app_state

    if not hasattr(app_state, "alert_publisher"):
        app_state.alert_publisher = AlertPublisher()

    return app_state.alert_publisher


async def get_filter_engine() -> AlertFilterEngine:
    """Get filter engine instance."""
    from ..core.app_state import app_state

    if not hasattr(app_state, "filter_engine"):
        app_state.filter_engine = AlertFilterEngine()

    return app_state.filter_engine


async def get_classification_service() -> Optional[AlertClassificationService]:
    """Get classification service instance."""
    from ..core.app_state import app_state

    if not hasattr(app_state, "classification_service"):
        try:
            app_state.classification_service = AlertClassificationService()
        except Exception as e:
            logger.warning(f"Classification service unavailable: {e}")
            app_state.classification_service = None

    return app_state.classification_service


async def get_webhook_processor() -> WebhookProcessor:
    """Get webhook processor instance."""
    from ..core.app_state import app_state

    if not hasattr(app_state, "webhook_processor"):
        app_state.webhook_processor = WebhookProcessor()

    return app_state.webhook_processor


async def get_metrics() -> LegacyMetrics:
    """Get metrics instance."""
    from ..core.app_state import app_state

    if not hasattr(app_state, "metrics"):
        app_state.metrics = LegacyMetrics()

    return app_state.metrics


@webhook_router.post("/", response_model=Dict[str, Any])
async def legacy_webhook(
    webhook_data: WebhookAlertRequest,
    background_tasks: BackgroundTasks,
    webhook_processor: WebhookProcessor = Depends(get_webhook_processor),
    metrics: LegacyMetrics = Depends(get_metrics),
):
    """
    Legacy webhook endpoint for backward compatibility.

    Принимает webhook от Alertmanager и сохраняет в базу данных
    без intelligent proxy функций.
    """
    try:
        logger.info(
            "Processing legacy webhook",
            alerts_count=len(webhook_data.alerts),
            receiver=webhook_data.receiver,
        )

        # Process alerts using existing webhook processor
        result = await webhook_processor.process_webhook(webhook_data.dict(), background_tasks)

        # Update metrics
        for alert_data in webhook_data.alerts:
            fingerprint = alert_data.get("fingerprint", "unknown")
            status = alert_data.get("status", "firing")
            metrics.increment_alerts_received(fingerprint, status)

        return {
            "message": "Webhook processed successfully",
            "processed_alerts": len(webhook_data.alerts),
            "mode": "legacy",
        }

    except Exception as e:
        logger.error(f"Legacy webhook processing failed: {e}")
        raise HTTPException(status_code=500, detail=f"Webhook processing failed: {str(e)}")


@webhook_router.post("/proxy", response_model=ProxyWebhookResponse)
async def intelligent_proxy_webhook(
    webhook_data: WebhookAlertRequest,
    background_tasks: BackgroundTasks,
    target_manager: DynamicTargetManager = Depends(get_target_manager),
    alert_publisher: AlertPublisher = Depends(get_alert_publisher),
    filter_engine: AlertFilterEngine = Depends(get_filter_engine),
    classification_service: Optional[AlertClassificationService] = Depends(
        get_classification_service
    ),
    webhook_processor: WebhookProcessor = Depends(get_webhook_processor),
    metrics: LegacyMetrics = Depends(get_metrics),
):
    """
    Intelligent proxy webhook endpoint (Phase 3).

    Полный intelligent proxy workflow:
    1. Принимает Alertmanager webhook
    2. Обрабатывает и сохраняет алерты (backward compatibility)
    3. Классифицирует алерты через LLM (если включено)
    4. Фильтрует алерты согласно правилам
    5. Публикует в настроенные targets (Rootly, PagerDuty, Slack)
    6. Возвращает детальный статус операций
    """
    start_time = datetime.utcnow()
    processed_alerts = 0
    published_alerts = 0
    filtered_alerts = 0
    classification_results = {}

    try:
        # Get active publishing targets
        active_targets = target_manager.get_active_targets()
        metrics_only_mode = target_manager.is_metrics_only_mode()

        logger.info(
            "Processing intelligent proxy webhook",
            alerts_count=len(webhook_data.alerts),
            active_targets=len(active_targets),
            metrics_only_mode=metrics_only_mode,
            receiver=webhook_data.receiver,
        )

        # Enrichment mode resolve (transparent vs enriched)
        from ..core.app_state import app_state as _as

        enrichment_mode = getattr(_as, "enrichment_mode", None)
        if enrichment_mode not in ("transparent", "enriched"):
            try:
                from .enrichment_endpoints import _get_mode_from_redis, _get_default_mode

                resolved = await _get_mode_from_redis()
                enrichment_mode = (
                    resolved if resolved in ("transparent", "enriched") else _get_default_mode()
                )
                _as.enrichment_mode = enrichment_mode
            except Exception:
                enrichment_mode = "enriched"

        # Step 1: Process alerts for backward compatibility (respect enrichment mode)
        if enrichment_mode == "transparent":
            # Record transparent mode processing
            metrics.enrichment_transparent_alerts.inc(len(webhook_data.alerts))

            original_flag = webhook_processor.enable_auto_classification
            webhook_processor.enable_auto_classification = False
            try:
                await webhook_processor.process_webhook(webhook_data.dict(), background_tasks)
            finally:
                webhook_processor.enable_auto_classification = original_flag
        else:
            # Record enriched mode processing
            metrics.enrichment_enriched_alerts.inc(len(webhook_data.alerts))
            await webhook_processor.process_webhook(webhook_data.dict(), background_tasks)

        # Process each alert through intelligent proxy pipeline
        publishing_results = {}

        for alert_data in webhook_data.alerts:
            try:
                processed_alerts += 1
                fingerprint = alert_data.get("fingerprint", f"unknown-{processed_alerts}")

                # Update metrics
                status = alert_data.get("status", "firing")
                metrics.increment_alerts_received(fingerprint, status)

                # Convert to Alert object
                alert = Alert(
                    fingerprint=fingerprint,
                    alert_name=alert_data.get("labels", {}).get("alertname", "Unknown"),
                    status=AlertStatus(status),
                    labels=alert_data.get("labels", {}),
                    annotations=alert_data.get("annotations", {}),
                    starts_at=datetime.fromisoformat(
                        alert_data.get("startsAt", datetime.utcnow().isoformat()).replace(
                            "Z", "+00:00"
                        )
                    ),
                    ends_at=(
                        datetime.fromisoformat(
                            alert_data.get("endsAt", datetime.utcnow().isoformat()).replace(
                                "Z", "+00:00"
                            )
                        )
                        if alert_data.get("endsAt")
                        else None
                    ),
                    generator_url=alert_data.get("generatorURL", ""),
                )

                # Step 2: Classify alert (if LLM available)
                enriched_alert = None
                if classification_service:
                    try:
                        classification_result = await classification_service.classify_alert(alert)
                        enriched_alert = EnrichedAlert(
                            alert=alert, classification=classification_result
                        )
                        classification_results[fingerprint] = {
                            "severity": classification_result.severity.value,
                            "confidence": classification_result.confidence,
                            "reasoning": (
                                classification_result.reasoning[:100] + "..."
                                if len(classification_result.reasoning) > 100
                                else classification_result.reasoning
                            ),
                        }

                        # Update classification metrics
                        metrics.increment_classification(classification_result.severity.value)

                        logger.debug(
                            f"Alert {fingerprint} classified as {classification_result.severity.value}"
                        )

                    except Exception as e:
                        logger.warning(f"Classification failed for {fingerprint}: {e}")
                        enriched_alert = EnrichedAlert(alert=alert)
                else:
                    enriched_alert = EnrichedAlert(alert=alert)

                # Step 3: Apply filters
                should_publish, delay = await filter_engine.should_publish(enriched_alert)

                if not should_publish:
                    filtered_alerts += 1
                    logger.debug(f"Alert {fingerprint} filtered out")
                    continue

                # Step 4: Publish to targets (if not metrics-only mode)
                if not metrics_only_mode and active_targets:
                    try:
                        # Use existing AlertPublisher to publish to multiple targets
                        publish_results = await alert_publisher.publish_to_multiple_targets(
                            enriched_alert, active_targets
                        )

                        publishing_results[fingerprint] = {
                            "targets": len(active_targets),
                            "successful": sum(
                                1 for r in publish_results if r.get("success", False)
                            ),
                            "failed": sum(
                                1 for r in publish_results if not r.get("success", False)
                            ),
                        }

                        if publishing_results[fingerprint]["successful"] > 0:
                            published_alerts += 1

                        logger.debug(
                            f"Alert {fingerprint} published to {publishing_results[fingerprint]['successful']}/{len(active_targets)} targets"
                        )

                    except Exception as e:
                        logger.error(f"Publishing failed for {fingerprint}: {e}")
                        publishing_results[fingerprint] = {
                            "error": str(e),
                            "targets": len(active_targets),
                            "successful": 0,
                            "failed": len(active_targets),
                        }

            except Exception as e:
                logger.error(f"Error processing alert {fingerprint}: {e}")
                continue

        # Calculate processing time
        processing_time = (datetime.utcnow() - start_time).total_seconds() * 1000

        logger.info(
            "Intelligent proxy webhook completed",
            processed_alerts=processed_alerts,
            published_alerts=published_alerts,
            filtered_alerts=filtered_alerts,
            processing_time_ms=int(processing_time),
            metrics_only_mode=metrics_only_mode,
        )

        return ProxyWebhookResponse(
            message="Intelligent proxy webhook processed successfully",
            processed_alerts=processed_alerts,
            published_alerts=published_alerts,
            filtered_alerts=filtered_alerts,
            classification_results=(classification_results if classification_results else None),
            publishing_results=publishing_results if publishing_results else None,
            metrics_only_mode=metrics_only_mode,
            processing_time_ms=int(processing_time),
        )

    except Exception as e:
        logger.error(f"Intelligent proxy webhook processing failed: {e}")
        raise HTTPException(
            status_code=500, detail=f"Intelligent proxy processing failed: {str(e)}"
        )


@webhook_router.get("/health")
async def webhook_health():
    """Health check для webhook endpoints."""
    return {
        "status": "healthy",
        "endpoints": ["/webhook (legacy)", "/webhook/proxy (intelligent)"],
        "timestamp": datetime.utcnow().isoformat(),
    }
