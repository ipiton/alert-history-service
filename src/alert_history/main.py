"""
Alert History Service - новая версия с сохранением полной функциональности.

Обеспечивает 100% совместимость с существующим API:
- POST /webhook — приём алертов от Alertmanager
- GET /history — получение истории (фильтры: alertname, status, fingerprint, время)
- GET /report — аналитика по истории алертов
- GET /metrics — Prometheus метрики
- GET /dashboard — HTML dashboard

Новые возможности:
- LLM классификация алертов
- Intelligent publishing в Rootly/PagerDuty/Slack
- Horizontal scaling (PostgreSQL + Redis)
- 12-Factor App compliance
"""

# Standard library imports
import asyncio
import signal
import sys
from contextlib import asynccontextmanager
from typing import AsyncGenerator

# Third-party imports
from fastapi import FastAPI, HTTPException

# Local imports
from .api.classification_endpoints import router as classification_router
from .api.legacy_adapter import LegacyAPIAdapter
from .api.metrics import LegacyMetrics
from .api.proxy_endpoints import proxy_router
from .api.webhook_endpoints import webhook_router
from .api.publishing_endpoints import publishing_router
from .config import get_config, validate_config
from .core.shutdown import lifespan_manager, health_checker, shutdown_handler
from .database.sqlite_adapter import SQLiteLegacyStorage
from .logging_config import get_logger, get_performance_logger, setup_logging
from .services.alert_classifier import AlertClassificationService
from .services.alert_formatter import AlertFormatter
from .services.alert_publisher import AlertPublisher
from .services.filter_engine import AlertFilterEngine
from .services.graceful_shutdown import GracefulShutdownHandler
from .services.llm_client import LLMProxyClient
from .services.target_discovery import DynamicTargetManager, TargetDiscoveryConfig
from .services.webhook_processor import WebhookProcessor

# Global configuration and logging
config = get_config()
logger = get_logger(__name__)
performance_logger = get_performance_logger()


# Using lifespan_manager from shutdown.py for proper lifecycle management


async def initialize_services() -> None:
    """Initialize all services maintaining existing functionality."""
    # Initialize storage (maintains existing SQLite compatibility)
    storage = SQLiteLegacyStorage(config.database.sqlite_path)
    # Expose storage via global app_state for APIs that depend on it
    try:
        from .core.app_state import app_state

        app_state.storage = storage
    except Exception:
        pass

    # Initialize Redis cache (T1.3: Redis Integration)
    redis_cache = None
    if config.redis.url:
        try:
            from .services.redis_cache import RedisCache

            redis_cache = RedisCache(
                redis_url=config.redis.url,
                default_ttl=3600,  # 1 hour
                max_connections=config.redis.pool_size,
                socket_timeout=float(config.redis.pool_timeout),
            )

            await redis_cache.initialize()
            app.state.redis_cache = redis_cache
            # Mirror to global app_state for convenience
            try:
                from .core.app_state import app_state as _as

                _as.redis_cache = redis_cache
            except Exception:
                pass

            logger.info(
                "Redis cache initialized",
                redis_url=(
                    config.redis.url.split("@")[-1] if "@" in config.redis.url else config.redis.url
                ),
                pool_size=config.redis.pool_size,
            )

        except Exception as e:
            logger.warning(
                "Failed to initialize Redis cache, running without caching",
                error=str(e),
            )
            redis_cache = None
    else:
        logger.info("Redis not configured, running without distributed caching")

    # Initialize Stateless Manager (T1.4: Stateless Application Design)
    from .core.stateless_manager import StatelessManager

    stateless_manager = StatelessManager(
        redis_cache=redis_cache,
        operation_ttl=3600,  # 1 hour for operation idempotency
    )

    # Update instance heartbeat
    await stateless_manager.update_instance_heartbeat()
    app.state.stateless_manager = stateless_manager

    logger.info(
        "Stateless manager initialized",
        instance_id=stateless_manager.instance_id,
        redis_available=redis_cache is not None,
    )

    # Initialize metrics
    metrics = LegacyMetrics()
    app.state.metrics = metrics
    try:
        from .core.app_state import app_state as _as

        _as.metrics = metrics
    except Exception:
        pass

    # Initialize LLM client (if enabled and configured)
    llm_client = None
    classification_service = None

    if config.alerts.enable_classification and config.llm.api_key:
        try:
            llm_client = LLMProxyClient(
                proxy_url=config.llm.proxy_url,
                api_key=config.llm.api_key,
                model=config.llm.model,
                timeout=config.llm.timeout,
                max_retries=config.llm.max_retries,
                retry_delay=config.llm.retry_delay,
            )

            # Initialize classification service
            classification_service = AlertClassificationService(
                llm_client=llm_client,
                storage=storage,
                metrics=metrics,
                cache=redis_cache,  # T1.3: Use Redis cache if available
                cache_ttl=config.llm.cache_ttl,
                enable_fallback=True,
            )

            await classification_service.initialize()

            # Store services in app state for dependency injection
            app.state.classification_service = classification_service
            try:
                from .core.app_state import app_state as _as

                _as.classification_service = classification_service
            except Exception:
                pass

            logger.info(
                "LLM Classification service initialized",
                llm_proxy_url=config.llm.proxy_url,
                llm_model=config.llm.model,
                cache_ttl=config.llm.cache_ttl,
            )

        except Exception as e:
            logger.error(
                "Failed to initialize LLM service, running without classification",
                error=str(e),
            )
            app.state.classification_service = None
    else:
        logger.info(
            "LLM Classification disabled",
            classification_enabled=config.alerts.enable_classification,
            api_key_configured=bool(config.llm.api_key),
        )
        app.state.classification_service = None

    # Initialize Intelligent Proxy components (Phase 3)
    logger.info("Initializing Intelligent Alert Proxy components (static publishers mode)")

    # Target Discovery disabled: use static publishers only
    class _StaticTargetManager:
        def is_metrics_only_mode(self) -> bool:
            return False

        def get_targets_count(self) -> int:
            return 0

        async def stop_monitoring(self) -> None:
            return None

    target_manager = _StaticTargetManager()
    app.state.target_manager = target_manager

    # Alert Formatter
    alert_formatter = AlertFormatter()

    # Filter Engine
    filter_engine = AlertFilterEngine()
    app.state.filter_engine = filter_engine

    # Alert Publisher
    alert_publisher = AlertPublisher(
        formatter=alert_formatter,
        filter_engine=filter_engine,
        metrics=metrics,
        max_concurrent_publishes=10,
        default_timeout=30.0,
        default_retries=3,
    )

    # Initialize publisher session
    await alert_publisher._init_session()
    app.state.alert_publisher = alert_publisher

    # Initialize webhook processor
    webhook_processor = WebhookProcessor(
        storage=storage,
        metrics=metrics,
        classification_service=classification_service,
        enable_auto_classification=config.alerts.enable_classification,
        classification_timeout=10.0,
    )
    app.state.webhook_processor = webhook_processor

    # Initialize legacy API adapter to maintain 100% compatibility
    app.legacy_adapter = LegacyAPIAdapter(
        app=app,
        storage=storage,
        db_path=config.database.sqlite_path,
        retention_days=config.alerts.retention_days,
        webhook_processor=webhook_processor,
    )

    # Log proxy initialization status
    metrics_only_mode = target_manager.is_metrics_only_mode()
    active_targets = target_manager.get_targets_count()

    logger.info(
        "Intelligent Alert Proxy initialized",
        metrics_only_mode=metrics_only_mode,
        active_targets=active_targets,
        target_discovery_enabled=discovery_config.enabled,
    )

    logger.info(
        "Services initialized successfully",
        database_path=config.database.sqlite_path,
        retention_days=config.alerts.retention_days,
        llm_enabled=classification_service is not None,
        proxy_enabled=True,
        metrics_only_mode=metrics_only_mode,
    )


