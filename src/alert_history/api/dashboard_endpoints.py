"""
Dashboard API endpoints для T6: Dashboard и UI интеграция.

Provides:
- Consolidated API для dashboard data
- Real-time statistics
- Alert analytics
- System health overview
- Chart data endpoints
"""

import os
from datetime import datetime, timedelta
from typing import Any, Dict, List, Optional

from fastapi import APIRouter, Depends, HTTPException, Query
from fastapi.responses import JSONResponse
from pydantic import BaseModel, Field

from ..api.metrics import LegacyMetrics
from ..core.app_state import app_state
from ..core.interfaces import AlertStatus
from ..database.sqlite_adapter import SQLiteLegacyStorage
from ..logging_config import get_logger
from ..services.alert_classifier import AlertClassificationService
from ..services.alert_publisher import AlertPublisher
from ..services.target_discovery import DynamicTargetManager

logger = get_logger(__name__)

# Create router
dashboard_router = APIRouter(prefix="/api/dashboard", tags=["Dashboard API"])


# Pydantic models
class DashboardOverview(BaseModel):
    """Main dashboard overview data."""

    # Alert statistics
    total_alerts: int = 0
    active_alerts: int = 0
    resolved_alerts: int = 0
    alerts_last_24h: int = 0

    # Classification statistics
    classification_enabled: bool = False
    classified_alerts: int = 0
    classification_cache_hit_rate: float = 0.0

    # Publishing statistics
    publishing_targets: int = 0
    publishing_mode: str = "unknown"
    successful_publishes: int = 0
    failed_publishes: int = 0

    # System health
    system_healthy: bool = True
    redis_connected: bool = False
    llm_service_available: bool = False

    # Timestamps
    last_updated: str = Field(default_factory=lambda: datetime.utcnow().isoformat())


class AlertTimeSeriesData(BaseModel):
    """Time series data for alert charts."""

    timestamp: str
    firing_alerts: int = 0
    resolved_alerts: int = 0
    critical_alerts: int = 0
    warning_alerts: int = 0
    info_alerts: int = 0


class TopAlertsData(BaseModel):
    """Top alerts by frequency."""

    alert_name: str
    namespace: Optional[str] = None
    count: int = 0
    last_seen: str
    severity: str = "unknown"
    status: str = "unknown"


class SystemHealthData(BaseModel):
    """System health overview."""

    component: str
    status: str  # healthy, unhealthy, degraded, unknown
    details: Optional[Dict[str, Any]] = None
    last_check: str = Field(default_factory=lambda: datetime.utcnow().isoformat())


class DashboardChartsData(BaseModel):
    """Chart data for dashboard visualizations."""

    # Time series for line charts
    alert_timeline: List[AlertTimeSeriesData] = []

    # Top alerts for bar charts
    top_alerts_by_frequency: List[TopAlertsData] = []
    top_alerts_by_severity: List[TopAlertsData] = []

    # Distribution data for pie charts
    severity_distribution: Dict[str, int] = {}
    status_distribution: Dict[str, int] = {}
    namespace_distribution: Dict[str, int] = {}


class RecommendationItem(BaseModel):
    fingerprint: str
    alert_name: Optional[str] = None
    severity: str
    confidence: float
    reasoning: Optional[str] = None
    recommendations: List[str] = []
    created_at: str


class RecommendationsResponse(BaseModel):
    total: int
    items: List[RecommendationItem]


# Dependency injection helpers
def get_storage() -> SQLiteLegacyStorage:
    """Get storage instance from app state."""
    storage = getattr(app_state, "storage", None)
    if storage:
        return storage

    # Fallback: initialize legacy SQLite storage if not present
    try:
        db_path = os.getenv("SQLITE_PATH", "alert_history.sqlite3")
        storage = SQLiteLegacyStorage(db_path)
        app_state.storage = storage
        logger.info(f"Initialized fallback SQLite storage at {db_path}")
        return storage
    except Exception as e:
        logger.error(f"Failed to initialize fallback storage: {e}")
        raise HTTPException(status_code=503, detail="Storage not available")


