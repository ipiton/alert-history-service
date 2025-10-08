# TN-031: Анализ задачи "Alert Domain Models"

**Дата анализа**: 2025-10-08
**Исполнитель**: AI Assistant
**Статус на начало анализа**: Не начата (галочки не проставлены)
**Ветка**: `feature/TN-031-alert-domain-models` (создана)

---

## 📋 EXECUTIVE SUMMARY

### ✅ Что уже реализовано (частично):
- ✅ Domain модели существуют в `internal/core/interfaces.go`
- ✅ Alert, ClassificationResult, PublishingTarget определены
- ✅ JSON serialization теги присутствуют
- ✅ Код компилируется без ошибок
- ✅ Модели используются в SQLite и PostgreSQL адаптерах
- ✅ Модели используются в миграциях БД

### ❌ Что НЕ выполнено согласно задаче:
- ❌ Модели НЕ вынесены в отдельную структуру `internal/core/domain/`
- ❌ Validation tags (`validator/v10`) НЕ добавлены
- ❌ Unit тесты для domain моделей НЕ созданы
- ❌ Зависимость `github.com/go-playground/validator/v10` НЕ добавлена

### ⚠️ Критические проблемы:
1. **ДУБЛИРОВАНИЕ МОДЕЛЕЙ** - в `internal/infrastructure/llm/client.go` есть дубликат Alert и Classification
2. **НЕСООТВЕТСТВИЕ SEVERITY LEVELS** между design.md и реализацией
3. **ОТСУТСТВИЕ ВАЛИДАЦИИ** - нет runtime validation для входных данных
4. **НЕСООТВЕТСТВИЕ СТРУКТУРЕ ЗАДАЧИ** - код не соответствует tasks.md

---

## 🔍 ДЕТАЛЬНЫЙ АНАЛИЗ

### 1. Соответствие Requirements ↔️ Design ↔️ Реализация

#### Requirements.md требует:
- ✅ Alert struct с полями Alertmanager
- ✅ Classification struct для LLM результатов
- ✅ PublishingTarget struct для внешних систем
- ❌ **Validation tags и JSON serialization** (validation tags отсутствуют)
- ✅ Type safety для всех полей

#### Design.md предлагает:

**Alert:**
```go
type Alert struct {
    Fingerprint  string            `json:"fingerprint" validate:"required"`
    Status       AlertStatus       `json:"status" validate:"required"`
    Labels       map[string]string `json:"labels"`
    Annotations  map[string]string `json:"annotations"`
    StartsAt     time.Time         `json:"startsAt"`
    EndsAt       *time.Time        `json:"endsAt,omitempty"`
    GeneratorURL string            `json:"generatorURL,omitempty"`
}
```

**Severity levels в Design:**
- critical, high, medium, low, info

**Реализация в interfaces.go:**
```go
type Alert struct {
    Fingerprint  string            `json:"fingerprint"`
    AlertName    string            `json:"alert_name"`    // ДОПОЛНИТЕЛЬНОЕ ПОЛЕ
    Status       AlertStatus       `json:"status"`
    Labels       map[string]string `json:"labels"`
    Annotations  map[string]string `json:"annotations"`
    StartsAt     time.Time         `json:"starts_at"`    // РАЗНЫЙ NAMING
    EndsAt       *time.Time        `json:"ends_at,omitempty"`
    GeneratorURL *string           `json:"generator_url,omitempty"` // РАЗНЫЙ ТИП
    Timestamp    *time.Time        `json:"timestamp,omitempty"` // ДОПОЛНИТЕЛЬНОЕ ПОЛЕ
}
```

**Severity levels в реализации:**
- critical, warning, info, noise ⚠️ **НЕСООТВЕТСТВИЕ**

#### ⚠️ ПРОБЛЕМА: Design vs Реализация

| Аспект | Design | Реализация | Статус |
|--------|--------|------------|--------|
| AlertName | ❌ Нет | ✅ Есть | ⚠️ Расширение |
| Timestamp | ❌ Нет | ✅ Есть | ⚠️ Расширение |
| StartsAt naming | `startsAt` | `starts_at` | ⚠️ Разные стили |
| GeneratorURL type | `string` | `*string` | ⚠️ Изменен тип |
| Severity levels | 5 уровней | 4 уровня | ❌ **Несоответствие** |
| Validation tags | ✅ Есть | ❌ Нет | ❌ **НЕ реализовано** |

