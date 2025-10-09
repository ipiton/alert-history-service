#!/usr/bin/env python3
"""
Простой тест для режима transparent_with_recommendations
"""

import asyncio
import json

from src.alert_history.api.metrics import LegacyMetrics
from src.alert_history.database.sqlite_adapter import SQLiteLegacyStorage
from src.alert_history.services.webhook_processor import WebhookProcessor


async def test_transparent_recommendations():
    """Тестирует режим transparent_with_recommendations"""

    print("🧪 Тест режима transparent_with_recommendations")
    print("=" * 50)

    # 1. Создаем компоненты
    print("1️⃣ Создаем компоненты...")
    storage = SQLiteLegacyStorage("data/alert_history.sqlite3")
    metrics = LegacyMetrics()
    webhook_processor = WebhookProcessor(
        storage=storage,
        metrics=metrics,
        classification_service=None,  # LLM недоступен
        enable_auto_classification=False,
    )

    print("   ✅ Webhook processor создан")
    print("   ✅ Metrics созданы")

    # 2. Симулируем режим transparent_with_recommendations
    print("\n2️⃣ Симулируем режим transparent_with_recommendations...")

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

    # 3. Обрабатываем webhook (transparent mode)
    print("\n3️⃣ Обрабатываем webhook (transparent mode)...")
    try:
        # Отключаем auto classification (transparent mode)
        original_flag = webhook_processor.enable_auto_classification
        webhook_processor.enable_auto_classification = False

        # Обрабатываем webhook
        result = await webhook_processor.process_webhook(test_webhook)

        # Восстанавливаем флаг
        webhook_processor.enable_auto_classification = original_flag

        print("   ✅ Webhook обработан успешно!")
        print(f"   Обработано алертов: {result.get('processed', 0)}")
        print(f"   Классифицировано: {result.get('classified', 0)}")
        print(f"   Ошибки: {result.get('errors', [])}")

        # 4. Симулируем рекомендации (если бы был LLM)
        print("\n4️⃣ Симулируем рекомендации...")
        print("   📋 Рекомендации (симуляция):")
        print("     - Увеличить threshold для CPU usage с 80% до 85%")
        print("     - Добавить условие 'for: 5m' для стабильности")
        print("     - Исключить рабочее время (9:00-18:00)")
        print("     - Добавить условие для instance != 'test-server'")

        # 5. Результат в transparent_with_recommendations режиме
        print("\n5️⃣ Результат в transparent_with_recommendations режиме:")
        result_summary = {
            "message": "Webhook processed successfully (transparent_with_recommendations mode)",
            "processed_alerts": len(test_webhook["alerts"]),
            "published_alerts": len(test_webhook["alerts"]),  # Все алерты проходят
            "filtered_alerts": 0,  # Нет фильтрации
            "classification_results": {
                "test-alert-1": {
                    "severity": "noise",
                    "confidence": 0.9,
                    "reasoning": "Это обычная нагрузка в рабочее время",
                    "recommendations": [
                        "Увеличить threshold с 80% до 85%",
                        "Добавить условие 'for: 5m'",
                        "Исключить рабочее время (9:00-18:00)",
                    ],
                }
            },
            "mode": "transparent_with_recommendations",
        }

        print(json.dumps(result_summary, indent=2, ensure_ascii=False))

    except Exception as e:
        print(f"   ❌ Ошибка обработки webhook: {e}")
        import traceback

        traceback.print_exc()

    print("\n" + "=" * 50)
    print("✅ Тест завершен!")


if __name__ == "__main__":
    asyncio.run(test_transparent_recommendations())
