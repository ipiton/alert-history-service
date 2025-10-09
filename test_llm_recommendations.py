#!/usr/bin/env python3
"""
Финальный тест режима transparent_with_recommendations с LLM
"""

import json
import os
import time

import requests

BASE_URL = "http://localhost:8000"


def test_llm_recommendations():
    """Тестирует режим transparent_with_recommendations с LLM"""

    print("🎯 Тест режима transparent_with_recommendations с LLM")
    print("=" * 60)

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

    # 3. Тестовые алерты для LLM анализа
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
                "description": "CPU usage is high (85% for 5 minutes)",
                "summary": "High CPU usage detected on web server",
            },
            "startsAt": "2024-01-01T10:00:00Z",
            "endsAt": "2024-01-01T10:05:00Z",
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
                "description": "Disk space is running low (5% free)",
                "summary": "Critical disk space issue on database server",
            },
            "startsAt": "2024-01-01T10:00:00Z",
            "endsAt": "2024-01-01T10:05:00Z",
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
                "description": "Memory usage is elevated (75% used)",
                "summary": "Memory usage monitoring alert",
            },
            "startsAt": "2024-01-01T10:00:00Z",
            "endsAt": "2024-01-01T10:05:00Z",
        },
    ]

    # 4. Отправляем webhook с тестовыми алертами
    print(f"\n3️⃣ Отправляем webhook с {len(test_alerts)} алертами...")
    webhook_data = {"receiver": "test-llm", "status": "firing", "alerts": test_alerts}

    start_time = time.time()
    response = requests.post(
        f"{BASE_URL}/webhook/",
        json=webhook_data,
        timeout=60,  # Увеличиваем timeout для LLM
    )
    processing_time = time.time() - start_time

    if response.status_code == 200:
        result = response.json()
        print("   ✅ Webhook обработан успешно!")
        print(f"   ⏱️  Время обработки: {processing_time:.2f}s")
        print(f"   📊 Обработано алертов: {result.get('processed_alerts', 0)}")
        print(f"   📤 Опубликовано алертов: {result.get('published_alerts', 0)}")
        print(f"   🚫 Отфильтровано алертов: {result.get('filtered_alerts', 0)}")
        print(f"   🎯 Режим обработки: {result.get('mode', 'unknown')}")

        # 5. Анализируем результаты LLM
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
            print("      - LLM недоступен (проверьте API ключ)")
            print("      - Проблемы с сетью")
            print("      - Timeout LLM запроса")
            print("   🎯 В transparent_with_recommendations режиме:")
            print("      - Все алерты проходят (безопасно)")
            print("      - LLM анализирует и дает рекомендации")
            print("      - Нет риска потери важных алертов")

        # 6. Проверяем статистику
        print("\n5️⃣ 📊 Проверяем статистику...")
        try:
            response = requests.get(f"{BASE_URL}/classification/stats")
            if response.status_code == 200:
                stats = response.json()
                print(f"   📈 Всего запросов: {stats.get('total_requests', 0)}")
                print(f"   🎯 Cache hits: {stats.get('cache_hits', 0)}")
                print(f"   🤖 LLM запросы: {stats.get('llm_requests', 0)}")
                print(
                    f"   ⚡ Среднее время: {stats.get('avg_processing_time', 0):.2f}s"
                )
            else:
                print(f"   ⚠️  Статистика недоступна: {response.status_code}")
        except Exception as e:
            print(f"   ⚠️  Ошибка получения статистики: {e}")

        # 7. Проверяем метрики
        print("\n6️⃣ 📈 Проверяем метрики...")
        try:
            response = requests.get(f"{BASE_URL}/metrics")
            if response.status_code == 200:
                metrics_text = response.text
                if "enrichment_mode_status" in metrics_text:
                    print("   ✅ Метрика enrichment_mode_status найдена")
                if "enrichment_transparent_alerts_total" in metrics_text:
                    print("   ✅ Метрика enrichment_transparent_alerts_total найдена")
                if "classification_total" in metrics_text:
                    print("   ✅ Метрика classification_total найдена")
            else:
                print(f"   ⚠️  Метрики недоступны: {response.status_code}")
        except Exception as e:
            print(f"   ⚠️  Ошибка получения метрик: {e}")

    else:
        print(f"   ❌ Ошибка webhook: {response.status_code}")
        print(f"   Ответ: {response.text}")

    # 8. Финальная сводка
    print(f"\n{'='*60}")
    print("📊 ФИНАЛЬНАЯ СВОДКА")
    print("=" * 60)
    print("✅ Тест завершен!")
    print("🎯 Режим: transparent_with_recommendations")
    print(f"🤖 LLM статус: {'доступен' if classification_results else 'недоступен'}")
    print(
        f"📊 Обработано алертов: {result.get('processed_alerts', 0) if response.status_code == 200 else 0}"
    )
    print(f"⏱️  Время обработки: {processing_time:.2f}s")

    print("\n💡 Рекомендации:")
    if classification_results:
        print("   ✅ LLM работает - можно использовать enriched режим")
        print("   📋 Изучите рекомендации для оптимизации алертов")
        print("   🎯 Постепенно переходите к enriched режиму")
    else:
        print("   🔧 Проверьте LLM конфигурацию:")
        print(f"      - API ключ: {os.getenv('LLM_API_KEY', 'не установлен')[:10]}...")
        print(f"      - Proxy URL: {os.getenv('LLM_PROXY_URL', 'не установлен')}")
        print(f"      - Модель: {os.getenv('LLM_MODEL', 'не установлена')}")
        print("   🛡️  Система работает безопасно в transparent режиме")
        print("   📊 Все алерты проходят без потерь")


if __name__ == "__main__":
    test_llm_recommendations()