---

### 2. Дублирование моделей

#### 🚨 КРИТИЧЕСКАЯ ПРОБЛЕМА: Дублирование в llm/client.go

**internal/infrastructure/llm/client.go:**
```go
// Alert represents an alert to be classified.
type Alert struct {
    AlertName   string            `json:"alertname"`
    Status      string            `json:"status"`
    Labels      map[string]string `json:"labels"`
    Annotations map[string]string `json:"annotations"`
    StartsAt    string            `json:"startsAt"`    // ВНИМАНИЕ: string вместо time.Time!
    EndsAt      string            `json:"endsAt"`      // ВНИМАНИЕ: string вместо time.Time!
    Fingerprint string            `json:"fingerprint"`
}

// Classification represents the LLM classification result.
type Classification struct {
    Severity    int      `json:"severity"`       // ВНИМАНИЕ: int вместо AlertSeverity!
    Category    string   `json:"category"`
    Summary     string   `json:"summary"`
    Confidence  float64  `json:"confidence"`
    Reasoning   string   `json:"reasoning"`
    Suggestions []string `json:"suggestions"`
}
```

**internal/core/interfaces.go:**
```go
type Alert struct {
    Fingerprint  string            `json:"fingerprint"`
    AlertName    string            `json:"alert_name"`
    Status       AlertStatus       `json:"status"`      // Typed!
    Labels       map[string]string `json:"labels"`
    Annotations  map[string]string `json:"annotations"`
    StartsAt     time.Time         `json:"starts_at"`   // time.Time!
    EndsAt       *time.Time        `json:"ends_at,omitempty"`
    GeneratorURL *string           `json:"generator_url,omitempty"`
    Timestamp    *time.Time        `json:"timestamp,omitempty"`
}

type ClassificationResult struct {
    Severity        AlertSeverity  `json:"severity"`    // Typed!
    Confidence      float64        `json:"confidence"`
    Reasoning       string         `json:"reasoning"`
    Recommendations []string       `json:"recommendations"`
    ProcessingTime  float64        `json:"processing_time"`
    Metadata        map[string]any `json:"metadata,omitempty"`
}
```

#### 🎯 Рекомендация:
Рефакторинг `llm.Alert` → использовать `core.Alert` + mapper/converter для API LLM proxy.

---

### 3. Отсутствие Validation

#### ❌ Проблема:
- Нет runtime validation через `validator/v10`
- Нет `validate` tags в структурах
- Нет проверки business rules (например, `confidence >= 0 && confidence <= 1`)

#### ✅ Должно быть (согласно Design):
```go
type Alert struct {
    Fingerprint  string            `json:"fingerprint" validate:"required"`
    Status       AlertStatus       `json:"status" validate:"required"`
    // ...
}

type Classification struct {
    Confidence float64 `json:"confidence" validate:"min=0,max=1"`
    // ...
}
```

#### 📦 Отсутствует зависимость:
```bash
$ go list -m all | grep validator
# Пусто!
```

---

### 4. Отсутствие Unit тестов

#### ❌ Проблема:
```bash
$ find go-app/internal/core -name "*_test.go"
# Нет файлов!
```

Согласно tasks.md, должен быть:
- `domain_test.go` с unit тестами
- JSON serialization тесты

#### ✅ Что нужно тестировать:
1. JSON marshaling/unmarshaling
2. Validation rules
3. Методы моделей (Alert.Namespace(), Alert.Severity())
4. Edge cases (nil pointers, empty strings, invalid enums)

---

### 5. Структура директорий

#### Tasks.md требует:
```
internal/core/domain/
  ├── alert.go
  ├── classification.go
  ├── publishing.go
  └── domain_test.go
```

#### Реализация:
```
internal/core/
  └── interfaces.go  (все в одном файле!)
```

