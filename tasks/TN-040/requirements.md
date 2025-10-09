# TN-040: Retry Logic с Exponential Backoff

## 1. Обоснование
Устойчивость к временным сбоям внешних сервисов.

## 2. Сценарий
При временных ошибках система автоматически повторяет запросы.

## 3. Требования
- Exponential backoff
- Jitter для избежания thundering herd
- Configurable retry policies
- Context cancellation support

## 4. Критерии приёмки
- [ ] Retry mechanism работает
- [ ] Backoff корректный
- [ ] Jitter добавлен
- [ ] Context поддерживается
