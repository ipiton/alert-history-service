# alert-history

Helm chart для деплоя Alert History Service (Intelligent Alert Proxy)

## Features
- Intelligent Alert Proxy (`POST /webhook/proxy`)
- Enrichment Mode Toggle (`GET/POST /enrichment/mode`)
- Dynamic Target Discovery (Rootly, PagerDuty, Slack)
- Prometheus метрики и ServiceMonitor

## Monitoring and Metrics

- Service exports Prometheus metrics on the `/metrics` endpoint (port 8080).
- ServiceMonitor is enabled for automatic metric collection (if kube-prometheus-stack is installed).
- Metrics exported:
  - `alert_history_webhook_events_total` — webhook events (by status, alertname, namespace)
  - `alert_history_webhook_errors_total` — webhook processing errors
  - `alert_history_history_queries_total` — history queries
  - `alert_history_report_queries_total` — report queries
  - `alert_history_db_alerts` — number of alerts in the database
  - `alert_history_request_latency_seconds` — request processing time (histogram)

## Quick Start

1. Build Docker image:
   ```bash
   docker build -t alert-history:latest .
   ```

2. Push image to your registry (if needed):
   ```bash
   docker tag alert-history:latest <your-registry>/alert-history:latest
   docker push <your-registry>/alert-history:latest
   ```

3. Install Helm chart:
   ```bash
   helm install alert-history ./helm/alert-history \
     --set image.repository=<your-registry>/alert-history \
      --set image.tag=latest \
      --set postgresql.enabled=true \
      --set redis.enabled=true
   ```

4. Forward port for local test:
   ```bash
   kubectl port-forward svc/alert-history-alert-history 8080:8080
   ```

5. Configure Alertmanager webhook:
   ```yaml
   receivers:
     - name: 'alert-history'
       webhook_configs:
          - url: 'http://alert-history-alert-history:8080/webhook/proxy'
   ```

## Example History Query

```bash
curl 'http://localhost:8080/history?alertname=CPUThrottlingHigh&status=firing&since=2024-06-01T00:00:00'
```

## values.yaml Variables
- `image.repository` — image name
- `image.tag` — image tag
- `persistence.enabled` — enable PVC
- `persistence.size` — PVC size
- `service.port` — service port
- `retentionDays` — alert history retention period in days (default: 30)

## PVC
Alert history is stored in `/data/alert_history.sqlite3` (persistent volume). Old records are automatically deleted after `retentionDays` days.

## ServiceMonitor Example

ServiceMonitor is created automatically:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: <release>-alert-history
spec:
  selector:
    matchLabels:
      app: alert-history
      release: <release>
  endpoints:
    - port: http
      path: /metrics
      interval: 30s
      scrapeTimeout: 10s
```

---

**Author:** @your-team
