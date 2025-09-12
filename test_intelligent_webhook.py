#!/usr/bin/env python3
"""
Тест intelligent webhook с LLM
"""

import json
import time

import requests

BASE_URL = "http://localhost:8000"


def test_intelligent_webhook():
    """Тестирует intelligent webhook с LLM"""

    print("🎯 Тест intelligent webhook с LLM")
    print("=" * 50)

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

    # 3. Отправляем тестовый webhook через intelligent endpoint
    print("\n3️⃣ Отправляем webhook через /webhook/proxy...")
    webhook_data = {
        "receiver": "test-intelligent",
        "status": "firing",
        "alerts": [
            {
                "fingerprint": "test-cpu-high-intelligent",
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
            },
            {
                "fingerprint": "test-disk-critical-intelligent",
                "status": "firing",
                "labels": {
                    "alertname": "DiskSpaceLow",
                    "instance": "db-server-1",
                    "severity": "critical",
                },
                "annotations": {
                    "description": "Disk space is running low (5% free)",
                    "summary": "Critical disk space issue on database server",
                },
                "startsAt": "2024-01-01T10:00:00Z",
                "endsAt": "2024-01-01T10:05:00Z",
            },
        ],
    }

    start_time = time.time()
    response = requests.post(f"{BASE_URL}/webhook/proxy", json=webhook_data, timeout=60)
    processing_time = time.time() - start_time

    print(f"   ⏱️  Время обработки: {processing_time:.2f}s")
    print(f"   📊 Статус ответа: {response.status_code}")

    if response.status_code == 200:
        result = response.json()
        print("   ✅ Intelligent webhook обработан успешно!")
        print(f"   📊 Обработано алертов: {result.get('processed_alerts', 0)}")
        print(f"   📤 Опубликовано алертов: {result.get('published_alerts', 0)}")
        print(f"   🚫 Отфильтровано алертов: {result.get('filtered_alerts', 0)}")
        print(f"   🎯 Metrics only mode: {result.get('metrics_only_mode', False)}")

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
            print("      - LLM недоступен")
            print("      - Проблемы с конфигурацией")
            print("      - Metrics only mode")
    else:
        print(f"   ❌ Ошибка webhook: {response.status_code}")
        print(f"   Ответ: {response.text}")

    # 5. Сравниваем с legacy webhook
    print("\n5️⃣ Сравниваем с legacy webhook...")
    legacy_response = requests.post(
        f"{BASE_URL}/webhook/", json=webhook_data, timeout=30
    )

    if legacy_response.status_code == 200:
        legacy_result = legacy_response.json()
        print("   📊 Legacy webhook:")
        print(f"      - Обработано: {legacy_result.get('processed_alerts', 0)}")
        print(f"      - Режим: {legacy_result.get('mode', 'unknown')}")
        print(
            f"      - LLM результаты: {'есть' if legacy_result.get('classification_results') else 'нет'}"
        )
    else:
        print(f"   ❌ Legacy webhook ошибка: {legacy_response.status_code}")


if __name__ == "__main__":
    test_intelligent_webhook()
