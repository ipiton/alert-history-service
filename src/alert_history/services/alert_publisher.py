"""
Alert Publisher для отправки алертов в различные системы.

Поддерживает:
- Retry логику с exponential backoff
- Circuit breaker pattern
- Async publishing с таймаутами
- Metrics и monitoring всех операций
- Graceful degradation при сбоях
"""

# Standard library imports
import asyncio
import time
from dataclasses import dataclass, field
from enum import Enum
from typing import Optional

# Third-party imports
import aiohttp

# Local imports
from ..core.interfaces import (
    EnrichedAlert,
    IAlertFormatter,
    IAlertPublisher,
    IFilterEngine,
    IMetricsCollector,
    PublishingTarget,
)
from ..logging_config import get_logger, get_performance_logger
from ..services.alert_formatter import AlertFormatter
from ..utils.decorators import measure_time, retry

logger = get_logger(__name__)
performance_logger = get_performance_logger()


class CircuitBreakerState(Enum):
    """Состояния Circuit Breaker."""

    CLOSED = "closed"  # Нормальная работа
    OPEN = "open"  # Блокировка запросов
    HALF_OPEN = "half_open"  # Тестирование восстановления


@dataclass
class CircuitBreaker:
    """Circuit breaker для защиты от cascading failures."""

    failure_threshold: int = 5
    timeout: float = 60.0  # Seconds
    half_open_max_calls: int = 3

    # State
    state: CircuitBreakerState = field(default=CircuitBreakerState.CLOSED)
    failure_count: int = field(default=0)
    last_failure_time: float = field(default=0.0)
    half_open_calls: int = field(default=0)

    def can_execute(self) -> bool:
        """Проверить можно ли выполнить запрос."""
        if self.state == CircuitBreakerState.CLOSED:
            return True

        if self.state == CircuitBreakerState.OPEN:
            # Check if timeout expired
            if time.time() - self.last_failure_time >= self.timeout:
                self.state = CircuitBreakerState.HALF_OPEN
                self.half_open_calls = 0
                return True
            return False

        if self.state == CircuitBreakerState.HALF_OPEN:
            return self.half_open_calls < self.half_open_max_calls

        return False

    def record_success(self) -> None:
        """Записать успешный запрос."""
        if self.state == CircuitBreakerState.HALF_OPEN:
            self.half_open_calls += 1
            if self.half_open_calls >= self.half_open_max_calls:
                self.state = CircuitBreakerState.CLOSED
                self.failure_count = 0
        elif self.state == CircuitBreakerState.CLOSED:
            self.failure_count = 0

    def record_failure(self) -> None:
        """Записать неудачный запрос."""
        self.failure_count += 1
        self.last_failure_time = time.time()

        if self.state == CircuitBreakerState.HALF_OPEN:
            self.state = CircuitBreakerState.OPEN
        elif (
            self.state == CircuitBreakerState.CLOSED
            and self.failure_count >= self.failure_threshold
        ):
            self.state = CircuitBreakerState.OPEN


@dataclass
class PublishingStats:
    """Статистика публикации для конкретного target."""

    target_name: str
    total_attempts: int = 0
    successful_publishes: int = 0
    failed_publishes: int = 0
    circuit_breaker_blocks: int = 0
    last_success_time: Optional[float] = None
    last_failure_time: Optional[float] = None
    average_response_time: float = 0.0

    @property
    def success_rate(self) -> float:
        """Процент успешных публикаций."""
        if self.total_attempts == 0:
            return 0.0
        return self.successful_publishes / self.total_attempts

    @property
    def is_healthy(self) -> bool:
        """Проверка здоровья target."""
        # Считается здоровым если success rate > 50% или недавно были успешные отправки
        recent_success = (
            self.last_success_time
            and time.time() - self.last_success_time < 300  # 5 minutes
        )
        return self.success_rate > 0.5 or recent_success