def get_classification_service() -> Optional[AlertClassificationService]:
    """Get classification service from app state."""
    return getattr(app_state, "classification_service", None)


def get_target_manager() -> Optional[DynamicTargetManager]:
    """Get target manager from app state."""
    return getattr(app_state, "target_manager", None)


def get_alert_publisher() -> Optional[AlertPublisher]:
    """Get alert publisher from app state."""
    return getattr(app_state, "alert_publisher", None)


def get_metrics() -> Optional[LegacyMetrics]:
    """Get metrics from app state."""
    return getattr(app_state, "metrics", None)


# API Endpoints
@dashboard_router.get("/overview", response_model=DashboardOverview)
async def get_dashboard_overview(
    storage: SQLiteLegacyStorage = Depends(get_storage),
    classification_service: Optional[AlertClassificationService] = Depends(
        get_classification_service
    ),
    target_manager: Optional[DynamicTargetManager] = Depends(get_target_manager),
    alert_publisher: Optional[AlertPublisher] = Depends(get_alert_publisher),
    metrics: Optional[LegacyMetrics] = Depends(get_metrics),
) -> DashboardOverview:
    """Get main dashboard overview data."""

    try:
        overview = DashboardOverview()

        # Get alert statistics
        all_alerts = await storage.get_alerts({}, limit=10000, offset=0)
        overview.total_alerts = len(all_alerts)

        # Count by status
        active_count = sum(
            1 for alert in all_alerts if alert.status == AlertStatus.FIRING
        )
        resolved_count = sum(
            1 for alert in all_alerts if alert.status == AlertStatus.RESOLVED
        )

        overview.active_alerts = active_count
        overview.resolved_alerts = resolved_count

        # Alerts in last 24h
        yesterday = datetime.utcnow() - timedelta(days=1)
        recent_alerts = [
            alert
            for alert in all_alerts
            if alert.timestamp and alert.timestamp >= yesterday
        ]
        overview.alerts_last_24h = len(recent_alerts)

        # Classification statistics
        if classification_service:
            overview.classification_enabled = True
            try:
                class_stats = await classification_service.get_classification_stats()
                overview.classified_alerts = class_stats.get("total_requests", 0)
                overview.classification_cache_hit_rate = class_stats.get(
                    "cache_hit_rate", 0.0
                )
                overview.llm_service_available = class_stats.get("llm_available", False)
            except Exception as e:
                logger.warning(f"Failed to get classification stats: {e}")

        # Publishing statistics
        if target_manager:
            overview.publishing_targets = target_manager.get_targets_count()
            overview.publishing_mode = (
                "metrics_only"
                if target_manager.is_metrics_only_mode()
                else "intelligent"
            )

        if alert_publisher:
            try:
                pub_stats = await alert_publisher.get_publishing_stats()
                overview.successful_publishes = pub_stats.get("successful_publishes", 0)
                overview.failed_publishes = pub_stats.get("failed_publishes", 0)
            except Exception as e:
                logger.warning(f"Failed to get publishing stats: {e}")

        # System health checks
        overview.system_healthy = True  # Will be updated by health checks

        # Check Redis connection
        redis_cache = getattr(app_state, "redis_cache", None)
        if redis_cache:
            try:
                redis_health = await redis_cache.health_check()
                overview.redis_connected = redis_health.get("status") == "healthy"
            except Exception:
                overview.redis_connected = False

        logger.debug(f"Dashboard overview generated: {overview.total_alerts} alerts")
        return overview

    except Exception as e:
        logger.error(f"Failed to generate dashboard overview: {e}")
        raise HTTPException(status_code=500, detail=f"Failed to get overview: {str(e)}")


