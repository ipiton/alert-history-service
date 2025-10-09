#!/usr/bin/env python3
"""
Тест для проверки legacy adapter
"""

import asyncio

from src.alert_history.api.legacy_adapter import LegacyAPIAdapter
from src.alert_history.api.metrics import LegacyMetrics
from src.alert_history.database.sqlite_adapter import SQLiteLegacyStorage
from src.alert_history.services.webhook_processor import WebhookProcessor


def test_legacy_adapter():
    """Тестирует legacy adapter"""

    print("🧪 Тестирование Legacy Adapter")
    print("=" * 40)

    # 1. Создаем компоненты
    print("1️⃣ Создаем компоненты...")
    storage = SQLiteLegacyStorage("data/alert_history.sqlite3")
    metrics = LegacyMetrics()
    webhook_processor = WebhookProcessor(
        storage=storage,
        metrics=metrics,
        classification_service=None,
        enable_auto_classification=False,
    )

    print(f"   ✅ Storage создан: {type(storage)}")
    print(f"   ✅ Metrics созданы: {type(metrics)}")
    print(f"   ✅ Webhook processor создан: {type(webhook_processor)}")

    # 2. Создаем mock FastAPI app
    print("\n2️⃣ Создаем mock FastAPI app...")
    from fastapi import FastAPI

    app = FastAPI()

    print(f"   ✅ FastAPI app создан: {type(app)}")

    # 3. Создаем legacy adapter
    print("\n3️⃣ Создаем legacy adapter...")
    try:
        legacy_adapter = LegacyAPIAdapter(
            app=app,
            storage=storage,
            db_path="data/alert_history.sqlite3",
            retention_days=30,
            webhook_processor=webhook_processor,
        )
        print(f"   ✅ Legacy adapter создан: {type(legacy_adapter)}")
    except Exception as e:
        print(f"   ❌ Ошибка создания legacy adapter: {e}")
        import traceback

        traceback.print_exc()
        return

    # 4. Проверяем endpoints
    print("\n4️⃣ Проверяем endpoints...")
    routes = [route.path for route in app.routes]
    print(f"   Зарегистрированные routes: {routes}")

    expected_routes = [
        "/webhook",
        "/history",
        "/report",
        "/metrics",
        "/dashboard",
        "/dashboard/grouped",
        "/health",
    ]

    for route in expected_routes:
        if route in routes:
            print(f"   ✅ {route} - найден")
        else:
            print(f"   ❌ {route} - НЕ найден")

    print("\n" + "=" * 40)
    print("✅ Тест завершен!")


if __name__ == "__main__":
    test_legacy_adapter()
