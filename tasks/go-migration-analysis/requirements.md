# Анализ возможности переписывания Alert History Service на Go

## Обоснование задачи

### Зачем рассматриваем миграцию на Go?

1. **Производительность**
   - Go компилируется в машинный код, обеспечивая высокую производительность
   - Эффективная работа с памятью и сборщик мусора
   - Нативная поддержка конкурентности через goroutines

2. **Deployment и эксплуатация**
   - Статически скомпилированные бинарники без зависимостей
   - Меньший размер Docker образов
   - Быстрый старт приложения

3. **Масштабируемость**
   - Отличная поддержка многопоточности
   - Низкое потребление ресурсов
   - Эффективная работа под высокой нагрузкой

## Текущее состояние проекта

### Архитектура Python приложения
- **Веб-фреймворк**: FastAPI с async/await
- **База данных**: PostgreSQL (production) + SQLite (dev)
- **Кэширование**: Redis с async клиентом
- **LLM интеграция**: HTTP клиент для LLM proxy
- **Kubernetes**: Native интеграция для target discovery
- **Мониторинг**: Prometheus metrics, structured logging
- **Deployment**: Docker + Helm chart

### Ключевые компоненты
1. **REST API** (9 основных endpoints)
2. **Webhook processing** (Alertmanager интеграция)
3. **LLM classification service** (внешний HTTP API)
4. **Alert publishing** (Rootly, PagerDuty, Slack)
5. **Target discovery** (Kubernetes secrets)
6. **Database layer** (PostgreSQL/SQLite)
7. **Redis caching** (distributed caching)
8. **Health checks** (liveness/readiness probes)
9. **Metrics collection** (Prometheus)
10. **Dashboard** (HTML5 + REST API)

## Требования к миграции

### Функциональные требования
- [ ] 100% совместимость API (все существующие endpoints)
- [ ] Поддержка всех режимов обработки (transparent, enriched, transparent_with_recommendations)
- [ ] LLM интеграция через HTTP proxy
- [ ] Publishing в внешние системы (Rootly, PagerDuty, Slack)
- [ ] Target discovery из Kubernetes secrets
- [ ] Database миграции (PostgreSQL + SQLite)
- [ ] Redis кэширование с TTL
- [ ] Prometheus метрики
- [ ] Health checks (K8s probes)
- [ ] Structured logging (JSON)

### Нефункциональные требования
- [ ] Производительность: >= текущей (FastAPI)
- [ ] Memory footprint: <= текущего
- [ ] Startup time: <= 5 секунд
- [ ] Docker image size: <= текущего размера
- [ ] 12-Factor App compliance
- [ ] Graceful shutdown
- [ ] Horizontal scaling support

### Ограничения
- **Время миграции**: Должна быть поэтапной без downtime
- **Команда**: Текущие разработчики должны освоить Go
- **Тестирование**: Полное покрытие всех сценариев
- **Документация**: Обновление всей документации
- **CI/CD**: Адаптация пайплайнов под Go

## Пользовательские сценарии

### Критически важные сценарии
1. **Webhook processing**: Прием алертов от Alertmanager
2. **LLM classification**: Классификация алертов через внешний API
3. **Alert publishing**: Автоматическая публикация в внешние системы
4. **API compatibility**: Все существующие клиенты должны работать
5. **Dashboard functionality**: HTML5 dashboard с REST API
6. **Health monitoring**: K8s probes и Prometheus metrics

### Вторичные сценарии
1. **Target discovery**: Автоматическое обнаружение publishing targets
2. **Cache management**: Эффективное кэширование классификаций
3. **Database migrations**: Управление схемой БД
4. **Configuration management**: 12-Factor конфигурация через ENV

## Внешние зависимости

### Сохраняемые зависимости
- **PostgreSQL**: Основная БД (asyncpg → pgx/gorm)
- **Redis**: Кэширование (aioredis → go-redis)
- **Kubernetes API**: Target discovery (kubernetes-python → client-go)
- **LLM Proxy**: HTTP API (aiohttp → net/http)
- **Prometheus**: Метрики (prometheus_client → prometheus/client_golang)

### Новые Go зависимости
- **Web framework**: Gin/Fiber/Echo
- **Database**: pgx (PostgreSQL), modernc.org/sqlite
- **Redis client**: go-redis
- **HTTP client**: net/http или resty
- **Kubernetes**: client-go
- **Logging**: logrus/zap
- **Configuration**: viper
- **Testing**: testify

## Критерии успеха

### Технические критерии
- [ ] Все API endpoints работают идентично
- [ ] Производительность не хуже текущей
- [ ] Memory usage снижен на 20-30%
- [ ] Docker image размер уменьшен на 50%+
- [ ] Startup time < 3 секунд
- [ ] Test coverage >= 80%

### Операционные критерии
- [ ] Zero-downtime migration
- [ ] Полная совместимость с Helm chart
- [ ] Все мониторинг dashboards работают
- [ ] Логи структурированы и читаемы
- [ ] Документация обновлена

### Бизнес критерии
- [ ] Снижение затрат на инфраструктуру
- [ ] Улучшение времени отклика
- [ ] Упрощение deployment процесса
- [ ] Повышение надежности системы
