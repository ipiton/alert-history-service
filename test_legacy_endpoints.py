#!/usr/bin/env python3
"""
Тест legacy endpoints
"""

import requests

BASE_URL = "http://localhost:8000"


def test_legacy_endpoints():
    """Тестирует legacy endpoints"""

    print("🔍 Тестирование legacy endpoints")
    print("=" * 40)

    # Список legacy endpoints для проверки
    endpoints = ["/webhook", "/history", "/report", "/metrics", "/dashboard", "/health"]

    for endpoint in endpoints:
        print(f"🔍 Проверяем {endpoint}...")
        try:
            if endpoint == "/webhook":
                # POST запрос для webhook
                response = requests.post(
                    f"{BASE_URL}{endpoint}",
                    json={"alerts": [], "receiver": "test"},
                    timeout=5,
                )
            else:
                # GET запрос для остальных
                response = requests.get(f"{BASE_URL}{endpoint}", timeout=5)

            if response.status_code == 200:
                print(f"   ✅ {endpoint} - работает (200)")
            elif response.status_code == 404:
                print(f"   ❌ {endpoint} - не найден (404)")
            else:
                print(f"   ⚠️  {endpoint} - статус {response.status_code}")

        except Exception as e:
            print(f"   ❌ {endpoint} - ошибка: {e}")

    print("\n📊 Проверяем доступные routes...")
    try:
        response = requests.get(f"{BASE_URL}/openapi.json")
        if response.status_code == 200:
            openapi = response.json()
            paths = list(openapi.get("paths", {}).keys())
            print(f"   Доступные paths: {len(paths)}")
            for path in sorted(paths):
                print(f"     - {path}")
        else:
            print(f"   ❌ Не удалось получить OpenAPI spec: {response.status_code}")
    except Exception as e:
        print(f"   ❌ Ошибка получения OpenAPI spec: {e}")


if __name__ == "__main__":
    test_legacy_endpoints()
