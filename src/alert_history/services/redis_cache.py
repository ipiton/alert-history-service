"""
Redis Cache и Distributed Locking для Alert History Service.

Implements:
- ICache interface для LLM result caching
- Distributed locking для coordination между instances
- Session management для stateless operations
- Performance optimizations через pipelining
"""

# Standard library imports
import asyncio
import json
import time
import uuid
from contextlib import asynccontextmanager
from datetime import datetime
from typing import Any, Dict, List, Optional

# Third-party imports
import aioredis
from aioredis import Redis

# Local imports
from ..core.interfaces import AlertSeverity, ClassificationResult, ICache
from ..logging_config import get_logger, get_performance_logger
from ..utils.common import safe_json_dumps

logger = get_logger(__name__)
performance_logger = get_performance_logger()


class RedisCache(ICache):
    """
    Redis-based cache implementation.

    Features:
    - LLM result caching с TTL
    - Distributed locking
    - Connection pooling
    - Serialization/deserialization
    - Performance monitoring
    """

    def __init__(
        self,
        redis_url: str,
        default_ttl: int = 3600,  # 1 hour
        max_connections: int = 20,
        retry_on_timeout: bool = True,
        socket_timeout: float = 5.0,
        socket_connect_timeout: float = 5.0,
    ):
        """Initialize Redis cache."""
        self.redis_url = redis_url
        self.default_ttl = default_ttl
        self.max_connections = max_connections
        self.retry_on_timeout = retry_on_timeout
        self.socket_timeout = socket_timeout
        self.socket_connect_timeout = socket_connect_timeout

        self.redis: Optional[Redis] = None
        self._connection_lock = asyncio.Lock()
        self._initialized = False

        # Performance tracking
        self._cache_hits = 0
        self._cache_misses = 0
        self._cache_errors = 0

    async def initialize(self) -> None:
        """Initialize Redis connection."""
        async with self._connection_lock:
            if self._initialized:
                return

            try:
                self.redis = await aioredis.from_url(
                    self.redis_url,
                    max_connections=self.max_connections,
                    retry_on_timeout=self.retry_on_timeout,
                    socket_timeout=self.socket_timeout,
                    socket_connect_timeout=self.socket_connect_timeout,
                    decode_responses=True,
                )

                # Test connection
                await self.redis.ping()

                logger.info(
                    "Redis connection initialized",
                    url=self.redis_url.split("@")[-1],  # Hide credentials
                    max_connections=self.max_connections,
                    default_ttl=self.default_ttl,
                )

                self._initialized = True

            except Exception as e:
                logger.error(f"Failed to initialize Redis connection: {e}")
                raise

    async def close(self) -> None:
        """Close Redis connection."""
        if self.redis:
            await self.redis.close()
            self.redis = None
            self._initialized = False
            logger.info("Redis connection closed")

    async def get(self, key: str) -> Optional[Any]:
        """Get value from cache."""
        if not self._initialized:
            await self.initialize()

        try:
            start_time = time.time()

            value = await self.redis.get(key)

            if value is not None:
                self._cache_hits += 1
                result = json.loads(value)

                performance_logger.debug(
                    "Cache hit", key=key, duration=time.time() - start_time
                )

                return result
            else:
                self._cache_misses += 1

                performance_logger.debug(
                    "Cache miss", key=key, duration=time.time() - start_time
                )

                return None

        except Exception as e:
            self._cache_errors += 1
            logger.warning(f"Cache get error for key {key}: {e}")
            return None

    async def set(self, key: str, value: Any, ttl: Optional[int] = None) -> bool:
        """Set value in cache with TTL."""
        if not self._initialized:
            await self.initialize()

        try:
            start_time = time.time()

            serialized_value = safe_json_dumps(value)
            effective_ttl = ttl or self.default_ttl

            await self.redis.setex(key, effective_ttl, serialized_value)

            performance_logger.debug(
                "Cache set",
                key=key,
                ttl=effective_ttl,
                size=len(serialized_value),
                duration=time.time() - start_time,
            )

            return True

        except Exception as e:
            self._cache_errors += 1
            logger.warning(f"Cache set error for key {key}: {e}")
            return False

    async def delete(self, key: str) -> bool:
        """Delete key from cache."""
        if not self._initialized:
            await self.initialize()

        try:
            result = await self.redis.delete(key)

            logger.debug(f"Cache delete key {key}, deleted: {result > 0}")

            return result > 0

        except Exception as e:
            self._cache_errors += 1
            logger.warning(f"Cache delete error for key {key}: {e}")
            return False

    async def exists(self, key: str) -> bool:
        """Check if key exists in cache."""
        if not self._initialized:
            await self.initialize()

        try:
            result = await self.redis.exists(key)
            return result > 0

        except Exception as e:
            logger.warning(f"Cache exists error for key {key}: {e}")
            return False

    async def get_many(self, keys: List[str]) -> Dict[str, Any]:
        """Get multiple values from cache."""
        if not self._initialized:
            await self.initialize()

        if not keys:
            return {}

        try:
            start_time = time.time()

            values = await self.redis.mget(keys)

            result = {}
            for key, value in zip(keys, values):
                if value is not None:
                    try:
                        result[key] = json.loads(value)
                        self._cache_hits += 1
                    except json.JSONDecodeError:
                        logger.warning(f"Invalid JSON in cache for key {key}")
                        self._cache_errors += 1
                else:
                    self._cache_misses += 1

            performance_logger.debug(
                "Cache mget",
                keys_requested=len(keys),
                keys_found=len(result),
                duration=time.time() - start_time,
            )

            return result

        except Exception as e:
            self._cache_errors += len(keys)
            logger.warning(f"Cache mget error: {e}")
            return {}

    async def set_many(self, items: Dict[str, Any], ttl: Optional[int] = None) -> bool:
        """Set multiple values in cache."""
        if not self._initialized:
            await self.initialize()

        if not items:
            return True

        try:
            start_time = time.time()
            effective_ttl = ttl or self.default_ttl

            # Use pipeline for efficiency
            pipe = self.redis.pipeline()

            for key, value in items.items():
                serialized_value = safe_json_dumps(value)
                pipe.setex(key, effective_ttl, serialized_value)

            await pipe.execute()

            performance_logger.debug(
                "Cache mset",
                keys_count=len(items),
                ttl=effective_ttl,
                duration=time.time() - start_time,
            )

            return True

        except Exception as e:
            self._cache_errors += len(items)
            logger.warning(f"Cache mset error: {e}")
            return False

    async def clear_pattern(self, pattern: str) -> int:
        """Clear keys matching pattern."""
        if not self._initialized:
            await self.initialize()

        try:
            keys = await self.redis.keys(pattern)

            if keys:
                deleted = await self.redis.delete(*keys)
                logger.info(f"Cleared {deleted} keys matching pattern: {pattern}")
                return deleted

            return 0

        except Exception as e:
            logger.warning(f"Cache clear pattern error: {e}")
            return 0

    # ===============================
    # LLM Classification Caching
    # ===============================

    async def cache_classification(
        self,
        fingerprint: str,
        classification: ClassificationResult,
        ttl: Optional[int] = None,
    ) -> bool:
        """Cache LLM classification result."""
        cache_key = f"classification:{fingerprint}"

        cache_data = {
            "severity": classification.severity.value,
            "confidence": classification.confidence,
            "reasoning": classification.reasoning,
            "recommendations": classification.recommendations,
            "processing_time": classification.processing_time,
            "metadata": classification.metadata,
            "cached_at": datetime.utcnow().isoformat(),
            "cache_hit": True,  # Mark as cached for future retrievals
        }

        return await self.set(cache_key, cache_data, ttl)

    async def get_cached_classification(
        self, fingerprint: str
    ) -> Optional[ClassificationResult]:
        """Get cached LLM classification."""
        cache_key = f"classification:{fingerprint}"

        cached_data = await self.get(cache_key)

        if cached_data:
            try:
                return ClassificationResult(
                    severity=AlertSeverity(cached_data["severity"]),
                    confidence=cached_data["confidence"],
                    reasoning=cached_data["reasoning"],
                    recommendations=cached_data["recommendations"],
                    processing_time=cached_data["processing_time"],
                    metadata=cached_data.get("metadata", {}),
                )
            except Exception as e:
                logger.warning(f"Failed to deserialize cached classification: {e}")
                # Invalid cache entry, delete it
                await self.delete(cache_key)

        return None

    # ===============================
    # Distributed Locking
    # ===============================

    @asynccontextmanager
    async def distributed_lock(
        self,
        lock_name: str,
        timeout: float = 30.0,
        blocking_timeout: Optional[float] = None,
    ):
        """
        Distributed lock implementation using Redis.

        Args:
            lock_name: Unique name for the lock
            timeout: Lock timeout in seconds
            blocking_timeout: Time to wait for lock acquisition
        """
        if not self._initialized:
            await self.initialize()

        lock_key = f"lock:{lock_name}"
        lock_value = str(uuid.uuid4())
        acquired = False

        try:
            # Try to acquire lock
            start_time = time.time()

            while True:
                # Atomic set with expiration
                acquired = await self.redis.set(
                    lock_key, lock_value, ex=int(timeout), nx=True
                )

                if acquired:
                    logger.debug(f"Acquired distributed lock: {lock_name}")
                    break

                # Check timeout
                if blocking_timeout is not None:
                    elapsed = time.time() - start_time
                    if elapsed >= blocking_timeout:
                        raise TimeoutError(
                            f"Failed to acquire lock {lock_name} within {blocking_timeout}s"
                        )

                # Short sleep before retry
                await asyncio.sleep(0.1)

            yield

        finally:
            if acquired:
                try:
                    # Release lock using Lua script for atomicity
                    lua_script = """
                    if redis.call("get", KEYS[1]) == ARGV[1] then
                        return redis.call("del", KEYS[1])
                    else
                        return 0
                    end
                    """

                    result = await self.redis.eval(lua_script, 1, lock_key, lock_value)

                    if result:
                        logger.debug(f"Released distributed lock: {lock_name}")
                    else:
                        logger.warning(
                            f"Lock {lock_name} was already expired or taken by another process"
                        )

                except Exception as e:
                    logger.error(f"Error releasing lock {lock_name}: {e}")

    async def is_locked(self, lock_name: str) -> bool:
        """Check if lock is currently held."""
        if not self._initialized:
            await self.initialize()

        lock_key = f"lock:{lock_name}"
        return await self.exists(lock_key)

    # ===============================
    # Session Management
    # ===============================

    async def create_session(
        self, session_id: str, data: Dict[str, Any], ttl: int = 3600
    ) -> bool:
        """Create or update session data."""
        session_key = f"session:{session_id}"

        session_data = {
            "data": data,
            "created_at": datetime.utcnow().isoformat(),
            "last_accessed": datetime.utcnow().isoformat(),
        }

        return await self.set(session_key, session_data, ttl)

    async def get_session(self, session_id: str) -> Optional[Dict[str, Any]]:
        """Get session data."""
        session_key = f"session:{session_id}"

        session_data = await self.get(session_key)

        if session_data:
            # Update last accessed time
            session_data["last_accessed"] = datetime.utcnow().isoformat()
            await self.set(session_key, session_data)  # Refresh TTL

            return session_data.get("data", {})

        return None

    async def delete_session(self, session_id: str) -> bool:
        """Delete session."""
        session_key = f"session:{session_id}"
        return await self.delete(session_key)

    # ===============================
    # Monitoring and Statistics
    # ===============================

    async def get_stats(self) -> Dict[str, Any]:
        """Get cache statistics."""
        if not self._initialized:
            await self.initialize()

        try:
            # Get Redis info
            info = await self.redis.info()

            total_requests = self._cache_hits + self._cache_misses
            hit_rate = (
                (self._cache_hits / total_requests * 100) if total_requests > 0 else 0
            )

            return {
                "cache_hits": self._cache_hits,
                "cache_misses": self._cache_misses,
                "cache_errors": self._cache_errors,
                "hit_rate_percent": round(hit_rate, 2),
                "redis_version": info.get("redis_version"),
                "used_memory_human": info.get("used_memory_human"),
                "connected_clients": info.get("connected_clients"),
                "total_connections_received": info.get("total_connections_received"),
                "total_commands_processed": info.get("total_commands_processed"),
                "keyspace_hits": info.get("keyspace_hits"),
                "keyspace_misses": info.get("keyspace_misses"),
            }

        except Exception as e:
            logger.error(f"Failed to get Redis stats: {e}")
            return {
                "cache_hits": self._cache_hits,
                "cache_misses": self._cache_misses,
                "cache_errors": self._cache_errors,
                "error": str(e),
            }

    async def health_check(self) -> Dict[str, Any]:
        """Health check for Redis connection."""
        try:
            start_time = time.time()

            # Test basic connectivity
            await self.redis.ping()

            # Test write/read
            test_key = f"health_check:{int(time.time())}"
            await self.redis.set(test_key, "test", ex=60)
            test_result = await self.redis.get(test_key)
            await self.redis.delete(test_key)

            response_time = time.time() - start_time

            return {
                "status": "healthy",
                "response_time": response_time,
                "ping_success": True,
                "read_write_test": test_result == "test",
                "cache": "redis",
            }

        except Exception as e:
            return {"status": "unhealthy", "error": str(e), "cache": "redis"}
