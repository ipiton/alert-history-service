#!/usr/bin/env python3
"""
Скрипт для настройки LLM тестирования
"""

import os
import subprocess
import sys


def setup_llm_test():
    """Настройка LLM для тестирования"""

    print("🤖 Настройка LLM для тестирования рекомендаций")
    print("=" * 50)

    print("\n📋 Необходимые параметры для LLM:")
    print("   - LLM_ENABLED=true")
    print("   - LLM_API_KEY=<ваш API ключ>")
    print("   - LLM_PROXY_URL=<URL прокси>")
    print("   - LLM_MODEL=<модель>")

    print("\n🔧 Пожалуйста, предоставьте следующие данные:")

    # Запрашиваем данные у пользователя
    api_key = input("🔑 LLM API Key: ").strip()
    if not api_key:
        print("❌ API Key обязателен!")
        return False

    proxy_url = input("🌐 LLM Proxy URL (по умолчанию http://localhost:8080): ").strip()
    if not proxy_url:
        proxy_url = "http://localhost:8080"

    model = input("🧠 LLM Model (по умолчанию gpt-4): ").strip()
    if not model:
        model = "gpt-4"

    timeout = input("⏱️  Timeout в секундах (по умолчанию 30): ").strip()
    if not timeout:
        timeout = "30"

    print("\n✅ Настройки LLM:")
    print(
        f"   API Key: {'*' * (len(api_key) - 4) + api_key[-4:] if len(api_key) > 4 else '***'}"
    )
    print(f"   Proxy URL: {proxy_url}")
    print(f"   Model: {model}")
    print(f"   Timeout: {timeout}s")

    # Устанавливаем переменные окружения
    env_vars = {
        "LLM_ENABLED": "true",
        "LLM_API_KEY": api_key,
        "LLM_PROXY_URL": proxy_url,
        "LLM_MODEL": model,
        "LLM_TIMEOUT": timeout,
        "LLM_MAX_RETRIES": "3",
        "LLM_RETRY_DELAY": "1.0",
        "LLM_BATCH_SIZE": "10",
        "LLM_CACHE_TTL": "3600",
    }

    print("\n🚀 Устанавливаем переменные окружения...")
    for key, value in env_vars.items():
        os.environ[key] = value
        print(f"   {key}={value if key != 'LLM_API_KEY' else '***'}")

    # Проверяем конфигурацию
    print("\n🔍 Проверяем конфигурацию...")
    try:
        from config import get_config

        config = get_config()

        if config.llm.enabled:
            print("   ✅ LLM включен")
            print(f"   ✅ Proxy URL: {config.llm.proxy_url}")
            print(f"   ✅ Model: {config.llm.model}")
            print(f"   ✅ Timeout: {config.llm.timeout}s")
            return True
        else:
            print("   ❌ LLM не включен")
            return False

    except Exception as e:
        print(f"   ❌ Ошибка проверки конфигурации: {e}")
        return False


def test_llm_connection():
    """Тестирование подключения к LLM"""

    print("\n🧪 Тестирование подключения к LLM...")

    try:
        from config import get_config
        from src.alert_history.services.llm_client import LLMProxyClient

        config = get_config()

        # Создаем LLM клиент
        llm_client = LLMProxyClient(
            proxy_url=config.llm.proxy_url,
            api_key=config.llm.api_key,
            model=config.llm.model,
            timeout=config.llm.timeout,
            max_retries=config.llm.max_retries,
        )

        # Тестовый запрос
        test_prompt = (
            "Классифицируй этот алерт: High CPU usage detected on web-server-1"
        )

        print("   📤 Отправляем тестовый запрос...")
        response = llm_client.classify_alert(test_prompt)

        if response:
            print("   ✅ LLM ответил успешно!")
            print(f"   📝 Ответ: {response[:100]}...")
            return True
        else:
            print("   ❌ LLM не ответил")
            return False

    except Exception as e:
        print(f"   ❌ Ошибка подключения к LLM: {e}")
        return False


def start_service_with_llm():
    """Запуск сервиса с LLM"""

    print("\n🚀 Запуск сервиса с LLM...")

    try:
        # Запускаем сервис в фоне
        cmd = [
            sys.executable,
            "-m",
            "uvicorn",
            "src.alert_history.main:app",
            "--host",
            "0.0.0.0",
            "--port",
            "8000",
            "--reload",
        ]

        print(f"   📋 Команда: {' '.join(cmd)}")
        print("   ⏳ Запускаем...")

        # Запускаем процесс
        process = subprocess.Popen(
            cmd,
            env=os.environ.copy(),
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
        )

        # Ждем немного
        import time

        time.sleep(5)

        # Проверяем, что процесс запустился
        if process.poll() is None:
            print(f"   ✅ Сервис запущен (PID: {process.pid})")
            return process
        else:
            stdout, stderr = process.communicate()
            print("   ❌ Ошибка запуска сервиса:")
            print(f"   STDOUT: {stdout}")
            print(f"   STDERR: {stderr}")
            return None

    except Exception as e:
        print(f"   ❌ Ошибка запуска: {e}")
        return None


def main():
    """Основная функция"""

    print("🎯 Настройка LLM для тестирования рекомендаций")
    print("=" * 60)

    # 1. Настройка LLM
    if not setup_llm_test():
        print("❌ Не удалось настроить LLM")
        return

    # 2. Тестирование подключения
    if not test_llm_connection():
        print("❌ Не удалось подключиться к LLM")
        print("💡 Проверьте:")
        print("   - Правильность API ключа")
        print("   - Доступность LLM прокси")
        print("   - Настройки сети")
        return

    # 3. Запуск сервиса
    process = start_service_with_llm()
    if not process:
        print("❌ Не удалось запустить сервис")
        return

    print("\n🎉 Настройка завершена!")
    print("📊 Сервис доступен по адресу: http://localhost:8000")
    print("🎛️  Dashboard: http://localhost:8000/dashboard")
    print("📋 API Docs: http://localhost:8000/docs")

    print("\n🧪 Для тестирования выполните:")
    print("   python3 test_all_enrichment_modes.py")

    print("\n⏹️  Для остановки сервиса нажмите Ctrl+C")

    try:
        # Ждем завершения процесса
        process.wait()
    except KeyboardInterrupt:
        print("\n🛑 Останавливаем сервис...")
        process.terminate()
        process.wait()
        print("✅ Сервис остановлен")


if __name__ == "__main__":
    main()
