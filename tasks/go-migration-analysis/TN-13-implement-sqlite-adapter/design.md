# TN-13: Дизайн SQLite Adapter

## Интерфейс БД
```go
type Database interface {
    SaveAlert(ctx context.Context, alert *Alert) error
    GetAlert(ctx context.Context, fingerprint string) (*Alert, error)
    HealthCheck(ctx context.Context) error
    Close() error
}

type SQLiteDB struct {
    db *sql.DB
}

func NewSQLiteDB(filepath string) (*SQLiteDB, error) {
    db, err := sql.Open("sqlite3", filepath)
    if err != nil {
        return nil, err
    }

    // Auto-migrate
    if err := createSchema(db); err != nil {
        return nil, err
    }

    return &SQLiteDB{db: db}, nil
}
```
