"""
Health check endpoints для Kubernetes readiness/liveness probes.
"""

from fastapi import APIRouter, HTTPException, status
from fastapi.responses import JSONResponse

from ..logging_config import get_logger
from ..services.health_checker import get_health_checker

logger = get_logger(__name__)

# Health router
health_router = APIRouter(prefix="/health", tags=["health"])


@health_router.get("/")
async def basic_health():
    """Базовый health check - всегда возвращает OK если приложение запущено."""
    return JSONResponse(
        status_code=status.HTTP_200_OK,
        content={
            "status": "healthy",
            "message": "Application is running",
            "service": "alert-history",
        },
    )


@health_router.get("/liveness")
async def liveness_probe():
    """
    Kubernetes liveness probe.
    Проверяет что приложение запущено и не зависло.
    Должен быть быстрым и простым.
    """
    try:
        import time

        return JSONResponse(
            status_code=status.HTTP_200_OK,
            content={
                "status": "alive",
                "timestamp": time.time(),
                "service": "alert-history",
                "message": "Application is alive",
            },
        )
    except Exception as e:
        logger.error("Liveness probe failed", error=str(e))
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Liveness check failed",
        )


@health_router.get("/readiness")
async def readiness_probe():
    """
    Kubernetes readiness probe.
    Проверяет что приложение готово принимать трафик.
    Проверяет внешние зависимости (БД, Redis).
    """
    try:
        health_checker = get_health_checker()
        health_result = await health_checker.check_overall_health()

        if health_result["ready_for_traffic"]:
            return JSONResponse(
                status_code=status.HTTP_200_OK,
                content={
                    "status": "ready",
                    "ready_for_traffic": True,
                    "timestamp": health_result["timestamp"],
                    "response_time_ms": health_result["response_time_ms"],
                    "components": {
                        comp["name"]: comp["status"]
                        for comp in health_result["components"]
                    },
                },
            )
        else:
            # Service not ready for traffic
            return JSONResponse(
                status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
                content={
                    "status": "not_ready",
                    "ready_for_traffic": False,
                    "timestamp": health_result["timestamp"],
                    "components": {
                        comp["name"]: comp["status"]
                        for comp in health_result["components"]
                    },
                    "message": "Service dependencies not ready",
                },
            )

    except Exception as e:
        logger.error("Readiness probe failed", error=str(e))
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail="Readiness check failed",
        )


@health_router.get("/detailed")
async def detailed_health():
    """
    Детальная проверка состояния всех компонентов.
    Для мониторинга и диагностики.
    """
    try:
        health_checker = get_health_checker()
        health_result = await health_checker.check_overall_health()

        # Определяем HTTP статус код на основе общего статуса
        if health_result["status"] == "healthy":
            status_code = status.HTTP_200_OK
        elif health_result["status"] == "warning":
            status_code = status.HTTP_200_OK  # Warning не критично
        else:
            status_code = status.HTTP_503_SERVICE_UNAVAILABLE

        return JSONResponse(status_code=status_code, content=health_result)

    except Exception as e:
        logger.error("Detailed health check failed", error=str(e))
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Health check failed",
        )


@health_router.get("/database")
async def database_health():
    """Проверка состояния базы данных."""
    try:
        health_checker = get_health_checker()
        db_health = await health_checker.check_database_health()

        status_code = (
            status.HTTP_200_OK
            if db_health["status"] == "healthy"
            else status.HTTP_503_SERVICE_UNAVAILABLE
        )

        return JSONResponse(status_code=status_code, content=db_health)

    except Exception as e:
        logger.error("Database health check failed", error=str(e))
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Database health check failed",
        )


@health_router.get("/cache")
async def cache_health():
    """Проверка состояния кеша (Redis/KeyDB)."""
    try:
        health_checker = get_health_checker()
        cache_health = await health_checker.check_redis_health()

        status_code = (
            status.HTTP_200_OK
            if cache_health["status"] == "healthy"
            else status.HTTP_503_SERVICE_UNAVAILABLE
        )

        return JSONResponse(status_code=status_code, content=cache_health)

    except Exception as e:
        logger.error("Cache health check failed", error=str(e))
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Cache health check failed",
        )


@health_router.get("/llm")
async def llm_health():
    """Проверка состояния LLM proxy."""
    try:
        health_checker = get_health_checker()
        llm_health = await health_checker.check_llm_proxy_health()

        status_code = (
            status.HTTP_200_OK
            if llm_health["status"] in ["healthy", "disabled"]
            else status.HTTP_503_SERVICE_UNAVAILABLE
        )

        return JSONResponse(status_code=status_code, content=llm_health)

    except Exception as e:
        logger.error("LLM health check failed", error=str(e))
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="LLM health check failed",
        )