@dashboard_router.get("/charts", response_model=DashboardChartsData)
async def get_dashboard_charts(
    hours: int = Query(24, ge=1, le=168, description="Hours of data to include"),
    storage: SQLiteLegacyStorage = Depends(get_storage),
) -> DashboardChartsData:
    """Get chart data for dashboard visualizations."""

    try:
        charts_data = DashboardChartsData()

        # Get alerts for the specified time range
        since = datetime.utcnow() - timedelta(hours=hours)
        all_alerts = await storage.get_alerts({}, limit=10000, offset=0)

        # Filter alerts within time range
        filtered_alerts = [
            alert
            for alert in all_alerts
            if alert.timestamp and alert.timestamp >= since
        ]

        # Generate time series data (hourly buckets)
        time_buckets = {}
        for i in range(hours):
            bucket_time = since + timedelta(hours=i)
            bucket_key = bucket_time.strftime("%Y-%m-%d %H:00")
            time_buckets[bucket_key] = AlertTimeSeriesData(
                timestamp=bucket_time.isoformat(),
                firing_alerts=0,
                resolved_alerts=0,
                critical_alerts=0,
                warning_alerts=0,
                info_alerts=0,
            )

        # Fill time buckets with alert data
        for alert in filtered_alerts:
            if not alert.timestamp:
                continue

            bucket_time = alert.timestamp.replace(minute=0, second=0, microsecond=0)
            bucket_key = bucket_time.strftime("%Y-%m-%d %H:00")

            if bucket_key in time_buckets:
                bucket = time_buckets[bucket_key]

                # Count by status
                if alert.status == AlertStatus.FIRING:
                    bucket.firing_alerts += 1
                elif alert.status == AlertStatus.RESOLVED:
                    bucket.resolved_alerts += 1

                # Count by severity (if available in labels)
                severity = alert.labels.get("severity", "unknown").lower()
                if severity == "critical":
                    bucket.critical_alerts += 1
                elif severity == "warning":
                    bucket.warning_alerts += 1
                elif severity in ["info", "low"]:
                    bucket.info_alerts += 1

        charts_data.alert_timeline = list(time_buckets.values())

        # Generate top alerts by frequency
        alert_counts = {}
        for alert in filtered_alerts:
            key = (alert.alert_name, alert.namespace)
            if key not in alert_counts:
                alert_counts[key] = {
                    "count": 0,
                    "last_seen": alert.timestamp,
                    "severity": alert.labels.get("severity", "unknown"),
                    "status": alert.status.value,
                }
            alert_counts[key]["count"] += 1
            if alert.timestamp > alert_counts[key]["last_seen"]:
                alert_counts[key]["last_seen"] = alert.timestamp
                alert_counts[key]["status"] = alert.status.value

        # Sort by frequency and take top 10
        top_by_frequency = sorted(
            alert_counts.items(), key=lambda x: x[1]["count"], reverse=True
        )[:10]

        charts_data.top_alerts_by_frequency = [
            TopAlertsData(
                alert_name=name,
                namespace=ns,
                count=data["count"],
                last_seen=data["last_seen"].isoformat(),
                severity=data["severity"],
                status=data["status"],
            )
            for (name, ns), data in top_by_frequency
        ]

        # Generate distribution data
        severity_counts = {}
        status_counts = {}
        namespace_counts = {}

        for alert in filtered_alerts:
            # Severity distribution
            severity = alert.labels.get("severity", "unknown")
            severity_counts[severity] = severity_counts.get(severity, 0) + 1

            # Status distribution
            status = alert.status.value
            status_counts[status] = status_counts.get(status, 0) + 1

            # Namespace distribution
            namespace = alert.namespace or "unknown"
            namespace_counts[namespace] = namespace_counts.get(namespace, 0) + 1

        charts_data.severity_distribution = severity_counts
        charts_data.status_distribution = status_counts
        charts_data.namespace_distribution = namespace_counts

        logger.debug(
            f"Generated charts data for {len(filtered_alerts)} alerts over {hours} hours"
        )
        return charts_data

    except Exception as e:
        logger.error(f"Failed to generate charts data: {e}")
        raise HTTPException(
            status_code=500, detail=f"Failed to get charts data: {str(e)}"
        )


