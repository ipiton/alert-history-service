# TN-036: Alert Deduplication & Fingerprinting

## 1. Обоснование
Система дедупликации алертов по fingerprint и группировка похожих алертов.

## 2. Сценарий
При получении алерта система проверяет, не является ли он дубликатом существующего.

## 3. Требования
- Fingerprint generation по Alertmanager алгоритму
- Deduplication по fingerprint
- Alert grouping по labels
- Update existing alerts при изменении статуса
- Metrics для дубликатов

## 4. Критерии приёмки
- [ ] Fingerprinting работает корректно
- [ ] Дубликаты не создаются
- [ ] Группировка функционирует
- [ ] Метрики собираются
- [ ] Unit тесты покрывают сценарии
