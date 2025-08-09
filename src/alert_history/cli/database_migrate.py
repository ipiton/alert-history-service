#!/usr/bin/env python3
"""
Database Migration CLI для Alert History Service.

Команды для миграции SQLite → PostgreSQL:
- schema: Применить schema migrations
- data: Мигрировать данные из SQLite в PostgreSQL
- status: Показать статус миграции
- validate: Проверить готовность к миграции
"""
import asyncio
import click
import os
import sys
from pathlib import Path

# Add project root to path
project_root = Path(__file__).parent.parent.parent.parent
sys.path.insert(0, str(project_root))

from src.alert_history.config import get_config
from src.alert_history.database.migration_manager import MigrationManager
from src.alert_history.logging_config import setup_logging, get_logger

# Setup logging
setup_logging()
logger = get_logger(__name__)


@click.group()
def cli():
    """Database Migration CLI for Alert History Service."""
    pass


@cli.command()
@click.option("--dry-run", is_flag=True, help="Show what would be migrated without applying")
async def schema():
    """Apply PostgreSQL schema migrations."""
    try:
        config = get_config()

        if not config.database.postgres_url:
            click.echo(
                "❌ PostgreSQL URL not configured. Set DATABASE_POSTGRES_URL environment variable."
            )
            return

        click.echo("🗄️ Applying PostgreSQL schema migrations...")

        manager = MigrationManager(
            postgresql_url=config.database.postgres_url, sqlite_path=config.database.sqlite_path
        )

        await manager.initialize()

        # Get current version
        current_version = await manager.get_current_schema_version()
        click.echo(f"📋 Current schema version: {current_version or 'None (first run)'}")

        # Get available migrations
        migrations = await manager.get_available_migrations()
        click.echo(f"📦 Available migrations: {len(migrations)}")

        for migration in migrations:
            version_status = (
                "✅ Applied"
                if current_version and migration["version"] <= current_version
                else "⏳ Pending"
            )
            click.echo(f"   {version_status} {migration['version']}: {migration['description']}")

        # Apply migrations
        success = await manager.apply_schema_migrations()

        if success:
            click.echo("✅ Schema migrations applied successfully!")
        else:
            click.echo("❌ Schema migration failed!")

        await manager.close()

    except Exception as e:
        click.echo(f"💥 Schema migration error: {e}")
        logger.error(f"Schema migration failed: {e}")


@cli.command()
@click.option("--batch-size", default=1000, help="Records per batch")
@click.option("--verify", is_flag=True, help="Verify migrated data")
@click.option("--backup", is_flag=True, default=True, help="Create SQLite backup")
async def data(batch_size: int, verify: bool, backup: bool):
    """Migrate data from SQLite to PostgreSQL."""
    try:
        config = get_config()

        if not config.database.postgres_url:
            click.echo(
                "❌ PostgreSQL URL not configured. Set DATABASE_POSTGRES_URL environment variable."
            )
            return

        if not os.path.exists(config.database.sqlite_path):
            click.echo(f"❌ SQLite database not found: {config.database.sqlite_path}")
            return

        click.echo("📦 Starting data migration SQLite → PostgreSQL...")
        click.echo(f"   📊 Batch size: {batch_size}")
        click.echo(f"   ✅ Verification: {'enabled' if verify else 'disabled'}")
        click.echo(f"   💾 Backup: {'enabled' if backup else 'disabled'}")

        manager = MigrationManager(
            postgresql_url=config.database.postgres_url, sqlite_path=config.database.sqlite_path
        )

        await manager.initialize()

        # Create backup if requested
        if backup:
            backup_path = await manager.create_migration_backup()
            if backup_path:
                click.echo(f"💾 Backup created: {backup_path}")
            else:
                click.echo("⚠️ Backup creation failed, continuing...")

        # Start migration
        success = await manager.migrate_data(batch_size=batch_size, verify_data=verify)

        # Show results
        status = manager.get_migration_status()

        click.echo("\n📊 Migration Results:")
        click.echo(f"   📝 Status: {status['status']}")
        click.echo(f"   📄 Alerts: {status['migrated_alerts']}/{status['total_alerts']}")
        click.echo(
            f"   🏷️ Classifications: {status['migrated_classifications']}/{status['total_classifications']}"
        )

        if status.get("duration_seconds"):
            click.echo(f"   ⏱️ Duration: {status['duration_seconds']:.1f}s")

        if status["errors"]:
            click.echo(f"   ❌ Errors: {len(status['errors'])}")
            for error in status["errors"]:
                click.echo(f"      • {error}")

        if success:
            click.echo("✅ Data migration completed successfully!")

            # Cleanup if successful
            await manager.cleanup_after_migration(keep_sqlite_backup=backup)

        else:
            click.echo("❌ Data migration failed!")

        await manager.close()

    except Exception as e:
        click.echo(f"💥 Data migration error: {e}")
        logger.error(f"Data migration failed: {e}")


