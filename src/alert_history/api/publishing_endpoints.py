"""
Publishing Management API endpoints.

Endpoints for managing dynamic publishing targets:
- GET /publishing/targets - список активных targets
- POST /publishing/targets/refresh - принудительное обновление
- GET /publishing/mode - проверка режима работы
- GET /publishing/stats - статистика по targets
- POST /publishing/test/{target} - тестирование targets
- GET /publishing/secrets/template - templates для secrets
"""

# Standard library imports
import asyncio
import base64
from datetime import datetime
from typing import Any, Dict, List, Optional

# Third-party imports
from fastapi import APIRouter, Depends, HTTPException, Path
from pydantic import BaseModel

from ..core.app_state import app_state

# Local imports
from ..logging_config import get_logger
from ..services.alert_publisher import AlertPublisher
from ..services.target_discovery import DynamicTargetManager

logger = get_logger(__name__)

# Create router
publishing_router = APIRouter(prefix="/publishing", tags=["Publishing Management"])


# Pydantic models
class PublishingTargetInfo(BaseModel):
    """Information about a publishing target."""

    name: str
    format: str
    url: str
    enabled: bool
    priority: Optional[int] = None
    last_successful_publish: Optional[str] = None
    total_publishes: int = 0
    successful_publishes: int = 0
    failed_publishes: int = 0
    health_status: str = "unknown"


class PublishingModeInfo(BaseModel):
    """Publishing mode information."""

    mode: str  # "intelligent" or "metrics_only"
    targets_available: bool
    targets_count: int
    last_discovery_refresh: Optional[str] = None
    kubernetes_available: bool
    description: str


class PublishingStatsResponse(BaseModel):
    """Publishing statistics response."""

    total_targets: int
    active_targets: int
    total_publishes: int
    successful_publishes: int
    failed_publishes: int
    success_rate: float
    targets: List[PublishingTargetInfo]
    last_updated: str


class SecretTemplate(BaseModel):
    """Template for creating publishing target secrets."""

    apiVersion: str = "v1"
    kind: str = "Secret"
    metadata: Dict[str, Any]
    type: str = "Opaque"
    data: Dict[str, str]


class TestTargetRequest(BaseModel):
    """Request for testing a publishing target."""

    test_alert: Optional[Dict[str, Any]] = None
    timeout_seconds: int = 30


class TestTargetResponse(BaseModel):
    """Response from testing a publishing target."""

    target_name: str
    success: bool
    status_code: Optional[int] = None
    response_time_ms: int
    error_message: Optional[str] = None
    test_timestamp: str


# Dependency injection
async def get_target_manager() -> DynamicTargetManager:
    """Get target manager instance."""
    if not hasattr(app_state, "target_manager") or app_state.target_manager is None:
        from ..services.target_discovery import (
            DynamicTargetManager,
            TargetDiscoveryConfig,
        )

        config = TargetDiscoveryConfig(
            enabled=True,
            secret_labels=["alert-history.io/target=true"],
            secret_namespaces=["default", "monitoring", "alert-history"],
        )
        app_state.target_manager = DynamicTargetManager(config)

    return app_state.target_manager


async def get_alert_publisher() -> AlertPublisher:
    """Get alert publisher instance."""
    if not hasattr(app_state, "alert_publisher") or app_state.alert_publisher is None:
        app_state.alert_publisher = AlertPublisher()

    return app_state.alert_publisher


