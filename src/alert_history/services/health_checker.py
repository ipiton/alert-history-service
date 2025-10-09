"""
Health checker для проверки состояния всех внешних зависимостей.
Необходим для Kubernetes readiness/liveness probes.
"""

import asyncio
import time
from datetime import datetime

import aioredis
import asyncpg

from config import get_config

from ..logging_config import get_logger

logger = get_logger(__name__)


class HealthChecker:
    """Проверка состояния всех компонентов приложения."""

    def __init__(self):
        self.config = get_config()

    async def check_database_health(self) -> dict[str, any]:
        """Проверка состояния базы данных."""
        start_time = time.time()
        result = {
            "name": "database",
            "status": "unknown",
            "response_time_ms": 0,
            "details": {},
            "timestamp": datetime.utcnow().isoformat(),
        }

        try:
            if (
                self.config.database.database_url
                and self.config.database.database_url.startswith("postgresql")
            ):
                # PostgreSQL health check
                conn = await asyncpg.connect(self.config.database.postgres_url)
                try:
                    # Simple query to check connectivity
                    await conn.execute("SELECT 1")
                    result["status"] = "healthy"
                    result["details"] = {
                        "type": "postgresql",
                        "host": self.config.database.postgres_host,
                        "port": self.config.database.postgres_port,
                        "database": self.config.database.postgres_database,
                    }
                finally:
                    await conn.close()
            else:
                # SQLite health check (always healthy if file exists)
                import os

                sqlite_path = self.config.database.sqlite_path
                if os.path.exists(sqlite_path):
                    result["status"] = "healthy"
                    result["details"] = {
                        "type": "sqlite",
                        "path": sqlite_path,
                        "size_bytes": os.path.getsize(sqlite_path),
                    }
                else:
                    result["status"] = "warning"
                    result["details"] = {
                        "type": "sqlite",
                        "path": sqlite_path,
                        "message": "Database file does not exist",
                    }

        except Exception as e:
            result["status"] = "unhealthy"
            result["details"] = {"error": str(e)}
            logger.error("Database health check failed", error=str(e))

        result["response_time_ms"] = round((time.time() - start_time) * 1000, 2)
        return result

    async def check_redis_health(self) -> dict[str, any]:
        """Проверка состояния Redis/KeyDB."""
        start_time = time.time()
        result = {
            "name": "cache",
            "status": "unknown",
            "response_time_ms": 0,
            "details": {},
            "timestamp": datetime.utcnow().isoformat(),
        }

        try:
            redis = aioredis.from_url(
                self.config.redis.redis_url,
                max_connections=1,
                retry_on_timeout=True,
                health_check_interval=30,
            )

            # Test connection with ping
            await redis.ping()

            # Test basic operations
            test_key = f"health_check_{int(time.time())}"
            await redis.set(test_key, "test", ex=5)  # 5 second expiry
            test_value = await redis.get(test_key)
            await redis.delete(test_key)

            if test_value == b"test":
                result["status"] = "healthy"
                result["details"] = {
                    "type": "redis",
                    "host": self.config.redis.host,
                    "port": self.config.redis.port,
                    "database": self.config.redis.database,
                }
            else:
                result["status"] = "warning"
                result["details"] = {
                    "message": "Redis responding but read/write test failed"
                }

            await redis.close()

        except Exception as e:
            result["status"] = "unhealthy"
            result["details"] = {"error": str(e)}
            logger.error("Redis health check failed", error=str(e))

        result["response_time_ms"] = round((time.time() - start_time) * 1000, 2)
        return result

    async def check_llm_proxy_health(self) -> dict[str, any]:
        """Проверка состояния LLM proxy (если настроен)."""
        start_time = time.time()
        result = {
            "name": "llm_proxy",
            "status": "unknown",
            "response_time_ms": 0,
            "details": {},
            "timestamp": datetime.utcnow().isoformat(),
        }

        try:
            if not self.config.llm or not self.config.llm.proxy_url:
                result["status"] = "disabled"
                result["details"] = {"message": "LLM proxy not configured"}
                return result

            import aiohttp

            async with aiohttp.ClientSession() as session:
                # Test LLM proxy endpoint
                async with session.get(
                    f"{self.config.llm.proxy_url}/health",
                    timeout=aiohttp.ClientTimeout(total=5),
                ) as response:
                    if response.status == 200:
                        result["status"] = "healthy"
                        result["details"] = {
                            "proxy_url": self.config.llm.proxy_url,
                            "response_status": response.status,
                        }
                    else:
                        result["status"] = "warning"
                        result["details"] = {
                            "proxy_url": self.config.llm.proxy_url,
                            "response_status": response.status,
                        }

        except Exception as e:
            result["status"] = "unhealthy"
            result["details"] = {"error": str(e)}
            logger.warning("LLM proxy health check failed", error=str(e))

        result["response_time_ms"] = round((time.time() - start_time) * 1000, 2)
        return result

    async def check_overall_health(self) -> dict[str, any]:
        """Полная проверка состояния всех компонентов."""
        start_time = time.time()

        # Параллельная проверка всех компонентов
        checks = await asyncio.gather(
            self.check_database_health(),
            self.check_redis_health(),
            self.check_llm_proxy_health(),
            return_exceptions=True,
        )

        # Фильтруем исключения
        health_checks = []
        for check in checks:
            if isinstance(check, Exception):
                logger.error("Health check failed with exception", error=str(check))
                health_checks.append(
                    {
                        "name": "unknown",
                        "status": "error",
                        "details": {"error": str(check)},
                        "timestamp": datetime.utcnow().isoformat(),
                    }
                )
            else:
                health_checks.append(check)

        # Определяем общий статус
        statuses = [check["status"] for check in health_checks]
        if "unhealthy" in statuses:
            overall_status = "unhealthy"
        elif "warning" in statuses:
            overall_status = "warning"
        elif all(status in ["healthy", "disabled"] for status in statuses):
            overall_status = "healthy"
        else:
            overall_status = "unknown"

        # Считаем готовность для Kubernetes
        # Для readiness нужны healthy database и cache (если включен)
        ready_for_traffic = True
        required_services = ["database"]

        if self.config.redis:
            required_services.append("cache")

        for check in health_checks:
            if check["name"] in required_services and check["status"] != "healthy":
                ready_for_traffic = False
                break

        result = {
            "status": overall_status,
            "ready_for_traffic": ready_for_traffic,
            "timestamp": datetime.utcnow().isoformat(),
            "response_time_ms": round((time.time() - start_time) * 1000, 2),
            "version": "1.0",
            "application": "alert-history-service",
            "components": health_checks,
            "summary": {
                "healthy": len([c for c in health_checks if c["status"] == "healthy"]),
                "warning": len([c for c in health_checks if c["status"] == "warning"]),
                "unhealthy": len(
                    [c for c in health_checks if c["status"] == "unhealthy"]
                ),
                "disabled": len(
                    [c for c in health_checks if c["status"] == "disabled"]
                ),
                "total": len(health_checks),
            },
        }

        return result


# Singleton instance для использования в роутерах
_health_checker = None


def get_health_checker() -> HealthChecker:
    """Получить singleton instance health checker."""
    global _health_checker
    if _health_checker is None:
        _health_checker = HealthChecker()
    return _health_checker
