#!/usr/bin/env python3
"""
Тест webhook с LLM
"""

import json
import time

import requests

BASE_URL = "http://localhost:8000"


def test_webhook_llm():
    """Тестирует webhook с LLM"""

    print("🎯 Тест webhook с LLM")
    print("=" * 40)

    # 1. Проверяем статус сервиса
    print("1️⃣ Проверяем статус сервиса...")
    try:
        response = requests.get(f"{BASE_URL}/healthz")
        if response.status_code == 200:
            print("   ✅ Сервис работает")
        else:
            print(f"   ❌ Сервис недоступен: {response.status_code}")
            return
    except Exception as e:
        print(f"   ❌ Ошибка подключения: {e}")
        return

    # 2. Устанавливаем режим transparent_with_recommendations
    print("\n2️⃣ Устанавливаем режим transparent_with_recommendations...")
    response = requests.post(
        f"{BASE_URL}/enrichment/mode", json={"mode": "transparent_with_recommendations"}
    )
    if response.status_code == 200:
        mode_info = response.json()
        print(f"   ✅ Режим установлен: {mode_info['mode']}")
    else:
        print(f"   ❌ Ошибка установки режима: {response.status_code}")
        return

    # 3. Отправляем тестовый webhook
    print("\n3️⃣ Отправляем тестовый webhook...")
    webhook_data = {
        "receiver": "test-llm",
        "status": "firing",
        "alerts": [
            {
                "fingerprint": "test-cpu-high-llm",
                "status": "firing",
                "labels": {
                    "alertname": "HighCPUUsage",
                    "instance": "web-server-1",
                    "severity": "warning",
                },
                "annotations": {
                    "description": "CPU usage is high (85% for 5 minutes)",
                    "summary": "High CPU usage detected on web server",
                },
                "startsAt": "2024-01-01T10:00:00Z",
                "endsAt": "2024-01-01T10:05:00Z",
            }
        ],
    }

    start_time = time.time()
    response = requests.post(f"{BASE_URL}/webhook/", json=webhook_data, timeout=60)
    processing_time = time.time() - start_time

    print(f"   ⏱️  Время обработки: {processing_time:.2f}s")
    print(f"   📊 Статус ответа: {response.status_code}")

    if response.status_code == 200:
        result = response.json()
        print("   ✅ Webhook обработан успешно!")
        print(f"   📊 Обработано алертов: {result.get('processed_alerts', 0)}")
        print(f"   📤 Опубликовано алертов: {result.get('published_alerts', 0)}")
        print(f"   🚫 Отфильтровано алертов: {result.get('filtered_alerts', 0)}")
        print(f"   🎯 Режим обработки: {result.get('mode', 'unknown')}")

        # Проверяем результаты LLM
        classification_results = result.get("classification_results", {})
        if classification_results:
            print("\n4️⃣ 📋 Результаты LLM классификации:")
            for fingerprint, data in classification_results.items():
                print(f"   🔍 Алерт: {fingerprint}")
                print(f"      Severity: {data.get('severity', 'unknown')}")
                print(f"      Confidence: {data.get('confidence', 0)}")
                print(f"      Reasoning: {data.get('reasoning', 'N/A')[:100]}...")

                recommendations = data.get("recommendations", [])
                if recommendations:
                    print("      💡 Рекомендации:")
                    for i, rec in enumerate(recommendations, 1):
                        print(f"         {i}. {rec}")
                else:
                    print("      💡 Рекомендации: нет")
                print()
        else:
            print("\n4️⃣ ⚠️  Нет результатов LLM классификации")
            print("   💡 Возможные причины:")
            print("      - Webhook работает в legacy mode")
            print("      - LLM не вызывается в webhook")
            print("      - Проблемы с конфигурацией")
    else:
        print(f"   ❌ Ошибка webhook: {response.status_code}")
        print(f"   Ответ: {response.text}")

    # 5. Проверяем, что алерт сохранился в базе
    print("\n5️⃣ Проверяем сохранение в базе...")
    try:
        # Проверяем через legacy endpoint (если работает)
        response = requests.get(f"{BASE_URL}/history")
        if response.status_code == 200:
            history = response.json()
            print(f"   ✅ История доступна: {len(history.get('alerts', []))} алертов")
        else:
            print(f"   ⚠️  История недоступна: {response.status_code}")
    except Exception as e:
        print(f"   ⚠️  Ошибка получения истории: {e}")


if __name__ == "__main__":
    test_webhook_llm()
