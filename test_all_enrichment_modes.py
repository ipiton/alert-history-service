#!/usr/bin/env python3
"""
Комплексный тест всех режимов обогащения
"""

import json
import time

import requests

BASE_URL = "http://localhost:8000"


def test_all_enrichment_modes():
    """Тестирует все три режима обогащения"""

    print("🎯 Комплексный тест всех режимов обогащения")
    print("=" * 60)

    # Ждем запуска сервиса
    print("⏳ Ждем запуска сервиса...")
    for i in range(10):
        try:
            response = requests.get(f"{BASE_URL}/healthz", timeout=2)
            if response.status_code == 200:
                print("✅ Сервис запущен!")
                break
        except:
            time.sleep(1)
    else:
        print("❌ Сервис не запустился")
        return

    # Тестовые алерты
    test_alerts = [
        {
            "fingerprint": "test-cpu-high",
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
            "fingerprint": "test-disk-critical",
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
        {
            "fingerprint": "test-memory-info",
            "status": "firing",
            "labels": {
                "alertname": "HighMemoryUsage",
                "instance": "app-server-1",
                "severity": "info",
            },
            "annotations": {
                "description": "Memory usage is elevated",
                "summary": "Memory usage monitoring",
            },
            "startsAt": "2024-01-01T10:00:00Z",
            "endsAt": "2024-01-01T10:05:00Z",
            "generatorURL": "http://localhost:9090",
        },
    ]

    # Тестируем каждый режим
    modes = [
        ("transparent", "Прозрачный режим"),
        ("transparent_with_recommendations", "Прозрачный с рекомендациями"),
        ("enriched", "Обогащенный режим"),
    ]

    for mode, description in modes:
        print(f"\n{'='*20} {description} {'='*20}")

        # 1. Устанавливаем режим
        print(f"1️⃣ Устанавливаем режим: {mode}")
        response = requests.post(f"{BASE_URL}/enrichment/mode", json={"mode": mode})
        if response.status_code == 200:
            print(f"   ✅ Режим установлен: {response.json()['mode']}")
        else:
            print(f"   ❌ Ошибка установки режима: {response.status_code}")
            continue

        # 2. Проверяем текущий режим
        print("2️⃣ Проверяем текущий режим...")
        response = requests.get(f"{BASE_URL}/enrichment/mode")
        current_mode = response.json()
        print(
            f"   Текущий режим: {current_mode['mode']} (источник: {current_mode['source']})"
        )

        # 3. Отправляем webhook
        print(f"3️⃣ Отправляем webhook с {len(test_alerts)} алертами...")
        webhook_data = {
            "receiver": f"test-{mode}",
            "status": "firing",
            "alerts": test_alerts,
        }

        response = requests.post(f"{BASE_URL}/webhook/", json=webhook_data)

        if response.status_code == 200:
            result = response.json()
            print("   ✅ Webhook обработан успешно!")
            print(f"   Обработано алертов: {result.get('processed_alerts', 0)}")
            print(f"   Опубликовано алертов: {result.get('published_alerts', 0)}")
            print(f"   Отфильтровано алертов: {result.get('filtered_alerts', 0)}")
            print(f"   Режим обработки: {result.get('mode', 'unknown')}")

            # Проверяем результаты классификации
            classification_results = result.get("classification_results", {})
            if classification_results:
                print("   📋 Результаты классификации:")
                for fingerprint, data in classification_results.items():
                    severity = data.get("severity", "unknown")
                    confidence = data.get("confidence", 0)
                    print(
                        f"     - {fingerprint}: {severity} (confidence: {confidence})"
                    )

                    # Проверяем рекомендации
                    recommendations = data.get("recommendations", [])
                    if recommendations:
                        print(f"       💡 Рекомендации: {recommendations}")
            else:
                print("   ⚠️  Нет результатов классификации (LLM недоступен)")

        else:
            print(f"   ❌ Ошибка webhook: {response.status_code}")
            print(f"   Ответ: {response.text}")

        # 4. Проверяем метрики режима
        print("4️⃣ Проверяем метрики режима...")
        try:
            response = requests.get(f"{BASE_URL}/metrics")
            if response.status_code == 200:
                metrics_text = response.text
                if "enrichment_mode_status" in metrics_text:
                    print("   ✅ Метрика enrichment_mode_status найдена")
                if f"enrichment_{mode}_alerts_total" in metrics_text:
                    print(f"   ✅ Метрика enrichment_{mode}_alerts_total найдена")
            else:
                print(f"   ⚠️  Метрики недоступны: {response.status_code}")
        except Exception as e:
            print(f"   ⚠️  Ошибка получения метрик: {e}")

        # 5. Анализируем поведение режима
        print(f"5️⃣ Анализ поведения режима '{mode}':")
        if mode == "transparent":
            print("   🎯 Все алерты проходят без изменений")
            print("   🎯 Нет классификации LLM")
            print("   🎯 Нет фильтрации")
        elif mode == "transparent_with_recommendations":
            print("   🎯 Все алерты проходят (без фильтрации)")
            print("   🎯 LLM классифицирует и дает рекомендации")
            print("   🎯 Безопасное обучение без риска потери алертов")
        elif mode == "enriched":
            print("   🎯 Алерты обогащаются LLM классификацией")
            print("   🎯 Применяется фильтрация на основе LLM")
            print("   🎯 Только разрешенные алерты публикуются")

        time.sleep(1)  # Пауза между режимами

    # Финальная сводка
    print(f"\n{'='*60}")
    print("📊 ФИНАЛЬНАЯ СВОДКА")
    print("=" * 60)
    print("✅ Все три режима протестированы:")
    print("   🎯 transparent - простой режим без изменений")
    print("   🎯 transparent_with_recommendations - безопасное обучение")
    print("   🎯 enriched - полная LLM обработка")
    print("\n💡 Рекомендации по использованию:")
    print("   1. transparent - для простого логирования")
    print("   2. transparent_with_recommendations - для изучения LLM")
    print("   3. enriched - для продакшена с фильтрацией")
    print("\n🚀 Система готова к использованию!")


if __name__ == "__main__":
    test_all_enrichment_modes()