@publishing_router.get("/targets", response_model=List[PublishingTargetInfo])
async def get_publishing_targets(
    target_manager: DynamicTargetManager = Depends(get_target_manager),
    alert_publisher: AlertPublisher = Depends(get_alert_publisher),
):
    """
    Получить список всех активных publishing targets.

    Возвращает информацию о всех настроенных targets для публикации алертов,
    включая их статус, статистику и health check результаты.
    """
    try:
        # Get active targets from discovery
        active_targets = target_manager.get_active_targets()

        # Get publishing statistics
        publisher_stats = alert_publisher.get_all_stats()

        target_infos = []

        for target in active_targets:
            # Get stats for this specific target
            target_stats = publisher_stats.get(target.name, {})

            # Determine health status
            health_status = "healthy"
            if target_stats:
                success_rate = target_stats.get("success_rate", 0)
                if success_rate < 0.5:
                    health_status = "unhealthy"
                elif success_rate < 0.8:
                    health_status = "degraded"

            target_info = PublishingTargetInfo(
                name=target.name,
                format=target.format.value,
                url=target.url,
                enabled=target.enabled,
                total_publishes=target_stats.get("total_publishes", 0),
                successful_publishes=target_stats.get("successful_publishes", 0),
                failed_publishes=target_stats.get("failed_publishes", 0),
                health_status=health_status,
                last_successful_publish=target_stats.get("last_success_time"),
            )

            target_infos.append(target_info)

        logger.info(f"Retrieved {len(target_infos)} publishing targets")
        return target_infos

    except Exception as e:
        logger.error(f"Failed to get publishing targets: {e}")
        raise HTTPException(status_code=500, detail=f"Failed to get targets: {str(e)}")


@publishing_router.post("/targets/refresh")
async def refresh_publishing_targets(
    target_manager: DynamicTargetManager = Depends(get_target_manager),
):
    """
    Принудительно обновить список publishing targets из Kubernetes secrets.

    Запускает немедленное обновление target discovery из secrets
    вместо ожидания следующего планового обновления.
    """
    try:
        # Trigger manual refresh
        success = await target_manager.refresh_targets()

        if success:
            # Get updated target count
            targets_count = target_manager.get_targets_count()

            logger.info(
                f"Publishing targets refreshed successfully: {targets_count} targets"
            )

            return {
                "message": "Publishing targets refreshed successfully",
                "targets_discovered": targets_count,
                "refresh_timestamp": datetime.utcnow().isoformat(),
                "success": True,
            }
        else:
            logger.warning("Publishing targets refresh completed with issues")
            return {
                "message": "Publishing targets refresh completed with issues",
                "targets_discovered": target_manager.get_targets_count(),
                "refresh_timestamp": datetime.utcnow().isoformat(),
                "success": False,
            }

    except Exception as e:
        logger.error(f"Failed to refresh publishing targets: {e}")
        raise HTTPException(status_code=500, detail=f"Refresh failed: {str(e)}")


@publishing_router.get("/mode", response_model=PublishingModeInfo)
async def get_publishing_mode(
    target_manager: DynamicTargetManager = Depends(get_target_manager),
):
    """
    Получить информацию о текущем режиме публикации.

    Возвращает информацию о том, работает ли система в intelligent режиме
    с активными targets или в metrics-only режиме.
    """
    try:
        # Get discovery stats
        discovery_stats = target_manager.get_discovery_stats()

        # Determine mode
        metrics_only = target_manager.is_metrics_only_mode()
        targets_count = target_manager.get_targets_count()

        mode_info = PublishingModeInfo(
            mode="metrics_only" if metrics_only else "intelligent",
            targets_available=not metrics_only,
            targets_count=targets_count,
            last_discovery_refresh=discovery_stats.get("last_refresh_time"),
            kubernetes_available=discovery_stats.get("kubernetes_available", False),
            description=(
                "Metrics-only mode: No publishing targets available, collecting metrics only"
                if metrics_only
                else f"Intelligent mode: {targets_count} publishing targets active"
            ),
        )

        logger.debug(f"Publishing mode: {mode_info.mode}, targets: {targets_count}")
        return mode_info

    except Exception as e:
        logger.error(f"Failed to get publishing mode: {e}")
        raise HTTPException(status_code=500, detail=f"Failed to get mode: {str(e)}")


