#!/usr/bin/env python3
"""
Test T6: Dashboard and UI Integration (HTML5 Dashboard + API)

Проверяет:
- /api/dashboard/overview JSON структуру
- /api/dashboard/charts JSON структуру
- /api/dashboard/health JSON структуру
- /api/dashboard/alerts/recent JSON структуру
- /dashboard/modern HTML отдачу
"""
import asyncio
import os
import sys
from typing import Tuple, Union

import httpx

BASE_URL = os.getenv("TEST_BASE_URL", "http://127.0.0.1:8000")


async def get(path: str) -> Tuple[int, Union[dict, str]]:
    async with httpx.AsyncClient(timeout=10.0) as client:
        resp = await client.get(f"{BASE_URL}{path}")
        content_type = resp.headers.get("content-type", "")
        if "application/json" in content_type:
            return resp.status_code, resp.json()
        return resp.status_code, resp.text


async def test_overview():
    code, data = await get("/api/dashboard/overview")
    assert code == 200
    assert isinstance(data, dict)
    for key in [
        "total_alerts",
        "active_alerts",
        "resolved_alerts",
        "alerts_last_24h",
        "publishing_targets",
        "publishing_mode",
        "classification_enabled",
    ]:
        assert key in data
    print("overview ok")


async def test_charts():
    code, data = await get("/api/dashboard/charts?hours=24")
    assert code == 200
    assert isinstance(data, dict)
    assert "alert_timeline" in data
    assert "severity_distribution" in data
    print("charts ok")


async def test_health():
    code, data = await get("/api/dashboard/health")
    assert code == 200
    assert isinstance(data, list)
    print("health ok")


async def test_recent_alerts():
    code, data = await get("/api/dashboard/alerts/recent?limit=5")
    assert code == 200
    assert isinstance(data, dict)
    assert "alerts" in data
    print("recent alerts ok")


async def test_html5_dashboard():
    code, html = await get("/dashboard/modern")
    assert code == 200
    assert isinstance(html, str)
    assert "HTML5 Dashboard" in html or "Alert History Dashboard" in html
    print("html5 dashboard ok")


async def main():
    tests = [
        test_overview,
        test_charts,
        test_health,
        test_recent_alerts,
        test_html5_dashboard,
    ]
    passed = 0
    for t in tests:
        try:
            await t()
            passed += 1
        except AssertionError as e:
            print(f"FAIL {t.__name__}: {e}")
        except Exception as e:
            print(f"ERROR {t.__name__}: {e}")
    print(f"Passed {passed}/{len(tests)}")
    return passed == len(tests)


if __name__ == "__main__":
    ok = asyncio.run(main())
    sys.exit(0 if ok else 1)
