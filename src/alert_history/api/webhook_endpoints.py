"""
Webhook API endpoints для приема алертов от Alertmanager.

Endpoints:
- POST /webhook/ - universal webhook (auto-switches between legacy and intelligent modes)
- POST /webhook/proxy - intelligent proxy webhook (Phase 3, explicit intelligent mode)

The universal webhook automatically detects available features and switches modes:
- Legacy mode: Simple processing and storage
- Intelligent mode: Full LLM classification, filtering, and publishing
"""

# Standard library imports
import os
from datetime import datetime
from typing import Any, Optional

# Third-party imports
from fastapi import APIRouter, BackgroundTasks, Depends, HTTPException, Request
from pydantic import BaseModel

# Local imports
from ..api.metrics import LegacyMetrics
from ..core.interfaces import Alert, AlertStatus, EnrichedAlert
from ..logging_config import get_logger
from ..services.alert_classifier import AlertClassificationService
from ..services.alert_publisher import AlertPublisher
from ..services.filter_engine import AlertFilterEngine
from ..services.target_discovery import DynamicTargetManager
from ..services.webhook_processor import WebhookProcessor

logger = get_logger(__name__)

# Create router
webhook_router = APIRouter(prefix="/webhook", tags=["Webhook"])


# Pydantic models
class WebhookAlertRequest(BaseModel):
    """Request model для webhook alerts."""

    alerts: list[dict[str, Any]]
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
    classification_results: Optional[dict[str, Any]] = None
    publishing_results: Optional[dict[str, Any]] = None
    metrics_only_mode: bool = False
    processing_time_ms: int


