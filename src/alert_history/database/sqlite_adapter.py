"""
SQLite adapter for backward compatibility.

Maintains 100% compatibility with existing SQLite database:
- Same table structure
- Same data types
- Same queries
- Smooth migration path to PostgreSQL
"""

# Standard library imports
import json
import sqlite3
import threading
from datetime import datetime, timedelta
from typing import Any, Dict, List, Optional

# Local imports
from ..core.interfaces import Alert, AlertStatus, ClassificationResult, IAlertStorage
from ..utils.common import parse_timestamp


class SQLiteLegacyStorage(IAlertStorage):
    """
    SQLite storage adapter that maintains full compatibility with legacy database.

    Provides exact same behavior as original alert_history_service.py
    while implementing the new IAlertStorage interface.
    """

    def __init__(self, db_path: str = "alert_history.sqlite3"):
        """Initialize SQLite storage with legacy schema."""
        self.db_path = db_path
        self._lock = threading.Lock()

        # Initialize database with exact same schema as original
        self._init_legacy_db()

    def _init_legacy_db(self) -> None:
        """Initialize database with legacy schema (exact same as original)."""
        with self._lock:
            conn = sqlite3.connect(self.db_path)
            c = conn.cursor()

            # Create alerts table (exact same as original)
            c.execute(
                """
                CREATE TABLE IF NOT EXISTS alerts (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    fingerprint TEXT NOT NULL,
                    alertname TEXT NOT NULL,
                    status TEXT NOT NULL,
                    labels TEXT NOT NULL,
                    annotations TEXT NOT NULL,
                    starts_at TEXT NOT NULL,
                    ends_at TEXT,
                    generator_url TEXT,
                    received_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
                    UNIQUE(fingerprint, starts_at)
                )
            """
            )

            # Create indexes (same as original)
            c.execute(
                "CREATE INDEX IF NOT EXISTS idx_fingerprint ON alerts(fingerprint)"
            )
            c.execute("CREATE INDEX IF NOT EXISTS idx_alertname ON alerts(alertname)")
            c.execute("CREATE INDEX IF NOT EXISTS idx_status ON alerts(status)")
            c.execute("CREATE INDEX IF NOT EXISTS idx_starts_at ON alerts(starts_at)")

            # Create classifications table (new, for LLM data)
            c.execute(
                """
                CREATE TABLE IF NOT EXISTS alert_classifications (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    fingerprint TEXT NOT NULL,
                    severity TEXT NOT NULL,
                    confidence REAL NOT NULL,
                    reasoning TEXT NOT NULL,
                    recommendations TEXT NOT NULL,
                    processing_time REAL NOT NULL,
                    metadata TEXT,
                    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
                    UNIQUE(fingerprint)
                )
            """
            )

            c.execute(
                "CREATE INDEX IF NOT EXISTS idx_classification_fingerprint ON alert_classifications(fingerprint)"
            )

            conn.commit()
            conn.close()

    async def save_alert(self, alert: Alert) -> bool:
        """Save alert using legacy database format."""
        try:
            with self._lock:
                conn = sqlite3.connect(self.db_path)
                c = conn.cursor()

                # Use exact same INSERT OR REPLACE logic as original
                c.execute(
                    """
                    INSERT OR REPLACE INTO alerts
                    (fingerprint, alertname, status, labels, annotations, starts_at, ends_at, generator_url, received_at)
                    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
                """,
                    (
                        alert.fingerprint,
                        alert.alert_name,
                        alert.status.value,
                        json.dumps(alert.labels, sort_keys=True),
                        json.dumps(alert.annotations, sort_keys=True),
                        alert.starts_at.isoformat() if alert.starts_at else None,
                        alert.ends_at.isoformat() if alert.ends_at else None,
                        alert.generator_url,
                        datetime.utcnow().isoformat(),
                    ),
                )

                conn.commit()
                conn.close()

            return True

        except Exception as e:
            print(f"Error saving alert: {e}")
            return False

    async def get_alert_by_fingerprint(self, fingerprint: str) -> Optional[Alert]:
        """Get alert by fingerprint using legacy database."""
        try:
            with self._lock:
                conn = sqlite3.connect(self.db_path)
                c = conn.cursor()

                c.execute(
                    """
                    SELECT fingerprint, alertname, status, labels, annotations,
                           starts_at, ends_at, generator_url
                    FROM alerts
                    WHERE fingerprint = ?
                    ORDER BY starts_at DESC
                    LIMIT 1
                """,
                    (fingerprint,),
                )

                row = c.fetchone()
                conn.close()

                if row:
                    return self._row_to_alert(row)

            return None

        except Exception as e:
            print(f"Error getting alert by fingerprint: {e}")
            return None

    async def get_alerts(
        self, filters: Dict[str, Any], limit: int = 100, offset: int = 0
    ) -> List[Alert]:
        """Get alerts with filters using legacy database (same logic as original)."""
        try:
            with self._lock:
                conn = sqlite3.connect(self.db_path)
                c = conn.cursor()

                # Build WHERE clause (same logic as original)
                where_clauses = []
                params = []

                if "alertname" in filters:
                    where_clauses.append("alertname = ?")
                    params.append(filters["alertname"])

                if "status" in filters:
                    where_clauses.append("status = ?")
                    params.append(filters["status"])

                if "fingerprint" in filters:
                    where_clauses.append("fingerprint = ?")
                    params.append(filters["fingerprint"])

                if "start_time" in filters:
                    where_clauses.append("starts_at >= ?")
                    params.append(filters["start_time"])

                if "end_time" in filters:
                    where_clauses.append("starts_at <= ?")
                    params.append(filters["end_time"])

                # Build query
                query = """
                    SELECT fingerprint, alertname, status, labels, annotations,
                           starts_at, ends_at, generator_url
                    FROM alerts
                """

                if where_clauses:
                    query += " WHERE " + " AND ".join(where_clauses)

                query += " ORDER BY starts_at DESC LIMIT ? OFFSET ?"
                params.extend([limit, offset])

                c.execute(query, params)
                rows = c.fetchall()
                conn.close()

                # Convert rows to Alert objects
                alerts = []
                for row in rows:
                    alert = self._row_to_alert(row)
                    if alert:
                        alerts.append(alert)

                return alerts

        except Exception as e:
            print(f"Error getting alerts: {e}")
            return []

    async def cleanup_old_alerts(self, retention_days: int) -> int:
        """Clean up old alerts (same logic as original)."""
        try:
            cutoff_date = datetime.utcnow() - timedelta(days=retention_days)

            with self._lock:
                conn = sqlite3.connect(self.db_path)
                c = conn.cursor()

                # Delete old alerts
                c.execute(
                    """
                    DELETE FROM alerts
                    WHERE starts_at < ?
                """,
                    (cutoff_date.isoformat(),),
                )

                deleted_count = c.rowcount

                # Also clean up old classifications
                c.execute(
                    """
                    DELETE FROM alert_classifications
                    WHERE fingerprint NOT IN (SELECT fingerprint FROM alerts)
                """,
                )

                conn.commit()
                conn.close()

                return deleted_count

        except Exception as e:
            print(f"Error cleaning up old alerts: {e}")
            return 0

    def _row_to_alert(self, row: tuple) -> Optional[Alert]:
        """Convert database row to Alert object."""
        try:
            (
                fingerprint,
                alertname,
                status,
                labels_json,
                annotations_json,
                starts_at,
                ends_at,
                generator_url,
            ) = row

            # Parse JSON fields
            labels = json.loads(labels_json) if labels_json else {}
            annotations = json.loads(annotations_json) if annotations_json else {}

            # Parse timestamps
            starts_at_dt = (
                parse_timestamp(starts_at) if starts_at else datetime.utcnow()
            )
            ends_at_dt = parse_timestamp(ends_at) if ends_at else None

            # Parse status
            alert_status = AlertStatus.FIRING
            if status == "resolved":
                alert_status = AlertStatus.RESOLVED

            return Alert(
                fingerprint=fingerprint,
                alert_name=alertname,
                status=alert_status,
                labels=labels,
                annotations=annotations,
                starts_at=starts_at_dt,
                ends_at=ends_at_dt,
                generator_url=generator_url,
            )

        except Exception as e:
            print(f"Error converting row to alert: {e}")
            return None

    # Additional methods for LLM classification storage

    async def save_classification(
        self, fingerprint: str, result: ClassificationResult
    ) -> bool:
        """Save classification result."""
        try:
            with self._lock:
                conn = sqlite3.connect(self.db_path)
                c = conn.cursor()

                c.execute(
                    """
                    INSERT OR REPLACE INTO alert_classifications
                    (fingerprint, severity, confidence, reasoning, recommendations,
                     processing_time, metadata, created_at)
                    VALUES (?, ?, ?, ?, ?, ?, ?, ?)
                """,
                    (
                        fingerprint,
                        result.severity.value,
                        result.confidence,
                        result.reasoning,
                        json.dumps(result.recommendations),
                        result.processing_time,
                        json.dumps(result.metadata) if result.metadata else None,
                        datetime.utcnow().isoformat(),
                    ),
                )

                conn.commit()
                conn.close()

            return True

        except Exception as e:
            print(f"Error saving classification: {e}")
            return False

    async def get_classification(
        self, fingerprint: str
    ) -> Optional[ClassificationResult]:
        """Get classification result by fingerprint."""
        try:
            with self._lock:
                conn = sqlite3.connect(self.db_path)
                c = conn.cursor()

                c.execute(
                    """
                    SELECT severity, confidence, reasoning, recommendations,
                           processing_time, metadata
                    FROM alert_classifications
                    WHERE fingerprint = ?
                """,
                    (fingerprint,),
                )

                row = c.fetchone()
                conn.close()

                if row:
                    (
                        severity,
                        confidence,
                        reasoning,
                        recommendations_json,
                        processing_time,
                        metadata_json,
                    ) = row

                    # Parse JSON fields
                    recommendations = (
                        json.loads(recommendations_json) if recommendations_json else []
                    )
                    metadata = json.loads(metadata_json) if metadata_json else None

                    # Local imports
                    from ..core.interfaces import AlertSeverity

                    return ClassificationResult(
                        severity=AlertSeverity(severity),
                        confidence=confidence,
                        reasoning=reasoning,
                        recommendations=recommendations,
                        processing_time=processing_time,
                        metadata=metadata,
                    )

            return None

        except Exception as e:
            print(f"Error getting classification: {e}")
            return None

    def get_database_stats(self) -> Dict[str, Any]:
        """Get database statistics for monitoring."""
        try:
            with self._lock:
                conn = sqlite3.connect(self.db_path)
                c = conn.cursor()

                # Count alerts
                c.execute("SELECT COUNT(*) FROM alerts")
                total_alerts = c.fetchone()[0]

                # Count by status
                c.execute("SELECT status, COUNT(*) FROM alerts GROUP BY status")
                status_counts = dict(c.fetchall())

                # Count classifications
                c.execute("SELECT COUNT(*) FROM alert_classifications")
                total_classifications = c.fetchone()[0]

                # Database size
                c.execute(
                    "SELECT page_count * page_size as size FROM pragma_page_count(), pragma_page_size()"
                )
                db_size = c.fetchone()[0]

                conn.close()

                return {
                    "total_alerts": total_alerts,
                    "status_counts": status_counts,
                    "total_classifications": total_classifications,
                    "database_size_bytes": db_size,
                    "database_path": self.db_path,
                }

        except Exception as e:
            print(f"Error getting database stats: {e}")
            return {}


