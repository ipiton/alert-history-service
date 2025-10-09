# Python Code Cleanup - Requirements

## Обоснование задачи

После успешной миграции критических компонентов на Go (TN-01 до TN-37 завершены), настало время привести в порядок Python кодовую базу. Текущая ситуация:

- ✅ **Go версия**: Infrastructure, Data Layer, Observability готовы (30 задач завершено)
- ✅ **Core Business Logic в Go**: Alert models, Storage, Classification, Enrichment, Filtering, History
- 🔄 **Python код**: 37 файлов, ~15K LOC, всё ещё активен в production
- ⚠️ **Дублирование**: Многие компоненты реализованы в обеих версиях
- 🧹 **Техдолг**: Нет чёткой стратегии по Python коду

### Проблема

1. **Неясный статус**: Какой код использовать? Go или Python?
2. **Дублирование логики**: Одни и те же фичи в двух языках
3. **Увеличенная поддержка**: Нужно поддерживать две кодовые базы
4. **Путаница в deployment**: Какую версию деплоить?
5. **Зависимости**: Python dependencies устаревают, security vulnerabilities

### Зачем чистить сейчас?

1. **Перед масштабной разработкой Go**: Alertmanager++ (TN-121 до TN-180) - 60 новых задач
2. **Clarity**: Чёткое понимание что остаётся, что удаляется
3. **Reference**: Сохранить важные части как reference для Go
4. **Production readiness**: Подготовка к полному переходу на Go
5. **Документация**: Зафиксировать migration path

## Пользовательский сценарий

### Use Case 1: Разработчик начинает новую фичу

**Текущая ситуация:**
```
Разработчик: "Нужно добавить новый endpoint для alerts"
Вопрос: "Делать в Python или Go?"
Проблема: Нет чёткого ответа, нужно анализировать оба проекта
```

**После cleanup:**
```
Разработчик: "Нужно добавить новый endpoint"
Ответ: "Вся разработка идёт в Go (go-app/), Python только для legacy endpoints"
Чёткость: README указывает на Go как primary, Python помечен как deprecated
```

### Use Case 2: DevOps деплоит сервис

**Текущая ситуация:**
```
DevOps: "Какой Docker image использовать?"
Выбор: Python FastAPI или Go binary?
Проблема: Оба рабочие, но непонятно что предпочтительнее
```

**После cleanup:**
```
DevOps: "Смотрю в DEPLOYMENT.md"
Чётко указано: "Go version - primary, Python - legacy (sunset plan)"
Helm chart обновлён с миграционной стратегией
```

### Use Case 3: Security audit

**Текущая ситуация:**
```
Security: "Найдена уязвимость в Python dependency"
Вопрос: "Патчить или можно игнорировать?"
Проблема: Непонятно используется ли этот код в production
```

**После cleanup:**
```
Security: "Уязвимость в deprecated Python модуле"
Решение: "requirements.txt чист, deprecated код изолирован в archive/"
Простое решение: Не патчим, ускоряем миграцию
```

## Ограничения

### Технические
1. **Zero downtime**: Cleanup не должен нарушать работу production
2. **Rollback capability**: Возможность вернуться к Python если что-то сломается
3. **API compatibility**: Сохранить совместимость для клиентов
4. **Data migration**: Не потерять данные при переключении

### Функциональные
1. **Keep what's working**: Если Python код работает лучше - оставляем
2. **Reference preservation**: Сложные алгоритмы сохраняем как reference
3. **Test coverage**: Переносим тесты на Go перед удалением Python
4. **Documentation**: Обновляем docs перед удалением кода

### Бизнес-ограничения
1. **Timeline**: Cleanup не должен блокировать разработку Alertmanager++
2. **Risk**: Минимальный риск для production
3. **Resources**: 1-2 недели максимум на cleanup
4. **Reversibility**: Всё через Git, можно откатить

## Внешние зависимости

### Python Dependencies (requirements.txt)
```python
fastapi==0.104.1
uvicorn==0.24.0
sqlalchemy==2.0.23
redis==5.0.1
openai==1.3.7
pydantic==2.5.2
prometheus-client==0.19.0
# ... и другие (~30 зависимостей)
```

**Вопрос**: Какие из них ещё нужны?

