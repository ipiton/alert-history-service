#!/usr/bin/env python3
"""
Тест для проверки LegacyMetrics
"""

from src.alert_history.api.metrics import LegacyMetrics


def test_legacy_metrics():
    """Тестирует LegacyMetrics"""

    print("🧪 Тестирование LegacyMetrics")
    print("=" * 40)

    # 1. Создаем LegacyMetrics
    print("1️⃣ Создаем LegacyMetrics...")
    metrics = LegacyMetrics()

    print(f"   metrics type: {type(metrics)}")
    print(f"   metrics is None: {metrics is None}")

    # 2. Проверяем legacy метрики
    print("\n2️⃣ Проверяем legacy метрики...")
    print(f"   webhook_events_total: {hasattr(metrics, 'webhook_events_total')}")
    print(f"   webhook_errors_total: {hasattr(metrics, 'webhook_errors_total')}")
    print(f"   request_latency_seconds: {hasattr(metrics, 'request_latency_seconds')}")
    print(f"   alerts_stored_total: {hasattr(metrics, 'alerts_stored_total')}")

    # 3. Проверяем new метрики
    print("\n3️⃣ Проверяем new метрики...")
    print(
        f"   enrichment_transparent_alerts: {hasattr(metrics, 'enrichment_transparent_alerts')}"
    )
    print(
        f"   enrichment_enriched_alerts: {hasattr(metrics, 'enrichment_enriched_alerts')}"
    )
    print(f"   enrichment_mode_status: {hasattr(metrics, 'enrichment_mode_status')}")
    print(f"   classification_total: {hasattr(metrics, 'classification_total')}")

    # 4. Проверяем методы
    print("\n4️⃣ Проверяем методы...")
    print(
        f"   increment_alerts_received: {hasattr(metrics, 'increment_alerts_received')}"
    )
    print(
        f"   increment_webhook_errors: {hasattr(metrics, 'increment_webhook_errors')}"
    )
    print(f"   set_enrichment_mode: {hasattr(metrics, 'set_enrichment_mode')}")

    # 5. Тестируем методы
    print("\n5️⃣ Тестируем методы...")
    try:
        metrics.increment_alerts_received("test-alert", "firing")
        print("   ✅ increment_alerts_received работает")
    except Exception as e:
        print(f"   ❌ increment_alerts_received ошибка: {e}")

    try:
        metrics.set_enrichment_mode("transparent_with_recommendations")
        print("   ✅ set_enrichment_mode работает")
    except Exception as e:
        print(f"   ❌ set_enrichment_mode ошибка: {e}")

    try:
        metrics.enrichment_transparent_alerts.inc(1)
        print("   ✅ enrichment_transparent_alerts работает")
    except Exception as e:
        print(f"   ❌ enrichment_transparent_alerts ошибка: {e}")

    print("\n" + "=" * 40)
    print("✅ Тест завершен!")


if __name__ == "__main__":
    test_legacy_metrics()