@publishing_router.get("/stats", response_model=PublishingStatsResponse)
async def get_publishing_stats(
    target_manager: DynamicTargetManager = Depends(get_target_manager),
    alert_publisher: AlertPublisher = Depends(get_alert_publisher),
):
    """
    Получить общую статистику публикации алертов.

    Возвращает агрегированную статистику по всем targets,
    включая успешность публикации и производительность.
    """
    try:
        # Get targets and stats
        active_targets = target_manager.get_active_targets()
        publisher_stats = alert_publisher.get_all_stats()

        # Calculate aggregate statistics
        total_targets = len(active_targets)
        active_targets_count = len([t for t in active_targets if t.enabled])

        total_publishes = 0
        successful_publishes = 0
        failed_publishes = 0

        target_infos = []

        for target in active_targets:
            target_stats = publisher_stats.get(target.name, {})

            target_total = target_stats.get("total_publishes", 0)
            target_success = target_stats.get("successful_publishes", 0)
            target_failed = target_stats.get("failed_publishes", 0)

            total_publishes += target_total
            successful_publishes += target_success
            failed_publishes += target_failed

            # Determine health status
            health_status = "healthy"
            if target_total > 0:
                success_rate = target_success / target_total
                if success_rate < 0.5:
                    health_status = "unhealthy"
                elif success_rate < 0.8:
                    health_status = "degraded"

            target_info = PublishingTargetInfo(
                name=target.name,
                format=target.format.value,
                url=target.url,
                enabled=target.enabled,
                total_publishes=target_total,
                successful_publishes=target_success,
                failed_publishes=target_failed,
                health_status=health_status,
                last_successful_publish=target_stats.get("last_success_time"),
            )

            target_infos.append(target_info)

        # Calculate overall success rate
        overall_success_rate = (
            successful_publishes / total_publishes if total_publishes > 0 else 0.0
        )

        stats_response = PublishingStatsResponse(
            total_targets=total_targets,
            active_targets=active_targets_count,
            total_publishes=total_publishes,
            successful_publishes=successful_publishes,
            failed_publishes=failed_publishes,
            success_rate=round(overall_success_rate, 3),
            targets=target_infos,
            last_updated=datetime.utcnow().isoformat(),
        )

        logger.debug(
            f"Publishing stats: {total_targets} targets, {overall_success_rate:.1%} success rate"
        )
        return stats_response

    except Exception as e:
        logger.error(f"Failed to get publishing stats: {e}")
        raise HTTPException(status_code=500, detail=f"Failed to get stats: {str(e)}")


@publishing_router.post("/test/{target_name}", response_model=TestTargetResponse)
async def test_publishing_target(
    target_name: str = Path(..., description="Name of the target to test"),
    test_request: TestTargetRequest = TestTargetRequest(),
    target_manager: DynamicTargetManager = Depends(get_target_manager),
    alert_publisher: AlertPublisher = Depends(get_alert_publisher),
):
    """
    Протестировать конкретный publishing target.

    Отправляет тестовый alert в указанный target и возвращает
    результат попытки публикации.
    """
    try:
        start_time = datetime.utcnow()

        # Get target by name
        target = target_manager.get_target_by_name(target_name)
        if not target:
            raise HTTPException(
                status_code=404, detail=f"Target '{target_name}' not found"
            )

        # Create test alert if not provided
        if not test_request.test_alert:
            test_alert_data = {
                "fingerprint": f"test-{target_name}-{int(start_time.timestamp())}",
                "labels": {
                    "alertname": "TestAlert",
                    "severity": "info",
                    "source": "alert-history-test",
                },
                "annotations": {
                    "description": f"Test alert for target {target_name}",
                    "summary": "Publishing target test",
                },
                "status": "firing",
                "startsAt": start_time.isoformat(),
                "generatorURL": "http://alert-history/test",
            }
        else:
            test_alert_data = test_request.test_alert

        # Convert to Alert object and EnrichedAlert
        from ..core.interfaces import Alert, AlertStatus, EnrichedAlert

        alert = Alert(
            fingerprint=test_alert_data.get("fingerprint", "test-alert"),
            alert_name=test_alert_data.get("labels", {}).get("alertname", "TestAlert"),
            status=AlertStatus(test_alert_data.get("status", "firing")),
            labels=test_alert_data.get("labels", {}),
            annotations=test_alert_data.get("annotations", {}),
            starts_at=start_time,
            ends_at=None,
            generator_url=test_alert_data.get("generatorURL", ""),
        )

        enriched_alert = EnrichedAlert(alert=alert)

        # Test publishing
        try:
            publish_result = await alert_publisher.publish_alert(enriched_alert, target)

            end_time = datetime.utcnow()
            response_time = int((end_time - start_time).total_seconds() * 1000)

            success = publish_result.get("success", False)
            status_code = publish_result.get("status_code")
            error_message = publish_result.get("error") if not success else None

            result = TestTargetResponse(
                target_name=target_name,
                success=success,
                status_code=status_code,
                response_time_ms=response_time,
                error_message=error_message,
                test_timestamp=start_time.isoformat(),
            )

            logger.info(
                f"Target test completed: {target_name}, success: {success}, time: {response_time}ms"
            )
            return result

        except asyncio.TimeoutError:
            response_time = test_request.timeout_seconds * 1000

            return TestTargetResponse(
                target_name=target_name,
                success=False,
                response_time_ms=response_time,
                error_message=f"Test timeout after {test_request.timeout_seconds} seconds",
                test_timestamp=start_time.isoformat(),
            )

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Failed to test target {target_name}: {e}")
        raise HTTPException(status_code=500, detail=f"Test failed: {str(e)}")


