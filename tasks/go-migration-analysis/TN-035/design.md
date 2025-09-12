# TN-035: Filter Engine Design

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
