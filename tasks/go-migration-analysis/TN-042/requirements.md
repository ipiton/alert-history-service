# TN-042: Universal Webhook Handler

## 1. Обоснование
Универсальный обработчик webhook с auto-detection формата.

## 2. Сценарий
Endpoint /webhook принимает различные форматы и автоматически их обрабатывает.

## 3. Требования
- Auto-detection формата payload
- Support Alertmanager, generic webhooks
- Routing к соответствующим parsers
- Error handling и logging

## 4. Критерии приёмки
- [ ] Auto-detection работает
- [ ] Различные форматы поддерживаются
- [ ] Routing корректный
- [ ] Errors логируются
