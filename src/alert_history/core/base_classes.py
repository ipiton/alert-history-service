"""
Base classes implementing common functionality.

Applies DRY principle by providing reusable implementations:
- Template Method Pattern for common algorithms
- Strategy Pattern for pluggable behavior
- Observer Pattern for event handling
"""

# Standard library imports
import asyncio
import time
from abc import ABC
from collections.abc import AsyncGenerator
from contextlib import asynccontextmanager
from typing import Any, Dict, Generic, List, Optional, TypeVar

# Local imports
from .interfaces import (
    Alert,
    ClassificationResult,
    EnrichedAlert,
    IAlertStorage,
    ICache,
    IHealthChecker,
    IMetricsCollector,
    IRepository,
)

T = TypeVar("T")


class BaseService(ABC):
    """Base service class with common functionality."""

    def __init__(
        self,
        metrics: Optional[IMetricsCollector] = None,
        cache: Optional[ICache] = None,
    ):
        """Initialize base service."""
        self.metrics = metrics
        self.cache = cache
        self._initialized = False
        self._shutdown = False

    async def initialize(self) -> None:
        """Initialize service (Template Method)."""
        if self._initialized:
            return

        await self._before_initialize()
        await self._do_initialize()
        await self._after_initialize()

        self._initialized = True
        self._record_metric("service_initialized", 1)

    async def shutdown(self) -> None:
        """Shutdown service gracefully (Template Method)."""
        if self._shutdown:
            return

        self._shutdown = True

        await self._before_shutdown()
        await self._do_shutdown()
        await self._after_shutdown()

        self._record_metric("service_shutdown", 1)

    async def _before_initialize(self) -> None:
        """Hook called before initialization."""
        pass

    async def _do_initialize(self) -> None:
        """Override in subclasses for actual initialization."""
        pass

    async def _after_initialize(self) -> None:
        """Hook called after initialization."""
        pass

    async def _before_shutdown(self) -> None:
        """Hook called before shutdown."""
        pass

    async def _do_shutdown(self) -> None:
        """Override in subclasses for actual shutdown."""
        pass

    async def _after_shutdown(self) -> None:
        """Hook called after shutdown."""
        pass

    def _record_metric(
        self, name: str, value: float, labels: Optional[Dict[str, str]] = None
    ) -> None:
        """Record metric if metrics collector is available."""
        if self.metrics:
            self.metrics.set_gauge(name, value, labels)

    @asynccontextmanager
    async def _timed_operation(self, operation_name: str) -> AsyncGenerator[None, None]:
        """Context manager for timing operations."""
        start_time = time.time()
        try:
            yield
        finally:
            duration = time.time() - start_time
            if self.metrics:
                self.metrics.observe_histogram(
                    f"{operation_name}_duration_seconds", duration
                )


class BaseRepository(IRepository, Generic[T]):
    """Base repository with common CRUD operations."""

    def __init__(
        self,
        storage: Any,  # Database connection
        metrics: Optional[IMetricsCollector] = None,
        cache: Optional[ICache] = None,
    ):
        """Initialize base repository."""
        self.storage = storage
        self.metrics = metrics
        self.cache = cache

    async def _get_from_cache(self, cache_key: str) -> Optional[T]:
        """Get entity from cache."""
        if not self.cache:
            return None
        return await self.cache.get(cache_key)

    async def _set_cache(
        self, cache_key: str, entity: T, ttl: Optional[int] = None
    ) -> None:
        """Set entity in cache."""
        if self.cache:
            await self.cache.set(cache_key, entity, ttl)

    async def _invalidate_cache(self, cache_key: str) -> None:
        """Invalidate cache entry."""
        if self.cache:
            await self.cache.delete(cache_key)

    def _generate_cache_key(self, entity_id: str) -> str:
        """Generate cache key for entity."""
        return f"{self.__class__.__name__}:{entity_id}"

    def _record_operation(self, operation: str, success: bool = True) -> None:
        """Record repository operation metric."""
        if self.metrics:
            labels = {
                "repository": self.__class__.__name__,
                "operation": operation,
                "success": str(success).lower(),
            }
            self.metrics.increment_counter("repository_operations_total", labels)