async def shutdown_services() -> None:
    """Shutdown all services gracefully."""
    # Shutdown proxy components
    if hasattr(app.state, "target_manager"):
        try:
            await app.state.target_manager.stop_monitoring()
            logger.info("Target manager shut down successfully")
        except Exception as e:
            logger.error("Error shutting down target manager", error=str(e))

    if hasattr(app.state, "alert_publisher"):
        try:
            await app.state.alert_publisher._close_session()
            logger.info("Alert publisher shut down successfully")
        except Exception as e:
            logger.error("Error shutting down alert publisher", error=str(e))

    # Shutdown classification service
    if hasattr(app.state, "classification_service") and app.state.classification_service:
        try:
            await app.state.classification_service.shutdown()
            logger.info("Classification service shut down successfully")
        except Exception as e:
            logger.error("Error shutting down classification service", error=str(e))

    logger.info("All services shut down")


def setup_signal_handlers(shutdown_handler: GracefulShutdownHandler) -> None:
    """Setup signal handlers for graceful shutdown."""

    def signal_handler(signum: int, frame) -> None:
        logger.info(f"Received signal {signum}, initiating graceful shutdown")
        shutdown_handler.initiate_shutdown()

    signal.signal(signal.SIGTERM, signal_handler)
    signal.signal(signal.SIGINT, signal_handler)