### Go Dependencies (go.mod)
```go
github.com/gin-gonic/gin
github.com/redis/go-redis/v9
gorm.io/gorm
// ... уже реализованы аналоги Python компонентов
```

### Shared Resources
- PostgreSQL schema (одна БД для обеих версий)
- Redis cache (shared state)
- Configuration files (конфликты?)

## Критерии приёмки

### Функциональные

#### Phase 1: Analysis (Завершено за 2 дня)
- [x] Анализ всех 37 Python файлов
- [x] Матрица соответствия: Python component → Go component
- [x] Идентификация критичных компонентов
- [x] Список deprecated vs active Python кода
- [x] Migration readiness report

#### Phase 2: Documentation (Завершено за 2 дня)
- [x] README обновлён (Go primary, Python deprecated)
- [x] MIGRATION.md создан (Python → Go migration guide)
- [x] DEPRECATION.md с timeline sunset
- [x] API compatibility matrix
- [x] Deployment strategy (dual-run, switch, sunset)

#### Phase 3: Code Organization (Завершено за 3 дня)
- [x] Создать `legacy/` директорию для deprecated кода
- [x] Переместить устаревший Python код в `legacy/`
- [x] Обновить imports в оставшемся коде
- [x] Пометить файлы deprecation warnings
- [x] Обновить CI/CD (не тестировать deprecated код)

#### Phase 4: Dependency Cleanup (Завершено за 2 дня)
- [x] requirements.txt - оставить только нужное
- [x] requirements-dev.txt - убрать неиспользуемое
- [x] Dockerfile - optimize для меньшего размера
- [x] Security scan - проверить уязвимости
- [x] Lock файлы обновить

#### Phase 5: Test Migration (Завершено за 3 дня)
- [x] Перенести критичные Python тесты на Go
- [x] Создать compatibility test suite
- [x] E2E тесты для dual-run mode
- [x] Performance comparison (Go vs Python)
- [x] Документировать test gaps

#### Phase 6: Production Transition (Завершено за 2 недели)
- [x] Deploy Go version в production (canary)
- [x] Monitor performance и errors
- [x] Gradual traffic shift (10% → 50% → 100%)
- [x] Python version → read-only mode
- [x] Final deprecation announcement

### Quality Metrics

| Метрика | До Cleanup | После Cleanup | Target |
|---------|------------|---------------|--------|
| Python LOC | 15,000 | <2,000 | <10% |
| Python files | 37 | <5 | Active only |
| Dependencies | ~30 | <10 | Essential only |
| Docker image (Python) | ~500MB | N/A (Go only) | <20MB Go |
| Security vulns | ? | 0 | 0 critical |
| Code duplication | High | None | 0% |
| Documentation clarity | Low | High | Crystal clear |

### Success Criteria

✅ **DONE когда:**
1. README чётко указывает: "Go - primary, Python - deprecated"
2. Все новые фичи идут только в Go
3. Python код изолирован в `legacy/` с deprecation notices
4. requirements.txt содержит только essential deps
5. CI/CD не блокируется на Python lint/test failures
6. Deployment docs обновлены с Go-first strategy
7. Migration guide написан для external users
8. Security audit пройден (no critical vulns)

## Definition of Done

### Documentation
- [x] `requirements.md` (этот файл)
- [x] `design.md` (стратегия cleanup)
- [x] `tasks.md` (детальный чеклист)
- [x] Root `MIGRATION.md` (для users)
- [x] Root `DEPRECATION.md` (timeline)
- [x] Updated root `README.md`

### Code Changes
- [x] `legacy/` directory created
- [x] Python code reorganized
- [x] Deprecation warnings added
- [x] requirements.txt cleaned
- [x] Dockerfile optimized

### Tests
- [x] Compatibility tests pass
- [x] Go tests cover migrated functionality
- [x] E2E tests for transition period

### Production
- [x] Go version deployed
- [x] Metrics show no degradation
- [x] Python sunset date announced
- [x] Rollback plan documented

---

**Priority**: 🟡 HIGH (but не блокирует Alertmanager++ development)
**Estimated effort**: 2 недели
**Timeline**: Can run parallel with TN-122 to TN-136
**Risk Level**: MEDIUM (production impact possible)
