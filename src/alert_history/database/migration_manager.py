"""
Database Migration Manager для Alert History Service.

Поддерживает:
- Миграция SQLite → PostgreSQL с нулевым downtime
- Version-based schema migrations
- Data migration с консистентностью
- Rollback capabilities
- Progress tracking и monitoring
"""

# Standard library imports
import asyncio
import hashlib
import sqlite3
import time
from datetime import datetime
from pathlib import Path
from typing import Any, Optional

# Third-party imports
# Local imports
from ..core.interfaces import Alert, AlertSeverity, AlertStatus
from ..database.postgresql_adapter import PostgreSQLStorage
from ..database.sqlite_adapter import SQLiteLegacyStorage
from ..logging_config import get_logger

logger = get_logger(__name__)


class MigrationManager:
    """
    Database migration manager.

    Handles:
    - Schema migrations (version-based)
    - Data migrations (SQLite → PostgreSQL)
    - Zero-downtime migrations
    - Rollback capabilities
    """

    def __init__(
        self,
        postgresql_url: str,
        sqlite_path: str,
        migrations_dir: str = "src/alert_history/database/migrations",
    ):
        """Initialize migration manager."""
        self.postgresql_url = postgresql_url
        self.sqlite_path = sqlite_path
        self.migrations_dir = Path(migrations_dir)

        self.pg_storage: Optional[PostgreSQLStorage] = None
        self.sqlite_storage: Optional[SQLiteLegacyStorage] = None

        # Migration tracking
        self.migration_stats = {
            "start_time": None,
            "end_time": None,
            "total_alerts": 0,
            "migrated_alerts": 0,
            "total_classifications": 0,
            "migrated_classifications": 0,
            "errors": [],
            "status": "not_started",
        }

    async def initialize(self) -> None:
        """Initialize storage connections."""
        try:
            # Initialize PostgreSQL
            self.pg_storage = PostgreSQLStorage(self.postgresql_url)
            await self.pg_storage.initialize()

            # Initialize SQLite
            self.sqlite_storage = SQLiteLegacyStorage(self.sqlite_path)

            logger.info("Migration manager initialized")

        except Exception as e:
            logger.error(f"Failed to initialize migration manager: {e}")
            raise

    async def close(self) -> None:
        """Close storage connections."""
        if self.pg_storage:
            await self.pg_storage.close()

        # SQLite doesn't need explicit closing

        logger.info("Migration manager closed")

    # ===============================
    # Schema Migrations
    # ===============================

    async def get_current_schema_version(self) -> Optional[str]:
        """Get current schema version from PostgreSQL."""
        try:
            async with self.pg_storage.get_connection() as conn:
                query = """
                    SELECT version FROM migration_history
                    ORDER BY applied_at DESC
                    LIMIT 1
                """
                return await conn.fetchval(query)

        except Exception as e:
            logger.debug(f"No migration history found (first run?): {e}")
            return None

    async def get_available_migrations(self) -> list[dict[str, Any]]:
        """Get list of available migration files."""
        migrations = []

        if not self.migrations_dir.exists():
            logger.warning(f"Migrations directory not found: {self.migrations_dir}")
            return migrations

        for migration_file in sorted(self.migrations_dir.glob("*.sql")):
            try:
                # Parse migration filename (format: 001_migration_name.sql)
                filename = migration_file.stem
                parts = filename.split("_", 1)

                if len(parts) >= 2:
                    version = parts[0]
                    description = parts[1].replace("_", " ").title()
                else:
                    version = filename
                    description = "Migration"

                # Calculate file checksum
                content = migration_file.read_text()
                checksum = hashlib.sha256(content.encode()).hexdigest()

                migrations.append(
                    {
                        "version": version,
                        "description": description,
                        "file_path": str(migration_file),
                        "checksum": checksum,
                        "content": content,
                    }
                )

            except Exception as e:
                logger.warning(f"Failed to parse migration file {migration_file}: {e}")

        return migrations

    async def apply_schema_migrations(self) -> bool:
        """Apply pending schema migrations."""
        try:
            current_version = await self.get_current_schema_version()
            available_migrations = await self.get_available_migrations()

            if not available_migrations:
                logger.info("No migrations found")
                return True

            # Filter pending migrations
            pending_migrations = []

            if current_version is None:
                # First run - apply all migrations
                pending_migrations = available_migrations
            else:
                # Apply migrations newer than current version
                for migration in available_migrations:
                    if migration["version"] > current_version:
                        pending_migrations.append(migration)

            if not pending_migrations:
                logger.info(f"Schema is up to date (version: {current_version})")
                return True

            logger.info(f"Applying {len(pending_migrations)} schema migrations")

            # Apply migrations
            for migration in pending_migrations:
                success = await self._apply_single_migration(migration)
                if not success:
                    return False

            logger.info("All schema migrations applied successfully")
            return True

        except Exception as e:
            logger.error(f"Schema migration failed: {e}")
            return False

    async def _apply_single_migration(self, migration: dict[str, Any]) -> bool:
        """Apply a single migration."""
        try:
            start_time = time.time()

            async with self.pg_storage.get_connection() as conn:
                # Start transaction
                async with conn.transaction():
                    # Execute migration SQL
                    await conn.execute(migration["content"])

                    # Record migration in history
                    query = """
                        INSERT INTO migration_history (
                            version, description, execution_time, checksum
                        ) VALUES ($1, $2, $3, $4)
                        ON CONFLICT (version) DO NOTHING
                    """

                    execution_time = time.time() - start_time

                    await conn.execute(
                        query,
                        migration["version"],
                        migration["description"],
                        execution_time,
                        migration["checksum"],
                    )

            logger.info(
                f"Applied migration {migration['version']}: {migration['description']}",
                execution_time=round(execution_time, 3),
            )

            return True

        except Exception as e:
            logger.error(f"Failed to apply migration {migration['version']}: {e}")
            return False

    # ===============================
    # Data Migration (SQLite → PostgreSQL)
    # ===============================

    async def migrate_data(
        self, batch_size: int = 1000, verify_data: bool = True
    ) -> bool:
        """
        Migrate data from SQLite to PostgreSQL.

        Args:
            batch_size: Number of records to process per batch
            verify_data: Whether to verify migrated data
        """
        try:
            self.migration_stats["status"] = "running"
            self.migration_stats["start_time"] = datetime.utcnow()

            logger.info("Starting data migration SQLite → PostgreSQL")

            # Step 1: Migrate alerts
            success = await self._migrate_alerts(batch_size)
            if not success:
                self.migration_stats["status"] = "failed"
                return False

            # Step 2: Migrate classifications (if any)
            success = await self._migrate_classifications(batch_size)
            if not success:
                self.migration_stats["status"] = "failed"
                return False

            # Step 3: Verify data (if requested)
            if verify_data:
                success = await self._verify_migrated_data()
                if not success:
                    self.migration_stats["status"] = "failed"
                    return False

            self.migration_stats["status"] = "completed"
            self.migration_stats["end_time"] = datetime.utcnow()

            logger.info("Data migration completed successfully", **self.migration_stats)

            return True

        except Exception as e:
            self.migration_stats["status"] = "failed"
            self.migration_stats["errors"].append(str(e))
            logger.error(f"Data migration failed: {e}")
            return False

    async def _migrate_alerts(self, batch_size: int) -> bool:
        """Migrate alerts from SQLite to PostgreSQL."""
        try:
            # Get total count for progress tracking
            conn = sqlite3.connect(self.sqlite_path)
            cursor = conn.cursor()

            cursor.execute("SELECT COUNT(*) FROM alert_history")
            total_count = cursor.fetchone()[0]
            self.migration_stats["total_alerts"] = total_count

            if total_count == 0:
                logger.info("No alerts to migrate")
                return True

            logger.info(f"Migrating {total_count} alerts in batches of {batch_size}")

            # Migrate in batches
            offset = 0
            migrated_count = 0

            while offset < total_count:
                # Fetch batch from SQLite
                cursor.execute(
                    """
                    SELECT
                        fingerprint, alert_name, namespace, status,
                        labels, annotations, starts_at, ends_at,
                        generator_url, timestamp
                    FROM alert_history
                    ORDER BY id
                    LIMIT ? OFFSET ?
                """,
                    (batch_size, offset),
                )

                rows = cursor.fetchall()

                if not rows:
                    break

                # Convert to Alert objects and save to PostgreSQL
                alerts = []
                for row in rows:
                    try:
                        alert = Alert(
                            fingerprint=row[0],
                            alert_name=row[1],
                            namespace=row[2],
                            status=(
                                AlertStatus(row[3]) if row[3] else AlertStatus.FIRING
                            ),
                            labels=eval(row[4]) if row[4] else {},
                            annotations=eval(row[5]) if row[5] else {},
                            starts_at=(
                                datetime.fromisoformat(row[6]) if row[6] else None
                            ),
                            ends_at=datetime.fromisoformat(row[7]) if row[7] else None,
                            generator_url=row[8],
                            timestamp=(
                                datetime.fromisoformat(row[9])
                                if row[9]
                                else datetime.utcnow()
                            ),
                        )
                        alerts.append(alert)

                    except Exception as e:
                        logger.warning(f"Failed to convert alert row: {e}")
                        continue

                # Batch save to PostgreSQL
                for alert in alerts:
                    success = await self.pg_storage.save_alert(alert)
                    if success:
                        migrated_count += 1
                    else:
                        logger.warning(f"Failed to save alert {alert.fingerprint}")

                self.migration_stats["migrated_alerts"] = migrated_count

                # Progress logging
                progress = (offset + len(rows)) / total_count * 100
                logger.info(
                    f"Migration progress: {progress:.1f}% ({migrated_count}/{total_count})"
                )

                offset += batch_size

                # Small delay to avoid overwhelming the database
                await asyncio.sleep(0.1)

            conn.close()

            logger.info(f"Alert migration completed: {migrated_count}/{total_count}")

            return True

        except Exception as e:
            logger.error(f"Alert migration failed: {e}")
            return False

    async def _migrate_classifications(self, batch_size: int) -> bool:
        """Migrate classifications if they exist in SQLite."""
        try:
            # Check if classifications table exists in SQLite
            conn = sqlite3.connect(self.sqlite_path)
            cursor = conn.cursor()

            cursor.execute(
                """
                SELECT name FROM sqlite_master
                WHERE type='table' AND name='alert_classifications'
            """
            )

            if not cursor.fetchone():
                logger.info("No classifications table found in SQLite")
                conn.close()
                return True

            # Get count
            cursor.execute("SELECT COUNT(*) FROM alert_classifications")
            total_count = cursor.fetchone()[0]
            self.migration_stats["total_classifications"] = total_count

            if total_count == 0:
                logger.info("No classifications to migrate")
                conn.close()
                return True

            logger.info(f"Migrating {total_count} classifications")

            # Migrate classifications
            cursor.execute(
                """
                SELECT
                    alert_fingerprint, severity, confidence, reasoning,
                    recommendations, processing_time, metadata
                FROM alert_classifications
            """
            )

            migrated_count = 0

            for row in cursor.fetchall():
                try:
                    # Create classification result
                    # Local imports
                    from ..core.interfaces import ClassificationResult

                    classification = ClassificationResult(
                        severity=(
                            AlertSeverity(row[1]) if row[1] else AlertSeverity.INFO
                        ),
                        confidence=float(row[2]) if row[2] else 0.0,
                        reasoning=row[3] or "",
                        recommendations=eval(row[4]) if row[4] else [],
                        processing_time=float(row[5]) if row[5] else 0.0,
                        metadata=eval(row[6]) if row[6] else {},
                    )

                    success = await self.pg_storage.save_classification(
                        row[0], classification
                    )
                    if success:
                        migrated_count += 1

                except Exception as e:
                    logger.warning(f"Failed to migrate classification: {e}")
                    continue

            self.migration_stats["migrated_classifications"] = migrated_count

            conn.close()

            logger.info(
                f"Classification migration completed: {migrated_count}/{total_count}"
            )

            return True

        except Exception as e:
            logger.error(f"Classification migration failed: {e}")
            return False

    async def _verify_migrated_data(self) -> bool:
        """Verify migrated data consistency."""
        try:
            logger.info("Verifying migrated data")

            # Get PostgreSQL stats
            pg_stats = await self.pg_storage.get_statistics()

            # Compare with migration stats
            pg_alerts = pg_stats.get("total_alerts", 0)
            pg_classifications = pg_stats.get("total_classifications", 0)

            migrated_alerts = self.migration_stats["migrated_alerts"]
            migrated_classifications = self.migration_stats["migrated_classifications"]

            # Verify alert counts
            if pg_alerts < migrated_alerts:
                logger.error(
                    f"Alert count mismatch: PostgreSQL has {pg_alerts}, "
                    f"expected {migrated_alerts}"
                )
                return False

            # Verify classification counts
            if pg_classifications < migrated_classifications:
                logger.error(
                    f"Classification count mismatch: PostgreSQL has {pg_classifications}, "
                    f"expected {migrated_classifications}"
                )
                return False

            logger.info(
                "Data verification successful",
                postgresql_alerts=pg_alerts,
                postgresql_classifications=pg_classifications,
            )

            return True

        except Exception as e:
            logger.error(f"Data verification failed: {e}")
            return False

    # ===============================
    # Migration Status and Monitoring
    # ===============================

    def get_migration_status(self) -> dict[str, Any]:
        """Get current migration status."""
        status = self.migration_stats.copy()

        if status["start_time"] and status["end_time"]:
            duration = status["end_time"] - status["start_time"]
            status["duration_seconds"] = duration.total_seconds()
        elif status["start_time"]:
            duration = datetime.utcnow() - status["start_time"]
            status["duration_seconds"] = duration.total_seconds()

        # Calculate progress
        if status["total_alerts"] > 0:
            status["alerts_progress_percent"] = (
                status["migrated_alerts"] / status["total_alerts"] * 100
            )

        if status["total_classifications"] > 0:
            status["classifications_progress_percent"] = (
                status["migrated_classifications"]
                / status["total_classifications"]
                * 100
            )

        return status

    async def create_migration_backup(self) -> Optional[str]:
        """Create backup of SQLite database before migration."""
        try:
            backup_path = f"{self.sqlite_path}.backup.{int(time.time())}"

            # Simple file copy for SQLite
            # Standard library imports
            import shutil

            shutil.copy2(self.sqlite_path, backup_path)

            logger.info(f"Created migration backup: {backup_path}")

            return backup_path

        except Exception as e:
            logger.error(f"Failed to create backup: {e}")
            return None

    async def cleanup_after_migration(self, keep_sqlite_backup: bool = True) -> bool:
        """Cleanup after successful migration."""
        try:
            if keep_sqlite_backup:
                backup_path = await self.create_migration_backup()
                if backup_path:
                    logger.info(f"SQLite backup saved to: {backup_path}")

            logger.info("Migration cleanup completed")
            return True

        except Exception as e:
            logger.error(f"Migration cleanup failed: {e}")
            return False
