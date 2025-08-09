"""
Legacy API Adapter for backward compatibility.

Maintains 100% compatibility with existing alert_history_service.py:
- Same endpoints (/webhook, /history, /report, /metrics, /dashboard)
- Same request/response formats
- Same SQLite database structure (with migration path)
- Same Prometheus metrics
"""

# Standard library imports
import json
import sqlite3
import time
from datetime import datetime, timedelta
from typing import Any, Dict, List, Optional, TYPE_CHECKING

if TYPE_CHECKING:
    from ..services.alert_classifier import AlertClassificationService

# Third-party imports
from fastapi import FastAPI, Form, HTTPException, Query, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import HTMLResponse, JSONResponse, Response
from fastapi.templating import Jinja2Templates
from prometheus_client import CONTENT_TYPE_LATEST, generate_latest

# Local imports
from ..core.interfaces import Alert, AlertStatus, IAlertStorage
from ..database.sqlite_adapter import SQLiteLegacyStorage
from ..services.webhook_processor import WebhookProcessor
from ..utils.common import generate_fingerprint, parse_timestamp
from .metrics import LegacyMetrics


class LegacyAPIAdapter:
    """
    Adapter that provides exact same API as original alert_history_service.py.

    This ensures zero-downtime migration and backward compatibility.
    """

    def __init__(
        self,
        app: FastAPI,
        storage: Optional[IAlertStorage] = None,
        db_path: str = "alert_history.sqlite3",
        retention_days: int = 30,
        webhook_processor: Optional[WebhookProcessor] = None,
    ):
        """Initialize legacy API adapter."""
        self.app = app
        self.db_path = db_path
        self.retention_days = retention_days

        # Use provided storage or create SQLite legacy storage
        self.storage = storage or SQLiteLegacyStorage(db_path)

        # Legacy metrics (exact same as original)
        self.metrics = LegacyMetrics()

        # Webhook processor for intelligent processing
        self.webhook_processor = webhook_processor

        # Optional classification service for LLM integration
        self._classification_service = self._init_classification_service()

        # Templates for dashboard (same path as original)
        self.templates = Jinja2Templates(directory="templates")

        # Setup middleware (exact same as original)
        self._setup_middleware()

        # Register legacy endpoints
        self._register_legacy_endpoints()

    def _init_classification_service(self) -> Optional["AlertClassificationService"]:
        """
        Инициализировать classification service, если возможно.

        Returns:
            AlertClassificationService или None если не настроен
        """
        try:
            # Проверяем конфигурацию
            from config import get_config

            config = get_config()

            # Проверяем что LLM настроен
            if not (hasattr(config, "llm") and config.llm and config.llm.enabled):
                return None

            # Импортируем компоненты
            from ..services.alert_classifier import AlertClassificationService
            from ..services.llm_client import LLMProxyClient
            from ..services.redis_cache import RedisCache

            # Инициализируем LLM клиент
            llm_client = LLMProxyClient(
                proxy_url=config.llm.proxy_url,
                api_key=config.llm.api_key,
                model=config.llm.model,
                timeout=config.llm.timeout,
                max_retries=config.llm.max_retries,
            )

            # Инициализируем кеш (опционально)
            cache = None
            if config.redis:
                cache = RedisCache(
                    redis_url=config.redis.redis_url,
                    default_ttl=3600,  # 1 час для классификаций
                )

            # Создаем classification service
            classification_service = AlertClassificationService(
                llm_client=llm_client,
                storage=self.storage,
                cache=cache,
                cache_ttl=3600,
                enable_fallback=True,
            )

            print("LLM Classification enabled for legacy webhook")
            return classification_service

        except Exception as e:
            print(f"Failed to initialize classification service: {e}")
            return None

    def _setup_middleware(self) -> None:
        """Setup CORS middleware (same as original)."""
        self.app.add_middleware(
            CORSMiddleware,
            allow_origins=["*"],
            allow_credentials=True,
            allow_methods=["*"],
            allow_headers=["*"],
        )

    def _register_legacy_endpoints(self) -> None:
        """Register all legacy endpoints with exact same signatures."""

        @self.app.post("/webhook")
        async def webhook_endpoint(request: Request) -> JSONResponse:
            """Legacy webhook endpoint - exact same as original."""
            return await self._handle_webhook(request)

        @self.app.get("/history")
        async def history_endpoint(
            alertname: Optional[str] = Query(None),
            status: Optional[str] = Query(None),
            fingerprint: Optional[str] = Query(None),
            start_time: Optional[str] = Query(None),
            end_time: Optional[str] = Query(None),
            limit: int = Query(100),
        ) -> JSONResponse:
            """Legacy history endpoint - exact same as original."""
            return await self._handle_history(
                alertname, status, fingerprint, start_time, end_time, limit
            )

        @self.app.get("/report")
        async def report_endpoint(
            days: int = Query(7), group_by: str = Query("alertname")
        ) -> JSONResponse:
            """Legacy report endpoint - exact same as original."""
            return await self._handle_report(days, group_by)

        @self.app.get("/metrics")
        async def metrics_endpoint() -> Response:
            """Legacy metrics endpoint - exact same as original."""
            return await self._handle_metrics()

        @self.app.get("/dashboard", response_class=HTMLResponse)
        async def dashboard_endpoint(request: Request) -> HTMLResponse:
            """Legacy dashboard endpoint - exact same as original."""
            return await self._handle_dashboard(request)

        @self.app.get("/dashboard/grouped", response_class=HTMLResponse)
        async def dashboard_grouped_endpoint(request: Request) -> HTMLResponse:
            """Legacy grouped dashboard endpoint."""
            return await self._handle_dashboard_grouped(request)

        @self.app.get("/health")
        async def health_endpoint() -> JSONResponse:
            """Health check endpoint."""
            return JSONResponse({"status": "healthy", "timestamp": time.time()})

    async def _handle_webhook(self, request: Request) -> JSONResponse:
        """Handle webhook request - same logic as original."""
        try:
            # Record request start time
            start_time = time.time()

            # Parse request body
            body = await request.body()
            webhook_data = json.loads(body.decode("utf-8"))

            # Process alerts (same logic as original)
            processed_count = 0

            for alert_data in webhook_data.get("alerts", []):
                try:
                    # Convert to internal Alert format
                    alert = self._convert_webhook_to_alert(alert_data)

                    # Save to storage (maintains SQLite compatibility)
                    await self.storage.save_alert(alert)

                    # Optional: LLM classification (if enabled)
                    await self._maybe_classify_alert(alert)

                    processed_count += 1

                    # Update metrics (same as original)
                    self.metrics.increment_alerts_received(alert.alert_name, alert.status.value)

                except Exception as e:
                    # Log error but continue processing other alerts
                    print(f"Error processing alert: {e}")
                    self.metrics.increment_webhook_errors()

            # Record processing time
            processing_time = time.time() - start_time
            self.metrics.observe_webhook_duration(processing_time)

            return JSONResponse(
                {
                    "status": "success",
                    "processed": processed_count,
                    "timestamp": start_time,
                }
            )

        except Exception as e:
            self.metrics.increment_webhook_errors()
            raise HTTPException(status_code=400, detail=str(e))

    async def _handle_history(
        self,
        alertname: Optional[str],
        status: Optional[str],
        fingerprint: Optional[str],
        start_time: Optional[str],
        end_time: Optional[str],
        limit: int,
    ) -> JSONResponse:
        """Handle history request - same logic as original."""
        try:
            # Build filters (same logic as original)
            filters = {}

            if alertname:
                filters["alertname"] = alertname
            if status:
                filters["status"] = status
            if fingerprint:
                filters["fingerprint"] = fingerprint
            if start_time:
                filters["start_time"] = start_time
            if end_time:
                filters["end_time"] = end_time

            # Get alerts from storage
            alerts = await self.storage.get_alerts(filters, limit, 0)

            # Convert to legacy format (same as original response)
            legacy_alerts = []
            for alert in alerts:
                legacy_alerts.append(
                    {
                        "fingerprint": alert.fingerprint,
                        "alertname": alert.alert_name,
                        "status": alert.status.value,
                        "labels": alert.labels,
                        "annotations": alert.annotations,
                        "startsAt": (alert.starts_at.isoformat() if alert.starts_at else None),
                        "endsAt": alert.ends_at.isoformat() if alert.ends_at else None,
                        "generatorURL": alert.generator_url,
                    }
                )

            return JSONResponse({"alerts": legacy_alerts, "total": len(legacy_alerts)})

        except Exception as e:
            raise HTTPException(status_code=500, detail=str(e))

    async def _handle_report(self, days: int, group_by: str) -> JSONResponse:
        """Handle report request - same logic as original."""
        try:
            # Calculate time range
            end_time = datetime.utcnow()
            start_time = end_time - timedelta(days=days)

            # Get alerts for the time range
            filters = {
                "start_time": start_time.isoformat(),
                "end_time": end_time.isoformat(),
            }

            alerts = await self.storage.get_alerts(filters, 10000, 0)  # Large limit for report

            # Group alerts (same logic as original)
            grouped_data = {}

            for alert in alerts:
                if group_by == "alertname":
                    key = alert.alert_name
                elif group_by == "status":
                    key = alert.status.value
                elif group_by == "namespace":
                    key = alert.namespace or "unknown"
                else:
                    key = "all"

                if key not in grouped_data:
                    grouped_data[key] = {"count": 0, "firing": 0, "resolved": 0}

                grouped_data[key]["count"] += 1
                if alert.status == AlertStatus.FIRING:
                    grouped_data[key]["firing"] += 1
                else:
                    grouped_data[key]["resolved"] += 1

            return JSONResponse(
                {
                    "period_days": days,
                    "group_by": group_by,
                    "data": grouped_data,
                    "total_alerts": len(alerts),
                }
            )

        except Exception as e:
            raise HTTPException(status_code=500, detail=str(e))

    async def _handle_metrics(self) -> Response:
        """Handle metrics request - same format as original."""
        # Generate Prometheus metrics (same format)
        metrics_output = generate_latest(self.metrics.registry)

        return Response(content=metrics_output, media_type=CONTENT_TYPE_LATEST)

    async def _handle_dashboard(self, request: Request) -> HTMLResponse:
        """Handle dashboard request - same template as original."""
        try:
            # Get recent alerts for dashboard
            recent_alerts = await self.storage.get_alerts({}, 50, 0)

            # Convert to template format (same as original)
            dashboard_alerts = []
            for alert in recent_alerts:
                dashboard_alerts.append(
                    {
                        "fingerprint": alert.fingerprint,
                        "alertname": alert.alert_name,
                        "status": alert.status.value,
                        "severity": alert.labels.get("severity", "unknown"),
                        "namespace": alert.namespace or "unknown",
                        "startsAt": (
                            alert.starts_at.strftime("%Y-%m-%d %H:%M:%S") if alert.starts_at else ""
                        ),
                        "labels": alert.labels,
                        "annotations": alert.annotations,
                    }
                )

            return self.templates.TemplateResponse(
                "html5_dashboard.html",
                {
                    "request": request,
                    "alerts": dashboard_alerts,
                    "title": "Alert History Dashboard",
                },
            )

        except Exception as e:
            raise HTTPException(status_code=500, detail=str(e))

    async def _handle_dashboard_grouped(self, request: Request) -> HTMLResponse:
        """Handle grouped dashboard request."""
        try:
            # Get recent alerts
            recent_alerts = await self.storage.get_alerts({}, 200, 0)

            # Group by alertname (same logic as original)
            grouped_alerts = {}

            for alert in recent_alerts:
                alert_name = alert.alert_name
                if alert_name not in grouped_alerts:
                    grouped_alerts[alert_name] = {
                        "alertname": alert_name,
                        "count": 0,
                        "firing": 0,
                        "resolved": 0,
                        "latest": None,
                    }

                grouped_alerts[alert_name]["count"] += 1
                if alert.status == AlertStatus.FIRING:
                    grouped_alerts[alert_name]["firing"] += 1
                else:
                    grouped_alerts[alert_name]["resolved"] += 1

                # Update latest alert
                if (
                    grouped_alerts[alert_name]["latest"] is None
                    or alert.starts_at > grouped_alerts[alert_name]["latest"].starts_at
                ):
                    grouped_alerts[alert_name]["latest"] = alert

            return self.templates.TemplateResponse(
                "html5_dashboard.html",
                {
                    "request": request,
                    "grouped_alerts": list(grouped_alerts.values()),
                    "title": "Alert History Dashboard (Grouped)",
                },
            )

        except Exception as e:
            raise HTTPException(status_code=500, detail=str(e))

    def _convert_webhook_to_alert(self, alert_data: Dict[str, Any]) -> Alert:
        """Convert webhook alert data to internal Alert format."""
        # Extract required fields
        labels = alert_data.get("labels", {})
        annotations = alert_data.get("annotations", {})

        # Generate fingerprint (same logic as original)
        fingerprint = alert_data.get("fingerprint")
        if not fingerprint:
            fingerprint = generate_fingerprint(labels)

        # Parse timestamps
        starts_at = parse_timestamp(alert_data.get("startsAt", datetime.utcnow().isoformat()))
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

    async def _maybe_classify_alert(self, alert: "Alert") -> None:
        """
        Опционально классифицировать алерт через LLM.

        Эта функция:
        1. Проверяет включена ли классификация
        2. Выполняет асинхронную классификацию без блокировки webhook
        3. Кеширует результаты для performance
        4. Gracefully fallback при ошибках LLM
        """
        try:
            # Проверяем включена ли классификация
            classification_service = getattr(self, "_classification_service", None)
            if not classification_service:
                return

            # Получаем конфигурацию классификации
            from config import get_config

            config = get_config()

            # Проверяем настройки LLM
            if not (hasattr(config, "llm") and config.llm and config.llm.enabled):
                return

            # Асинхронная классификация (не блокирует webhook)
            import asyncio

            asyncio.create_task(self._classify_alert_background(alert))

        except Exception as e:
            # Логируем ошибку, но не роняем webhook
            print(f"Classification setup error: {e}")

    async def _classify_alert_background(self, alert: "Alert") -> None:
        """Фоновая классификация алерта."""
        try:
            # Получаем классификационный сервис
            classification_service = getattr(self, "_classification_service", None)
            if not classification_service:
                return

            # Выполняем классификацию
            classification_result = await classification_service.classify_alert(
                alert=alert, context={"source": "legacy_webhook", "async": True}
            )

            # Обновляем метрики
            if hasattr(self.metrics, "increment_classifications"):
                self.metrics.increment_classifications(
                    severity=classification_result.severity.value, cached=False
                )

            print(
                f"Alert classified: {alert.fingerprint} -> {classification_result.severity.value}"
            )

        except Exception as e:
            # Логируем ошибку классификации
            print(f"Background classification error for {alert.fingerprint}: {e}")

            # Обновляем метрики ошибок
            if hasattr(self.metrics, "increment_classification_errors"):
                self.metrics.increment_classification_errors()