def create_app() -> FastAPI:
    """Create FastAPI application with 12-Factor compliance and graceful shutdown."""

    # Load configuration (12-Factor: config via environment)
    config = get_config()
    if not validate_config(config):
        raise RuntimeError("Invalid configuration")

    # Setup structured logging (12-Factor: logs to stdout)
    setup_logging()

    logger.info(
        "Starting Alert History Service",
        service_name=config.service_name,
        version=config.service_version,
        environment=config.environment,
        llm_enabled=config.llm.enabled,
        proxy_enabled=config.proxy.enabled,
    )

    app = FastAPI(
        title="Alert History Service with LLM Intelligence",
        lifespan=lifespan_manager,
        description="""
Alert History Service for Alertmanager webhook processing with LLM classification.

## Legacy Endpoints (100% compatible)
- **POST /webhook** - Receive alert events from Alertmanager
- **GET /history** - Get alert history with filters
- **GET /report** - Get alert analytics and reports
- **GET /metrics** - Prometheus metrics
- **GET /dashboard** - HTML dashboard
- **GET /dashboard/grouped** - Grouped HTML dashboard

## New Endpoints (LLM + Publishing)
- **GET /classification/{fingerprint}** - Get alert classification
- **POST /classification/refresh/{fingerprint}** - Force refresh classification
- **POST /classification/bulk** - Bulk classification
- **GET /classification/stats** - Classification statistics
- **GET /classification/health** - Classification service health

## Intelligent Alert Proxy (Phase 3)
- **POST /proxy/webhook** - Intelligent webhook proxy with filtering and publishing
- **GET /proxy/targets** - List active publishing targets
- **GET /proxy/stats** - Proxy operation statistics
- **GET /proxy/health** - Proxy components health check
- **POST /proxy/targets/refresh** - Force refresh publishing targets
- **POST /proxy/filters/rules** - Add custom filter rule
- **DELETE /proxy/filters/rules/{rule_name}** - Remove filter rule
        """,
        version=config.service_version,
    )

    # Ensure minimal dependencies for dashboard APIs (storage in app_state)
    try:
        from .core.app_state import app_state as _as

        if not hasattr(_as, "storage") or _as.storage is None:
            _as.storage = SQLiteLegacyStorage(config.database.sqlite_path)
    except Exception:
        pass

    # Register classification endpoints (new functionality)
    app.include_router(classification_router)

    # Register webhook endpoints (including intelligent proxy)
    app.include_router(webhook_router)

    # Register proxy management endpoints (Phase 3 - Intelligent Alert Proxy)
    app.include_router(proxy_router)

    # Register publishing management endpoints (Phase 3 - Publishing Management)
    app.include_router(publishing_router)

    # Register dashboard API endpoints (T6: Dashboard и UI интеграция)
    from .api.dashboard_endpoints import dashboard_router

    app.include_router(dashboard_router)

    # Register enrichment mode endpoints
    from .api.enrichment_endpoints import router as enrichment_router

    app.include_router(enrichment_router)

    # Add modern dashboard endpoint (T6: Dashboard и UI интеграция)
    from fastapi.responses import HTMLResponse
    from fastapi.templating import Jinja2Templates
    from fastapi import Request

    templates = Jinja2Templates(directory="templates")

    @app.get("/dashboard/modern", response_class=HTMLResponse, tags=["Dashboard"])
    async def modern_dashboard(request: Request):
        """Modern HTML5 dashboard with CSS Grid, Flexbox and minimal JavaScript."""
        return templates.TemplateResponse("html5_dashboard.html", {"request": request})

    # Friendly routes to open specific sections in the same HTML5 dashboard
    @app.get("/dashboard/publishing", response_class=HTMLResponse, tags=["Dashboard"])
    async def dashboard_publishing(request: Request):
        return templates.TemplateResponse("html5_dashboard.html", {"request": request})

    @app.get("/dashboard/targets", response_class=HTMLResponse, tags=["Dashboard"])
    async def dashboard_targets(request: Request):
        return templates.TemplateResponse("html5_dashboard.html", {"request": request})

    @app.get("/dashboard/llm-metrics", response_class=HTMLResponse, tags=["Dashboard"])
    async def dashboard_llm_metrics(request: Request):
        return templates.TemplateResponse("html5_dashboard.html", {"request": request})

    # Add health check endpoints (12-Factor: health checks)
    @app.get("/healthz", tags=["Health"])
    async def health_check():
        """Kubernetes liveness probe."""
        status = health_checker.get_status()
        if status["healthy"]:
            return {"status": "healthy", "timestamp": status["timestamp"]}
        else:
            raise HTTPException(status_code=503, detail="Service unhealthy")

    @app.get("/readyz", tags=["Health"])
    async def readiness_check():
        """Kubernetes readiness probe."""
        status = health_checker.get_status()
        if status["ready"]:
            return {
                "status": "ready",
                "uptime": status["uptime_seconds"],
                "dependencies": status["dependencies"],
                "timestamp": status["timestamp"],
            }
        else:
            raise HTTPException(status_code=503, detail="Service not ready")

    return app


# Create application instance
app = create_app()


def main() -> None:
    """Main entry point for the application."""
    try:
        # Setup logging first
        setup_logging()

        logger.info(
            "Initializing Alert History Service",
            version=config.version,
            environment=config.environment,
        )

        # Import uvicorn here to avoid circular imports
        # Third-party imports
        import uvicorn

        # Run server with configuration
        uvicorn.run(
            "src.alert_history.main:app",
            host=config.server.host,
            port=config.server.port,
            workers=1,  # Start with single worker for compatibility
            log_level=config.server.log_level.lower(),
            reload=config.server.reload and config.is_development(),
            access_log=True,
            server_header=False,
            date_header=False,
        )

    except KeyboardInterrupt:
        logger.info("Received keyboard interrupt, shutting down")
        sys.exit(0)
    except Exception as e:
        logger.critical("Failed to start server", error=str(e))
        sys.exit(1)


if __name__ == "__main__":
    main()
