# TN-23: Webhook Endpoint MVP

## 1. Обоснование
Основной endpoint для приёма алертов от Alertmanager.

## 2. Сценарий
Alertmanager отправляет POST /webhook с алертами.

## 3. Требования
- POST /webhook endpoint.
- Парсинг Alertmanager payload.
- Валидация данных.
- Сохранение в БД.
- Ответ 200 OK.

## 4. Критерии приёмки
- [ ] Endpoint принимает webhook.
- [ ] Payload корректно парсится.
- [ ] Данные сохраняются в БД.
- [ ] Интеграционный тест.
