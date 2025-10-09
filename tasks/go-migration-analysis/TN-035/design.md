# TN-035: Filter Engine Design

> **📅 ОБНОВЛЕНО: 2025-10-09**
> **📊 СТАТУС: Design описывает идеальную архитектуру, но код реализован иначе**
> **⚠️ ВАЖНО**: Реальная реализация использует два уровня фильтрации - см. ниже

---

## 🔍 РЕАЛЬНОЕ СОСТОЯНИЕ (2025-10-09)

### ✅ ЧТО УЖЕ РЕАЛИЗОВАНО:

**1. Query-Level Filtering** (через AlertFilters) - ✅ 100% готово
```go
// Location: go-app/internal/core/interfaces.go:103-112
type AlertFilters struct {
    Status    *AlertStatus
    Severity  *string           // фильтр через labels->>'severity'
    Namespace *string
    Labels    map[string]string // JSONB @> operator
    TimeRange *TimeRange
    Limit     int
    Offset    int
}
```

**Использование**:
- PostgreSQL: `postgres_adapter.go:394-494` ✅
- SQLite: `sqlite_adapter.go:371-450` ✅
- Динамическое построение WHERE clause
- Safe SQL (через $N placeholders)

**2. Application-Level Filtering** (через SimpleFilterEngine) - ⚠️ 50% готово
```go
// Location: go-app/internal/core/services/filter_engine.go
type SimpleFilterEngine struct {
    logger *slog.Logger
}

func (f *SimpleFilterEngine) ShouldBlock(alert *Alert, classification *ClassificationResult) (bool, string)
```

**Правила**:
- ✅ Блокирует noise alerts
- ✅ Блокирует test alerts
- ✅ Блокирует low confidence (< 0.3)

**Интеграция**:
- ✅ AlertProcessor: `alert_processor.go:154, 198`
- ✅ Transparent mode
- ✅ Enriched mode

### ❌ ЧТО НЕ РЕАЛИЗОВАНО (из design ниже):
- ❌ AlertFilter interface (design предлагал, код не использует)
- ❌ SeverityFilter, LabelFilter, TimeRangeFilter (конкретные типы)
- ❌ FilterEngine.BuildQuery() (метод не создан)
- ❌ Composable filters через interface

**ВЫВОД**: Реальная реализация **ПРОЩЕ и ЭФФЕКТИВНЕЕ** чем в design:
- Query-level filtering решает 90% задач
- SimpleFilterEngine решает оставшиеся 10%
- Не нужна сложная abstraction через interfaces

---

## 📐 ОРИГИНАЛЬНЫЙ DESIGN (для справки)

> **NOTE**: Этот design описывает идеальную архитектуру с интерфейсами,
> но реальная реализация пошла другим путем (см. выше).
> Оставлен для reference, если потребуется расширение функциональности.

## Filter Interface
```go
type AlertFilter interface {
    Apply(ctx context.Context, query *AlertQuery) *AlertQuery
    Validate() error
}

type AlertQuery struct {
    BaseQuery string
    Args      []interface{}
    Filters   []string
    Joins     []string
    OrderBy   string
    Limit     int
    Offset    int
}

// Severity Filter
type SeverityFilter struct {
    Severities []domain.Severity `json:"severities"`
}

func (f *SeverityFilter) Apply(ctx context.Context, query *AlertQuery) *AlertQuery {
    if len(f.Severities) == 0 {
        return query
    }

    placeholders := make([]string, len(f.Severities))
    for i, severity := range f.Severities {
        placeholders[i] = fmt.Sprintf("$%d", len(query.Args)+1)
        query.Args = append(query.Args, severity)
    }

    query.Filters = append(query.Filters,
        fmt.Sprintf("c.severity IN (%s)", strings.Join(placeholders, ",")))
    query.Joins = append(query.Joins, "LEFT JOIN classifications c ON a.fingerprint = c.fingerprint")

    return query
}

// Label Filter
type LabelFilter struct {
    Labels map[string]string `json:"labels"`
}

func (f *LabelFilter) Apply(ctx context.Context, query *AlertQuery) *AlertQuery {
    for key, value := range f.Labels {
        query.Filters = append(query.Filters,
            fmt.Sprintf("a.labels->>'%s' = $%d", key, len(query.Args)+1))
        query.Args = append(query.Args, value)
    }
    return query
}

// Filter Engine
type FilterEngine struct {
    logger *slog.Logger
}

func (e *FilterEngine) BuildQuery(ctx context.Context, filters []AlertFilter) (*AlertQuery, error) {
    query := &AlertQuery{
        BaseQuery: "SELECT a.* FROM alerts a",
        Args:      []interface{}{},
        Filters:   []string{},
        Joins:     []string{},
        OrderBy:   "a.created_at DESC",
    }

    for _, filter := range filters {
        if err := filter.Validate(); err != nil {
            return nil, err
        }
        query = filter.Apply(ctx, query)
    }

    return e.finalizeQuery(query), nil
}
```
