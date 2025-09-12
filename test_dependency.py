#!/usr/bin/env python3
"""
Тест для проверки dependency injection
"""

import asyncio

from src.alert_history.api.webhook_endpoints import get_metrics, get_webhook_processor


async def test_dependencies():
    """Тестирует dependency injection"""

    print("🧪 Тестирование dependency injection")
    print("=" * 50)

    # 1. Проверяем metrics
    print("1️⃣ Проверяем metrics...")
    try:
        metrics = await get_metrics()
        print(f"   metrics type: {type(metrics)}")
        print(f"   metrics is None: {metrics is None}")
    except Exception as e:
        print(f"   ❌ Error getting metrics: {e}")

    # 2. Проверяем webhook processor
    print("\n2️⃣ Проверяем webhook processor...")
    try:
        webhook_processor = await get_webhook_processor()
        print(f"   webhook_processor type: {type(webhook_processor)}")
        print(f"   webhook_processor is None: {webhook_processor is None}")

        if webhook_processor is not None:
            print(
                f"   webhook_processor has process_webhook: {hasattr(webhook_processor, 'process_webhook')}"
            )
            print(f"   webhook_processor storage: {type(webhook_processor.storage)}")
            print(f"   webhook_processor metrics: {type(webhook_processor.metrics)}")
        else:
            print("   ❌ webhook_processor is None!")

    except Exception as e:
        print(f"   ❌ Error getting webhook processor: {e}")
        import traceback

        traceback.print_exc()

    print("\n" + "=" * 50)
    print("✅ Тест завершен!")


if __name__ == "__main__":
    asyncio.run(test_dependencies())
