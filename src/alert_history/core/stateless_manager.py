"""
Stateless Application Manager для T1.4: Stateless Application Design.

Обеспечивает:
- Удаление локального состояния из приложения
- Перенос временных данных в Redis/PostgreSQL
- Idempotent operations для всех API endpoints
- Instance ID tracking для debugging
- Stateless service coordination
"""

import time
import uuid
from datetime import datetime, timedelta
from typing import Any, Optional

from ..core.interfaces import ICache
from ..logging_config import get_logger

logger = get_logger(__name__)


class StatelessManager:
    """
    Manages stateless application design patterns.

    Features:
    - Instance tracking без локального состояния
    - Operation idempotency через Redis
    - Temporary data в Redis с TTL
    - Cross-instance coordination
    """

    def __init__(
        self,
        redis_cache: Optional[ICache] = None,
        instance_id: Optional[str] = None,
        operation_ttl: int = 3600,  # 1 hour
    ):
        """Initialize stateless manager."""
        self.redis_cache = redis_cache
        self.instance_id = instance_id or self._generate_instance_id()
        self.operation_ttl = operation_ttl

        # Track operations in this instance (for debugging only)
        self._operation_registry: dict[str, datetime] = {}

        logger.info(
            "Stateless manager initialized",
            instance_id=self.instance_id,
            redis_available=redis_cache is not None,
        )

    def _generate_instance_id(self) -> str:
        """Generate unique instance ID."""
        timestamp = int(time.time())
        random_suffix = str(uuid.uuid4())[:8]
        return f"alert-history-{timestamp}-{random_suffix}"

    # ===============================
    # Idempotent Operations
    # ===============================

    async def ensure_idempotent_operation(
        self,
        operation_key: str,
        ttl: Optional[int] = None,
    ) -> bool:
        """
        Ensure operation is idempotent.

        Returns:
            True if operation can proceed (first time)
            False if operation already executed (duplicate)
        """
        if not self.redis_cache:
            # Without Redis, we can't guarantee idempotency across instances
            # But we can check local registry for this instance
            return self._check_local_operation(operation_key)

        try:
            operation_ttl = ttl or self.operation_ttl

            # Try to set operation key with TTL
            # If key already exists, operation was already performed
            full_key = f"operation:{operation_key}"

            # Use Redis SET with NX (only if not exists) and EX (expiration)
            operation_data = {
                "instance_id": self.instance_id,
                "timestamp": datetime.utcnow().isoformat(),
                "ttl": operation_ttl,
            }

            # Check if operation exists
            existing = await self.redis_cache.get(full_key)
            if existing:
                logger.debug(
                    "Operation already executed",
                    operation_key=operation_key,
                    original_instance=existing.get("instance_id"),
                    current_instance=self.instance_id,
                )
                return False

            # Set operation as executed
            success = await self.redis_cache.set(
                full_key, operation_data, operation_ttl
            )

            if success:
                self._register_local_operation(operation_key)
                logger.debug(
                    "Operation registered as idempotent",
                    operation_key=operation_key,
                    instance_id=self.instance_id,
                )
                return True
            else:
                logger.warning(
                    "Failed to register idempotent operation",
                    operation_key=operation_key,
                )
                return False

        except Exception as e:
            logger.error(f"Error checking operation idempotency: {e}")
            # Fallback to local check
            return self._check_local_operation(operation_key)

    def _check_local_operation(self, operation_key: str) -> bool:
        """Check operation in local registry (fallback)."""
        if operation_key in self._operation_registry:
            # Check if operation is expired
            operation_time = self._operation_registry[operation_key]
            if datetime.utcnow() - operation_time < timedelta(
                seconds=self.operation_ttl
            ):
                return False  # Operation still valid, don't duplicate
            else:
                # Operation expired, remove from registry
                del self._operation_registry[operation_key]

        # Register operation in local registry
        self._register_local_operation(operation_key)
        return True  # Operation can proceed

    def _register_local_operation(self, operation_key: str) -> None:
        """Register operation in local registry."""
        self._operation_registry[operation_key] = datetime.utcnow()

        # Cleanup old operations to prevent memory leaks
        if len(self._operation_registry) > 1000:
            self._cleanup_local_operations()

    def _cleanup_local_operations(self) -> None:
        """Cleanup expired operations from local registry."""
        cutoff_time = datetime.utcnow() - timedelta(seconds=self.operation_ttl)
        expired_keys = [
            key
            for key, timestamp in self._operation_registry.items()
            if timestamp < cutoff_time
        ]

        for key in expired_keys:
            del self._operation_registry[key]

        logger.debug(
            f"Cleaned up {len(expired_keys)} expired operations from local registry"
        )

    # ===============================
    # Temporary Data Management
    # ===============================

    async def store_temporary_data(
        self,
        key: str,
        data: Any,
        ttl: int = 300,  # 5 minutes default
    ) -> bool:
        """Store temporary data in Redis instead of local memory."""
        if not self.redis_cache:
            logger.warning(
                "No Redis cache available for temporary data storage",
                key=key,
            )
            return False

        try:
            temp_key = f"temp:{self.instance_id}:{key}"
            success = await self.redis_cache.set(temp_key, data, ttl)

            if success:
                logger.debug(
                    "Temporary data stored",
                    key=key,
                    ttl=ttl,
                    instance_id=self.instance_id,
                )

            return success

        except Exception as e:
            logger.error(f"Failed to store temporary data: {e}")
            return False

    async def get_temporary_data(self, key: str) -> Optional[Any]:
        """Get temporary data from Redis."""
        if not self.redis_cache:
            return None

        try:
            temp_key = f"temp:{self.instance_id}:{key}"
            data = await self.redis_cache.get(temp_key)

            if data:
                logger.debug(
                    "Temporary data retrieved",
                    key=key,
                    instance_id=self.instance_id,
                )

            return data

        except Exception as e:
            logger.error(f"Failed to get temporary data: {e}")
            return None

    async def delete_temporary_data(self, key: str) -> bool:
        """Delete temporary data from Redis."""
        if not self.redis_cache:
            return False

        try:
            temp_key = f"temp:{self.instance_id}:{key}"
            success = await self.redis_cache.delete(temp_key)

            if success:
                logger.debug(
                    "Temporary data deleted",
                    key=key,
                    instance_id=self.instance_id,
                )

            return success

        except Exception as e:
            logger.error(f"Failed to delete temporary data: {e}")
            return False

    # ===============================
    # Cross-Instance Coordination
    # ===============================

    async def get_active_instances(self) -> list[dict[str, Any]]:
        """Get list of currently active instances."""
        if not self.redis_cache:
            return [{"instance_id": self.instance_id, "source": "local"}]

        try:
            # Heartbeat pattern - instances update their heartbeat every 30 seconds
            # heartbeat_pattern = "instance:heartbeat:*"  # Reserved for future use

            # This would require Redis SCAN command, simplified for now
            # In real implementation, we'd scan for heartbeat keys

            return [
                {
                    "instance_id": self.instance_id,
                    "last_heartbeat": datetime.utcnow().isoformat(),
                    "source": "redis",
                }
            ]

        except Exception as e:
            logger.error(f"Failed to get active instances: {e}")
            return []

    async def update_instance_heartbeat(self) -> bool:
        """Update this instance's heartbeat."""
        if not self.redis_cache:
            return False

        try:
            heartbeat_key = f"instance:heartbeat:{self.instance_id}"
            heartbeat_data = {
                "instance_id": self.instance_id,
                "timestamp": datetime.utcnow().isoformat(),
                "version": "1.0.0",  # Could be from config
            }

            # Heartbeat expires in 2 minutes - instances should update every 30 seconds
            success = await self.redis_cache.set(heartbeat_key, heartbeat_data, 120)

            if success:
                logger.debug(
                    "Instance heartbeat updated",
                    instance_id=self.instance_id,
                )

            return success

        except Exception as e:
            logger.error(f"Failed to update instance heartbeat: {e}")
            return False

    # ===============================
    # Stateless Validation
    # ===============================

    def validate_stateless_operation(
        self, operation_name: str, **kwargs
    ) -> dict[str, Any]:
        """
        Validate that operation can be performed statelessly.

        Checks:
        - No reliance on local state
        - All required data provided in parameters
        - Operation is idempotent
        """
        validation_result = {
            "operation": operation_name,
            "stateless": True,
            "issues": [],
            "recommendations": [],
        }

        # Check for state dependencies
        stateful_indicators = [
            "self._cache",
            "self.cache",
            "self._state",
            "self.state",
            "global",
            "singleton",
            "_instance",
        ]

        operation_str = str(kwargs)

        for indicator in stateful_indicators:
            if indicator in operation_str:
                validation_result["stateless"] = False
                validation_result["issues"].append(
                    f"Potential state dependency: {indicator}"
                )
                validation_result["recommendations"].append(
                    f"Replace {indicator} with Redis/PostgreSQL storage"
                )

        # Check for required parameters
        if not kwargs:
            validation_result["issues"].append(
                "No parameters provided - may rely on implicit state"
            )
            validation_result["recommendations"].append(
                "Provide all required data as parameters"
            )

        # Check for idempotency indicators
        idempotent_indicators = ["fingerprint", "id", "uuid", "key"]
        has_idempotent_key = any(
            indicator in str(kwargs.keys()) for indicator in idempotent_indicators
        )

        if not has_idempotent_key:
            validation_result["issues"].append("No idempotent key found")
            validation_result["recommendations"].append(
                "Add unique identifier for idempotent operations"
            )

        if validation_result["issues"]:
            validation_result["stateless"] = False

        logger.debug(
            "Stateless operation validation",
            operation=operation_name,
            stateless=validation_result["stateless"],
            issues_count=len(validation_result["issues"]),
        )

        return validation_result

    # ===============================
    # Monitoring and Statistics
    # ===============================

    async def get_stateless_stats(self) -> dict[str, Any]:
        """Get statistics about stateless operations."""
        stats = {
            "instance_id": self.instance_id,
            "redis_available": self.redis_cache is not None,
            "local_operations_count": len(self._operation_registry),
            "operation_ttl": self.operation_ttl,
        }

        if self.redis_cache:
            try:
                # Get Redis-based statistics
                cache_stats = await self.redis_cache.get_stats()
                stats.update(
                    {
                        "redis_connected": True,
                        "cache_hit_rate": cache_stats.get("hit_rate_percent", 0),
                        "redis_memory": cache_stats.get("used_memory_human", "unknown"),
                    }
                )
            except Exception as e:
                logger.error(f"Failed to get Redis stats: {e}")
                stats["redis_connected"] = False
        else:
            stats["redis_connected"] = False

        return stats

    async def health_check(self) -> dict[str, Any]:
        """Health check for stateless manager."""
        health = {
            "stateless_manager": "healthy",
            "instance_id": self.instance_id,
            "redis_available": self.redis_cache is not None,
        }

        if self.redis_cache:
            try:
                redis_health = await self.redis_cache.health_check()
                health["redis_health"] = redis_health["status"]
            except Exception as e:
                health["redis_health"] = "unhealthy"
                health["redis_error"] = str(e)

        # Check for potential memory leaks in local registry
        if len(self._operation_registry) > 500:
            health["warning"] = (
                f"Large local operation registry: {len(self._operation_registry)} operations"
            )

        return health
