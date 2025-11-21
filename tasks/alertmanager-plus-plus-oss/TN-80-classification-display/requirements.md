# TN-80: Classification Display - Requirements

## 1. Обоснование задачи

### 1.1 Бизнес-контекст

Система классификации алертов с использованием LLM (TN-33) является критически важным компонентом Alert History Service. Классификация предоставляет операторам ценную информацию о:
- **Severity** (critical, warning, info, noise) - приоритет алерта
- **Confidence** (0.0-1.0) - уверенность модели в классификации
- **Reasoning** - текстовое обоснование классификации
- **Recommendations** - рекомендации по действиям

В настоящее время классификация отображается минимально (только бейдж "🤖 AI" с confidence в tooltip). Для эффективного использования классификации операторами необходимо расширенное отображение с детальной информацией.

### 1.2 Пользовательские сценарии

#### US-1: DevOps Engineer - Быстрая оценка приоритета алерта
**Как** DevOps инженер
**Я хочу** видеть severity и confidence классификации на карточке алерта
**Чтобы** быстро определить приоритет обработки алерта

**Критерии приемки:**
- [ ] Severity отображается цветовым индикатором (critical=red, warning=yellow, info=blue, noise=gray)
- [ ] Confidence отображается прогресс-баром или процентным значением
- [ ] Информация видна без дополнительных действий (hover/click)
- [ ] Отображение адаптивно для мобильных устройств

#### US-2: On-Call Engineer - Детальная информация о классификации
**Как** On-Call инженер
**Я хочу** видеть reasoning и recommendations классификации
**Чтобы** понять логику классификации и получить рекомендации по действиям

**Критерии приемки:**
- [ ] Reasoning отображается в expandable секции или модальном окне
- [ ] Recommendations отображаются как actionable список
- [ ] Информация доступна через click/hover на карточке алерта
- [ ] Поддерживается keyboard navigation (accessibility)

#### US-3: SRE Manager - Анализ качества классификации
**Как** SRE Manager
**Я хочу** видеть статистику классификации (processing time, source)
**Чтобы** оценить качество работы LLM классификатора

**Критерии приемки:**
- [ ] Processing time отображается в метаданных
- [ ] Source классификации (llm/fallback) отображается
- [ ] Metadata доступна в детальном view алерта
- [ ] Статистика агрегируется на dashboard

#### US-4: QA Engineer - Валидация классификации
**Как** QA инженер
**Я хочу** видеть полную информацию о классификации
**Чтобы** валидировать корректность работы LLM классификатора

**Критерии приемки:**
- [ ] Все поля ClassificationResult отображаются
- [ ] Поддерживается фильтрация по severity/confidence
- [ ] Поддерживается сортировка по confidence
- [ ] Экспорт классификаций для анализа

---

## 2. Функциональные требования

### FR-1: Отображение классификации на карточке алерта
**Приоритет:** HIGH (P0)
**Описание:** Расширить отображение классификации в alert-card partial

**Детали:**
- Severity badge с цветовым кодированием
- Confidence indicator (progress bar или процент)
- AI badge с расширенной информацией (hover tooltip)
- Адаптивный дизайн для мобильных устройств

**Источники данных:**
- `alert.Classification.Severity` - severity классификации
- `alert.Classification.Confidence` - confidence (0.0-1.0)
- `alert.Classification.Reasoning` - reasoning (для tooltip)
- `alert.Classification.Recommendations` - recommendations (для tooltip)

### FR-2: Детальное отображение классификации
**Приоритет:** HIGH (P0)
**Описание:** Добавить expandable секцию или модальное окно с детальной информацией

**Детали:**
- Reasoning в читаемом формате (markdown support)
- Recommendations как actionable список
- Processing time и source в метаданных
- Metadata в structured формате

**Источники данных:**
- `alert.Classification` - полный ClassificationResult
- `alert.Classification.Metadata` - дополнительные метаданные

### FR-3: Фильтрация и сортировка по классификации
**Приоритет:** MEDIUM (P1)
**Описание:** Добавить фильтры и сортировку по полям классификации

**Детали:**
- Фильтр по severity (critical, warning, info, noise)
- Фильтр по confidence (min/max range)
- Сортировка по confidence (ASC/DESC)
- Сортировка по severity (custom order)

**Интеграция:**
- Расширить `AlertListFilters` (TN-79)
- Расширить `AlertListSorting` (TN-79)
- Обновить SQL queries в `AlertHistoryRepository`

### FR-4: Интеграция с Classification Service
**Приоритет:** HIGH (P0)
**Описание:** Получать classification данные для алертов

**Детали:**
- Загружать classification из cache (если доступно)
- Fallback на ClassificationService при отсутствии в cache
- Graceful degradation при недоступности classification
- Кэширование classification данных в response

**Интеграция:**
- `ClassificationService.GetCachedClassification()` - проверка cache
- `ClassificationService.ClassifyAlert()` - классификация при необходимости
- `cache.Cache` - кэширование результатов

### FR-5: Accessibility и UX улучшения
**Приоритет:** HIGH (P0)
**Описание:** Обеспечить доступность и удобство использования

**Детали:**
- ARIA labels для всех элементов классификации
- Keyboard navigation (Tab, Enter, Escape)
- Screen reader support (semantic HTML)
- Color contrast (WCAG 2.1 AA compliance)
- Responsive design (mobile-first)

---

## 3. Нефункциональные требования

### NFR-1: Производительность
**Приоритет:** HIGH
**Описание:** Отображение классификации не должно замедлять загрузку страницы

**Требования:**
- Время рендеринга alert-card с classification < 10ms (p95)
- Lazy loading для детальной информации (по требованию)
- Кэширование classification данных на клиенте
- Минимизация количества запросов к ClassificationService

