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
- Отсутствие дублирования моделей между пакетами

## 4. Критерии приёмки
- [x] Все domain models определены (в `internal/core/interfaces.go`)
- [x] JSON tags корректны
- [x] Validation tags добавлены и работают (`validator/v10`)
- [x] Unit тесты для моделей (`models_test.go`)
- [x] Дублирование моделей в `llm/client.go` устранено

## 5. Текущий статус (2025-10-08)
- ✅ **Модели определены**: Alert, ClassificationResult, PublishingTarget в `internal/core/interfaces.go`
- ✅ **Используются в коде**: SQLite, PostgreSQL, handlers, migrations
- ✅ **Validation добавлена**: зависимость `validator/v10`, validation tags на всех моделях
- ✅ **Unit тесты созданы**: `models_test.go` (530+ строк comprehensive тестов)
- ✅ **Дублирование устранено**: создан `llm/mapper.go` для конвертации core.Alert ↔️ LLM API

**Прогресс**: 100% ЗАВЕРШЕНО ✅
**Блокеры**: Нет блокеров. Задача полностью завершена.
