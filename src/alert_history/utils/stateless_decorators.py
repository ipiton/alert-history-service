"""
Decorators для stateless application design.

Provides:
- @idempotent - ensure operation idempotency
- @stateless - validate stateless operation
- @instance_tracked - track operations across instances
"""

import functools
import hashlib
import inspect
from typing import Callable, Optional

from ..logging_config import get_logger

logger = get_logger(__name__)


def idempotent(
    key_func: Optional[Callable] = None,
    ttl: int = 3600,
    use_args: bool = True,
    use_kwargs: bool = True,
):
    """
    Decorator to ensure operation idempotency.

    Args:
        key_func: Function to generate operation key
        ttl: Time-to-live for idempotency check (seconds)
        use_args: Include function args in key generation
        use_kwargs: Include function kwargs in key generation
    """

    def decorator(func: Callable) -> Callable:
        @functools.wraps(func)
        async def wrapper(*args, **kwargs):
            # Extract FastAPI request or app state if available
            stateless_manager = None

            # Try to get stateless_manager from various sources
            if (
                args
                and hasattr(args[0], "state")
                and hasattr(args[0].state, "stateless_manager")
            ):
                # FastAPI app instance
                stateless_manager = args[0].state.stateless_manager
            elif "request" in kwargs and hasattr(
                kwargs["request"].app.state, "stateless_manager"
            ):
                # FastAPI request
                stateless_manager = kwargs["request"].app.state.stateless_manager
            elif hasattr(func, "__self__") and hasattr(
                func.__self__, "stateless_manager"
            ):
                # Class method with stateless_manager attribute
                stateless_manager = func.__self__.stateless_manager

            if not stateless_manager:
                logger.debug(
                    f"No stateless manager available for {func.__name__}, proceeding without idempotency check"
                )
                return await func(*args, **kwargs)

            # Generate operation key
            if key_func:
                operation_key = key_func(*args, **kwargs)
            else:
                operation_key = _generate_operation_key(
                    func, args if use_args else (), kwargs if use_kwargs else {}
                )

            # Check if operation can proceed (idempotency check)
            can_proceed = await stateless_manager.ensure_idempotent_operation(
                operation_key, ttl
            )

            if not can_proceed:
                logger.info(
                    f"Idempotent operation skipped: {func.__name__}",
                    operation_key=operation_key,
                    instance_id=stateless_manager.instance_id,
                )
                # Return a default response or cached result
                return {"status": "already_processed", "operation_key": operation_key}

            # Execute operation
            logger.debug(
                f"Executing idempotent operation: {func.__name__}",
                operation_key=operation_key,
                instance_id=stateless_manager.instance_id,
            )

            return await func(*args, **kwargs)

        return wrapper

    return decorator


def stateless(
    validate_parameters: bool = True,
    require_redis: bool = False,
):
    """
    Decorator to validate stateless operation.

    Args:
        validate_parameters: Check that all required data is in parameters
        require_redis: Require Redis for stateless coordination
    """

    def decorator(func: Callable) -> Callable:
        @functools.wraps(func)
        async def wrapper(*args, **kwargs):
            # Get function signature for parameter validation
            sig = inspect.signature(func)

            if validate_parameters:
                # Check that all required parameters are provided
                bound = sig.bind(*args, **kwargs)
                bound.apply_defaults()

                missing_params = []
                for param_name, param in sig.parameters.items():
                    if (
                        param.default == inspect.Parameter.empty
                        and param_name not in bound.arguments
                    ):
                        missing_params.append(param_name)

                if missing_params:
                    logger.warning(
                        f"Stateless operation {func.__name__} missing required parameters",
                        missing_params=missing_params,
                    )

            # Check Redis requirement
            if require_redis:
                stateless_manager = _get_stateless_manager(*args, **kwargs)
                if not stateless_manager or not stateless_manager.redis_cache:
                    raise RuntimeError(
                        f"Redis required for stateless operation {func.__name__}"
                    )

            # Validate stateless operation
            stateless_manager = _get_stateless_manager(*args, **kwargs)
            if stateless_manager:
                validation = stateless_manager.validate_stateless_operation(
                    func.__name__,
                    args=args,
                    kwargs=kwargs,
                )

                if not validation["stateless"]:
                    logger.warning(
                        f"Operation {func.__name__} may not be stateless",
                        issues=validation["issues"],
                        recommendations=validation["recommendations"],
                    )

            return await func(*args, **kwargs)

        return wrapper

    return decorator


