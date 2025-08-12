"""
Utility decorators implementing DRY principle.

Common decorators to avoid code duplication:
- Retry logic
- Performance monitoring
- Error handling
- Caching
- Validation
"""

# Standard library imports
import asyncio
import functools
import time
from typing import Any, Callable, Optional, TypeVar, Union

# Local imports
from ..core.interfaces import ICache, IMetricsCollector

T = TypeVar("T")


def retry(
    max_attempts: int = 3,
    delay: float = 1.0,
    backoff_factor: float = 2.0,
    exceptions: tuple = (Exception,),
) -> Callable:
    """
    Retry decorator with exponential backoff.

    Args:
        max_attempts: Maximum number of retry attempts
        delay: Initial delay between retries in seconds
        backoff_factor: Multiplier for delay after each attempt
        exceptions: Tuple of exceptions to catch and retry on
    """

    def decorator(func: Callable[..., T]) -> Callable[..., T]:
        @functools.wraps(func)
        async def async_wrapper(*args: Any, **kwargs: Any) -> T:
            last_exception = None
            current_delay = delay

            for attempt in range(max_attempts):
                try:
                    if asyncio.iscoroutinefunction(func):
                        return await func(*args, **kwargs)
                    else:
                        return func(*args, **kwargs)
                except exceptions as e:
                    last_exception = e
                    if attempt < max_attempts - 1:
                        await asyncio.sleep(current_delay)
                        current_delay *= backoff_factor
                    continue

            # All attempts failed
            raise last_exception

        @functools.wraps(func)
        def sync_wrapper(*args: Any, **kwargs: Any) -> T:
            last_exception = None
            current_delay = delay

            for attempt in range(max_attempts):
                try:
                    return func(*args, **kwargs)
                except exceptions as e:
                    last_exception = e
                    if attempt < max_attempts - 1:
                        time.sleep(current_delay)
                        current_delay *= backoff_factor
                    continue

            # All attempts failed
            raise last_exception

        if asyncio.iscoroutinefunction(func):
            return async_wrapper
        else:
            return sync_wrapper

    return decorator


def measure_time(
    metrics_collector: Optional[IMetricsCollector] = None,
    metric_name: Optional[str] = None,
    labels: Optional[dict[str, str]] = None,
) -> Callable:
    """
    Decorator to measure execution time.

    Args:
        metrics_collector: Metrics collector instance
        metric_name: Name of the metric (defaults to function name)
        labels: Additional labels for the metric
    """

    def decorator(func: Callable[..., T]) -> Callable[..., T]:
        @functools.wraps(func)
        async def async_wrapper(*args: Any, **kwargs: Any) -> T:
            start_time = time.time()
            try:
                result = await func(*args, **kwargs)
                return result
            finally:
                duration = time.time() - start_time
                if metrics_collector:
                    name = metric_name or f"{func.__name__}_duration_seconds"
                    metrics_collector.observe_histogram(name, duration, labels)

        @functools.wraps(func)
        def sync_wrapper(*args: Any, **kwargs: Any) -> T:
            start_time = time.time()
            try:
                result = func(*args, **kwargs)
                return result
            finally:
                duration = time.time() - start_time
                if metrics_collector:
                    name = metric_name or f"{func.__name__}_duration_seconds"
                    metrics_collector.observe_histogram(name, duration, labels)

        if asyncio.iscoroutinefunction(func):
            return async_wrapper
        else:
            return sync_wrapper

    return decorator


def cache_result(
    cache: ICache,
    ttl: Optional[int] = None,
    key_generator: Optional[Callable[..., str]] = None,
) -> Callable:
    """
    Decorator to cache function results.

    Args:
        cache: Cache instance
        ttl: Time to live in seconds
        key_generator: Function to generate cache key from arguments
    """

    def decorator(func: Callable[..., T]) -> Callable[..., T]:
        def generate_key(*args: Any, **kwargs: Any) -> str:
            if key_generator:
                return key_generator(*args, **kwargs)

            # Default key generation
            key_parts = [func.__name__]
            key_parts.extend(str(arg) for arg in args)
            key_parts.extend(f"{k}={v}" for k, v in sorted(kwargs.items()))
            return ":".join(key_parts)

        @functools.wraps(func)
        async def async_wrapper(*args: Any, **kwargs: Any) -> T:
            cache_key = generate_key(*args, **kwargs)

            # Try to get from cache
            cached_result = await cache.get(cache_key)
            if cached_result is not None:
                return cached_result

            # Execute function and cache result
            result = await func(*args, **kwargs)
            await cache.set(cache_key, result, ttl)
            return result

        @functools.wraps(func)
        def sync_wrapper(*args: Any, **kwargs: Any) -> T:
            # For sync functions, we can't use async cache
            # Fall back to direct execution
            return func(*args, **kwargs)

        if asyncio.iscoroutinefunction(func):
            return async_wrapper
        else:
            return sync_wrapper

    return decorator


