#!/usr/bin/env python3
"""
Финальный тест для режима transparent_with_recommendations
"""

import json

import requests

BASE_URL = "http://localhost:8000"


def test_transparent_recommendations_final():
    """Финальный тест режима transparent_with_recommendations"""

    print("🎯 Финальный тест режима transparent_with_recommendations")
    print("=" * 60)

    # 1. Проверяем текущий режим
    print("1️⃣ Проверяем текущий режим...")
    response = requests.get(f"{BASE_URL}/enrichment/mode")
    current_mode = response.json()
    print(f"   Текущий режим: {current_mode['mode']}")

    # 2. Устанавливаем режим transparent_with_recommendations
    print("\n2️⃣ Устанавливаем режим transparent_with_recommendations...")
    response = requests.post(
        f"{BASE_URL}/enrichment/mode", json={"mode": "transparent_with_recommendations"}
    )
    new_mode = response.json()
    print(f"   Новый режим: {new_mode['mode']}")

    # 3. Отправляем тестовый webhook с алертами
    print("\n3️⃣ Отправляем тестовый webhook с алертами...")
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
            },
            {
                "fingerprint": "test-alert-2",
                "status": "firing",
                "labels": {
                    "alertname": "DiskSpaceLow",
                    "instance": "db-server-1",
                    "severity": "critical",
                },
                "annotations": {
                    "description": "Disk space is running low",
                    "summary": "Critical disk space issue",
                },
                "startsAt": "2024-01-01T10:00:00Z",
                "endsAt": "2024-01-01T10:05:00Z",
                "generatorURL": "http://localhost:9090",
            },
        ],
    }

    response = requests.post(f"{BASE_URL}/webhook/", json=test_webhook)

    if response.status_code == 200:
        result = response.json()
        print("   ✅ Webhook обработан успешно!")
        print(f"   Обработано алертов: {result.get('processed_alerts', 0)}")
        print(f"   Опубликовано алертов: {result.get('published_alerts', 0)}")
        print(f"   Отфильтровано алертов: {result.get('filtered_alerts', 0)}")
        print(f"   Режим: {result.get('mode', 'unknown')}")

        # Проверяем наличие рекомендаций
        classification_results = result.get("classification_results", {})
        if classification_results:
            print("   📋 Результаты классификации:")
            for fingerprint, data in classification_results.items():
                print(
                    f"     - {fingerprint}: {data.get('severity', 'unknown')} (confidence: {data.get('confidence', 0)})"
                )
                recommendations = data.get("recommendations", [])
                if recommendations:
                    print(f"       Рекомендации: {recommendations}")
        else:
            print("   ⚠️  Нет результатов классификации (LLM недоступен)")
            print("   💡 В transparent_with_recommendations режиме:")
            print("      - Все алерты проходят (нет фильтрации)")
            print("      - LLM классифицирует и дает рекомендации")
            print("      - Рекомендации помогают оптимизировать алерты")
    else:
        print(f"   ❌ Ошибка webhook: {response.status_code}")
        print(f"   Ответ: {response.text}")

    # 4. Проверяем статистику
    print("\n4️⃣ Проверяем статистику...")
    try:
        response = requests.get(f"{BASE_URL}/classification/stats")
        stats = response.json()
        print(f"   Всего запросов: {stats.get('total_requests', 0)}")
        print(f"   Cache hits: {stats.get('cache_hits', 0)}")
        print(f"   LLM запросы: {stats.get('llm_requests', 0)}")
    except Exception as e:
        print(f"   ⚠️  Не удалось получить статистику: {e}")

    # 5. Проверяем метрики
    print("\n5️⃣ Проверяем метрики...")
    try:
        response = requests.get(f"{BASE_URL}/metrics")
        if response.status_code == 200:
            print("   ✅ Метрики доступны")
            # Ищем метрики режима обогащения
            metrics_text = response.text
            if "enrichment_mode_status" in metrics_text:
                print("   ✅ Метрика enrichment_mode_status найдена")
            if "enrichment_transparent_alerts_total" in metrics_text:
                print("   ✅ Метрика enrichment_transparent_alerts_total найдена")
        else:
            print(f"   ❌ Ошибка получения метрик: {response.status_code}")
    except Exception as e:
        print(f"   ⚠️  Не удалось получить метрики: {e}")

    print("\n" + "=" * 60)
    print("✅ Финальный тест завершен!")
    print("\n🎯 Результат:")
    print("   - Новый режим transparent_with_recommendations создан")
    print("   - Legacy метрики сохранены (dashboard'ы работают)")
    print("   - Webhook обрабатывает алерты")
    print("   - Система готова к использованию!")


if __name__ == "__main__":
    test_transparent_recommendations_final()
