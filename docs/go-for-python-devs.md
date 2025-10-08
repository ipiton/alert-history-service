# Go для Python разработчиков

Добро пожаловать в мир Go! Этот guide поможет Python разработчикам быстро освоить Go и понять ключевые отличия между языками.

## 📋 Table of Contents

- [Основные отличия](#-основные-отличия)
- [Синтаксис и структуры](#-синтаксис-и-структуры)
- [Сравнительная таблица](#-сравнительная-таблица)
- [Практические примеры](#-практические-примеры)
- [Инструменты разработки](#-инструменты-разработки)
- [Паттерны и идиомы](#-паттерны-и-идиомы)
- [Практические задания](#-практические-задания)
- [Ресурсы для изучения](#-ресурсы-для-изучения)

## 🔄 Основные отличия

### 1. Статическая типизация vs Динамическая

**Python (динамическая типизация):**
```python
def process_alert(alert):
    # Тип переменной определяется во время выполнения
    if isinstance(alert, dict):
        return alert.get('status', 'unknown')
    return str(alert)
```

**Go (статическая типизация):**
```go
// Типы определяются во время компиляции
func processAlert(alert Alert) string {
    return alert.Status
}

type Alert struct {
    Status string `json:"status"`
    Name   string `json:"name"`
}
```

### 2. Компиляция vs Интерпретация

**Python:**
- Интерпретируемый язык
- Ошибки обнаруживаются во время выполнения
- Медленнее выполнение

**Go:**
- Компилируемый язык
- Ошибки обнаруживаются во время компиляции
- Быстрое выполнение
- Один исполняемый файл

### 3. Управление памятью

**Python:**
- Автоматическое управление памятью (GC)
- Reference counting + cycle detection

**Go:**
- Автоматическое управление памятью (GC)
- Concurrent mark-and-sweep GC
- Более предсказуемые паузы GC

### 4. Конкурентность

**Python:**
```python
import asyncio
import aiohttp

async def fetch_data(url):
    async with aiohttp.ClientSession() as session:
        async with session.get(url) as response:
            return await response.text()

async def main():
    tasks = [fetch_data(url) for url in urls]
    results = await asyncio.gather(*tasks)
```

**Go:**
```go
func fetchData(url string, ch chan<- string) {
    resp, err := http.Get(url)
    if err != nil {
        ch <- ""
        return
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    ch <- string(body)
}

func main() {
    ch := make(chan string, len(urls))

    for _, url := range urls {
        go fetchData(url, ch) // goroutine
    }

    for i := 0; i < len(urls); i++ {
        result := <-ch
        fmt.Println(result)
    }
}
```

## 📝 Синтаксис и структуры

### Переменные и константы

**Python:**
```python
# Переменные
name = "Alert History"
port = 8080
is_enabled = True

# "Константы" (по соглашению)
DEFAULT_PORT = 8080
MAX_RETRIES = 3
```

**Go:**
```go
// Переменные
var name string = "Alert History"
var port int = 8080
var isEnabled bool = true

// Короткая форма объявления
name := "Alert History"
port := 8080
isEnabled := true

// Константы
const DefaultPort = 8080
const MaxRetries = 3
```

### Функции

**Python:**
```python
def calculate_score(alerts, severity_weight=1.0):
    """Calculate alert score with optional severity weight."""
    if not alerts:
        return 0.0

    total_score = 0
    for alert in alerts:
        score = alert.get('severity', 1) * severity_weight
        total_score += score

    return total_score / len(alerts)
```

**Go:**
```go
// calculateScore calculates alert score with optional severity weight
func calculateScore(alerts []Alert, severityWeight float64) float64 {
    if len(alerts) == 0 {
        return 0.0
    }

    var totalScore float64
    for _, alert := range alerts {
        score := float64(alert.Severity) * severityWeight
        totalScore += score
    }

    return totalScore / float64(len(alerts))
}

// Функция с множественным возвращаемым значением
func processAlert(alert Alert) (string, error) {
    if alert.Name == "" {
        return "", errors.New("alert name is required")
    }

    result := fmt.Sprintf("Processed: %s", alert.Name)
    return result, nil
}
```

### Структуры данных

**Python:**
```python
# Словари
alert = {
    'name': 'HighCPUUsage',
    'severity': 3,
    'labels': {'instance': 'server-01'},
    'annotations': {'summary': 'High CPU usage detected'}
}

# Классы
class Alert:
    def __init__(self, name, severity, labels=None):
        self.name = name
        self.severity = severity
        self.labels = labels or {}

    def is_critical(self):
        return self.severity >= 4

    def __str__(self):
        return f"Alert({self.name}, severity={self.severity})"
```

**Go:**
```go
// Структуры
type Alert struct {
    Name        string            `json:"name"`
    Severity    int               `json:"severity"`
    Labels      map[string]string `json:"labels"`
    Annotations map[string]string `json:"annotations"`
}

// Методы для структур
func (a Alert) IsCritical() bool {
    return a.Severity >= 4
}

func (a Alert) String() string {
    return fmt.Sprintf("Alert(%s, severity=%d)", a.Name, a.Severity)
}

// Конструктор (по соглашению)
func NewAlert(name string, severity int) Alert {
    return Alert{
        Name:        name,
        Severity:    severity,
        Labels:      make(map[string]string),
        Annotations: make(map[string]string),
    }
}
```

### Обработка ошибок

**Python:**
```python
def save_alert(alert):
    try:
        # Валидация
        if not alert.get('name'):
            raise ValueError("Alert name is required")

        # Сохранение в БД
        db.save(alert)
        logger.info(f"Alert saved: {alert['name']}")

    except ValueError as e:
        logger.error(f"Validation error: {e}")
        raise
    except DatabaseError as e:
        logger.error(f"Database error: {e}")
        raise
    except Exception as e:
        logger.error(f"Unexpected error: {e}")
        raise
```

**Go:**
```go
func saveAlert(alert Alert) error {
    // Валидация
    if alert.Name == "" {
        return fmt.Errorf("alert name is required")
    }

    // Сохранение в БД
    if err := db.Save(alert); err != nil {
        return fmt.Errorf("failed to save alert: %w", err)
    }

    slog.Info("Alert saved", "name", alert.Name)
    return nil
}

// Использование
func main() {
    alert := NewAlert("HighCPUUsage", 3)

    if err := saveAlert(alert); err != nil {
        slog.Error("Failed to save alert", "error", err)
        os.Exit(1)
    }
}
```

## 📊 Сравнительная таблица

| Аспект | Python | Go | Комментарий |
|--------|--------|----|-----------|
| **Типизация** | Динамическая | Статическая | Go ловит ошибки на этапе компиляции |
| **Производительность** | ~100ms | ~10ms | Go в 5-10 раз быстрее |
| **Память** | ~50MB | ~10MB | Go более эффективен с памятью |
| **Компиляция** | Нет | Да | Go создает один исполняемый файл |
| **Конкурентность** | async/await | goroutines | Go проще в использовании |
| **Обработка ошибок** | try/except | if err != nil | Go явная, Python скрытая |
| **Пакеты** | pip/poetry | go mod | Go встроенное управление зависимостями |
| **Тестирование** | pytest | go test | Go встроенное тестирование |
| **Форматирование** | black/autopep8 | gofmt | Go стандартное форматирование |
| **Линтинг** | flake8/pylint | golangci-lint | Go комплексный линтер |

### Соответствие библиотек

| Python | Go | Назначение |
|--------|----|-----------|
| `requests` | `net/http` | HTTP клиент |
| `flask/fastapi` | `net/http`, `gin`, `fiber` | Web фреймворки |
| `sqlalchemy` | `database/sql`, `gorm` | ORM/Database |
| `redis-py` | `go-redis` | Redis клиент |
| `pydantic` | `struct tags` | Валидация данных |
| `logging` | `log/slog` | Логирование |
| `json` | `encoding/json` | JSON обработка |
| `os` | `os` | Системные вызовы |
| `time` | `time` | Работа со временем |
| `re` | `regexp` | Регулярные выражения |

## 💡 Практические примеры

### 1. HTTP Server

**Python (Flask):**
```python
from flask import Flask, request, jsonify
import logging

app = Flask(__name__)
logger = logging.getLogger(__name__)

@app.route('/webhook', methods=['POST'])
def webhook():
    try:
        data = request.get_json()

        if not data or 'alertname' not in data:
            return jsonify({'error': 'Invalid payload'}), 400

        # Обработка webhook
        result = process_webhook(data)

        return jsonify({
            'status': 'success',
            'alert_id': result['id']
        })

    except Exception as e:
        logger.error(f"Webhook error: {e}")
        return jsonify({'error': 'Internal error'}), 500

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8080)
```

**Go:**
```go
package main

import (
    "encoding/json"
    "log/slog"
    "net/http"
    "os"
)

type WebhookRequest struct {
    AlertName string `json:"alertname"`
    Status    string `json:"status"`
}

type WebhookResponse struct {
    Status  string `json:"status"`
    AlertID string `json:"alert_id"`
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req WebhookRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        slog.Error("Failed to decode JSON", "error", err)
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    if req.AlertName == "" {
        http.Error(w, "Invalid payload", http.StatusBadRequest)
        return
    }

    // Обработка webhook
    alertID, err := processWebhook(req)
    if err != nil {
        slog.Error("Webhook processing failed", "error", err)
        http.Error(w, "Internal error", http.StatusInternalServerError)
        return
    }

    response := WebhookResponse{
        Status:  "success",
        AlertID: alertID,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func main() {
    http.HandleFunc("/webhook", webhookHandler)

    slog.Info("Server starting on :8080")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        slog.Error("Server failed", "error", err)
        os.Exit(1)
    }
}
```

### 2. Database Operations

**Python (SQLAlchemy):**
```python
from sqlalchemy import create_engine, Column, Integer, String
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import sessionmaker

Base = declarative_base()

class Alert(Base):
    __tablename__ = 'alerts'

    id = Column(Integer, primary_key=True)
    name = Column(String(255), nullable=False)
    status = Column(String(50), nullable=False)

engine = create_engine('postgresql://user:pass@localhost/db')
Session = sessionmaker(bind=engine)

def save_alert(alert_data):
    session = Session()
    try:
        alert = Alert(
            name=alert_data['name'],
            status=alert_data['status']
        )
        session.add(alert)
        session.commit()
        return alert.id
    except Exception as e:
        session.rollback()
        raise
    finally:
        session.close()
```

**Go:**
```go
package main

import (
    "database/sql"
    "fmt"
    _ "github.com/lib/pq"
)

type Alert struct {
    ID     int    `db:"id"`
    Name   string `db:"name"`
    Status string `db:"status"`
}

type AlertRepository struct {
    db *sql.DB
}

func NewAlertRepository(db *sql.DB) *AlertRepository {
    return &AlertRepository{db: db}
}

func (r *AlertRepository) Save(alert Alert) (int, error) {
    query := `
        INSERT INTO alerts (name, status)
        VALUES ($1, $2)
        RETURNING id`

    var id int
    err := r.db.QueryRow(query, alert.Name, alert.Status).Scan(&id)
    if err != nil {
        return 0, fmt.Errorf("failed to save alert: %w", err)
    }

    return id, nil
}

func main() {
    db, err := sql.Open("postgres", "postgresql://user:pass@localhost/db")
    if err != nil {
        panic(err)
    }
    defer db.Close()

    repo := NewAlertRepository(db)

    alert := Alert{
        Name:   "HighCPUUsage",
        Status: "firing",
    }

    id, err := repo.Save(alert)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Alert saved with ID: %d\n", id)
}
```

### 3. Конкурентная обработка

**Python (asyncio):**
```python
import asyncio
import aiohttp
import time

async def process_alert(session, alert):
    try:
        async with session.post('http://api.example.com/classify',
                               json=alert) as response:
            result = await response.json()
            return result
    except Exception as e:
        print(f"Error processing alert: {e}")
        return None

async def process_alerts_batch(alerts):
    async with aiohttp.ClientSession() as session:
        tasks = [process_alert(session, alert) for alert in alerts]
        results = await asyncio.gather(*tasks, return_exceptions=True)
        return results

# Использование
alerts = [{'name': f'alert-{i}'} for i in range(100)]
start = time.time()
results = asyncio.run(process_alerts_batch(alerts))
print(f"Processed {len(results)} alerts in {time.time() - start:.2f}s")
```

**Go:**
```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "sync"
    "time"
)

type Alert struct {
    Name string `json:"name"`
}

type Result struct {
    Alert Alert
    Data  map[string]interface{}
    Error error
}

func processAlert(client *http.Client, alert Alert, results chan<- Result) {
    alertJSON, _ := json.Marshal(alert)

    resp, err := client.Post("http://api.example.com/classify",
                            "application/json",
                            bytes.NewBuffer(alertJSON))
    if err != nil {
        results <- Result{Alert: alert, Error: err}
        return
    }
    defer resp.Body.Close()

    var data map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
        results <- Result{Alert: alert, Error: err}
        return
    }

    results <- Result{Alert: alert, Data: data}
}

func processAlertsBatch(alerts []Alert) []Result {
    client := &http.Client{Timeout: 10 * time.Second}
    results := make(chan Result, len(alerts))

    // Запускаем goroutines
    for _, alert := range alerts {
        go processAlert(client, alert, results)
    }

    // Собираем результаты
    var allResults []Result
    for i := 0; i < len(alerts); i++ {
        result := <-results
        allResults = append(allResults, result)
    }

    return allResults
}

func main() {
    alerts := make([]Alert, 100)
    for i := 0; i < 100; i++ {
        alerts[i] = Alert{Name: fmt.Sprintf("alert-%d", i)}
    }

    start := time.Now()
    results := processAlertsBatch(alerts)
    duration := time.Since(start)

    fmt.Printf("Processed %d alerts in %v\n", len(results), duration)
}
```

## 🛠️ Инструменты разработки

### Управление зависимостями

**Python:**
```bash
# pip
pip install requests flask sqlalchemy

# poetry
poetry add requests flask sqlalchemy
poetry install

# requirements.txt
pip freeze > requirements.txt
pip install -r requirements.txt
```

**Go:**
```bash
# Инициализация модуля
go mod init github.com/user/project

# Добавление зависимости
go get github.com/gin-gonic/gin
go get github.com/lib/pq

# Обновление зависимостей
go mod tidy

# Vendor зависимости
go mod vendor
```

### Тестирование

**Python (pytest):**
```python
import pytest
from myapp import process_alert

def test_process_alert_success():
    alert = {'name': 'test', 'status': 'firing'}
    result = process_alert(alert)
    assert result['status'] == 'processed'

def test_process_alert_invalid():
    with pytest.raises(ValueError):
        process_alert({})

@pytest.fixture
def sample_alert():
    return {'name': 'test', 'status': 'firing'}

def test_with_fixture(sample_alert):
    result = process_alert(sample_alert)
    assert result is not None
```

**Go:**
```go
package main

import (
    "testing"
)

func TestProcessAlert(t *testing.T) {
    tests := []struct {
        name    string
        alert   Alert
        want    string
        wantErr bool
    }{
        {
            name:    "valid alert",
            alert:   Alert{Name: "test", Status: "firing"},
            want:    "processed",
            wantErr: false,
        },
        {
            name:    "empty alert name",
            alert:   Alert{Name: "", Status: "firing"},
            want:    "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := processAlert(tt.alert)
            if (err != nil) != tt.wantErr {
                t.Errorf("processAlert() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("processAlert() = %v, want %v", got, tt.want)
            }
        })
    }
}

// Benchmark test
func BenchmarkProcessAlert(b *testing.B) {
    alert := Alert{Name: "test", Status: "firing"}

    for i := 0; i < b.N; i++ {
        processAlert(alert)
    }
}
```

### Форматирование и линтинг

**Python:**
```bash
# Форматирование
black src/
autopep8 --in-place --recursive src/

# Линтинг
flake8 src/
pylint src/
mypy src/

# Сортировка импортов
isort src/
```

**Go:**
```bash
# Форматирование (встроенное)
go fmt ./...
goimports -w .

# Линтинг
golangci-lint run

# Vet проверки
go vet ./...

# Модули
go mod tidy
```

## 🎯 Паттерны и идиомы

### 1. Error Handling

**Python:**
```python
def risky_operation():
    try:
        result = dangerous_function()
        return result
    except SpecificError as e:
        logger.error(f"Specific error: {e}")
        return None
    except Exception as e:
        logger.error(f"Unexpected error: {e}")
        raise
```

**Go:**
```go
func riskyOperation() (*Result, error) {
    result, err := dangerousFunction()
    if err != nil {
        // Wrap error with context
        return nil, fmt.Errorf("risky operation failed: %w", err)
    }

    return result, nil
}

// Usage
result, err := riskyOperation()
if err != nil {
    slog.Error("Operation failed", "error", err)
    return
}
```

### 2. Configuration

**Python:**
```python
import os
from dataclasses import dataclass

@dataclass
class Config:
    host: str = "localhost"
    port: int = 8080
    debug: bool = False

    @classmethod
    def from_env(cls):
        return cls(
            host=os.getenv("HOST", "localhost"),
            port=int(os.getenv("PORT", "8080")),
            debug=os.getenv("DEBUG", "false").lower() == "true"
        )
```

**Go:**
```go
type Config struct {
    Host  string `mapstructure:"host"`
    Port  int    `mapstructure:"port"`
    Debug bool   `mapstructure:"debug"`
}

func LoadConfig() (*Config, error) {
    viper.SetDefault("host", "localhost")
    viper.SetDefault("port", 8080)
    viper.SetDefault("debug", false)

    viper.AutomaticEnv()

    var config Config
    if err := viper.Unmarshal(&config); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }

    return &config, nil
}
```

### 3. Dependency Injection

**Python:**
```python
class AlertService:
    def __init__(self, db, logger, metrics):
        self.db = db
        self.logger = logger
        self.metrics = metrics

    def process(self, alert):
        self.logger.info(f"Processing alert: {alert.name}")
        self.db.save(alert)
        self.metrics.increment("alerts_processed")
```

**Go:**
```go
type AlertService struct {
    db      AlertRepository
    logger  *slog.Logger
    metrics MetricsCollector
}

func NewAlertService(db AlertRepository, logger *slog.Logger, metrics MetricsCollector) *AlertService {
    return &AlertService{
        db:      db,
        logger:  logger,
        metrics: metrics,
    }
}

func (s *AlertService) Process(alert Alert) error {
    s.logger.Info("Processing alert", "name", alert.Name)

    if err := s.db.Save(alert); err != nil {
        return fmt.Errorf("failed to save alert: %w", err)
    }

    s.metrics.Increment("alerts_processed")
    return nil
}
```

## 📚 Практические задания

### Задание 1: HTTP API
Создайте простой HTTP API с endpoints:
- `GET /alerts` - получить список алертов
- `POST /alerts` - создать новый алерт
- `GET /alerts/{id}` - получить алерт по ID

**Требования:**
- Валидация входных данных
- Обработка ошибок
- JSON ответы
- Логирование

### Задание 2: Database Integration
Реализуйте слой работы с базой данных:
- Подключение к PostgreSQL
- CRUD операции для алертов
- Connection pooling
- Миграции

### Задание 3: Concurrent Processing
Создайте систему обработки алертов:
- Worker pool для обработки
- Graceful shutdown
- Метрики производительности
- Rate limiting

### Задание 4: Testing
Напишите тесты для созданного кода:
- Unit тесты для бизнес-логики
- Integration тесты для API
- Benchmark тесты
- Mocking внешних зависимостей

## 📖 Ресурсы для изучения

### Официальная документация
- [Go Tour](https://tour.golang.org/) - интерактивное введение в Go
- [Go Documentation](https://golang.org/doc/) - официальная документация
- [Effective Go](https://golang.org/doc/effective_go.html) - идиомы и паттерны

### Книги
- **"The Go Programming Language"** by Alan Donovan, Brian Kernighan
- **"Go in Action"** by William Kennedy, Brian Ketelsen, Erik St. Martin
- **"Concurrency in Go"** by Katherine Cox-Buday
- **"Go Web Programming"** by Sau Sheong Chang

### Онлайн курсы
- [Go by Example](https://gobyexample.com/) - практические примеры
- [Gophercises](https://gophercises.com/) - упражнения для практики
- [Go Web Examples](https://gowebexamples.com/) - веб-разработка на Go

### Практические ресурсы
- [Go Playground](https://play.golang.org/) - онлайн редактор
- [Go Time Podcast](https://changelog.com/gotime) - подкаст о Go
- [Awesome Go](https://awesome-go.com/) - список полезных библиотек

### Сообщества
- [Go Forum](https://forum.golangbridge.org/) - форум сообщества
- [r/golang](https://reddit.com/r/golang) - Reddit сообщество
- [Gophers Slack](https://gophers.slack.com/) - Slack сообщество

### YouTube каналы
- **JustForFunc** - программирование на Go
- **GopherCon** - записи конференций
- **Go Class** - обучающие видео

### Полезные инструменты
- [Go Report Card](https://goreportcard.com/) - анализ качества кода
- [pkg.go.dev](https://pkg.go.dev/) - документация пакетов
- [Go Modules](https://blog.golang.org/using-go-modules) - управление зависимостями

## 🎯 Следующие шаги

1. **Изучите основы** - пройдите Go Tour
2. **Практикуйтесь** - решайте задачи на Go by Example
3. **Читайте код** - изучайте популярные Go проекты на GitHub
4. **Пишите код** - создавайте собственные проекты
5. **Участвуйте в сообществе** - задавайте вопросы, помогайте другим

## 🔗 Быстрые ссылки

- [Go Installation](https://golang.org/doc/install)
- [VS Code Go Extension](https://marketplace.visualstudio.com/items?itemName=golang.go)
- [golangci-lint](https://golangci-lint.run/)
- [Go Modules Reference](https://golang.org/ref/mod)

---

**Помните:** Переход с Python на Go - это не только изучение нового синтаксиса, но и изменение мышления. Go поощряет простоту, явность и производительность. Не бойтесь писать больше кода, если это делает его более понятным!

*Удачи в изучении Go! 🚀*

*Последнее обновление: 2025-09-12*
