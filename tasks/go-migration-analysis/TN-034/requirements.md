# TN-034: Enrichment Mode System

**Обновлено**: 2025-10-09
**Статус**: ❌ НЕ НАЧАТА (0%)

## 1. Обоснование
Система переключения между режимами обработки алертов для поддержки различных use case:
- **transparent**: Простое проксирование без LLM (минимальная latency)
- **enriched**: Обогащение через LLM классификацию (production default)
- **transparent_with_recommendations**: Проксирование без фильтрации (bypass all filters)

## 2. Сценарий использования

### Use Case 1: Transparent Mode (Production Debugging)
**Когда**: При отладке проблем с алертами нужно видеть оригинальные данные
**Поведение**:
- Алерты сохраняются без LLM обработки
- Применяется обычная фильтрация
- Минимальная latency

### Use Case 2: Enriched Mode (Default Production)
**Когда**: Нормальная работа с intelligent filtering
**Поведение**:
- Алерты классифицируются через LLM
- Применяется фильтрация по severity/confidence
- Обогащение метаданными

### Use Case 3: Transparent with Recommendations (Special Targets)
**Когда**: Некоторые targets должны получать ВСЕ алерты без фильтрации
**Поведение**:
- Алерты сохраняются без LLM обработки
- **Фильтрация ПРОПУСКАЕТСЯ** (все алерты публикуются)
- Может добавлять рекомендации (опционально)

## 3. Требования

### 3.1 Функциональные требования

#### FR-1: Режимы обработки
- ✅ Три режима: `transparent`, `enriched`, `transparent_with_recommendations`
- ✅ Default режим: `enriched`
- ✅ Режим применяется ко всем входящим алертам

#### FR-2: Хранение состояния (Fallback Chain)
1. **Redis** (primary): `enrichment:mode` key
2. **In-memory**: app state cache
3. **Environment variable**: `ENRICHMENT_MODE`
4. **Default**: `enriched`

#### FR-3: API для управления
- `GET /enrichment/mode` - получить текущий режим + источник
- `POST /enrichment/mode` - установить новый режим
- Validation режима перед установкой
- Response format: `{"mode": "enriched", "source": "redis"}`

#### FR-4: Метрики
- `enrichment_mode_switches_total{from_mode, to_mode}` - счетчик переключений
- `enrichment_mode_status` - текущий режим (gauge: 0=transparent, 1=enriched, 2=transparent_with_recommendations)
- `enrichment_mode_requests_total{method, mode}` - счетчик API requests

#### FR-5: Интеграция с обработкой
- **transparent**: Classification Service НЕ вызывается, фильтрация применяется
- **enriched**: Classification Service вызывается, фильтрация применяется
- **transparent_with_recommendations**: Classification Service НЕ вызывается, фильтрация ПРОПУСКАЕТСЯ

### 3.2 Нефункциональные требования

#### NFR-1: Performance
- Mode resolution < 1ms (из in-memory cache)
- Redis fallback < 10ms
- API response time < 50ms

#### NFR-2: Reliability
- Graceful fallback при недоступности Redis
- Не прерывать активные requests при переключении режима
- Синхронизация между pod'ами через Redis

#### NFR-3: Observability
- Логирование всех переключений режимов
- Метрики по каждому режиму
- Алерты при ошибках Redis

## 4. Критерии приёмки

### Phase 1: Core Infrastructure
- [ ] EnrichmentModeManager реализован с тремя режимами
- [ ] Fallback chain работает (Redis → memory → ENV → default)
- [ ] API endpoints GET/POST /enrichment/mode работают
- [ ] Валидация режима реализована
- [ ] Метрики собираются и экспортируются
- [ ] Unit tests coverage > 80%

### Phase 2: Integration
- [ ] Интеграция с Classification Service (skip в transparent modes)
- [ ] Интеграция с Filter Engine (skip в transparent_with_recommendations)
- [ ] Интеграция в Webhook Processing
- [ ] Integration tests проходят
- [ ] E2E tests для всех трех режимов

### Phase 3: Production Readiness
- [ ] Graceful mode switching (не прерывает requests)
- [ ] Redis Pub/Sub для синхронизации между pods (опционально)
- [ ] API документирован (OpenAPI/Swagger)
- [ ] ENRICHMENT_MODES.md guide создан
- [ ] Performance tests пройдены

## 5. Зависимости

### Требуется (Blockers):
- ✅ TN-16: Redis Cache Wrapper (ГОТОВО)
- ✅ TN-21: Prometheus Metrics (ГОТОВО)
- ⚠️ TN-33: Classification Service (В РАЗРАБОТКЕ) - только для Phase 2

### Блокирует (Downstream):
- TN-35: Alert Filtering Engine (нужен enrichment mode check)
- TN-43: Webhook Validation (может зависеть от режима)

## 6. Ограничения и допущения

### Ограничения:
1. Режим глобальный для всего сервиса (нельзя per-target режимы)
2. Переключение не мгновенное (depends on cache refresh)
3. Redis - single point of truth (при split-brain pods могут расходиться)

### Допущения:
1. Redis доступен в production (fallback на ENV/default для dev)
2. Classification Service опциональный (graceful degradation)
3. Режим меняется редко (не под каждый request)

## 7. Риски

| Риск | Вероятность | Влияние | Митигация |
|------|-------------|---------|-----------|
| Redis недоступен | Medium | High | Fallback на memory + ENV |
| Split-brain в K8s | Low | Medium | Redis Pub/Sub для sync |
| Race condition при переключении | Low | Low | Context-based resolution |
| TN-33 не завершена | High | Medium | Реализовать Phase 1 независимо |

## 8. Ссылки

- Python implementation: `src/alert_history/api/enrichment_endpoints.py`
- Design document: `TN-034/design.md`
- Analysis report: `TN-034/ANALYSIS_REPORT.md`
