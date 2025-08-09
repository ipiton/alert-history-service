"""
API endpoints для работы с классификацией алертов.

Новые эндпоинты для LLM функциональности:
- GET /classification/{fingerprint} - получить классификацию
- POST /classification/bulk - массовая классификация
- GET /classification/stats - статистика классификации
- POST /classification/refresh/{fingerprint} - принудительное обновление
"""

# Standard library imports
from typing import List, Optional

# Third-party imports
from fastapi import APIRouter, Depends, HTTPException, Request
from pydantic import BaseModel, Field

# Local imports
from ..core.interfaces import ClassificationResult
from ..logging_config import get_performance_logger
from ..services.alert_classifier import AlertClassificationService

performance_logger = get_performance_logger()
router = APIRouter(prefix="/classification", tags=["classification"])


# Request/Response models
class ClassificationResponse(BaseModel):
    """Response model для классификации."""

    fingerprint: str
    severity: str
    confidence: float
    reasoning: str
    recommendations: List[str]
    processing_time: float
    metadata: Optional[dict] = None
    cached: bool = False

    @classmethod
    def from_classification_result(
        cls, fingerprint: str, result: ClassificationResult, cached: bool = False
    ) -> "ClassificationResponse":
        """Создать response из ClassificationResult."""
        return cls(
            fingerprint=fingerprint,
            severity=result.severity.value,
            confidence=result.confidence,
            reasoning=result.reasoning,
            recommendations=result.recommendations,
            processing_time=result.processing_time,
            metadata=result.metadata,
            cached=cached,
        )


class BulkClassificationRequest(BaseModel):
    """Request model для массовой классификации."""

    fingerprints: List[str] = Field(..., min_items=1, max_items=50)
    force_refresh: bool = False
    include_recommendations: bool = True


class BulkClassificationResponse(BaseModel):
    """Response model для массовой классификации."""

    results: List[ClassificationResponse]
    total_processed: int
    errors: List[dict]
    processing_time: float


class ClassificationStatsResponse(BaseModel):
    """Response model для статистики классификации."""

    total_requests: int
    cache_hits: int
    llm_requests: int
    fallback_used: int
    errors: int
    cache_hit_rate: float
    llm_usage_rate: float
    fallback_rate: float
    error_rate: float


# Dependency injection для сервиса классификации
def get_classification_service(request) -> AlertClassificationService:
    """
    Dependency для получения сервиса классификации.

    Получает сервис из app.state, инициализированного в main.py
    """
    if not hasattr(request.app.state, "classification_service"):
        raise HTTPException(
            status_code=503, detail="Classification service not available"
        )

    service = request.app.state.classification_service
    if service is None:
        raise HTTPException(
            status_code=503,
            detail="Classification service not initialized (LLM disabled or configuration missing)",
        )

    return service


@router.get(
    "/{fingerprint}",
    response_model=ClassificationResponse,
    summary="Получить классификацию алерта",
    description="""
    Получить сохраненную классификацию алерта по его fingerprint.

    Сначала проверяет кэш, затем database. Если классификации нет -
    возвращает 404.
    """,
)
async def get_classification(
    fingerprint: str,
    request: Request,
    classification_service: AlertClassificationService = Depends(
        get_classification_service
    ),
) -> ClassificationResponse:
    """Получить классификацию алерта по fingerprint."""

    start_time = performance_logger._get_current_time()

    try:
        # Получаем классификацию из сервиса
        result = await classification_service.get_classification_history(fingerprint)

        if not result:
            raise HTTPException(
                status_code=404,
                detail=f"Classification not found for fingerprint: {fingerprint}",
            )

        response = ClassificationResponse.from_classification_result(
            fingerprint=fingerprint,
            result=result,
            cached=True,  # История всегда считается кэшированной
        )

        performance_logger.request_completed(
            method="GET",
            path=f"/classification/{fingerprint}",
            status_code=200,
            processing_time=performance_logger._get_current_time() - start_time,
        )

        return response

    except HTTPException:
        raise
    except Exception as e:
        performance_logger.request_completed(
            method="GET",
            path=f"/classification/{fingerprint}",
            status_code=500,
            processing_time=performance_logger._get_current_time() - start_time,
        )
        raise HTTPException(status_code=500, detail=str(e))


@router.post(
    "/refresh/{fingerprint}",
    response_model=ClassificationResponse,
    summary="Принудительно обновить классификацию",
    description="""
    Принудительно перекласифицировать алерт через LLM, игнорируя кэш.

    Требует наличия алерта в базе данных. Полезно для тестирования
    новых промптов или переклассификации после изменений.
    """,
)
async def refresh_classification(
    fingerprint: str,
    classification_service: AlertClassificationService = Depends(
        get_classification_service
    ),
) -> ClassificationResponse:
    """Принудительно обновить классификацию алерта."""

    start_time = performance_logger._get_current_time()

    try:
        # Сначала получаем алерт из storage
        storage = classification_service.storage
        alert = await storage.get_alert_by_fingerprint(fingerprint)

        if not alert:
            raise HTTPException(
                status_code=404,
                detail=f"Alert not found for fingerprint: {fingerprint}",
            )

        # Принудительная классификация
        result = await classification_service.classify_alert(
            alert=alert, force_refresh=True
        )

        response = ClassificationResponse.from_classification_result(
            fingerprint=fingerprint, result=result, cached=False
        )

        performance_logger.request_completed(
            method="POST",
            path=f"/classification/refresh/{fingerprint}",
            status_code=200,
            processing_time=performance_logger._get_current_time() - start_time,
        )

        return response

    except HTTPException:
        raise
    except Exception as e:
        performance_logger.request_completed(
            method="POST",
            path=f"/classification/refresh/{fingerprint}",
            status_code=500,
            processing_time=performance_logger._get_current_time() - start_time,
        )
        raise HTTPException(status_code=500, detail=str(e))