@cli.command()
async def status():
    """Show migration status."""
    try:
        config = get_config()

        click.echo("📋 Migration Status Report")
        click.echo("=" * 40)

        # Check configuration
        click.echo("\n🔧 Configuration:")
        click.echo(f"   SQLite: {config.database.sqlite_path}")
        click.echo(
            f"   SQLite exists: {'✅' if os.path.exists(config.database.sqlite_path) else '❌'}"
        )
        click.echo(f"   PostgreSQL: {config.database.postgres_url}")
        click.echo(f"   PostgreSQL configured: {'✅' if config.database.postgres_url else '❌'}")

        if not config.database.postgres_url:
            click.echo("\n❌ PostgreSQL not configured. Migration not available.")
            return

        # Check PostgreSQL connection
        try:
            manager = MigrationManager(
                postgresql_url=config.database.postgres_url, sqlite_path=config.database.sqlite_path
            )

            await manager.initialize()

            # Get schema version
            current_version = await manager.get_current_schema_version()
            click.echo(f"\n🗄️ Schema:")
            click.echo(f"   Current version: {current_version or 'None'}")

            # Get available migrations
            migrations = await manager.get_available_migrations()
            pending = [
                m for m in migrations if not current_version or m["version"] > current_version
            ]

            click.echo(f"   Available migrations: {len(migrations)}")
            click.echo(f"   Pending migrations: {len(pending)}")

            # Migration readiness
            click.echo(f"\n🚀 Readiness:")

            ready_checks = [
                ("PostgreSQL connection", True),  # We connected successfully
                ("Schema migrations available", len(migrations) > 0),
                ("SQLite database exists", os.path.exists(config.database.sqlite_path)),
            ]

            all_ready = True
            for check_name, check_result in ready_checks:
                status_icon = "✅" if check_result else "❌"
                click.echo(f"   {status_icon} {check_name}")
                if not check_result:
                    all_ready = False

            if all_ready:
                click.echo("\n🎉 Ready for migration!")
            else:
                click.echo("\n⚠️ Not ready for migration. Fix issues above.")

            await manager.close()

        except Exception as e:
            click.echo(f"\n❌ PostgreSQL connection failed: {e}")

    except Exception as e:
        click.echo(f"💥 Status check error: {e}")
        logger.error(f"Status check failed: {e}")


@cli.command()
async def validate():
    """Validate migration readiness."""
    try:
        config = get_config()

        click.echo("🔍 Validating Migration Readiness...")

        checks = []

        # Check 1: Configuration
        checks.append(("PostgreSQL URL configured", bool(config.database.postgres_url)))
        checks.append(("SQLite path configured", bool(config.database.sqlite_path)))

        # Check 2: Files
        if config.database.sqlite_path:
            checks.append(("SQLite database exists", os.path.exists(config.database.sqlite_path)))

        # Check 3: Migration files
        migrations_dir = Path("src/alert_history/database/migrations")
        checks.append(("Migration directory exists", migrations_dir.exists()))

        if migrations_dir.exists():
            migration_files = list(migrations_dir.glob("*.sql"))
            checks.append(("Migration files available", len(migration_files) > 0))

        # Check 4: PostgreSQL connection (if configured)
        if config.database.postgres_url:
            try:
                manager = MigrationManager(
                    postgresql_url=config.database.postgres_url,
                    sqlite_path=config.database.sqlite_path,
                )
                await manager.initialize()
                checks.append(("PostgreSQL connection", True))
                await manager.close()
            except Exception:
                checks.append(("PostgreSQL connection", False))

        # Display results
        click.echo("\n📋 Validation Results:")

        passed = 0
        for check_name, result in checks:
            status_icon = "✅" if result else "❌"
            click.echo(f"   {status_icon} {check_name}")
            if result:
                passed += 1

        success_rate = passed / len(checks) * 100

        click.echo(f"\n📊 Validation Summary:")
        click.echo(f"   Passed: {passed}/{len(checks)}")
        click.echo(f"   Success Rate: {success_rate:.1f}%")

        if success_rate >= 80:
            click.echo("\n✅ Migration readiness validation PASSED!")
            click.echo("🚀 Ready to proceed with migration.")
        else:
            click.echo("\n❌ Migration readiness validation FAILED!")
            click.echo("🔧 Fix failing checks before proceeding.")

    except Exception as e:
        click.echo(f"💥 Validation error: {e}")
        logger.error(f"Validation failed: {e}")


# Async command wrapper
def async_command(f):
    """Decorator to run async click commands."""

    def wrapper(*args, **kwargs):
        return asyncio.run(f(*args, **kwargs))

    return wrapper


# Apply async wrapper to all commands
for command in [schema, data, status, validate]:
    command.callback = async_command(command.callback)


if __name__ == "__main__":
    cli()