@dashboard_router.get("/health", response_model=List[SystemHealthData])
async def get_system_health(
    classification_service: Optional[AlertClassificationService] = Depends(
        get_classification_service
    ),
    target_manager: Optional[DynamicTargetManager] = Depends(get_target_manager),
    alert_publisher: Optional[AlertPublisher] = Depends(get_alert_publisher),
) -> List[SystemHealthData]:
    """Get system health status for all components."""

    health_data = []

    # Database health
    try:
        storage = get_storage()
        # Simple test query
        await storage.get_alerts({}, limit=1, offset=0)
        health_data.append(
            SystemHealthData(
                component="database",
                status="healthy",
                details={"type": "sqlite", "available": True},
            )
        )
    except Exception as e:
        health_data.append(
            SystemHealthData(
                component="database", status="unhealthy", details={"error": str(e)}
            )
        )

    # Redis health
    redis_cache = getattr(app_state, "redis_cache", None)
    if redis_cache:
        try:
            redis_health = await redis_cache.health_check()
            health_data.append(
                SystemHealthData(
                    component="redis",
                    status=redis_health.get("status", "unknown"),
                    details=redis_health,
                )
            )
        except Exception as e:
            health_data.append(
                SystemHealthData(
                    component="redis", status="unhealthy", details={"error": str(e)}
                )
            )
    else:
        health_data.append(
            SystemHealthData(
                component="redis",
                status="not_configured",
                details={"message": "Redis not configured"},
            )
        )

    # Classification service health
    if classification_service:
        try:
            class_health = await classification_service.health_check()
            health_data.append(
                SystemHealthData(
                    component="classification",
                    status=(
                        "healthy" if class_health.get("llm_available") else "degraded"
                    ),
                    details=class_health,
                )
            )
        except Exception as e:
            health_data.append(
                SystemHealthData(
                    component="classification",
                    status="unhealthy",
                    details={"error": str(e)},
                )
            )
    else:
        health_data.append(
            SystemHealthData(
                component="classification",
                status="not_configured",
                details={"message": "Classification service not configured"},
            )
        )

    # Publishing targets health
    if target_manager:
        try:
            discovery_stats = target_manager.get_discovery_stats()
            health_data.append(
                SystemHealthData(
                    component="publishing",
                    status=(
                        "healthy"
                        if not target_manager.is_metrics_only_mode()
                        else "degraded"
                    ),
                    details={
                        "targets_count": target_manager.get_targets_count(),
                        "mode": (
                            "metrics_only"
                            if target_manager.is_metrics_only_mode()
                            else "intelligent"
                        ),
                        "discovery_stats": discovery_stats,
                    },
                )
            )
        except Exception as e:
            health_data.append(
                SystemHealthData(
                    component="publishing",
                    status="unhealthy",
                    details={"error": str(e)},
                )
            )
    else:
        health_data.append(
            SystemHealthData(
                component="publishing",
                status="not_configured",
                details={"message": "Publishing not configured"},
            )
        )

    return health_data


