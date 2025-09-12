# TN-21: Prometheus Metrics Middleware

## 1. Обоснование
Метрики необходимы для мониторинга производительности и SLA.

## 2. Сценарий
Prometheus собирает метрики с /metrics endpoint.

## 3. Требования
- HTTP метрики (RPS, latency, errors).
- Custom метрики бизнес-логики.
- /metrics endpoint.
- Гистограммы и счётчики.

## 4. Критерии приёмки
- [ ] Prometheus клиент интегрирован.
- [ ] Middleware считает метрики.
- [ ] /metrics endpoint работает.
- [ ] Custom метрики добавлены.
