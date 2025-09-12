#!/usr/bin/env python3
"""
Тест инициализации legacy adapter
"""

import asyncio

from fastapi import FastAPI

from src.alert_history.api.legacy_adapter import LegacyAPIAdapter
from src.alert_history.api.metrics import LegacyMetrics
from src.alert_history.database.sqlite_adapter import SQLiteLegacyStorage
from src.alert_history.services.webhook_processor import WebhookProcessor


def test_legacy_adapter_init():
    """Тестирует инициализацию legacy adapter"""

    print("🔍 Тестирование инициализации legacy adapter")
    print("=" * 50)

    try:
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

        # 2. Создаем FastAPI app
        print("\n2️⃣ Создаем FastAPI app...")
        app = FastAPI()
        print(f"   ✅ FastAPI app создан: {type(app)}")

        # 3. Создаем legacy adapter
        print("\n3️⃣ Создаем legacy adapter...")
        legacy_adapter = LegacyAPIAdapter(
            app=app,
            storage=storage,
            db_path="data/alert_history.sqlite3",
            retention_days=30,
            webhook_processor=webhook_processor,
        )
        print(f"   ✅ Legacy adapter создан: {type(legacy_adapter)}")

        # 4. Проверяем endpoints
        print("\n4️⃣ Проверяем зарегистрированные endpoints...")
        routes = [route.path for route in app.routes]
        print(f"   Всего routes: {len(routes)}")

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
                print(f"   ✅ {route} - зарегистрирован")
            else:
                print(f"   ❌ {route} - НЕ зарегистрирован")

        # 5. Проверяем, что legacy adapter сохранился в app
        print("\n5️⃣ Проверяем app.legacy_adapter...")
        if hasattr(app, "legacy_adapter"):
            print(f"   ✅ app.legacy_adapter существует: {type(app.legacy_adapter)}")
        else:
            print("   ❌ app.legacy_adapter не существует")

        return True

    except Exception as e:
        print(f"   ❌ Ошибка инициализации: {e}")
        import traceback

        traceback.print_exc()
        return False


if __name__ == "__main__":
    success = test_legacy_adapter_init()
    if success:
        print("\n✅ Тест завершен успешно!")
    else:
        print("\n❌ Тест завершен с ошибками!")