@publishing_router.get("/secrets/template", response_model=List[SecretTemplate])
async def get_secret_templates():
    """
    Получить templates для создания Kubernetes secrets с publishing targets.

    Возвращает примеры YAML конфигураций для различных типов targets
    (Rootly, PagerDuty, Slack, etc).
    """
    try:
        templates = []

        # Rootly template
        rootly_template = SecretTemplate(
            metadata={
                "name": "rootly-production",
                "namespace": "default",
                "labels": {
                    "alert-history.io/target": "true",
                    "alert-history.io/format": "rootly",
                    "alert-history.io/priority": "100",
                },
            },
            data={
                "url": base64.b64encode(
                    b"https://api.rootly.com/v1/incidents"
                ).decode(),
                "token": base64.b64encode(b"your-rootly-api-key").decode(),
                "format": base64.b64encode(b"rootly").decode(),
                "enabled": base64.b64encode(b"true").decode(),
                "timeout": base64.b64encode(b"30").decode(),
            },
        )
        templates.append(rootly_template)

        # PagerDuty template
        pagerduty_template = SecretTemplate(
            metadata={
                "name": "pagerduty-oncall",
                "namespace": "default",
                "labels": {
                    "alert-history.io/target": "true",
                    "alert-history.io/format": "pagerduty",
                    "alert-history.io/priority": "1",
                },
            },
            data={
                "url": base64.b64encode(
                    b"https://events.pagerduty.com/v2/enqueue"
                ).decode(),
                "routing_key": base64.b64encode(b"your-pagerduty-routing-key").decode(),
                "format": base64.b64encode(b"pagerduty").decode(),
                "enabled": base64.b64encode(b"true").decode(),
                "timeout": base64.b64encode(b"30").decode(),
            },
        )
        templates.append(pagerduty_template)

        # Slack template
        slack_template = SecretTemplate(
            metadata={
                "name": "slack-alerts",
                "namespace": "default",
                "labels": {
                    "alert-history.io/target": "true",
                    "alert-history.io/format": "slack",
                    "alert-history.io/priority": "200",
                },
            },
            data={
                "url": base64.b64encode(
                    b"https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK"
                ).decode(),
                "format": base64.b64encode(b"slack").decode(),
                "enabled": base64.b64encode(b"true").decode(),
                "timeout": base64.b64encode(b"30").decode(),
                "channel": base64.b64encode(b"#alerts").decode(),
            },
        )
        templates.append(slack_template)

        logger.info(f"Generated {len(templates)} secret templates")
        return templates

    except Exception as e:
        logger.error(f"Failed to generate secret templates: {e}")
        raise HTTPException(
            status_code=500, detail=f"Failed to generate templates: {str(e)}"
        )


@publishing_router.get("/health")
async def publishing_health_check():
    """Health check для publishing management API."""
    return {
        "status": "healthy",
        "service": "publishing-management",
        "endpoints": [
            "GET /publishing/targets",
            "POST /publishing/targets/refresh",
            "GET /publishing/mode",
            "GET /publishing/stats",
            "POST /publishing/test/{target}",
            "GET /publishing/secrets/template",
        ],
        "timestamp": datetime.utcnow().isoformat(),
    }
