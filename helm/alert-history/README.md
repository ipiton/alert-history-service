# alert-history

Helm-чарт для деплоя Alertmanager Alert History Webhook Receiver

## Возможности
- Приём webhook событий от Alertmanager (`/webhook`)
- Хранение истории алертов в SQLite (stateful PVC)
- Выдача истории по HTTP (`/history`)

## Быстрый старт

1. Соберите Docker-образ:
   ```bash
   docker build -t alert-history:latest .
   ```

2. Залейте образ в ваш реестр (если нужно):
   ```bash
   docker tag alert-history:latest <your-registry>/alert-history:latest
   docker push <your-registry>/alert-history:latest
   ```

3. Установите Helm-чарт:
   ```bash
   helm install alert-history ./helm/alert-history \
     --set image.repository=<your-registry>/alert-history \
     --set image.tag=latest
   ```

4. Пробросьте порт для локального теста:
   ```bash
   kubectl port-forward svc/alert-history-alert-history 8080:8080
   ```

5. Настройте Alertmanager webhook:
   ```yaml
   receivers:
     - name: 'alert-history'
       webhook_configs:
         - url: 'http://alert-history-alert-history:8080/webhook'
   ```

## Пример запроса истории

```bash
curl 'http://localhost:8080/history?alertname=CPUThrottlingHigh&status=firing&since=2024-06-01T00:00:00'
```

## Переменные values.yaml
- `image.repository` — имя образа
- `image.tag` — тег образа
- `persistence.enabled` — включить PVC
- `persistence.size` — размер PVC
- `service.port` — порт сервиса

## PVC
История алертов хранится в `/data/alert_history.sqlite3` (persistent volume).

---

**Автор:** @your-team
