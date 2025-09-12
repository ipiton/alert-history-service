# TN-039: Circuit Breaker для LLM Calls

## 1. Обоснование
Защита от каскадных отказов при недоступности LLM сервиса.

## 2. Сценарий
При сбоях LLM circuit breaker открывается и переключается на fallback.

## 3. Требования
- Circuit breaker pattern
- Configurable thresholds
- Automatic recovery
- Metrics для состояний

## 4. Критерии приёмки
- [ ] Circuit breaker реализован
- [ ] Fallback работает
- [ ] Recovery автоматический
- [ ] Метрики собираются