class AlertPublisher(IAlertPublisher):
    """
    Сервис публикации алертов с продвинутыми возможностями.

    Features:
    - Multiple target support
    - Circuit breaker per target
    - Retry logic с exponential backoff
    - Comprehensive metrics
    - Graceful degradation
    """

    def __init__(
        self,
        formatter: Optional[IAlertFormatter] = None,
        filter_engine: Optional[IFilterEngine] = None,
        metrics: Optional[IMetricsCollector] = None,
        max_concurrent_publishes: int = 10,
        default_timeout: float = 30.0,
        default_retries: int = 3,
    ):
        """Initialize alert publisher."""
        self.formatter = formatter or AlertFormatter()
        self.filter_engine = filter_engine
        self.metrics = metrics
        self.max_concurrent_publishes = max_concurrent_publishes
        self.default_timeout = default_timeout
        self.default_retries = default_retries

        # HTTP session for connection pooling
        self._session: Optional[aiohttp.ClientSession] = None

        # Circuit breakers per target
        self._circuit_breakers: dict[str, CircuitBreaker] = {}

        # Publishing statistics per target
        self._publishing_stats: dict[str, PublishingStats] = {}

        # Concurrency control
        self._publish_semaphore = asyncio.Semaphore(max_concurrent_publishes)

        # Active publishing tasks tracking
        self._active_publishes: set[asyncio.Task] = set()

    async def __aenter__(self) -> "AlertPublisher":
        """Async context manager entry."""
        await self._init_session()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb) -> None:
        """Async context manager exit."""
        await self._close_session()

    async def _init_session(self) -> None:
        """Initialize HTTP session."""
        connector = aiohttp.TCPConnector(
            limit=100,
            limit_per_host=20,
            keepalive_timeout=30,
            enable_cleanup_closed=True,
        )

        timeout = aiohttp.ClientTimeout(total=self.default_timeout)

        self._session = aiohttp.ClientSession(
            connector=connector,
            timeout=timeout,
            headers={"User-Agent": "AlertHistory-Publisher/1.0"},
        )

    async def _close_session(self) -> None:
        """Close HTTP session and cancel active tasks."""
        # Cancel all active publishing tasks
        if self._active_publishes:
            logger.info(
                f"Cancelling {len(self._active_publishes)} active publishing tasks"
            )
            for task in self._active_publishes:
                task.cancel()

            # Wait for cancellation with timeout
            try:
                await asyncio.wait_for(
                    asyncio.gather(*self._active_publishes, return_exceptions=True),
                    timeout=10.0,
                )
            except asyncio.TimeoutError:
                logger.warning("Some publishing tasks did not cancel in time")

        # Close HTTP session
        if self._session:
            await self._session.close()
            self._session = None

    async def publish_alert(
        self, enriched_alert: EnrichedAlert, target: PublishingTarget
    ) -> bool:
        """
        Опубликовать алерт в конкретный target.

        Args:
            enriched_alert: Обогащенный алерт для публикации
            target: Целевая система

        Returns:
            True если публикация успешна
        """
        if not target.enabled:
            logger.debug(f"Target {target.name} is disabled, skipping")
            return False

        # Check filter if available
        if self.filter_engine:
            should_publish = await self.filter_engine.should_publish(
                enriched_alert, target
            )
            if not should_publish:
                logger.debug(
                    f"Alert filtered out for target {target.name}",
                    alert_name=enriched_alert.alert.alert_name,
                    fingerprint=enriched_alert.alert.fingerprint,
                )
                return False

        # Check circuit breaker
        circuit_breaker = self._get_circuit_breaker(target.name)
        if not circuit_breaker.can_execute():
            logger.warning(
                f"Circuit breaker OPEN for target {target.name}, skipping publish",
                target=target.name,
                state=circuit_breaker.state.value,
            )
            self._record_circuit_breaker_block(target.name)
            return False

        # Publish with concurrency control
        async with self._publish_semaphore:
            return await self._publish_with_retry(
                enriched_alert, target, circuit_breaker
            )

    async def publish_to_multiple_targets(
        self, enriched_alert: EnrichedAlert, targets: list[PublishingTarget]
    ) -> dict[str, bool]:
        """
        Опубликовать алерт в несколько targets параллельно.

        Args:
            enriched_alert: Обогащенный алерт
            targets: Список целевых систем

        Returns:
            Словарь target_name -> success_status
        """
        if not targets:
            return {}

        # Create tasks for parallel publishing
        tasks = []
        target_names = []

        for target in targets:
            if target.enabled:
                task = asyncio.create_task(self.publish_alert(enriched_alert, target))
                tasks.append(task)
                target_names.append(target.name)
                self._active_publishes.add(task)

        if not tasks:
            logger.debug("No enabled targets for publishing")
            return {}

        try:
            # Wait for all publishing tasks
            results = await asyncio.gather(*tasks, return_exceptions=True)

            # Process results
            publishing_results = {}
            for target_name, result in zip(target_names, results):
                if isinstance(result, Exception):
                    logger.error(
                        f"Publishing to {target_name} failed with exception",
                        target=target_name,
                        error=str(result),
                    )
                    publishing_results[target_name] = False
                else:
                    publishing_results[target_name] = result

            # Log summary
            successful_targets = [
                name for name, success in publishing_results.items() if success
            ]
            failed_targets = [
                name for name, success in publishing_results.items() if not success
            ]

            logger.info(
                "Multi-target publishing completed",
                alert_name=enriched_alert.alert.alert_name,
                fingerprint=enriched_alert.alert.fingerprint,
                successful_targets=len(successful_targets),
                failed_targets=len(failed_targets),
                targets_success=successful_targets,
                targets_failed=failed_targets,
            )

            return publishing_results

        finally:
            # Clean up task references
            for task in tasks:
                self._active_publishes.discard(task)

    @measure_time()
    async def _publish_with_retry(
        self,
        enriched_alert: EnrichedAlert,
        target: PublishingTarget,
        circuit_breaker: CircuitBreaker,
    ) -> bool:
        """Публикация с retry логикой."""
        start_time = time.time()

        try:
            # Format alert for target
            formatted_alert = await self.formatter.format_alert(
                enriched_alert, target.format
            )

            # Publish with retries
            success = await self._perform_http_publish(
                formatted_alert, target, retries=self.default_retries
            )

            # Update circuit breaker and stats
            if success:
                circuit_breaker.record_success()
                self._record_publish_success(target.name, time.time() - start_time)
            else:
                circuit_breaker.record_failure()
                self._record_publish_failure(target.name, "HTTP request failed")

            return success

        except Exception as e:
            circuit_breaker.record_failure()
            self._record_publish_failure(target.name, str(e))

            logger.error(
                "Publishing failed with exception",
                target=target.name,
                alert_name=enriched_alert.alert.alert_name,
                error=str(e),
            )

            return False

    @retry(max_attempts=3, delay=1.0, backoff_factor=2.0)
    async def _perform_http_publish(
        self, formatted_alert: dict[str, any], target: PublishingTarget, retries: int
    ) -> bool:
        """Выполнить HTTP запрос для публикации."""
        if not self._session:
            await self._init_session()

        try:
            async with self._session.post(
                target.url,
                json=formatted_alert,
                headers=target.headers,
                timeout=aiohttp.ClientTimeout(total=self.default_timeout),
            ) as response:

                if response.status < 400:
                    logger.debug(
                        "Alert published successfully",
                        target=target.name,
                        status_code=response.status,
                        url=target.url,
                    )
                    return True
                else:
                    error_text = await response.text()
                    logger.warning(
                        "Alert publishing failed with HTTP error",
                        target=target.name,
                        status_code=response.status,
                        error=error_text[:200],
                        url=target.url,
                    )
                    return False

        except asyncio.TimeoutError:
            logger.warning(f"Publishing to {target.name} timed out")
            return False
        except aiohttp.ClientError as e:
            logger.warning(f"HTTP client error publishing to {target.name}: {e}")
            return False

    def _get_circuit_breaker(self, target_name: str) -> CircuitBreaker:
        """Получить circuit breaker для target."""
        if target_name not in self._circuit_breakers:
            self._circuit_breakers[target_name] = CircuitBreaker()
        return self._circuit_breakers[target_name]

    def _get_publishing_stats(self, target_name: str) -> PublishingStats:
        """Получить статистику для target."""
        if target_name not in self._publishing_stats:
            self._publishing_stats[target_name] = PublishingStats(
                target_name=target_name
            )
        return self._publishing_stats[target_name]

    def _record_publish_success(self, target_name: str, response_time: float) -> None:
        """Записать успешную публикацию."""
        stats = self._get_publishing_stats(target_name)
        stats.total_attempts += 1
        stats.successful_publishes += 1
        stats.last_success_time = time.time()

        # Update average response time (simple moving average)
        if stats.average_response_time == 0:
            stats.average_response_time = response_time
        else:
            stats.average_response_time = (
                stats.average_response_time + response_time
            ) / 2

        # Record metrics
        if self.metrics:
            self.metrics.increment_counter(
                "alert_publishing_total", {"target": target_name, "status": "success"}
            )
            self.metrics.observe_histogram(
                "alert_publishing_duration_seconds",
                response_time,
                {"target": target_name},
            )

    def _record_publish_failure(self, target_name: str, error: str) -> None:
        """Записать неудачную публикацию."""
        stats = self._get_publishing_stats(target_name)
        stats.total_attempts += 1
        stats.failed_publishes += 1
        stats.last_failure_time = time.time()

        # Record metrics
        if self.metrics:
            self.metrics.increment_counter(
                "alert_publishing_total",
                {"target": target_name, "status": "failure", "error": error[:50]},
            )

    def _record_circuit_breaker_block(self, target_name: str) -> None:
        """Записать блокировку circuit breaker."""
        stats = self._get_publishing_stats(target_name)
        stats.circuit_breaker_blocks += 1

        if self.metrics:
            self.metrics.increment_counter(
                "alert_publishing_circuit_breaker_blocks_total", {"target": target_name}
            )

    def get_target_stats(self, target_name: str) -> Optional[PublishingStats]:
        """Получить статистику для конкретного target."""
        return self._publishing_stats.get(target_name)

    def get_all_stats(self) -> dict[str, PublishingStats]:
        """Получить статистику для всех targets."""
        return self._publishing_stats.copy()

    def get_publishing_stats(self) -> dict[str, any]:
        """Получить агрегированную статистику публикации."""
        all_stats = self.get_all_stats()

        total_attempts = 0
        total_successful = 0
        total_failed = 0
        total_blocks = 0

        for stats in all_stats.values():
            total_attempts += stats.total_attempts
            total_successful += stats.successful_publishes
            total_failed += stats.failed_publishes
            total_blocks += stats.circuit_breaker_blocks

        return {
            "total_attempts": total_attempts,
            "total_successful": total_successful,
            "total_failed": total_failed,
            "total_blocks": total_blocks,
            "success_rate": (
                total_successful / total_attempts if total_attempts > 0 else 0.0
            ),
            "targets_count": len(all_stats),
        }

    def get_circuit_breaker_status(self, target_name: str) -> dict[str, any]:
        """Получить статус circuit breaker для target."""
        circuit_breaker = self._circuit_breakers.get(target_name)
        if not circuit_breaker:
            return {"state": "unknown", "target": target_name}

        return {
            "target": target_name,
            "state": circuit_breaker.state.value,
            "failure_count": circuit_breaker.failure_count,
            "last_failure_time": circuit_breaker.last_failure_time,
            "can_execute": circuit_breaker.can_execute(),
        }

    def reset_circuit_breaker(self, target_name: str) -> bool:
        """Сбросить circuit breaker для target."""
        if target_name in self._circuit_breakers:
            self._circuit_breakers[target_name] = CircuitBreaker()
            logger.info(f"Circuit breaker reset for target {target_name}")
            return True
        return False
