#!/usr/bin/env python3
"""
Диагностика проблем с LLM
"""

import asyncio
import os

import aiohttp


async def test_llm_directly():
    """Прямое тестирование LLM прокси"""

    print("🔍 Диагностика LLM")
    print("=" * 40)

    # Проверяем переменные окружения
    api_key = os.getenv("LLM_API_KEY")
    proxy_url = os.getenv("LLM_PROXY_URL")
    model = os.getenv("LLM_MODEL", "gpt-4")

    print(f"API Key: {api_key[:10]}..." if api_key else "не установлен")
    print(f"Proxy URL: {proxy_url}")
    print(f"Model: {model}")

    if not api_key or not proxy_url:
        print("❌ Не хватает переменных окружения")
        return

    # Тестируем прямое подключение к LLM прокси
    print("\n🧪 Тестируем прямое подключение к LLM...")

    headers = {"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"}

    payload = {
        "model": model,
        "messages": [
            {
                "role": "system",
                "content": "Ты эксперт по классификации алертов. Классифицируй этот алерт.",
            },
            {"role": "user", "content": "High CPU usage detected on web-server-1"},
        ],
        "max_tokens": 100,
    }

    try:
        async with aiohttp.ClientSession() as session:
            print(f"📤 Отправляем запрос к {proxy_url}/v1/chat/completions")

            async with session.post(
                f"{proxy_url}/v1/chat/completions",
                headers=headers,
                json=payload,
                timeout=aiohttp.ClientTimeout(total=30),
            ) as response:
                print(f"📥 Получен ответ: {response.status}")

                if response.status == 200:
                    data = await response.json()
                    print("✅ LLM ответил успешно!")
                    if "choices" in data and len(data["choices"]) > 0:
                        content = data["choices"][0]["message"]["content"]
                        print(f"📝 Ответ: {content[:200]}...")
                else:
                    text = await response.text()
                    print(f"❌ Ошибка: {response.status}")
                    print(f"📝 Ответ: {text}")

    except Exception as e:
        print(f"❌ Ошибка подключения: {e}")


def test_llm_client():
    """Тестирование LLM клиента"""

    print("\n🔧 Тестирование LLM клиента...")

    try:
        from datetime import datetime

        from src.alert_history.core.interfaces import Alert, AlertStatus
        from src.alert_history.services.llm_client import LLMProxyClient

        # Создаем LLM клиент
        llm_client = LLMProxyClient(
            proxy_url=os.getenv("LLM_PROXY_URL"),
            api_key=os.getenv("LLM_API_KEY"),
            model=os.getenv("LLM_MODEL", "gpt-4"),
            timeout=int(os.getenv("LLM_TIMEOUT", "30")),
            max_retries=int(os.getenv("LLM_MAX_RETRIES", "3")),
        )

        print("✅ LLM клиент создан")

        # Создаем тестовый алерт
        test_alert = Alert(
            fingerprint="test-debug",
            alert_name="TestAlert",
            status=AlertStatus.FIRING,
            labels={"instance": "test-server", "severity": "warning"},
            annotations={"description": "Test alert for debugging"},
            starts_at=datetime.now(),
            generator_url="http://localhost:9090",
        )

        print("✅ Тестовый алерт создан")

        # Тестируем классификацию
        print("📤 Отправляем запрос на классификацию...")
        result = asyncio.run(llm_client.classify_alert(test_alert))

        if result:
            print("✅ Классификация успешна!")
            print(f"📝 Результат: {result}")
        else:
            print("❌ Классификация не удалась")

    except Exception as e:
        print(f"❌ Ошибка LLM клиента: {e}")
        import traceback

        traceback.print_exc()


if __name__ == "__main__":
    # Тестируем прямое подключение
    asyncio.run(test_llm_directly())

    # Тестируем LLM клиент
    test_llm_client()