class SQLiteCompatibilityLayer:
    """
    Compatibility layer for migrating from SQLite to PostgreSQL.

    Provides smooth transition path while maintaining legacy behavior.
    """

    def __init__(self, sqlite_path: str, postgres_storage: Optional[Any] = None):
        """Initialize compatibility layer."""
        self.sqlite_storage = SQLiteLegacyStorage(sqlite_path)
        self.postgres_storage = postgres_storage
        self._migration_mode = postgres_storage is not None

    async def migrate_to_postgres(self, batch_size: int = 1000) -> Dict[str, int]:
        """Migrate data from SQLite to PostgreSQL."""
        if not self.postgres_storage:
            raise ValueError("PostgreSQL storage not configured")

        migration_stats = {
            "alerts_migrated": 0,
            "classifications_migrated": 0,
            "errors": 0,
        }

        try:
            # Migrate alerts in batches
            offset = 0
            while True:
                alerts = await self.sqlite_storage.get_alerts({}, batch_size, offset)
                if not alerts:
                    break

                for alert in alerts:
                    try:
                        await self.postgres_storage.save_alert(alert)
                        migration_stats["alerts_migrated"] += 1
                    except Exception as e:
                        print(f"Error migrating alert {alert.fingerprint}: {e}")
                        migration_stats["errors"] += 1

                offset += batch_size

            return migration_stats

        except Exception as e:
            print(f"Error during migration: {e}")
            migration_stats["errors"] += 1
            return migration_stats