def instance_tracked(
    heartbeat_interval: int = 30,
    track_operation: bool = True,
):
    """
    Decorator to track operations across instances.

    Args:
        heartbeat_interval: Interval to update instance heartbeat (seconds)
        track_operation: Whether to track this specific operation
    """

    def decorator(func: Callable) -> Callable:
        @functools.wraps(func)
        async def wrapper(*args, **kwargs):
            stateless_manager = _get_stateless_manager(*args, **kwargs)

            if stateless_manager:
                # Update instance heartbeat
                await stateless_manager.update_instance_heartbeat()

                if track_operation:
                    # Store operation in temporary data for tracking
                    operation_data = {
                        "function": func.__name__,
                        "timestamp": kwargs.get("timestamp", "unknown"),
                        "instance_id": stateless_manager.instance_id,
                    }

                    tracking_key = f"operation_tracking:{func.__name__}"
                    await stateless_manager.store_temporary_data(
                        tracking_key,
                        operation_data,
                        ttl=300,  # 5 minutes
                    )

            return await func(*args, **kwargs)

        return wrapper

    return decorator


def _get_stateless_manager(*args, **kwargs):
    """Helper to extract stateless manager from various sources."""
    # Try to get stateless_manager from various sources
    if (
        args
        and hasattr(args[0], "state")
        and hasattr(args[0].state, "stateless_manager")
    ):
        # FastAPI app instance
        return args[0].state.stateless_manager
    elif "request" in kwargs and hasattr(
        kwargs["request"].app.state, "stateless_manager"
    ):
        # FastAPI request
        return kwargs["request"].app.state.stateless_manager
    elif "app" in kwargs and hasattr(kwargs["app"].state, "stateless_manager"):
        # App passed as parameter
        return kwargs["app"].state.stateless_manager

    return None


def _generate_operation_key(func: Callable, args: tuple, kwargs: dict) -> str:
    """Generate operation key for idempotency."""
    # Create hash from function name and parameters
    key_components = [func.__name__]

    # Add args (excluding 'self' and common framework objects)
    for arg in args:
        if not (
            hasattr(arg, "__dict__") and hasattr(arg, "__module__")
        ):  # Skip complex objects
            key_components.append(str(arg))

    # Add kwargs (excluding common framework parameters)
    excluded_kwargs = {"request", "app", "self", "cls"}
    for key, value in kwargs.items():
        if key not in excluded_kwargs:
            if not (hasattr(value, "__dict__") and hasattr(value, "__module__")):
                key_components.append(f"{key}:{value}")

    # Create hash
    key_string = "|".join(key_components)
    operation_hash = hashlib.sha256(key_string.encode()).hexdigest()[:16]

    return f"{func.__name__}:{operation_hash}"


# Convenience decorators for common patterns
def idempotent_alert_processing(ttl: int = 3600):
    """Decorator specifically for alert processing operations."""

    def key_func(*args, **kwargs):
        # Extract alert fingerprint for key
        fingerprint = None

        if args and hasattr(args[0], "fingerprint"):
            fingerprint = args[0].fingerprint
        elif "fingerprint" in kwargs:
            fingerprint = kwargs["fingerprint"]
        elif "alert" in kwargs and hasattr(kwargs["alert"], "fingerprint"):
            fingerprint = kwargs["alert"].fingerprint

        if fingerprint:
            return f"alert_processing:{fingerprint}"
        else:
            return _generate_operation_key(
                args[0] if args else lambda: None, args, kwargs
            )

    return idempotent(key_func=key_func, ttl=ttl)


def idempotent_webhook(ttl: int = 300):  # 5 minutes for webhooks
    """Decorator for webhook processing idempotency."""

    def key_func(*args, **kwargs):
        # Extract webhook ID or generate from payload
        webhook_id = kwargs.get("webhook_id") or kwargs.get("request_id")

        if webhook_id:
            return f"webhook:{webhook_id}"

        # Generate from payload hash
        payload = kwargs.get("payload") or kwargs.get("data")
        if payload:
            payload_hash = hashlib.sha256(str(payload).encode()).hexdigest()[:16]
            return f"webhook:payload:{payload_hash}"

        return _generate_operation_key(args[0] if args else lambda: None, args, kwargs)

    return idempotent(key_func=key_func, ttl=ttl)