#### ⚠️ Оценка:
- **Текущая структура**: Все модели в одном `interfaces.go` (~250 строк)
- **Плюсы текущей структуры**: Проще импортировать, меньше файлов
- **Минусы**: Не соответствует задаче, сложнее поддерживать при росте

#### 🎯 Рекомендация:
**НЕ МЕНЯТЬ структуру** по следующим причинам:
1. Код уже используется в 5+ местах (SQLite, PostgreSQL, handlers, migrations)
2. Рефакторинг потребует изменений во всех зависимых файлах
3. Текущая структура работает и понятна
4. **ОБНОВИТЬ DESIGN И TASKS** чтобы соответствовали реальности

---

## 🚦 ОЦЕНКА СТАТУСА ЗАДАЧИ

### Чек-лист из tasks.md:

- [x] ~~1. Создать internal/core/domain/alert.go~~ ➡️ **Модели в interfaces.go**
- [x] ~~2. Создать internal/core/domain/classification.go~~ ➡️ **Модели в interfaces.go**
- [x] ~~3. Создать internal/core/domain/publishing.go~~ ➡️ **Модели в interfaces.go**
- [ ] **4. Добавить validation tags: `go get github.com/go-playground/validator/v10`** ❌ **НЕ ВЫПОЛНЕНО**
- [ ] **5. Создать domain_test.go с unit тестами** ❌ **НЕ ВЫПОЛНЕНО**
- [ ] **6. Добавить JSON serialization тесты** ❌ **НЕ ВЫПОЛНЕНО**
- [ ] **7. Коммит: `feat(go): TN-031 add domain models`** ❌ **НЕ ВЫПОЛНЕНО**

### Критерии приёмки из requirements.md:

- [x] **Все domain models определены** ✅ **ВЫПОЛНЕНО**
- [x] **JSON tags корректны** ✅ **ВЫПОЛНЕНО**
- [ ] **Validation работает** ❌ **НЕ ВЫПОЛНЕНО**
- [ ] **Unit тесты для моделей** ❌ **НЕ ВЫПОЛНЕНО**

### 📊 Прогресс задачи: **50% завершено**

---

## 🔄 ЗАВИСИМОСТИ И БЛОКЕРЫ

### Связанные задачи:

1. **TN-032: AlertStorage Interface** ✅ УЖЕ ИСПОЛЬЗУЕТ `core.Alert`
2. **TN-033: Alert classification service** ⚠️ ПОТРЕБУЕТ рефакторинг `llm.Alert` → `core.Alert`
3. **TN-041: Alertmanager webhook parser** ⚠️ ПОТРЕБУЕТ валидированные модели
4. **Все задачи Фазы 4+** ⚠️ Зависят от domain models

### ⚠️ Блокеры:

#### ❌ БЛОКЕР #1: Дублирование llm.Alert
**Статус**: Активный блокер
**Влияние**: TN-033, TN-039
**Решение**: Рефакторинг llm client для использования core.Alert

#### ⚠️ БЛОКЕР #2: Отсутствие validation
**Статус**: Средний приоритет
**Влияние**: TN-041 (webhook validation), TN-043 (error handling)
**Решение**: Добавить validator/v10 + validation tags

#### ⚠️ БЛОКЕР #3: Несоответствие severity levels
**Статус**: Требует решение архитектора
**Влияние**: TN-033 (classification service)
**Опции**:
- A) Использовать design (critical, high, medium, low, info)
- B) Использовать реализацию (critical, warning, info, noise) ✅ **РЕКОМЕНДУЕТСЯ**
- C) Объединить (critical, high, medium, low, warning, info, noise)

---

## 🎯 РЕКОМЕНДАЦИИ

### Приоритет 1: КРИТИЧЕСКИЕ ИСПРАВЛЕНИЯ

#### 1.1 Рефакторинг llm.Alert (БЛОКЕР)
```bash
# Создать mapper/converter
internal/infrastructure/llm/
  ├── client.go
  ├── mapper.go          # NEW: core.Alert <-> llm API format
  └── client_test.go
```