# Dependency injection functions
async def get_target_manager() -> DynamicTargetManager:
    """Get target manager instance."""
    # Import here to avoid circular imports
    from ..core.app_state import app_state

    if not hasattr(app_state, "target_manager"):
        from ..services.target_discovery import (
            DynamicTargetManager,
            TargetDiscoveryConfig,
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


def get_storage_instance():
    """Get storage instance using the same logic as main application."""
    from ..config import get_config
    from ..core.app_state import app_state

    # Try to get from app state first
    if hasattr(app_state, "storage") and app_state.storage is not None:
        return app_state.storage

    config = get_config()

    if config.database.url and config.database.url.startswith("postgresql://"):
        # Try PostgreSQL first
        try:
            from ..database.postgresql_adapter import PostgreSQLStorage

            storage = PostgreSQLStorage(config.database.url)
            logger.info("Using PostgreSQL storage for webhook processing")
            return storage
        except Exception as e:
            logger.warning(f"PostgreSQL fallback failed: {e}, using SQLite")

    # Use SQLite fallback
    from ..database.sqlite_adapter import SQLiteLegacyStorage

    storage = SQLiteLegacyStorage("data/alert_history.sqlite3")
    logger.info("Using SQLite storage for webhook processing")
    return storage


async def get_classification_service() -> Optional[AlertClassificationService]:
    """Get classification service instance."""
    from ..core.app_state import app_state

    if not hasattr(app_state, "classification_service"):
        try:
            # Create LLM client first
            from ..services.llm_client import LLMProxyClient

            llm_client = LLMProxyClient(
                api_key=os.getenv("LLM_API_KEY"),
                proxy_url=os.getenv("LLM_PROXY_URL"),
                model=os.getenv("LLM_MODEL", "openai/gpt-4o"),
            )

            storage = get_storage_instance()

            app_state.classification_service = AlertClassificationService(
                llm_client=llm_client, storage=storage
            )
        except Exception as e:
            logger.warning(f"Classification service unavailable: {e}")
            app_state.classification_service = None

    return app_state.classification_service


async def get_webhook_processor() -> WebhookProcessor:
    """Get webhook processor instance."""
    # Try to get from global app state
    try:
        from ..core.app_state import app_state

        if (
            hasattr(app_state, "webhook_processor")
            and app_state.webhook_processor is not None
        ):
            return app_state.webhook_processor
    except Exception:
        pass

    # Fallback: create new instance
    from ..api.metrics import get_metrics

    storage = get_storage_instance()
    metrics = get_metrics()

    # Try to get classification service from app state
    classification_service = None
    try:
        from ..core.app_state import app_state

        if hasattr(app_state, "classification_service"):
            classification_service = app_state.classification_service
    except Exception:
        pass

    webhook_processor = WebhookProcessor(
        storage=storage,
        metrics=metrics,
        classification_service=classification_service,
        enable_auto_classification=True,
    )

    # Store in global app state for future use
    try:
        from ..core.app_state import app_state

        app_state.webhook_processor = webhook_processor
    except Exception:
        pass

    return webhook_processor


async def get_metrics() -> LegacyMetrics:
    """Get metrics instance."""
    from ..core.app_state import app_state

    if not hasattr(app_state, "metrics") or app_state.metrics is None:
        app_state.metrics = LegacyMetrics()

    return app_state.metrics


@webhook_router.post("/", response_model=dict[str, Any])
async def universal_webhook(
    webhook_data: WebhookAlertRequest,
    background_tasks: BackgroundTasks,
    target_manager: DynamicTargetManager = Depends(get_target_manager),
    alert_publisher: AlertPublisher = Depends(get_alert_publisher),
    filter_engine: AlertFilterEngine = Depends(get_filter_engine),
    classification_service: Optional[AlertClassificationService] = Depends(
        get_classification_service
    ),
    metrics: LegacyMetrics = Depends(get_metrics),
):
    """
    Universal webhook endpoint that combines legacy and intelligent proxy functionality.

    Automatically switches between modes:
    - Legacy mode: Simple processing and storage
    - Intelligent mode: Full LLM classification, filtering, and publishing
    """
    start_time = datetime.utcnow()

    try:
        # Create webhook processor directly
        logger.info("Creating webhook processor...")
        storage = get_storage_instance()
        webhook_processor = WebhookProcessor(
            storage=storage,
            metrics=metrics,
            classification_service=classification_service,
            enable_auto_classification=True,
        )
        logger.info(f"Webhook processor created: {type(webhook_processor)}")

        # Check if intelligent features are available
        active_targets = []
        try:
            if target_manager and hasattr(target_manager, "get_active_targets"):
                active_targets = target_manager.get_active_targets()
        except Exception as e:
            logger.warning(f"Failed to get active targets: {e}")

        # Force intelligent mode if LLM is available
        has_intelligent_features = (
            classification_service is not None
            or len(active_targets) > 0
            or getattr(webhook_processor, "enable_auto_classification", False)
        )

        logger.info(
            f"Intelligent features check: classification_service={classification_service is not None}, active_targets={len(active_targets)}, auto_classification={getattr(webhook_processor, 'enable_auto_classification', False)}"
        )

        if not has_intelligent_features:
            # Legacy mode - simple processing
            logger.info(
                "Processing webhook in legacy mode",
                alerts_count=len(webhook_data.alerts),
                receiver=webhook_data.receiver,
            )

            # Process alerts using existing webhook processor
            result = await webhook_processor.process_webhook(webhook_data.dict())

            # Update metrics
            for alert_data in webhook_data.alerts:
                fingerprint = alert_data.get("fingerprint", "unknown")
                status = alert_data.get("status", "firing")
                metrics.increment_alerts_received(fingerprint, status)

            return {
                "message": "Webhook processed successfully (legacy mode)",
                "processed_alerts": len(webhook_data.alerts),
                "mode": "legacy",
                "processing_time_ms": int(
                    (datetime.utcnow() - start_time).total_seconds() * 1000
                ),
            }
        else:
            # Intelligent mode - full proxy functionality
            logger.info(
                "Processing webhook in intelligent mode",
                alerts_count=len(webhook_data.alerts),
                active_targets=len(active_targets),
                receiver=webhook_data.receiver,
            )

            # Use the same logic as intelligent proxy
            processed_alerts = 0
            published_alerts = 0
            filtered_alerts = 0
            classification_results = {}

            # Enrichment mode resolve
            from ..core.app_state import app_state as _as

            enrichment_mode = getattr(_as, "enrichment_mode", "enriched")

            # Process alerts with enrichment mode
            if enrichment_mode == "transparent":
                metrics.enrichment_transparent_alerts.inc(len(webhook_data.alerts))
                original_flag = webhook_processor.enable_auto_classification
                webhook_processor.enable_auto_classification = False
                try:
                    await webhook_processor.process_webhook(webhook_data.dict())
                finally:
                    webhook_processor.enable_auto_classification = original_flag
            else:
                metrics.enrichment_enriched_alerts.inc(len(webhook_data.alerts))
                await webhook_processor.process_webhook(webhook_data.dict())

            # Process each alert through intelligent pipeline
            for alert_data in webhook_data.alerts:
                try:
                    processed_alerts += 1
                    fingerprint = alert_data.get(
                        "fingerprint", f"unknown-{processed_alerts}"
                    )

                    # Update metrics
                    status = alert_data.get("status", "firing")
                    metrics.increment_alerts_received(fingerprint, status)

                    # Convert to Alert object
                    alert = Alert(
                        fingerprint=fingerprint,
                        alert_name=alert_data.get("labels", {}).get(
                            "alertname", "Unknown"
                        ),
                        status=AlertStatus(status),
                        labels=alert_data.get("labels", {}),
                        annotations=alert_data.get("annotations", {}),
                        starts_at=datetime.fromisoformat(
                            alert_data.get(
                                "startsAt", datetime.utcnow().isoformat()
                            ).replace("Z", "+00:00")
                        ),
                        ends_at=(
                            datetime.fromisoformat(
                                alert_data.get(
                                    "endsAt", datetime.utcnow().isoformat()
                                ).replace("Z", "+00:00")
                            )
                            if alert_data.get("endsAt")
                            else None
                        ),
                        generator_url=alert_data.get("generatorURL", ""),
                        timestamp=datetime.utcnow(),
                    )

                    # Classify alert (if LLM available)
                    if classification_service:
                        try:
                            classification_result = (
                                await classification_service.classify_alert(alert)
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
                        except Exception as e:
                            logger.warning(
                                f"Classification failed for {fingerprint}: {e}"
                            )
                            classification_results[fingerprint] = {"error": str(e)}

                    # Filter and publish
                    if active_targets:
                        try:
                            # Apply filters
                            filter_result = filter_engine.apply_filters(alert)
                            if filter_result.action == FilterAction.ALLOW:
                                # Publish to targets
                                publish_result = await alert_publisher.publish_alert(
                                    alert, active_targets
                                )
                                published_alerts += publish_result.successful_publishes
                            elif filter_result.action == FilterAction.DENY:
                                filtered_alerts += 1
                        except Exception as e:
                            logger.warning(f"Publishing failed for {fingerprint}: {e}")

                except Exception as e:
                    logger.error(f"Error processing alert {fingerprint}: {e}")

            return {
                "message": "Webhook processed successfully (intelligent mode)",
                "processed_alerts": processed_alerts,
                "published_alerts": published_alerts,
                "filtered_alerts": filtered_alerts,
                "classification_results": classification_results,
                "mode": "intelligent",
                "processing_time_ms": int(
                    (datetime.utcnow() - start_time).total_seconds() * 1000
                ),
            }

    except Exception as e:
        logger.error(f"Webhook processing failed: {e}")
        raise HTTPException(
            status_code=500, detail=f"Webhook processing failed: {str(e)}"
        ) from e


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

        # Enrichment mode resolve (transparent vs enriched vs transparent_with_recommendations)
        from ..core.app_state import app_state as _as

        enrichment_mode = getattr(_as, "enrichment_mode", None)
        if enrichment_mode not in (
            "transparent",
            "enriched",
            "transparent_with_recommendations",
        ):
            try:
                from .enrichment_endpoints import (
                    _get_default_mode,
                    _get_mode_from_redis,
                )

                resolved = await _get_mode_from_redis()
                enrichment_mode = (
                    resolved
                    if resolved
                    in ("transparent", "enriched", "transparent_with_recommendations")
                    else _get_default_mode()
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
                await webhook_processor.process_webhook(webhook_data.dict())
            finally:
                webhook_processor.enable_auto_classification = original_flag
        elif enrichment_mode == "transparent_with_recommendations":
            # Record transparent with recommendations mode processing
            metrics.enrichment_transparent_alerts.inc(len(webhook_data.alerts))

            # Process alerts normally (transparent) but collect LLM recommendations
            original_flag = webhook_processor.enable_auto_classification
            webhook_processor.enable_auto_classification = False
            try:
                await webhook_processor.process_webhook(webhook_data.dict())
            finally:
                webhook_processor.enable_auto_classification = original_flag
        else:
            # Record enriched mode processing
            metrics.enrichment_enriched_alerts.inc(len(webhook_data.alerts))
            await webhook_processor.process_webhook(webhook_data.dict())

        # Process each alert through intelligent proxy pipeline
        publishing_results = {}

        for alert_data in webhook_data.alerts:
            try:
                processed_alerts += 1
                fingerprint = alert_data.get(
                    "fingerprint", f"unknown-{processed_alerts}"
                )

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
                        alert_data.get(
                            "startsAt", datetime.utcnow().isoformat()
                        ).replace("Z", "+00:00")
                    ),
                    ends_at=(
                        datetime.fromisoformat(
                            alert_data.get(
                                "endsAt", datetime.utcnow().isoformat()
                            ).replace("Z", "+00:00")
                        )
                        if alert_data.get("endsAt")
                        else None
                    ),
                    generator_url=alert_data.get("generatorURL", ""),
                    timestamp=datetime.utcnow(),
                )

                # Step 2: Classify alert (if LLM available)
                enriched_alert = None
                if classification_service:
                    try:
                        classification_result = (
                            await classification_service.classify_alert(alert)
                        )
                        enriched_alert = EnrichedAlert(
                            alert=alert, classification=classification_result
                        )

                        # Store classification results
                        classification_results[fingerprint] = {
                            "severity": classification_result.severity.value,
                            "confidence": classification_result.confidence,
                            "reasoning": (
                                classification_result.reasoning[:100] + "..."
                                if len(classification_result.reasoning) > 100
                                else classification_result.reasoning
                            ),
                            "recommendations": classification_result.recommendations,
                        }

                        # Update classification metrics
                        metrics.increment_classification(
                            classification_result.severity.value
                        )

                        logger.debug(
                            f"Alert {fingerprint} classified as {classification_result.severity.value}"
                        )

                    except Exception as e:
                        logger.warning(f"Classification failed for {fingerprint}: {e}")
                        enriched_alert = EnrichedAlert(alert=alert)
                else:
                    enriched_alert = EnrichedAlert(alert=alert)

                # Step 3: Apply filters (skip filtering in transparent_with_recommendations mode)
                should_publish = True
                if enrichment_mode != "transparent_with_recommendations":
                    should_publish, delay = await filter_engine.should_publish(
                        enriched_alert
                    )

                if not should_publish:
                    filtered_alerts += 1
                    logger.debug(f"Alert {fingerprint} filtered out")
                    continue

                # Step 4: Publish to targets (if not metrics-only mode)
                if not metrics_only_mode and active_targets:
                    try:
                        # Use existing AlertPublisher to publish to multiple targets
                        publish_results = (
                            await alert_publisher.publish_to_multiple_targets(
                                enriched_alert, active_targets
                            )
                        )

                        publishing_results[fingerprint] = {
                            "targets": len(active_targets),
                            "successful": sum(
                                1 for r in publish_results if r.get("success", False)
                            ),
                            "failed": sum(
                                1
                                for r in publish_results
                                if not r.get("success", False)
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
            classification_results=(
                classification_results if classification_results else None
            ),
            publishing_results=publishing_results if publishing_results else None,
            metrics_only_mode=metrics_only_mode,
            processing_time_ms=int(processing_time),
        )

    except Exception as e:
        logger.error(f"Intelligent proxy webhook processing failed: {e}")
        raise HTTPException(
            status_code=500, detail=f"Intelligent proxy processing failed: {str(e)}"
        ) from e


@webhook_router.get("/health")
async def webhook_health():
    """Health check для webhook endpoints."""
    return {
        "status": "healthy",
        "endpoints": ["/webhook (legacy)", "/webhook/proxy (intelligent)"],
        "timestamp": datetime.utcnow().isoformat(),
    }
