"""
Proxy API endpoints для intelligent alert routing.

Новые эндпоинты:
- POST /webhook/proxy - основной entry point для intelligent proxy
- GET /proxy/targets - управление publishing targets
- GET /proxy/stats - статистика proxy операций
- GET /proxy/health - health check для proxy компонентов
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
from ..services.filter_engine import AlertFilterEngine, FilterAction, FilterRule
from ..services.target_discovery import DynamicTargetManager
from ..services.webhook_processor import WebhookProcessor

logger = get_logger(__name__)

# Create router
proxy_router = APIRouter(prefix="/proxy", tags=["Intelligent Proxy"])


# Pydantic models
class ProxyWebhookRequest(BaseModel):
    """Request model для proxy webhook."""

    alerts: List[Dict[str, Any]]
    receiver: str = "default"
    status: str = "firing"
    version: str = "4"
    groupKey: Optional[str] = None
    groupLabels: Optional[Dict[str, str]] = None
    commonLabels: Optional[Dict[str, str]] = None
    commonAnnotations: Optional[Dict[str, str]] = None
    externalURL: Optional[str] = None
    truncatedAlerts: int = 0


class ProxyStatsResponse(BaseModel):
    """Response model для proxy статистики."""

    proxy_enabled: bool
    metrics_only_mode: bool
    active_targets: int
    total_processed_alerts: int
    successful_publishes: int
    failed_publishes: int
    filtered_alerts: int
    classification_enabled: bool
    target_discovery_stats: Dict[str, Any]
    filter_stats: Dict[str, Any]
    publisher_stats: Dict[str, Any]


class TargetInfo(BaseModel):
    """Information about publishing target."""

    name: str
    type: str
    url: str
    format: str
    enabled: bool
    health_status: str
    success_rate: float
    last_success_time: Optional[datetime] = None
    circuit_breaker_state: str


class FilterRuleRequest(BaseModel):
    """Request model для добавления filter rule."""

    name: str
    action: str = Field(..., pattern="^(allow|deny|delay)$")
    conditions: Dict[str, Any]
    priority: int = Field(default=100, ge=1, le=1000)
    enabled: bool = True
    target_name: Optional[str] = None


# Dependency injection
def get_target_manager(request: Request) -> DynamicTargetManager:
    """Get target manager from app state."""
    if not hasattr(request.app.state, "target_manager"):
        raise HTTPException(status_code=503, detail="Target manager not available")
    return request.app.state.target_manager


def get_alert_publisher(request: Request) -> AlertPublisher:
    """Get alert publisher from app state."""
    if not hasattr(request.app.state, "alert_publisher"):
        raise HTTPException(status_code=503, detail="Alert publisher not available")
    return request.app.state.alert_publisher


def get_filter_engine(request: Request) -> AlertFilterEngine:
    """Get filter engine from app state."""
    if not hasattr(request.app.state, "filter_engine"):
        raise HTTPException(status_code=503, detail="Filter engine not available")
    return request.app.state.filter_engine


def get_classification_service(
    request: Request,
) -> Optional[AlertClassificationService]:
    """Get classification service from app state."""
    return getattr(request.app.state, "classification_service", None)


def get_webhook_processor(request: Request) -> WebhookProcessor:
    """Get webhook processor from app state."""
    if not hasattr(request.app.state, "webhook_processor"):
        raise HTTPException(status_code=503, detail="Webhook processor not available")
    return request.app.state.webhook_processor


def get_metrics(request: Request) -> LegacyMetrics:
    """Get metrics collector from app state."""
    if not hasattr(request.app.state, "metrics"):
        raise HTTPException(status_code=503, detail="Metrics not available")
    return request.app.state.metrics


@proxy_router.post("/webhook")
async def proxy_webhook(
    webhook_data: ProxyWebhookRequest,
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
    Intelligent proxy webhook endpoint.

    Этот endpoint:
    1. Принимает Alertmanager webhook
    2. Обрабатывает и сохраняет алерты (backward compatibility)
    3. Классифицирует алерты через LLM (если включено)
    4. Фильтрует алерты согласно правилам
    5. Публикует в настроенные targets (Rootly, PagerDuty, Slack)
    6. Возвращает статус операций
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
            "Processing proxy webhook",
            alerts_count=len(webhook_data.alerts),
            active_targets=len(active_targets),
            metrics_only_mode=metrics_only_mode,
            receiver=webhook_data.receiver,
        )

        # Process each alert
        publishing_results = {}

        for alert_data in webhook_data.alerts:
            try:
                # Process through webhook processor (backward compatibility)
                await webhook_processor.process_webhook_data(
                    {
                        "alerts": [alert_data],
                        "receiver": webhook_data.receiver,
                        "status": webhook_data.status,
                        "version": webhook_data.version,
                    }
                )

                processed_alerts += 1

                # Convert to internal Alert format
                alert = await _convert_webhook_to_alert(alert_data)

                # Create base enriched alert
                enriched_alert = EnrichedAlert(
                    alert=alert,
                    classification=None,
                    enrichment_metadata={
                        "proxy_processed": True,
                        "proxy_timestamp": start_time.isoformat(),
                        "receiver": webhook_data.receiver,
                    },
                    processing_timestamp=start_time,
                )

                # Classify alert if service available and not metrics-only mode
                if classification_service and not metrics_only_mode:
                    try:
                        classification = await classification_service.classify_alert(alert)
                        if classification:
                            enriched_alert.classification = classification
                            classification_results[alert.fingerprint] = {
                                "severity": classification.severity.value,
                                "confidence": classification.confidence,
                                "reasoning": classification.reasoning[
                                    :100
                                ],  # Truncate for response
                            }
                    except Exception as e:
                        logger.warning(f"Classification failed for alert {alert.fingerprint}: {e}")

                # Publish to targets if not metrics-only mode
                if not metrics_only_mode and active_targets:
                    # Filter and publish to each target
                    alert_publishing_results = {}

                    for target in active_targets:
                        try:
                            # Check filter
                            should_publish = await filter_engine.should_publish(
                                enriched_alert, target
                            )

                            if should_publish:
                                # Publish in background for performance
                                background_tasks.add_task(
                                    _publish_alert_to_target,
                                    enriched_alert,
                                    target,
                                    alert_publisher,
                                    metrics,
                                )
                                alert_publishing_results[target.name] = "scheduled"
                            else:
                                alert_publishing_results[target.name] = "filtered"
                                filtered_alerts += 1

                        except Exception as e:
                            logger.error(f"Error processing target {target.name}: {e}")
                            alert_publishing_results[target.name] = "error"

                    publishing_results[alert.fingerprint] = alert_publishing_results

                    # Count published alerts (scheduled counts as published)
                    published_count = sum(
                        1 for status in alert_publishing_results.values() if status == "scheduled"
                    )
                    published_alerts += published_count

                # Record metrics
                metrics.increment_counter("alert_proxy_processed_total")

                if enriched_alert.classification:
                    metrics.increment_counter(
                        "alert_proxy_classified_total",
                        {"severity": enriched_alert.classification.severity.value},
                    )

            except Exception as e:
                logger.error(f"Error processing individual alert: {e}")
                metrics.increment_counter("alert_proxy_errors_total", {"error": "processing"})
                continue

        # Record processing metrics
        processing_time = (datetime.utcnow() - start_time).total_seconds()
        metrics.observe_histogram("alert_proxy_processing_duration_seconds", processing_time)
        metrics.increment_counter(
            "alert_proxy_webhook_requests_total", {"receiver": webhook_data.receiver}
        )

        # Build response
        response_data = {
            "status": "success",
            "processed_alerts": processed_alerts,
            "published_alerts": published_alerts,
            "filtered_alerts": filtered_alerts,
            "active_targets": len(active_targets),
            "metrics_only_mode": metrics_only_mode,
            "processing_time_seconds": processing_time,
            "classification_results": classification_results,
            "publishing_results": publishing_results,
        }

        logger.info("Proxy webhook processing completed", **response_data)

        return JSONResponse(content=response_data)

    except Exception as e:
        logger.error(f"Proxy webhook processing failed: {e}")
        metrics.increment_counter("alert_proxy_errors_total", {"error": "general"})

        raise HTTPException(status_code=500, detail=f"Proxy processing failed: {str(e)}")


@proxy_router.get("/targets", response_model=List[TargetInfo])
async def get_publishing_targets(
    target_manager: DynamicTargetManager = Depends(get_target_manager),
    alert_publisher: AlertPublisher = Depends(get_alert_publisher),
):
    """Получить список активных publishing targets."""
    try:
        targets = target_manager.get_active_targets()
        target_infos = []

        for target in targets:
            # Get publishing stats
            stats = alert_publisher.get_target_stats(target.name)
            circuit_breaker_status = alert_publisher.get_circuit_breaker_status(target.name)

            target_info = TargetInfo(
                name=target.name,
                type=target.type,
                url=target.url,
                format=target.format.value,
                enabled=target.enabled,
                health_status=("healthy" if (stats and stats.is_healthy) else "unhealthy"),
                success_rate=stats.success_rate if stats else 0.0,
                last_success_time=(
                    datetime.fromtimestamp(stats.last_success_time)
                    if (stats and stats.last_success_time)
                    else None
                ),
                circuit_breaker_state=circuit_breaker_status.get("state", "unknown"),
            )

            target_infos.append(target_info)

        return target_infos

    except Exception as e:
        logger.error(f"Error getting publishing targets: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@proxy_router.get("/stats", response_model=ProxyStatsResponse)
async def get_proxy_stats(
    target_manager: DynamicTargetManager = Depends(get_target_manager),
    alert_publisher: AlertPublisher = Depends(get_alert_publisher),
    filter_engine: AlertFilterEngine = Depends(get_filter_engine),
    classification_service: Optional[AlertClassificationService] = Depends(
        get_classification_service
    ),
):
    """Получить статистику proxy операций."""
    try:
        # Get target discovery stats
        discovery_stats = target_manager.get_discovery_stats()

        # Get filter stats
        filter_stats = filter_engine.get_filter_stats()

        # Get publisher stats
        all_publisher_stats = alert_publisher.get_all_stats()

        # Calculate totals
        total_successful = sum(stats.successful_publishes for stats in all_publisher_stats.values())
        total_failed = sum(stats.failed_publishes for stats in all_publisher_stats.values())

        publisher_summary = {
            "total_attempts": sum(stats.total_attempts for stats in all_publisher_stats.values()),
            "successful_publishes": total_successful,
            "failed_publishes": total_failed,
            "targets_count": len(all_publisher_stats),
            "healthy_targets": sum(1 for stats in all_publisher_stats.values() if stats.is_healthy),
            "per_target_stats": {
                name: {
                    "success_rate": stats.success_rate,
                    "total_attempts": stats.total_attempts,
                    "is_healthy": stats.is_healthy,
                }
                for name, stats in all_publisher_stats.items()
            },
        }

        response = ProxyStatsResponse(
            proxy_enabled=True,
            metrics_only_mode=target_manager.is_metrics_only_mode(),
            active_targets=target_manager.get_targets_count(),
            total_processed_alerts=0,  # TODO: Add global counter
            successful_publishes=total_successful,
            failed_publishes=total_failed,
            filtered_alerts=0,  # TODO: Add global counter
            classification_enabled=classification_service is not None,
            target_discovery_stats=discovery_stats,
            filter_stats=filter_stats,
            publisher_stats=publisher_summary,
        )

        return response

    except Exception as e:
        logger.error(f"Error getting proxy stats: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@proxy_router.get("/health")
async def proxy_health_check(
    target_manager: DynamicTargetManager = Depends(get_target_manager),
    alert_publisher: AlertPublisher = Depends(get_alert_publisher),
    filter_engine: AlertFilterEngine = Depends(get_filter_engine),
    classification_service: Optional[AlertClassificationService] = Depends(
        get_classification_service
    ),
):
    """Health check для proxy компонентов."""
    health_status = {
        "proxy_enabled": True,
        "components": {
            "target_manager": "healthy",
            "alert_publisher": "healthy",
            "filter_engine": "healthy",
            "classification_service": ("healthy" if classification_service else "disabled"),
        },
        "metrics_only_mode": target_manager.is_metrics_only_mode(),
        "active_targets": target_manager.get_targets_count(),
        "timestamp": datetime.utcnow().isoformat(),
    }

    # Check component health
    try:
        discovery_stats = target_manager.get_discovery_stats()
        if not discovery_stats.get("kubernetes_available", False):
            health_status["components"]["target_manager"] = "degraded"
    except Exception:
        health_status["components"]["target_manager"] = "unhealthy"

    try:
        all_stats = alert_publisher.get_all_stats()
        unhealthy_targets = sum(1 for stats in all_stats.values() if not stats.is_healthy)
        if unhealthy_targets > 0:
            health_status["components"]["alert_publisher"] = "degraded"
    except Exception:
        health_status["components"]["alert_publisher"] = "unhealthy"

    # Overall health
    component_statuses = list(health_status["components"].values())
    if "unhealthy" in component_statuses:
        health_status["overall"] = "unhealthy"
        status_code = 503
    elif "degraded" in component_statuses:
        health_status["overall"] = "degraded"
        status_code = 200
    else:
        health_status["overall"] = "healthy"
        status_code = 200

    return JSONResponse(content=health_status, status_code=status_code)


@proxy_router.post("/targets/refresh")
async def refresh_targets(
    target_manager: DynamicTargetManager = Depends(get_target_manager),
):
    """Принудительно обновить список publishing targets."""
    try:
        success = await target_manager.refresh_targets()

        if success:
            active_targets = target_manager.get_targets_count()
            return {
                "status": "success",
                "message": "Targets refreshed successfully",
                "active_targets": active_targets,
            }
        else:
            raise HTTPException(status_code=500, detail="Failed to refresh targets")

    except Exception as e:
        logger.error(f"Error refreshing targets: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@proxy_router.post("/filters/rules")
async def add_filter_rule(
    rule_request: FilterRuleRequest,
    filter_engine: AlertFilterEngine = Depends(get_filter_engine),
):
    """Добавить новое правило фильтрации."""
    try:
        # Convert string action to enum
        action = FilterAction(rule_request.action.lower())

        rule = FilterRule(
            name=rule_request.name,
            action=action,
            conditions=rule_request.conditions,
            priority=rule_request.priority,
            enabled=rule_request.enabled,
        )

        if rule_request.target_name:
            filter_engine.add_target_rule(rule_request.target_name, rule)
        else:
            filter_engine.add_global_rule(rule)

        return {
            "status": "success",
            "message": f"Filter rule '{rule_request.name}' added successfully",
            "rule": {
                "name": rule.name,
                "action": rule.action.value,
                "priority": rule.priority,
                "target": rule_request.target_name or "global",
            },
        }

    except Exception as e:
        logger.error(f"Error adding filter rule: {e}")
        raise HTTPException(status_code=400, detail=str(e))


@proxy_router.delete("/filters/rules/{rule_name}")
async def remove_filter_rule(
    rule_name: str,
    target_name: Optional[str] = None,
    filter_engine: AlertFilterEngine = Depends(get_filter_engine),
):
    """Удалить правило фильтрации."""
    try:
        success = filter_engine.remove_rule(rule_name, target_name)

        if success:
            return {
                "status": "success",
                "message": f"Filter rule '{rule_name}' removed successfully",
            }
        else:
            raise HTTPException(status_code=404, detail=f"Filter rule '{rule_name}' not found")

    except Exception as e:
        logger.error(f"Error removing filter rule: {e}")
        raise HTTPException(status_code=500, detail=str(e))


# Helper functions
async def _convert_webhook_to_alert(alert_data: Dict[str, Any]) -> Alert:
    """Convert webhook alert data to internal Alert format."""
    # Parse timestamps
    starts_at = None
    ends_at = None

    if "startsAt" in alert_data:
        try:
            starts_at = datetime.fromisoformat(alert_data["startsAt"].replace("Z", "+00:00"))
        except (ValueError, AttributeError):
            pass

    if "endsAt" in alert_data:
        try:
            ends_at = datetime.fromisoformat(alert_data["endsAt"].replace("Z", "+00:00"))
        except (ValueError, AttributeError):
            pass

    # Extract basic fields
    fingerprint = alert_data.get("fingerprint", "unknown")
    labels = alert_data.get("labels", {})
    annotations = alert_data.get("annotations", {})

    alert = Alert(
        fingerprint=fingerprint,
        alert_name=labels.get("alertname", "Unknown"),
        namespace=labels.get("namespace"),
        status=AlertStatus(alert_data.get("status", "firing")),
        labels=labels,
        annotations=annotations,
        starts_at=starts_at,
        ends_at=ends_at,
        generator_url=alert_data.get("generatorURL"),
        timestamp=datetime.utcnow(),
    )

    return alert


async def _publish_alert_to_target(
    enriched_alert: EnrichedAlert,
    target: PublishingTarget,
    publisher: AlertPublisher,
    metrics: LegacyMetrics,
) -> None:
    """Background task для публикации алерта в target."""
    try:
        success = await publisher.publish_alert(enriched_alert, target)

        if success:
            metrics.increment_counter(
                "alert_proxy_published_total",
                {"target": target.name, "format": target.format.value},
            )
        else:
            metrics.increment_counter(
                "alert_proxy_publish_failures_total",
                {"target": target.name, "format": target.format.value},
            )

    except Exception as e:
        logger.error(f"Background publishing failed for target {target.name}: {e}")
        metrics.increment_counter(
            "alert_proxy_publish_errors_total",
            {"target": target.name, "error": "exception"},
        )
