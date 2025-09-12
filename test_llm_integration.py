#!/usr/bin/env python3
"""
Тест LLM интеграции в webhook
"""

import json
import os
import time

import requests

# Настройка переменных окружения
os.environ["LLM_ENABLED"] = "true"
os.environ["LLM_API_KEY"] = "sk-eEyKBRlxsrWB81yZT5Mc1w"
os.environ["LLM_PROXY_URL"] = "https://llm-proxy.b2broker.tech"
os.environ["LLM_MODEL"] = "openai/gpt-4o"

BASE_URL = "http://localhost:8000"


def test_llm_integration():
    """Тест LLM интеграции"""
    print("🔍 Тест LLM интеграции в webhook")
    print("=" * 50)

    # 1. Проверяем статус сервиса
    try:
        response = requests.get(f"{BASE_URL}/healthz")
        if response.status_code == 200:
            print("✅ Сервис работает")
        else:
            print(f"❌ Сервис недоступен: {response.status_code}")
            return
    except Exception as e:
        print(f"❌ Ошибка подключения к сервису: {e}")
        return

    # 2. Устанавливаем режим transparent_with_recommendations
    try:
        response = requests.post(
            f"{BASE_URL}/enrichment/mode",
            json={"mode": "transparent_with_recommendations"},
        )
        if response.status_code == 200:
            print("✅ Режим transparent_with_recommendations установлен")
        else:
            print(f"❌ Ошибка установки режима: {response.status_code}")
            return
    except Exception as e:
        print(f"❌ Ошибка установки режима: {e}")
        return

    # 3. Отправляем webhook через /webhook/proxy
    webhook_data = {
        "receiver": "test",
        "alerts": [
            {
                "fingerprint": "test-llm-1",
                "status": "firing",
                "labels": {
                    "alertname": "HighCPUUsage",
                    "instance": "server-01",
                    "severity": "warning",
                },
                "annotations": {
                    "summary": "High CPU usage detected",
                    "description": "CPU usage is above 90% for more than 5 minutes",
                },
                "startsAt": "2024-01-01T00:00:00Z",
            }
        ],
    }

    try:
        start_time = time.time()
        response = requests.post(
            f"{BASE_URL}/webhook/proxy", json=webhook_data, timeout=30
        )
        processing_time = time.time() - start_time

        print(f"✅ Webhook обработан за {processing_time:.2f}s")
        print(f"📊 Статус: {response.status_code}")

        if response.status_code == 200:
            result = response.json()
            print(f"📝 Результат: {json.dumps(result, indent=2)}")

            # Анализируем результат
            if result.get("classification_results"):
                print("🎉 LLM классификация работает!")
                for fingerprint, classification in result[
                    "classification_results"
                ].items():
                    print(f"   🔍 {fingerprint}: {classification}")
            else:
                print("⚠️  Нет результатов LLM классификации")
                print("   💡 Возможные причины:")
                print("      - LLM недоступен")
                print("      - Проблемы с конфигурацией")
                print("      - Timeout запроса")
        else:
            print(f"❌ Ошибка webhook: {response.text}")

    except Exception as e:
        print(f"❌ Ошибка отправки webhook: {e}")


if __name__ == "__main__":
    test_llm_integration()
