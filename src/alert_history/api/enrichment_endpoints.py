"""
Enrichment Mode API endpoints.

Позволяет читать/устанавливать режим обработки алертов:
- transparent: проксирование без LLM-обогащения
- enriched: с LLM-классификацией/обогащением

Хранение текущего режима в Redis (fallback: in-memory app_state).
"""

from typing import Literal, Optional

from fastapi import APIRouter, HTTPException
from pydantic import BaseModel

from ..logging_config import get_logger
from ..core.app_state import app_state

logger = get_logger(__name__)


router = APIRouter(prefix="/enrichment", tags=["Enrichment Mode"])


class EnrichmentModeResponse(BaseModel):
    mode: Literal["transparent", "enriched"]
    source: str


class EnrichmentModeRequest(BaseModel):
    mode: Literal["transparent", "enriched"]


REDIS_KEY = "enrichment:mode"


async def _get_mode_from_redis() -> Optional[str]:
    redis_cache = getattr(app_state, "redis_cache", None)
    if not redis_cache:
        return None
    try:
        data = await redis_cache.get(REDIS_KEY)
        if isinstance(data, dict):
            return data.get("mode")
        if isinstance(data, str):
            return data
    except Exception as e:
        logger.warning(f"Failed to read enrichment mode from Redis: {e}")
    return None


async def _set_mode_to_redis(mode: str) -> bool:
    redis_cache = getattr(app_state, "redis_cache", None)
    if not redis_cache:
        return False
    try:
        return await redis_cache.set(REDIS_KEY, {"mode": mode})
    except Exception as e:
        logger.warning(f"Failed to write enrichment mode to Redis: {e}")
        return False


def _get_default_mode() -> str:
    import os

    return os.getenv("ENRICHMENT_MODE", "enriched")


@router.get("/mode", response_model=EnrichmentModeResponse)
async def get_enrichment_mode() -> EnrichmentModeResponse:
    from ..api.metrics import get_metrics
    from fastapi import Depends

    # Record API request metric
    try:
        metrics = get_metrics()
        metrics.enrichment_mode_requests.labels(method="GET", mode="unknown").inc()
    except Exception:
        pass  # Metrics are optional

    # 1) Try Redis
    mode = await _get_mode_from_redis()
    if mode in ("transparent", "enriched"):
        return EnrichmentModeResponse(mode=mode, source="redis")

    # 2) Try in-memory app_state
    state_mode = getattr(app_state, "enrichment_mode", None)
    if state_mode in ("transparent", "enriched"):
        return EnrichmentModeResponse(mode=state_mode, source="memory")

    # 3) Default from ENV
    default_mode = _get_default_mode()
    return EnrichmentModeResponse(mode=default_mode, source="default")


@router.post("/mode", response_model=EnrichmentModeResponse)
async def set_enrichment_mode(req: EnrichmentModeRequest) -> EnrichmentModeResponse:
    from ..api.metrics import get_metrics

    mode = req.mode

    # Get current mode to track transitions
    current_mode = getattr(app_state, "enrichment_mode", None)
    if current_mode is None:
        current_response = await get_enrichment_mode()
        current_mode = current_response.mode

    # Record metrics
    try:
        metrics = get_metrics()
        metrics.enrichment_mode_requests.labels(method="POST", mode=mode).inc()

        # Track mode switches if different
        if current_mode != mode:
            metrics.enrichment_mode_switches.labels(from_mode=current_mode, to_mode=mode).inc()

        # Update mode status gauge
        metrics.enrichment_mode_status.set(1 if mode == "enriched" else 0)

    except Exception:
        pass  # Metrics are optional

    # Save to Redis if available
    saved = await _set_mode_to_redis(mode)
    if not saved:
        # Fallback to memory
        app_state.enrichment_mode = mode

    logger.info("Enrichment mode updated", mode=mode, saved_to="redis" if saved else "memory")
    return EnrichmentModeResponse(mode=mode, source="redis" if saved else "memory")
