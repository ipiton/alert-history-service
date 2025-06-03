# Alert History Service

Сервис для сбора и анализа истории алертов из Alertmanager через webhook.

---

## Возможности
- Приём событий алертов (firing/resolved) через POST /webhook
- Хранение истории в SQLite (stateful, поддержка PVC)
- Выдача истории по GET /history (фильтры: alertname, status, fingerprint, время)
- Аналитика по истории через GET /report (топ-алерты, flapping, summary)
- Готов к деплою в Kubernetes через Helm-чарт

---

## Быстрый старт (локально)

1. Установите зависимости:
   ```bash
   python3 -m venv .venv
   source .venv/bin/activate
   pip install -r requirements.txt
   ```

2. Запустите сервис:
   ```bash
   uvicorn alert_history_service:app --host 0.0.0.0 --port 8080
   ```

3. Проверьте здоровье:
   ```bash
   curl http://localhost:8080/healthz
   # {"status": "ok"}
   ```

---

## Эндпоинты

### POST /webhook
- Приём событий от Alertmanager (стандартный webhook формат)
- Пример payload: см. [документацию Alertmanager webhook](https://prometheus.io/docs/alerting/latest/configuration/#webhook_config)
- Пример:
  ```bash
  curl -X POST http://localhost:8080/webhook -H 'Content-Type: application/json' -d @example_alert.json
  ```
- Ответ: `{ "result": "ok", "saved": <кол-во событий> }`

### GET /history
- Получение истории алертов
- Поддерживает фильтры:
  - `alertname` — имя алерта
  - `status` — firing/resolved
  - `fingerprint` — уникальный id алерта
  - `since`, `until` — временной диапазон (ISO8601)
  - `limit`, `offset` — пагинация
- Пример:
  ```bash
  curl 'http://localhost:8080/history?alertname=CPUThrottlingHigh&status=firing&since=2024-06-01T00:00:00'
  ```
- Ответ: JSON-массив событий

### GET /report
- Аналитика по истории алертов
- Фильтры:
  - `alertname` — только по конкретному алерту
  - `since`, `until` — временной диапазон
  - `min_flap` — минимальное число смен статуса для flapping (по умолчанию 2)
  - `top` — сколько топ-алертов выводить (по умолчанию 10)
- Пример:
  ```bash
  curl 'http://localhost:8080/report?top=5&min_flap=2'
  ```
- Ответ: summary (топ-алерты, flapping, общее количество событий)

### GET /healthz
- Проверка работоспособности
- Ответ: `{ "status": "ok" }`

---

## Деплой в Kubernetes через Helm

1. Соберите Docker-образ:
   ```bash
   docker build -t alert-history:latest .
   # (опционально) docker push <your-registry>/alert-history:latest
   ```

2. Установите Helm-чарт:
   ```bash
   helm install alert-history ./helm/alert-history \
     --set image.repository=<your-registry>/alert-history \
     --set image.tag=latest
   ```

3. PVC для истории создаётся автоматически (по умолчанию 1Gi).

4. Пробросьте порт для локального теста:
   ```bash
   kubectl port-forward svc/alert-history-alert-history 8080:8080
   ```

---

## Интеграция с Alertmanager

В конфиге Alertmanager добавьте receiver:
```yaml
receivers:
  - name: 'alert-history'
    webhook_configs:
      - url: 'http://alert-history-alert-history:8080/webhook'
```

---

## Примеры аналитики

- **Топ-10 самых шумных алертов:**
  ```bash
  curl 'http://localhost:8080/report?top=10'
  ```
- **Топ-5 flapping-алертов:**
  ```bash
  curl 'http://localhost:8080/report?top=5&min_flap=2'
  ```
- **История по алерту за сутки:**
  ```bash
  curl 'http://localhost:8080/history?alertname=CPUThrottlingHigh&since=2024-06-01T00:00:00'
  ```

---

## Хранение данных
- Все события пишутся в SQLite-файл (`/data/alert_history.sqlite3`)
- Для production рекомендуется использовать PVC (persistent volume)

---

## Расширение
- Можно добавить фильтры по labels, агрегацию, авторизацию, экспорт в CSV/Markdown, фронтенд.
- Пишите пожелания!

---

**Автор:** @your-team