**Изменения:**
```go
// mapper.go
func CoreAlertToLLMRequest(alert *core.Alert) *LLMAlertRequest {
    return &LLMAlertRequest{
        AlertName:   alert.AlertName,
        Status:      string(alert.Status),
        Labels:      alert.Labels,
        Annotations: alert.Annotations,
        StartsAt:    alert.StartsAt.Format(time.RFC3339),
        EndsAt:      formatTimePtr(alert.EndsAt),
        Fingerprint: alert.Fingerprint,
    }
}

func LLMClassificationToCoreResult(llmClass *Classification) *core.ClassificationResult {
    return &core.ClassificationResult{
        Severity:        mapIntToSeverity(llmClass.Severity),
        Confidence:      llmClass.Confidence,
        Reasoning:       llmClass.Reasoning,
        Recommendations: llmClass.Suggestions,
    }
}
```

#### 1.2 Добавить Validation
```bash
cd go-app
go get github.com/go-playground/validator/v10
```

**Обновить interfaces.go:**
```go
type Alert struct {
    Fingerprint  string            `json:"fingerprint" validate:"required"`
    AlertName    string            `json:"alert_name" validate:"required"`
    Status       AlertStatus       `json:"status" validate:"required,oneof=firing resolved"`
    Labels       map[string]string `json:"labels"`
    Annotations  map[string]string `json:"annotations"`
    StartsAt     time.Time         `json:"starts_at" validate:"required"`
    // ...
}

type ClassificationResult struct {
    Severity        AlertSeverity `json:"severity" validate:"required,oneof=critical warning info noise"`
    Confidence      float64       `json:"confidence" validate:"gte=0,lte=1"`
    Reasoning       string        `json:"reasoning" validate:"required"`
    Recommendations []string      `json:"recommendations"`
    ProcessingTime  float64       `json:"processing_time" validate:"gte=0"`
    // ...
}
```

#### 1.3 Создать Unit тесты
```bash
# Создать tests
touch go-app/internal/core/models_test.go
```

**Содержимое models_test.go:**
```go
package core_test

import (
    "encoding/json"
    "testing"
    "time"

    "github.com/go-playground/validator/v10"
    "github.com/stretchr/testify/assert"
    "github.com/vitaliisemenov/alert-history/internal/core"
)

func TestAlertValidation(t *testing.T) {
    validate := validator.New()

    tests := []struct {
        name    string
        alert   core.Alert
        wantErr bool
    }{
        {
            name: "valid alert",
            alert: core.Alert{
                Fingerprint: "abc123",
                AlertName:   "TestAlert",
                Status:      core.StatusFiring,
                Labels:      map[string]string{"severity": "critical"},
                Annotations: map[string]string{},
                StartsAt:    time.Now(),
            },
            wantErr: false,
        },
        {
            name: "missing fingerprint",
            alert: core.Alert{
                AlertName: "TestAlert",
                Status:    core.StatusFiring,
                StartsAt:  time.Now(),
            },
            wantErr: true,
        },
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validate.Struct(tt.alert)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}

func TestAlertJSONSerialization(t *testing.T) {
    now := time.Now()
    alert := core.Alert{
        Fingerprint: "test123",
        AlertName:   "TestAlert",
        Status:      core.StatusFiring,
        Labels:      map[string]string{"severity": "critical"},
        Annotations: map[string]string{"description": "Test"},
        StartsAt:    now,
    }

    // Marshal
    data, err := json.Marshal(alert)
    assert.NoError(t, err)

    // Unmarshal
    var decoded core.Alert
    err = json.Unmarshal(data, &decoded)
    assert.NoError(t, err)

    // Compare
    assert.Equal(t, alert.Fingerprint, decoded.Fingerprint)
    assert.Equal(t, alert.AlertName, decoded.AlertName)
    assert.Equal(t, alert.Status, decoded.Status)
}

func TestAlertMethods(t *testing.T) {
    alert := core.Alert{
        Labels: map[string]string{
            "namespace": "production",
            "severity":  "critical",
        },
    }

    ns := alert.Namespace()
    assert.NotNil(t, ns)
    assert.Equal(t, "production", *ns)

    sev := alert.Severity()
    assert.NotNil(t, sev)
    assert.Equal(t, "critical", *sev)
}
```