class BaseHealthChecker(IHealthChecker):
    """Base health checker with common checks."""

    def __init__(self, service_name: str, version: str):
        """Initialize base health checker."""
        self.service_name = service_name
        self.version = version
        self.startup_time = time.time()
        self._dependencies: List[IHealthChecker] = []

    def add_dependency(self, dependency: IHealthChecker) -> None:
        """Add dependency health checker."""
        self._dependencies.append(dependency)

    async def check_health(self) -> Dict[str, Any]:
        """Perform comprehensive health check."""
        health_status = {
            "service": self.service_name,
            "version": self.version,
            "status": "healthy",
            "timestamp": time.time(),
            "uptime": time.time() - self.startup_time,
            "checks": {},
        }

        # Check self health
        try:
            self_check = await self._check_self_health()
            health_status["checks"]["self"] = self_check
        except Exception as e:
            health_status["checks"]["self"] = {"status": "unhealthy", "error": str(e)}
            health_status["status"] = "unhealthy"

        # Check dependencies
        for i, dependency in enumerate(self._dependencies):
            try:
                dep_check = await dependency.check_health()
                health_status["checks"][f"dependency_{i}"] = dep_check
                if dep_check.get("status") != "healthy":
                    health_status["status"] = "degraded"
            except Exception as e:
                health_status["checks"][f"dependency_{i}"] = {
                    "status": "unhealthy",
                    "error": str(e),
                }
                health_status["status"] = "unhealthy"

        return health_status

    async def check_readiness(self) -> Dict[str, Any]:
        """Perform readiness check."""
        readiness_status = {
            "service": self.service_name,
            "ready": True,
            "timestamp": time.time(),
            "checks": {},
        }

        # Check if service is ready to serve traffic
        try:
            ready_check = await self._check_readiness()
            readiness_status["checks"]["ready"] = ready_check
            if not ready_check.get("ready", False):
                readiness_status["ready"] = False
        except Exception as e:
            readiness_status["checks"]["ready"] = {"ready": False, "error": str(e)}
            readiness_status["ready"] = False

        return readiness_status

    async def _check_self_health(self) -> Dict[str, Any]:
        """Override in subclasses for specific health checks."""
        return {"status": "healthy"}

    async def _check_readiness(self) -> Dict[str, Any]:
        """Override in subclasses for specific readiness checks."""
        return {"ready": True}


class BaseClassificationService(BaseService):
    """Base classification service with common patterns."""

    def __init__(
        self,
        llm_client: Any,
        storage: IAlertStorage,
        metrics: Optional[IMetricsCollector] = None,
        cache: Optional[ICache] = None,
    ):
        """Initialize base classification service."""
        super().__init__(metrics, cache)
        self.llm_client = llm_client
        self.storage = storage
        self._classification_cache_ttl = 3600  # 1 hour

    async def classify_with_cache(self, alert: Alert) -> ClassificationResult:
        """Classify alert with caching (Template Method)."""
        cache_key = self._generate_classification_cache_key(alert)

        # Try cache first
        if self.cache:
            cached_result = await self.cache.get(cache_key)
            if cached_result:
                self._record_metric("classification_cache_hits", 1)
                return cached_result

        # Perform classification
        async with self._timed_operation("alert_classification"):
            result = await self._perform_classification(alert)

        # Cache result
        if self.cache and result:
            await self.cache.set(cache_key, result, self._classification_cache_ttl)
            self._record_metric("classification_cache_misses", 1)

        return result

    async def _perform_classification(self, alert: Alert) -> ClassificationResult:
        """Override in subclasses for actual classification logic."""
        raise NotImplementedError

    def _generate_classification_cache_key(self, alert: Alert) -> str:
        """Generate cache key for alert classification."""
        return f"classification:{alert.fingerprint}"


