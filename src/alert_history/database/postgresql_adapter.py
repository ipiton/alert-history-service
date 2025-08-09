"""
PostgreSQL Storage Adapter для Alert History Service.

Implements IAlertStorage interface for PostgreSQL database:
- Connection pooling с asyncpg
- Transaction management
- Query optimization
- Concurrent access support
- Backward compatibility с SQLite queries
"""

# Standard library imports
import asyncio
import json
import time
from contextlib import asynccontextmanager
from datetime import datetime, timedelta
from typing import Any, Dict, List, Optional

# Third-party imports
import asyncpg
from asyncpg.pool import Pool

# Local imports
from ..core.interfaces import (
    Alert,
    AlertSeverity,
    AlertStatus,
    ClassificationResult,
    IAlertStorage,
)
from ..logging_config import get_logger, get_performance_logger

logger = get_logger(__name__)
performance_logger = get_performance_logger()


class PostgreSQLStorage(IAlertStorage):
    """
    PostgreSQL implementation of alert storage.

    Features:
    - Connection pooling for high performance
    - Transaction support for consistency
    - JSON field support для labels/annotations
    - Async operations для non-blocking I/O
    - Migration compatibility с SQLite
    """

    def __init__(
        self,
        database_url: str,
        min_pool_size: int = 5,
        max_pool_size: int = 20,
        command_timeout: float = 60.0,
        query_timeout: float = 30.0,
    ):
        """Initialize PostgreSQL storage."""
        self.database_url = database_url
        self.min_pool_size = min_pool_size
        self.max_pool_size = max_pool_size
        self.command_timeout = command_timeout
        self.query_timeout = query_timeout

        self.pool: Optional[Pool] = None
        self._connection_lock = asyncio.Lock()
        self._initialized = False

    async def initialize(self) -> None:
        """Initialize connection pool and schema."""
        async with self._connection_lock:
            if self._initialized:
                return

            try:
                # Create connection pool
                self.pool = await asyncpg.create_pool(
                    self.database_url,
                    min_size=self.min_pool_size,
                    max_size=self.max_pool_size,
                    command_timeout=self.command_timeout,
                    server_settings={
                        "jit": "off",  # Disable JIT for better connection speed
                        "application_name": "alert-history-service",
                    },
                )

                # Test connection
                async with self.pool.acquire() as conn:
                    await conn.fetchval("SELECT 1")

                logger.info(
                    "PostgreSQL connection pool initialized",
                    min_size=self.min_pool_size,
                    max_size=self.max_pool_size,
                    database_url=self.database_url.split("@")[-1],  # Hide credentials
                )

                self._initialized = True

            except Exception as e:
                logger.error(f"Failed to initialize PostgreSQL connection: {e}")
                raise

    async def close(self) -> None:
        """Close connection pool."""
        if self.pool:
            await self.pool.close()
            self.pool = None
            self._initialized = False
            logger.info("PostgreSQL connection pool closed")

    @asynccontextmanager
    async def get_connection(self):
        """Get database connection from pool."""
        if not self._initialized:
            await self.initialize()

        if not self.pool:
            raise RuntimeError("Database pool not initialized")

        async with self.pool.acquire() as connection:
            yield connection

    async def save_alert(self, alert: Alert) -> bool:
        """Save alert to PostgreSQL database."""
        try:
            async with self.get_connection() as conn:
                # Prepare data
                labels_json = json.dumps(alert.labels) if alert.labels else "{}"
                annotations_json = (
                    json.dumps(alert.annotations) if alert.annotations else "{}"
                )

                # Insert or update alert
                query = """
                    INSERT INTO alerts (
                        fingerprint, alert_name, namespace, status,
                        labels, annotations, starts_at, ends_at,
                        generator_url, timestamp
                    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
                    ON CONFLICT (fingerprint) DO UPDATE SET
                        status = EXCLUDED.status,
                        ends_at = EXCLUDED.ends_at,
                        updated_at = NOW()
                    RETURNING id
                """

                alert_id = await conn.fetchval(
                    query,
                    alert.fingerprint,
                    alert.alert_name,
                    alert.namespace,
                    alert.status.value,
                    labels_json,
                    annotations_json,
                    alert.starts_at,
                    alert.ends_at,
                    alert.generator_url,
                    alert.timestamp,
                )

                logger.debug(
                    "Alert saved to PostgreSQL",
                    alert_id=alert_id,
                    fingerprint=alert.fingerprint,
                    alert_name=alert.alert_name,
                )

                return True

        except Exception as e:
            logger.error(
                "Failed to save alert to PostgreSQL",
                fingerprint=alert.fingerprint,
                error=str(e),
            )
            return False

    async def get_alert_by_fingerprint(self, fingerprint: str) -> Optional[Alert]:
        """Get alert by fingerprint."""
        try:
            async with self.get_connection() as conn:
                query = """
                    SELECT
                        fingerprint, alert_name, namespace, status,
                        labels, annotations, starts_at, ends_at,
                        generator_url, timestamp
                    FROM alerts
                    WHERE fingerprint = $1
                    ORDER BY created_at DESC
                    LIMIT 1
                """

                row = await conn.fetchrow(query, fingerprint)

                if row:
                    return self._row_to_alert(row)

                return None

        except Exception as e:
            logger.error(f"Failed to get alert by fingerprint: {e}")
            return None

    async def get_alerts(
        self,
        alert_name: Optional[str] = None,
        status: Optional[str] = None,
        namespace: Optional[str] = None,
        start_time: Optional[datetime] = None,
        end_time: Optional[datetime] = None,
        limit: int = 1000,
        offset: int = 0,
    ) -> List[Alert]:
        """Get alerts with filters."""
        try:
            async with self.get_connection() as conn:
                # Build dynamic query
                conditions = []
                params = []
                param_count = 0

                if alert_name:
                    param_count += 1
                    conditions.append(f"alert_name = ${param_count}")
                    params.append(alert_name)

                if status:
                    param_count += 1
                    conditions.append(f"status = ${param_count}")
                    params.append(status)

                if namespace:
                    param_count += 1
                    conditions.append(f"namespace = ${param_count}")
                    params.append(namespace)

                if start_time:
                    param_count += 1
                    conditions.append(f"timestamp >= ${param_count}")
                    params.append(start_time)

                if end_time:
                    param_count += 1
                    conditions.append(f"timestamp <= ${param_count}")
                    params.append(end_time)

                where_clause = "WHERE " + " AND ".join(conditions) if conditions else ""

                # Add limit and offset
                param_count += 1
                limit_clause = f"LIMIT ${param_count}"
                params.append(limit)

                param_count += 1
                offset_clause = f"OFFSET ${param_count}"
                params.append(offset)

                query = f"""
                    SELECT
                        fingerprint, alert_name, namespace, status,
                        labels, annotations, starts_at, ends_at,
                        generator_url, timestamp
                    FROM alerts
                    {where_clause}
                    ORDER BY timestamp DESC
                    {limit_clause} {offset_clause}
                """

                rows = await conn.fetch(query, *params)

                return [self._row_to_alert(row) for row in rows]

        except Exception as e:
            logger.error(f"Failed to get alerts: {e}")
            return []

    async def get_alert_history(
        self,
        hours: int = 24,
        alert_name: Optional[str] = None,
        status: Optional[str] = None,
    ) -> List[Dict[str, Any]]:
        """Get alert history for reports."""
        try:
            async with self.get_connection() as conn:
                # Build conditions
                conditions = ["timestamp >= NOW() - INTERVAL '%s hours'" % hours]
                params = []
                param_count = 0

                if alert_name:
                    param_count += 1
                    conditions.append(f"alert_name = ${param_count}")
                    params.append(alert_name)

                if status:
                    param_count += 1
                    conditions.append(f"status = ${param_count}")
                    params.append(status)

                where_clause = "WHERE " + " AND ".join(conditions)

                query = f"""
                    SELECT
                        fingerprint, alert_name, namespace, status,
                        labels, annotations, starts_at, ends_at,
                        generator_url, timestamp,
                        created_at, updated_at
                    FROM alerts
                    {where_clause}
                    ORDER BY timestamp DESC
                """

                rows = await conn.fetch(query, *params)

                # Convert to dict format for compatibility
                return [
                    {
                        "fingerprint": row["fingerprint"],
                        "alert_name": row["alert_name"],
                        "namespace": row["namespace"],
                        "status": row["status"],
                        "labels": json.loads(row["labels"]) if row["labels"] else {},
                        "annotations": (
                            json.loads(row["annotations"]) if row["annotations"] else {}
                        ),
                        "starts_at": (
                            row["starts_at"].isoformat() if row["starts_at"] else None
                        ),
                        "ends_at": (
                            row["ends_at"].isoformat() if row["ends_at"] else None
                        ),
                        "generator_url": row["generator_url"],
                        "timestamp": row["timestamp"].isoformat(),
                        "created_at": row["created_at"].isoformat(),
                        "updated_at": row["updated_at"].isoformat(),
                    }
                    for row in rows
                ]

        except Exception as e:
            logger.error(f"Failed to get alert history: {e}")
            return []

    async def save_classification(
        self, fingerprint: str, classification: ClassificationResult
    ) -> bool:
        """Save LLM classification result."""
        try:
            async with self.get_connection() as conn:
                # Calculate expiration time
                expires_at = None
                if hasattr(classification, "ttl") and classification.ttl:
                    expires_at = datetime.utcnow() + timedelta(
                        seconds=classification.ttl
                    )

                query = """
                    INSERT INTO alert_classifications (
                        alert_fingerprint, severity, confidence, reasoning,
                        recommendations, processing_time, metadata,
                        llm_model, llm_version, cache_hit, expires_at
                    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
                    RETURNING id
                """

                classification_id = await conn.fetchval(
                    query,
                    fingerprint,
                    classification.severity.value,
                    classification.confidence,
                    classification.reasoning,
                    (
                        json.dumps(classification.recommendations)
                        if classification.recommendations
                        else "[]"
                    ),
                    classification.processing_time,
                    (
                        json.dumps(classification.metadata)
                        if classification.metadata
                        else "{}"
                    ),
                    getattr(classification, "llm_model", None),
                    getattr(classification, "llm_version", None),
                    getattr(classification, "cache_hit", False),
                    expires_at,
                )

                logger.debug(
                    "Classification saved to PostgreSQL",
                    classification_id=classification_id,
                    fingerprint=fingerprint,
                    severity=classification.severity.value,
                )

                return True

        except Exception as e:
            logger.error(f"Failed to save classification: {e}")
            return False

    async def get_classification(
        self, fingerprint: str
    ) -> Optional[ClassificationResult]:
        """Get latest classification for alert."""
        try:
            async with self.get_connection() as conn:
                query = """
                    SELECT
                        severity, confidence, reasoning, recommendations,
                        processing_time, metadata, llm_model, llm_version,
                        cache_hit, created_at
                    FROM alert_classifications
                    WHERE alert_fingerprint = $1
                    AND (expires_at IS NULL OR expires_at > NOW())
                    ORDER BY created_at DESC
                    LIMIT 1
                """

                row = await conn.fetchrow(query, fingerprint)

                if row:
                    return ClassificationResult(
                        severity=AlertSeverity(row["severity"]),
                        confidence=float(row["confidence"]),
                        reasoning=row["reasoning"],
                        recommendations=(
                            json.loads(row["recommendations"])
                            if row["recommendations"]
                            else []
                        ),
                        processing_time=(
                            float(row["processing_time"])
                            if row["processing_time"]
                            else 0.0
                        ),
                        metadata=json.loads(row["metadata"]) if row["metadata"] else {},
                    )

                return None

        except Exception as e:
            logger.error(f"Failed to get classification: {e}")
            return None

    async def save_publishing_result(
        self,
        fingerprint: str,
        target_name: str,
        target_type: str,
        target_format: str,
        status: str,
        attempt_number: int = 1,
        response_code: Optional[int] = None,
        response_message: Optional[str] = None,
        processing_time: Optional[float] = None,
        error_details: Optional[Dict[str, Any]] = None,
    ) -> bool:
        """Save publishing result."""
        try:
            async with self.get_connection() as conn:
                query = """
                    INSERT INTO alert_publishing_history (
                        alert_fingerprint, target_name, target_type, target_format,
                        status, attempt_number, response_code, response_message,
                        processing_time, error_details
                    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
                    RETURNING id
                """

                publish_id = await conn.fetchval(
                    query,
                    fingerprint,
                    target_name,
                    target_type,
                    target_format,
                    status,
                    attempt_number,
                    response_code,
                    response_message,
                    processing_time,
                    json.dumps(error_details) if error_details else None,
                )

                # Update target statistics
                await self._update_target_stats(
                    conn, target_name, status, processing_time
                )

                logger.debug(
                    "Publishing result saved",
                    publish_id=publish_id,
                    fingerprint=fingerprint,
                    target=target_name,
                    status=status,
                )

                return True

        except Exception as e:
            logger.error(f"Failed to save publishing result: {e}")
            return False

    async def _update_target_stats(
        self, conn, target_name: str, status: str, processing_time: Optional[float]
    ) -> None:
        """Update target statistics."""
        try:
            # Update или insert target stats
            if status == "success":
                query = """
                    UPDATE publishing_targets SET
                        last_success_at = NOW(),
                        success_count = success_count + 1,
                        total_attempts = total_attempts + 1,
                        failure_count = 0,
                        updated_at = NOW()
                    WHERE name = $1
                """
            else:
                query = """
                    UPDATE publishing_targets SET
                        last_failure_at = NOW(),
                        failure_count = failure_count + 1,
                        total_attempts = total_attempts + 1,
                        updated_at = NOW()
                    WHERE name = $1
                """

            await conn.execute(query, target_name)

        except Exception as e:
            logger.warning(f"Failed to update target stats: {e}")

    async def cleanup_old_data(self, retention_days: int = 30) -> int:
        """Clean up old data based on retention policy."""
        try:
            async with self.get_connection() as conn:
                # Use the stored procedure
                deleted_count = await conn.fetchval(
                    "SELECT cleanup_old_data($1)", retention_days
                )

                logger.info(
                    "Cleanup completed",
                    deleted_alerts=deleted_count,
                    retention_days=retention_days,
                )

                return deleted_count

        except Exception as e:
            logger.error(f"Failed to cleanup old data: {e}")
            return 0

    async def cleanup_old_alerts(self, retention_days: int) -> int:
        """Clean up old alerts and return count of deleted records (IAlertStorage interface)."""
        return await self.cleanup_old_data(retention_days)

    async def get_statistics(self) -> Dict[str, Any]:
        """Get database statistics."""
        try:
            async with self.get_connection() as conn:
                # Get various statistics
                stats_queries = {
                    "total_alerts": "SELECT COUNT(*) FROM alerts",
                    "alerts_last_24h": "SELECT COUNT(*) FROM alerts WHERE created_at >= NOW() - INTERVAL '24 hours'",
                    "total_classifications": "SELECT COUNT(*) FROM alert_classifications",
                    "classifications_last_24h": "SELECT COUNT(*) FROM alert_classifications WHERE created_at >= NOW() - INTERVAL '24 hours'",
                    "total_publishes": "SELECT COUNT(*) FROM alert_publishing_history",
                    "publishes_last_24h": "SELECT COUNT(*) FROM alert_publishing_history WHERE created_at >= NOW() - INTERVAL '24 hours'",
                    "active_targets": "SELECT COUNT(*) FROM publishing_targets WHERE enabled = TRUE",
                }

                stats = {}
                for stat_name, query in stats_queries.items():
                    stats[stat_name] = await conn.fetchval(query)

                # Get database size
                db_size_query = """
                    SELECT pg_size_pretty(pg_database_size(current_database())) as size
                """
                stats["database_size"] = await conn.fetchval(db_size_query)

                return stats

        except Exception as e:
            logger.error(f"Failed to get statistics: {e}")
            return {}

    def _row_to_alert(self, row) -> Alert:
        """Convert database row to Alert object."""
        return Alert(
            fingerprint=row["fingerprint"],
            alert_name=row["alert_name"],
            namespace=row["namespace"],
            status=AlertStatus(row["status"]),
            labels=json.loads(row["labels"]) if row["labels"] else {},
            annotations=json.loads(row["annotations"]) if row["annotations"] else {},
            starts_at=row["starts_at"],
            ends_at=row["ends_at"],
            generator_url=row["generator_url"],
            timestamp=row["timestamp"],
        )

    async def health_check(self) -> Dict[str, Any]:
        """Health check for PostgreSQL connection."""
        try:
            start_time = time.time()

            async with self.get_connection() as conn:
                # Test basic connectivity
                result = await conn.fetchval("SELECT 1")

                # Test query performance
                await conn.fetchval("SELECT COUNT(*) FROM alerts LIMIT 1")

            response_time = time.time() - start_time

            return {
                "status": "healthy",
                "response_time": response_time,
                "pool_size": self.pool.get_size() if self.pool else 0,
                "pool_max_size": self.max_pool_size,
                "database": "postgresql",
            }

        except Exception as e:
            return {"status": "unhealthy", "error": str(e), "database": "postgresql"}