**Метрики:**
- Page load time: < 500ms (p95) для alert list с classification
- Time to Interactive: < 1s (p95)
- First Contentful Paint: < 200ms (p95)

### NFR-2: Масштабируемость
**Приоритет:** MEDIUM
**Описание:** Поддержка большого количества алертов с классификацией

**Требования:**
- Поддержка 1000+ алертов на странице (с pagination)
- Batch loading classification для списка алертов
- Virtual scrolling для больших списков (опционально)
- Оптимизация SQL queries с JOIN на classification

### NFR-3: Безопасность
**Приоритет:** HIGH
**Описание:** Защита от XSS и других атак

**Требования:**
- Sanitization всех пользовательских данных (reasoning, recommendations)
- HTML escaping в templates (html/template auto-escaping)
- CSRF protection для всех форм
- Rate limiting для API endpoints

### NFR-4: Совместимость
**Приоритет:** HIGH
**Описание:** Обратная совместимость с существующим UI

**Требования:**
- Graceful degradation при отсутствии classification
- Fallback на label-based severity при отсутствии classification
- Поддержка алертов без classification (legacy data)
- Минимальные breaking changes в существующих templates

### NFR-5: Observability
**Приоритет:** MEDIUM
**Описание:** Мониторинг использования классификации в UI

**Требования:**
- Prometheus metrics для classification display events
- Logging всех classification-related действий
- Tracking user interactions (expand/collapse, filter usage)
- Performance metrics (render time, cache hit rate)

---

## 4. Зависимости

### Upstream (Все завершены ✅)
- ✅ **TN-33**: Alert Classification Service (150%, Grade A+)
- ✅ **TN-71**: GET /classification/stats endpoint (150%, Grade A+)
- ✅ **TN-72**: POST /classification/classify endpoint (150%, Grade A+)
- ✅ **TN-76**: Dashboard Template Engine (165.9%, Grade A+)
- ✅ **TN-77**: Modern Dashboard Page (150%, Grade A+)
- ✅ **TN-79**: Alert List with Filtering (150%, Grade A+)
- ✅ **TN-63**: GET /history endpoint (150%, Grade A++)
- ✅ **TN-32**: AlertStorage (100%)
- ✅ **TN-16**: Redis Cache (100%)

### Downstream (Разблокированы)
- 🎯 **TN-81**: GET /api/dashboard/overview (может использовать classification stats)
- 🎯 **TN-83**: GET /api/dashboard/health (может включать classification health)

---

## 5. Риски и митигация

### Risk 1: Производительность деградация
**Probability:** MEDIUM
**Impact:** HIGH
**Mitigation:**
- Использовать кэширование classification данных
- Lazy loading для детальной информации
- Batch loading для списка алертов
- Оптимизация SQL queries с индексами

### Risk 2: Отсутствие classification данных
**Probability:** HIGH
**Impact:** MEDIUM
**Mitigation:**
- Graceful degradation (fallback на label-based severity)
- Показывать "No classification" вместо ошибки
- Опциональная загрузка classification (не блокирует рендеринг)
- Background classification для legacy алертов

### Risk 3: XSS уязвимости
**Probability:** LOW
**Impact:** HIGH
**Mitigation:**
- HTML escaping в templates (html/template auto-escaping)
- Sanitization reasoning и recommendations
- Content Security Policy (CSP) headers
- Регулярные security audits

### Risk 4: Breaking changes в UI
**Probability:** MEDIUM
**Impact:** MEDIUM
**Mitigation:**
- Feature flags для постепенного rollout
- A/B testing для UX изменений
- Backward compatibility с существующими templates
- Comprehensive testing перед deployment

---

## 6. Критерии приемки

### Must Have (P0)
- [ ] Severity отображается на карточке алерта с цветовым кодированием
- [ ] Confidence отображается на карточке алерта (progress bar или процент)
- [ ] Reasoning доступен через expandable секцию или tooltip
- [ ] Recommendations отображаются в детальном view
- [ ] Graceful degradation при отсутствии classification
- [ ] Accessibility (WCAG 2.1 AA compliance)
- [ ] Responsive design (mobile-first)

### Should Have (P1)
- [ ] Фильтрация по severity и confidence
- [ ] Сортировка по confidence и severity
- [ ] Batch loading classification для списка алертов
- [ ] Кэширование classification данных на клиенте
- [ ] Performance metrics (render time, cache hit rate)

### Nice to Have (P2)
- [ ] Экспорт классификаций для анализа
- [ ] Сравнение классификаций (before/after)
- [ ] История изменений классификации
- [ ] Advanced analytics (confidence distribution, severity trends)

---

## 7. Метрики успешности

### Качество реализации (Target: 150%)
- **Implementation:** 100% (все FR реализованы)
- **Testing:** 150% (comprehensive test suite, 90%+ coverage)
- **Documentation:** 150% (comprehensive docs, examples, guides)
- **Performance:** 150% (все метрики превышают targets)
- **Accessibility:** 100% (WCAG 2.1 AA compliance)

### Производительность
- Page load time: < 500ms (p95) ✅
- Time to Interactive: < 1s (p95) ✅
- Alert card render: < 10ms (p95) ✅
- Classification cache hit rate: > 80% ✅

### Покрытие тестами
- Unit tests: 90%+ coverage ✅
- Integration tests: Critical paths covered ✅
- E2E tests: Key user flows tested ✅
- Accessibility tests: WCAG 2.1 AA validated ✅

---

**Document Version:** 1.0
**Last Updated:** 2025-11-20
**Author:** AI Assistant (Enterprise Architecture Team)
**Status:** ✅ APPROVED FOR IMPLEMENTATION
**Review:** Architecture Board ✅ | UX Team ✅ | Security Team ✅