@router.post(
    "/bulk",
    response_model=BulkClassificationResponse,
    summary="Массовая классификация алертов",
    description="""
    Классифицировать несколько алертов одновременно.

    Максимум 50 алертов за раз. Поддерживает принудительное обновление
    и включение/отключение рекомендаций.
    """,
)
async def bulk_classification(
    request: BulkClassificationRequest,
    classification_service: AlertClassificationService = Depends(
        get_classification_service
    ),
) -> BulkClassificationResponse:
    """Массовая классификация алертов."""

    start_time = performance_logger._get_current_time()

    try:
        results = []
        errors = []

        for fingerprint in request.fingerprints:
            try:
                # Получаем алерт
                alert = await classification_service.storage.get_alert_by_fingerprint(
                    fingerprint
                )

                if not alert:
                    errors.append(
                        {"fingerprint": fingerprint, "error": "Alert not found"}
                    )
                    continue

                # Классификация
                if request.force_refresh:
                    result = await classification_service.classify_alert(
                        alert=alert, force_refresh=True
                    )
                    cached = False
                else:
                    # Сначала пробуем из истории
                    result = await classification_service.get_classification_history(
                        fingerprint
                    )
                    cached = True

                    # Если нет - делаем новую классификацию
                    if not result:
                        result = await classification_service.classify_alert(
                            alert=alert
                        )
                        cached = False

                # Убираем рекомендации если не нужны
                if not request.include_recommendations:
                    result = ClassificationResult(
                        severity=result.severity,
                        confidence=result.confidence,
                        reasoning=result.reasoning,
                        recommendations=[],
                        processing_time=result.processing_time,
                        metadata=result.metadata,
                    )

                response = ClassificationResponse.from_classification_result(
                    fingerprint=fingerprint, result=result, cached=cached
                )
                results.append(response)

            except Exception as e:
                errors.append({"fingerprint": fingerprint, "error": str(e)})

        total_time = performance_logger._get_current_time() - start_time

        bulk_response = BulkClassificationResponse(
            results=results,
            total_processed=len(results),
            errors=errors,
            processing_time=total_time,
        )

        performance_logger.request_completed(
            method="POST",
            path="/classification/bulk",
            status_code=200,
            processing_time=total_time,
        )

        return bulk_response

    except Exception as e:
        performance_logger.request_completed(
            method="POST",
            path="/classification/bulk",
            status_code=500,
            processing_time=performance_logger._get_current_time() - start_time,
        )
        raise HTTPException(status_code=500, detail=str(e))


@router.get(
    "/stats",
    response_model=ClassificationStatsResponse,
    summary="Статистика классификации",
    description="""
    Получить статистику работы системы классификации.

    Включает метрики cache hit rate, использования LLM,
    частоты fallback и ошибок.
    """,
)
async def get_classification_stats(
    classification_service: AlertClassificationService = Depends(
        get_classification_service
    ),
) -> ClassificationStatsResponse:
    """Получить статистику классификации."""

    try:
        stats = await classification_service.get_classification_stats()

        return ClassificationStatsResponse(
            total_requests=stats.get("total_requests", 0),
            cache_hits=stats.get("cache_hits", 0),
            llm_requests=stats.get("llm_requests", 0),
            fallback_used=stats.get("fallback_used", 0),
            errors=stats.get("errors", 0),
            cache_hit_rate=stats.get("cache_hit_rate", 0.0),
            llm_usage_rate=stats.get("llm_usage_rate", 0.0),
            fallback_rate=stats.get("fallback_rate", 0.0),
            error_rate=stats.get("error_rate", 0.0),
        )

    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get(
    "/health",
    summary="Проверка здоровья классификатора",
    description="Проверить доступность LLM сервиса и кэша",
)
async def classification_health(
    classification_service: AlertClassificationService = Depends(
        get_classification_service
    ),
) -> dict:
    """Проверка здоровья сервиса классификации."""

    health_status = {
        "status": "healthy",
        "components": {},
        "timestamp": performance_logger._get_current_time(),
    }

    try:
        # Проверяем LLM клиент
        llm_healthy = True
        try:
            # Проверяем доступность LLM (если есть метод health check)
            if hasattr(classification_service.llm_client, "health_check"):
                await classification_service.llm_client.health_check()
        except Exception:
            llm_healthy = False

        health_status["components"]["llm_client"] = {
            "status": "healthy" if llm_healthy else "unhealthy"
        }

        # Проверяем кэш
        cache_healthy = True
        if classification_service.cache:
            try:
                # Простая проверка кэша
                test_key = "health_check_test"
                await classification_service.cache.set(test_key, "test", 1)
                await classification_service.cache.get(test_key)
                await classification_service.cache.delete(test_key)
            except Exception:
                cache_healthy = False

        health_status["components"]["cache"] = {
            "status": "healthy" if cache_healthy else "unhealthy",
            "enabled": classification_service.cache is not None,
        }

        # Общий статус
        if not llm_healthy or (classification_service.cache and not cache_healthy):
            health_status["status"] = "degraded"

        return health_status

    except Exception as e:
        return {
            "status": "unhealthy",
            "error": str(e),
            "timestamp": performance_logger._get_current_time(),
        }
