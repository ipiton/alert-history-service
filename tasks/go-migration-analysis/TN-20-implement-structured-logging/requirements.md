# TN-20: Structured Logging с slog

## 1. Обоснование
Структурированные логи упрощают анализ и мониторинг в production.

## 2. Сценарий
Приложение выводит JSON логи с контекстом и уровнями.

## 3. Требования
- Использовать slog (Go 1.21+).
- JSON формат для production.
- Уровни: DEBUG, INFO, WARN, ERROR.
- Контекст (request ID, user ID).

## 4. Критерии приёмки
- [x] Logger инициализирован через pkg/logger.NewLogger().
- [x] JSON вывод работает с настройкой format="json".
- [x] Request ID генерируется и добавляется в контекст.
- [x] Middleware для HTTP логирования реализован для net/http.
- [x] Поддержка ротации файлов через lumberjack.
- [x] Интеграция с internal/config.LogConfig.
- [x] Тесты покрывают основную функциональность.
- [x] Замена inline slog в cmd/server/main.go.
- [x] Логирование включает: method, path, status, duration, request_id.
- [x] Поддержка уровней: DEBUG, INFO, WARN, ERROR.
- [x] Поддержка вывода: stdout, stderr, file.
