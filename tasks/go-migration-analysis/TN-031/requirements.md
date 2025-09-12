# TN-031: Alert Domain Models

## 1. Обоснование
Определить основные доменные модели для работы с алертами, классификацией и публикацией.

## 2. Сценарий
Разработчик импортирует domain модели и использует их в сервисах.

## 3. Требования
- Alert struct с полями Alertmanager
- Classification struct для LLM результатов
- PublishingTarget struct для внешних систем
- Validation tags и JSON serialization
- Type safety для всех полей

## 4. Критерии приёмки
- [ ] Все domain models определены
- [ ] JSON tags корректны
- [ ] Validation работает
- [ ] Unit тесты для моделей
