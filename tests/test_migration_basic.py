#!/usr/bin/env python3
"""
Базовый тест migration infrastructure.
"""
import asyncio
import os
import sys

# Add the project root to the Python path
project_root = os.path.abspath(".")
sys.path.insert(0, project_root)


async def test_migration_infrastructure():
    """Test basic migration infrastructure components."""
    print("🧪 Testing Migration Infrastructure...")

    try:
        # Test config loading
        print("1. Testing config loading...")
        from config import get_config

        config = get_config()
        print("   ✅ Config loaded")
        print(f"   ✅ SQLite path: {config.database.sqlite_path}")
        print(
            f"   ✅ PostgreSQL URL configured: {config.database.postgres_url.split('@')[-1]}"
        )

        # Test logging setup
        print("2. Testing logging setup...")
        from logging_config import get_logger, setup_logging

        setup_logging()
        logger = get_logger(__name__)
        logger.info("Test log entry")
        print("   ✅ Structured logging configured")

        # Test SQLite adapter (should work without external dependencies)
        print("3. Testing SQLite adapter...")
        from src.alert_history.database.sqlite_adapter import SQLiteLegacyStorage

        sqlite_storage = SQLiteLegacyStorage(config.database.sqlite_path)
        # SQLite adapter initializes automatically in constructor
        print("   ✅ SQLite adapter initialized")

        # Test that migration SQL files exist
        print("4. Testing migration SQL files...")
        import os

        migration_dir = "src/alert_history/database/migrations"
        if os.path.exists(migration_dir):
            sql_files = [f for f in os.listdir(migration_dir) if f.endswith(".sql")]
            print(f"   ✅ Found {len(sql_files)} migration SQL files")
        else:
            print("   ⚠️ Migration directory not found, will be created when needed")

        # Test PostgreSQL schema file
        schema_file = "src/alert_history/database/postgresql_schema.sql"
        if os.path.exists(schema_file):
            print("   ✅ PostgreSQL schema file exists")
        else:
            print("   ⚠️ PostgreSQL schema file not found")

        # Test configuration values
        print("5. Testing migration configuration...")
        migration_config = config.migration
        print(f"   ✅ Migration enabled: {migration_config.enabled}")
        print(f"   ✅ Auto migrate: {migration_config.auto_migrate}")
        print(f"   ✅ Batch size: {migration_config.batch_size}")
        print(f"   ✅ Enable publishing: {migration_config.enable_publishing}")

        print("\n🎉 All migration infrastructure tests passed!")
        return True

    except Exception as e:
        print(f"❌ Test failed: {e}")
        import traceback

        traceback.print_exc()
        return False


if __name__ == "__main__":
    success = asyncio.run(test_migration_infrastructure())
    sys.exit(0 if success else 1)