def validate_input(
    validator: Callable[[Any], bool], error_message: str = "Invalid input"
) -> Callable:
    """
    Decorator to validate function input.

    Args:
        validator: Function that returns True if input is valid
        error_message: Error message for invalid input
    """

    def decorator(func: Callable[..., T]) -> Callable[..., T]:
        @functools.wraps(func)
        async def async_wrapper(*args: Any, **kwargs: Any) -> T:
            if not validator(*args, **kwargs):
                raise ValueError(error_message)
            return await func(*args, **kwargs)

        @functools.wraps(func)
        def sync_wrapper(*args: Any, **kwargs: Any) -> T:
            if not validator(*args, **kwargs):
                raise ValueError(error_message)
            return func(*args, **kwargs)

        if asyncio.iscoroutinefunction(func):
            return async_wrapper
        else:
            return sync_wrapper

    return decorator


def handle_exceptions(
    exceptions: tuple = (Exception,),
    default_return: Any = None,
    log_errors: bool = True,
) -> Callable:
    """
    Decorator to handle exceptions gracefully.

    Args:
        exceptions: Tuple of exceptions to catch
        default_return: Default value to return on exception
        log_errors: Whether to log caught exceptions
    """

    def decorator(func: Callable[..., T]) -> Callable[..., Union[T, Any]]:
        @functools.wraps(func)
        async def async_wrapper(*args: Any, **kwargs: Any) -> Union[T, Any]:
            try:
                return await func(*args, **kwargs)
            except exceptions as e:
                if log_errors:
                    # Simple logging - in real implementation, use proper logger
                    print(f"Error in {func.__name__}: {e}")
                return default_return

        @functools.wraps(func)
        def sync_wrapper(*args: Any, **kwargs: Any) -> Union[T, Any]:
            try:
                return func(*args, **kwargs)
            except exceptions as e:
                if log_errors:
                    # Simple logging - in real implementation, use proper logger
                    print(f"Error in {func.__name__}: {e}")
                return default_return

        if asyncio.iscoroutinefunction(func):
            return async_wrapper
        else:
            return sync_wrapper

    return decorator


def singleton(cls: type) -> type:
    """
    Singleton decorator for classes.

    Args:
        cls: Class to make singleton
    """
    instances = {}

    def get_instance(*args: Any, **kwargs: Any) -> Any:
        if cls not in instances:
            instances[cls] = cls(*args, **kwargs)
        return instances[cls]

    return get_instance


def rate_limit(
    max_calls: int,
    time_window: float,
    key_generator: Optional[Callable[..., str]] = None,
) -> Callable:
    """
    Rate limiting decorator.

    Args:
        max_calls: Maximum number of calls allowed
        time_window: Time window in seconds
        key_generator: Function to generate rate limit key
    """
    call_history: dict[str, list] = {}

    def decorator(func: Callable[..., T]) -> Callable[..., T]:
        def generate_key(*args: Any, **kwargs: Any) -> str:
            if key_generator:
                return key_generator(*args, **kwargs)
            return "default"

        @functools.wraps(func)
        async def async_wrapper(*args: Any, **kwargs: Any) -> T:
            key = generate_key(*args, **kwargs)
            current_time = time.time()

            # Initialize or clean up call history
            if key not in call_history:
                call_history[key] = []

            # Remove old calls outside time window
            call_history[key] = [
                call_time
                for call_time in call_history[key]
                if current_time - call_time < time_window
            ]

            # Check rate limit
            if len(call_history[key]) >= max_calls:
                raise RuntimeError(
                    f"Rate limit exceeded: {max_calls} calls per {time_window} seconds"
                )

            # Record this call
            call_history[key].append(current_time)

            return await func(*args, **kwargs)

        @functools.wraps(func)
        def sync_wrapper(*args: Any, **kwargs: Any) -> T:
            key = generate_key(*args, **kwargs)
            current_time = time.time()

            # Initialize or clean up call history
            if key not in call_history:
                call_history[key] = []

            # Remove old calls outside time window
            call_history[key] = [
                call_time
                for call_time in call_history[key]
                if current_time - call_time < time_window
            ]

            # Check rate limit
            if len(call_history[key]) >= max_calls:
                raise RuntimeError(
                    f"Rate limit exceeded: {max_calls} calls per {time_window} seconds"
                )

            # Record this call
            call_history[key].append(current_time)

            return func(*args, **kwargs)

        if asyncio.iscoroutinefunction(func):
            return async_wrapper
        else:
            return sync_wrapper

    return decorator


def deprecated(reason: str = "") -> Callable:
    """
    Decorator to mark functions as deprecated.

    Args:
        reason: Reason for deprecation
    """

    def decorator(func: Callable[..., T]) -> Callable[..., T]:
        @functools.wraps(func)
        def wrapper(*args: Any, **kwargs: Any) -> T:
            warning_msg = f"Function {func.__name__} is deprecated"
            if reason:
                warning_msg += f": {reason}"
            print(
                f"WARNING: {warning_msg}"
            )  # In real implementation, use proper logger
            return func(*args, **kwargs)

        return wrapper

    return decorator
