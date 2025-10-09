#!/usr/bin/env python3
"""
Тест для проверки app_state
"""

import asyncio

from src.alert_history.api.webhook_endpoints import get_webhook_processor
from src.alert_history.core.app_state import app_state


async def test_app_state():
    """Тестирует app_state и webhook processor"""

    print("🧪 Тестирование app_state")
    print("=" * 40)

    # 1. Проверяем app_state
    print("1️⃣ Проверяем app_state...")
    print(f"   app_state type: {type(app_state)}")
    print(f"   app_state attributes: {dir(app_state)}")

    # 2. Проверяем webhook processor
    print("\n2️⃣ Проверяем webhook processor...")
    try:
        webhook_processor = await get_webhook_processor()
        print(f"   webhook_processor type: {type(webhook_processor)}")
        print(f"   webhook_processor attributes: {dir(webhook_processor)}")

        if hasattr(webhook_processor, "process_webhook"):
            print("   ✅ process_webhook method exists")
        else:
            print("   ❌ process_webhook method missing")

    except Exception as e:
        print(f"   ❌ Error getting webhook processor: {e}")

    # 3. Проверяем app_state после получения webhook processor
    print("\n3️⃣ Проверяем app_state после получения webhook processor...")
    if hasattr(app_state, "webhook_processor"):
        print(f"   webhook_processor in app_state: {type(app_state.webhook_processor)}")
    else:
        print("   webhook_processor not in app_state")

    print("\n" + "=" * 40)
    print("✅ Тест завершен!")


if __name__ == "__main__":
    asyncio.run(test_app_state())