@dashboard_router.get("/alerts/recent")
async def get_recent_alerts(
    limit: int = Query(50, ge=1, le=200, description="Number of recent alerts"),
    severity: Optional[str] = Query(None, description="Filter by severity"),
    status: Optional[str] = Query(None, description="Filter by status"),
    min_confidence: Optional[float] = Query(
        None, ge=0.0, le=1.0, description="Filter by minimum classification confidence"
    ),
    include_classification: bool = Query(
        False, description="Include classification details"
    ),
    namespace: Optional[str] = Query(None, description="Filter by namespace"),
    storage: SQLiteLegacyStorage = Depends(get_storage),
) -> JSONResponse:
    """Get recent alerts with optional filtering."""

    try:
        # Build filters
        filters = {}
        if namespace:
            filters["namespace"] = namespace
        if status:
            filters["status"] = status

        # Get alerts
        alerts = await storage.get_alerts(filters, limit=limit, offset=0)

        # Convert to JSON-serializable format
        alert_data = []
        for alert in alerts:
            # Apply severity filter if specified
            if severity and alert.labels.get("severity") != severity:
                continue

            item = {
                "fingerprint": alert.fingerprint,
                "alert_name": alert.alert_name,
                "namespace": alert.namespace,
                "status": alert.status.value,
                "severity": alert.labels.get("severity", "unknown"),
                "labels": alert.labels,
                "annotations": alert.annotations,
                "timestamp": alert.timestamp.isoformat() if alert.timestamp else None,
                "starts_at": alert.starts_at.isoformat() if alert.starts_at else None,
                "ends_at": alert.ends_at.isoformat() if alert.ends_at else None,
            }

            if include_classification:
                try:
                    cls = await storage.get_classification(alert.fingerprint)
                    if cls:
                        # Filter by min confidence if provided
                        if (min_confidence is not None) and (
                            cls.confidence < float(min_confidence)
                        ):
                            continue
                        item.update(
                            {
                                "classified_severity": cls.severity.value,
                                "classified_confidence": cls.confidence,
                                "classified_reasoning": cls.reasoning,
                                "classified_recommendations": cls.recommendations,
                            }
                        )
                except Exception:
                    pass

            alert_data.append(item)

        return JSONResponse(
            {
                "alerts": alert_data[
                    :limit
                ],  # Ensure we don't exceed limit after filtering
                "total": len(alert_data),
                "filters_applied": {
                    "severity": severity,
                    "status": status,
                    "namespace": namespace,
                    "limit": limit,
                },
            }
        )

    except Exception as e:
        logger.error(f"Failed to get recent alerts: {e}")
        raise HTTPException(
            status_code=500, detail=f"Failed to get recent alerts: {str(e)}"
        )


@dashboard_router.get("/recommendations", response_model=RecommendationsResponse)
async def get_recommendations(
    limit: int = Query(20, ge=1, le=200),
    min_confidence: float = Query(0.5, ge=0.0, le=1.0),
    storage: SQLiteLegacyStorage = Depends(get_storage),
) -> RecommendationsResponse:
    """Get recent recommendations from classification results."""
    try:
        import sqlite3

        items: List[RecommendationItem] = []

        # Open direct connection using storage path
        db_path = storage.db_path  # type: ignore[attr-defined]
        conn = sqlite3.connect(db_path)
        c = conn.cursor()

        c.execute(
            """
            SELECT ac.fingerprint, ac.severity, ac.confidence, ac.reasoning, ac.recommendations, ac.created_at,
                   a.alertname
            FROM alert_classifications ac
            LEFT JOIN alerts a ON a.fingerprint = ac.fingerprint
            WHERE ac.confidence >= ?
            ORDER BY ac.created_at DESC
            LIMIT ?
            """,
            (min_confidence, limit),
        )

        rows = c.fetchall()
        conn.close()

        for row in rows:
            (
                fingerprint,
                severity,
                confidence,
                reasoning,
                rec_json,
                created_at,
                alertname,
            ) = row
            try:
                import json

                rec_list = json.loads(rec_json) if rec_json else []
            except Exception:
                rec_list = []
            items.append(
                RecommendationItem(
                    fingerprint=fingerprint,
                    alert_name=alertname,
                    severity=severity,
                    confidence=float(confidence),
                    reasoning=reasoning,
                    recommendations=rec_list,
                    created_at=created_at,
                )
            )

        return RecommendationsResponse(total=len(items), items=items)

    except Exception as e:
        logger.error(f"Failed to get recommendations: {e}")
        raise HTTPException(status_code=500, detail="Failed to get recommendations")
