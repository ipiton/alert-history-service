#!/usr/bin/env python3
"""
Простой тест webhook без FastAPI
"""

import asyncio

from src.alert_history.api.metrics import LegacyMetrics
from src.alert_history.database.sqlite_adapter import SQLiteLegacyStorage
from src.alert_history.services.webhook_processor import WebhookProcessor


async def test_simple_webhook():
    """Тестирует webhook processor напрямую"""

    print("🧪 Простой тест webhook")
    print("=" * 40)

    # 1. Создаем webhook processor напрямую
    print("1️⃣ Создаем webhook processor...")
    storage = SQLiteLegacyStorage("data/alert_history.sqlite3")
    metrics = LegacyMetrics()
    webhook_processor = WebhookProcessor(
        storage=storage,
        metrics=metrics,
        classification_service=None,
        enable_auto_classification=False,
    )

    print(f"   webhook_processor type: {type(webhook_processor)}")
    print(f"   webhook_processor is None: {webhook_processor is None}")
    print(f"   has process_webhook: {hasattr(webhook_processor, 'process_webhook')}")

    # 2. Тестируем webhook processing
    print("\n2️⃣ Тестируем webhook processing...")
    test_webhook = {
        "receiver": "test-receiver",
        "status": "firing",
        "alerts": [
            {
                "fingerprint": "test-alert-1",
                "status": "firing",
                "labels": {
                    "alertname": "HighCPUUsage",
                    "instance": "web-server-1",
                    "severity": "warning",
                },
                "annotations": {
                    "description": "CPU usage is high",
                    "summary": "High CPU usage detected",
                },
                "startsAt": "2024-01-01T10:00:00Z",
                "endsAt": "2024-01-01T10:05:00Z",
                "generatorURL": "http://localhost:9090",
            }
        ],
    }

    try:
        result = await webhook_processor.process_webhook(test_webhook)
        print("   ✅ Webhook processing successful!")
        print(f"   Result: {result}")
    except Exception as e:
        print(f"   ❌ Webhook processing failed: {e}")
        import traceback

        traceback.print_exc()

    print("\n" + "=" * 40)
    print("✅ Тест завершен!")


if __name__ == "__main__":
    asyncio.run(test_simple_webhook())