### Приоритет 2: ДОКУМЕНТАЦИЯ

#### 2.1 Обновить design.md
- ✅ Заменить severity levels: critical, high, medium, low, info → **critical, warning, info, noise**
- ✅ Добавить поля `AlertName` и `Timestamp` в Alert
- ✅ Изменить `GeneratorURL string` → `*string`
- ✅ Изменить naming: `startsAt` → `starts_at` (snake_case для JSON)

#### 2.2 Обновить tasks.md
- ✅ Заменить "Создать internal/core/domain/" → "Модели в internal/core/interfaces.go"
- ✅ Добавить реальный чек-лист выполнения

#### 2.3 Обновить requirements.md
- ✅ Добавить criteria: "Validation tags добавлены и работают"
- ✅ Добавить criteria: "Нет дублирования моделей в llm пакете"

### Приоритет 3: НЕ КРИТИЧНО

#### 3.1 Не рефакторить структуру директорий
**Обоснование**: Код уже работает и используется. Перемещение в `domain/` потребует изменений в 10+ файлах без добавления ценности.

---

## 📈 ПЛАН ЗАВЕРШЕНИЯ ЗАДАЧИ

### Вариант A: Минимальное завершение (2-3 часа)
1. ✅ Добавить `validator/v10` в go.mod
2. ✅ Добавить validation tags в interfaces.go
3. ✅ Создать models_test.go с базовыми тестами
4. ✅ Обновить документацию (design.md, tasks.md)
5. ❌ НЕ рефакторить llm.Alert (отложить в отдельную задачу)

**Результат**: Задача TN-031 формально завершена, но остается технический долг с дублированием.

### Вариант B: Полное завершение (4-6 часов)
1. ✅ Все из Варианта A
2. ✅ Рефакторинг llm.Alert → использовать core.Alert
3. ✅ Создать llm/mapper.go для конвертации
4. ✅ Обновить llm/client_test.go
5. ✅ Обновить llm/integration_test.go

**Результат**: Задача TN-031 полностью завершена, технический долг устранен, код готов для TN-033.

### ✅ РЕКОМЕНДАЦИЯ: Вариант B (Полное завершение)

**Обоснование**:
- Дублирование llm.Alert - это БЛОКЕР для TN-033
- Рефакторинг сейчас проще, чем потом
- Улучшит качество кода и уменьшит технический долг

---

## 📝 ВЫВОДЫ

### ✅ Позитивные аспекты:
1. ✅ Модели определены и работают
2. ✅ JSON serialization корректна
3. ✅ Type safety обеспечена через typed enums
4. ✅ Модели используются во всех компонентах (DB, handlers, migrations)
5. ✅ Код компилируется и работает

### ❌ Проблемы:
1. ❌ **КРИТИЧНО**: Дублирование llm.Alert (блокер для TN-033)
2. ❌ **ВЫСОКИЙ ПРИОРИТЕТ**: Отсутствие validation
3. ❌ **ВЫСОКИЙ ПРИОРИТЕТ**: Отсутствие unit тестов
4. ⚠️ **СРЕДНИЙ ПРИОРИТЕТ**: Несоответствие design.md и реализации
5. ⚠️ **НИЗКИЙ ПРИОРИТЕТ**: Несоответствие структуры директорий

### 📊 Итоговая оценка:
- **Прогресс**: 50% завершено
- **Качество**: 6/10 (работает, но не соответствует задаче)
- **Готовность к production**: ⚠️ Требует доработки (validation + tests)
- **Блокирует другие задачи**: ⚠️ Да (TN-033, TN-041, TN-043)

### 🎯 Следующие шаги:
1. Выбрать вариант завершения (A или B) ✅ **Рекомендуется B**
2. Создать PR с изменениями
3. Обновить статус задачи в tasks.md
4. Убедиться что все тесты проходят
5. Обновить главный tasks.md с новой датой и процентом завершения

---

**Дата завершения анализа**: 2025-10-08
**Время анализа**: ~30 минут
**Следующая ревью**: После имплементации рекомендаций