class BasePublishingService(BaseService):
    """Base publishing service with retry logic."""

    def __init__(
        self,
        metrics: Optional[IMetricsCollector] = None,
        max_retries: int = 3,
        retry_delay: float = 1.0,
    ):
        """Initialize base publishing service."""
        super().__init__(metrics)
        self.max_retries = max_retries
        self.retry_delay = retry_delay

    async def publish_with_retry(
        self, enriched_alert: EnrichedAlert, target_config: Dict[str, Any]
    ) -> bool:
        """Publish alert with retry logic (Template Method)."""
        last_exception = None

        for attempt in range(self.max_retries + 1):
            try:
                async with self._timed_operation("alert_publishing"):
                    success = await self._perform_publish(enriched_alert, target_config)

                if success:
                    self._record_publishing_success(
                        target_config.get("name", "unknown")
                    )
                    return True

            except Exception as e:
                last_exception = e
                if attempt < self.max_retries:
                    await asyncio.sleep(
                        self.retry_delay * (2**attempt)
                    )  # Exponential backoff
                continue

        # All retries failed
        self._record_publishing_failure(
            target_config.get("name", "unknown"),
            str(last_exception) if last_exception else "Unknown error",
        )
        return False

    async def _perform_publish(
        self, enriched_alert: EnrichedAlert, target_config: Dict[str, Any]
    ) -> bool:
        """Override in subclasses for actual publishing logic."""
        raise NotImplementedError

    def _record_publishing_success(self, target_name: str) -> None:
        """Record successful publishing metric."""
        if self.metrics:
            labels = {"target": target_name, "status": "success"}
            self.metrics.increment_counter("alert_publishing_total", labels)

    def _record_publishing_failure(self, target_name: str, error: str) -> None:
        """Record failed publishing metric."""
        if self.metrics:
            labels = {"target": target_name, "status": "failure", "error": error[:100]}
            self.metrics.increment_counter("alert_publishing_total", labels)


class BaseEventProcessor(BaseService):
    """Base event processor with common event handling patterns."""

    def __init__(
        self,
        metrics: Optional[IMetricsCollector] = None,
        batch_size: int = 100,
        processing_timeout: int = 60,
    ):
        """Initialize base event processor."""
        super().__init__(metrics)
        self.batch_size = batch_size
        self.processing_timeout = processing_timeout
        self._event_queue: asyncio.Queue = asyncio.Queue()
        self._processor_task: Optional[asyncio.Task] = None

    async def _do_initialize(self) -> None:
        """Start event processing task."""
        self._processor_task = asyncio.create_task(self._process_events_loop())

    async def _do_shutdown(self) -> None:
        """Stop event processing task."""
        if self._processor_task:
            self._processor_task.cancel()
            try:
                await self._processor_task
            except asyncio.CancelledError:
                pass

    async def submit_event(self, event_data: Dict[str, Any]) -> None:
        """Submit event for processing."""
        await self._event_queue.put(event_data)
        self._record_metric("events_submitted", 1)

    async def _process_events_loop(self) -> None:
        """Main event processing loop."""
        while not self._shutdown:
            try:
                # Collect batch of events
                events = []
                deadline = time.time() + self.processing_timeout

                while len(events) < self.batch_size and time.time() < deadline:
                    try:
                        event = await asyncio.wait_for(
                            self._event_queue.get(),
                            timeout=min(1.0, deadline - time.time()),
                        )
                        events.append(event)
                    except asyncio.TimeoutError:
                        break

                if events:
                    await self._process_event_batch(events)

            except Exception:
                # Log error but continue processing
                self._record_metric("event_processing_errors", 1)
                await asyncio.sleep(1)  # Brief pause before retrying

    async def _process_event_batch(self, events: List[Dict[str, Any]]) -> None:
        """Process batch of events."""
        async with self._timed_operation("event_batch_processing"):
            for event in events:
                try:
                    await self._process_single_event(event)
                    self._record_metric("events_processed_success", 1)
                except Exception:
                    self._record_metric("events_processed_failure", 1)

    async def _process_single_event(self, event_data: Dict[str, Any]) -> None:
        """Override in subclasses for actual event processing."""
        raise NotImplementedError
