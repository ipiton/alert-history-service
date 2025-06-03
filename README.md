# Alert History Service

A service for collecting and analyzing alert history from Alertmanager via webhook. Supports SQLite storage, REST API, HTML dashboards, Docker and Helm deployment.

---

## Features
- Receives alert events (firing/resolved) via POST /webhook
- Stores alert history in SQLite (stateful, PVC support)
- REST API for history and analytics: /history, /report
- HTML dashboards: classic and grouped (by alert and namespace)
- Analytics: top alerts, flapping, unique incidents
- Ready for Kubernetes deployment via Helm chart

---

## Quick Start

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
uvicorn alert_history_service:app --host 0.0.0.0 --port 8080
```

Health check:
```bash
curl http://localhost:8080/healthz
```

---

## Main Endpoints

- **POST /webhook** — receive events from Alertmanager (webhook)
- **GET /history** — alert history (filters: alertname, status, fingerprint, namespace, time)
- **GET /report** — analytics (top alerts, flapping, summary)
- **GET /dashboard** — HTML dashboard (top, flapping, filters)
- **GET /dashboard_grouped** — grouped dashboard (top alerts and namespaces, with unique incidents)
- **POST /fill_namespaces** — retroactively fill namespace for old events
- **GET /healthz** — health check

---

## Dashboards

- `/dashboard` — top alerts and flapping, filters by alertname, namespace, time
- `/dashboard_grouped` — top alerts and namespaces, for each: total events and unique incidents (repeat_interval excluded)

---

## Deployment: Docker & Helm

1. Build Docker image:
   ```bash
   docker build -t alert-history:latest .
   # docker push <your-registry>/alert-history:latest
   ```
2. Install Helm chart:
   ```bash
   helm install alert-history ./helm/alert-history \
     --set image.repository=<your-registry>/alert-history \
     --set image.tag=latest
   ```
3. PVC for history is created automatically (default 1Gi).

---

## Alertmanager Integration

In Alertmanager config:
```yaml
receivers:
  - name: 'alert-history'
    webhook_configs:
      - url: 'http://alert-history-alert-history:8080/webhook'
```

---

## Analytics Examples

- Top-10 noisiest alerts:
  ```bash
  curl 'http://localhost:8080/report?top=10'
  ```
- Top-5 flapping alerts:
  ```bash
  curl 'http://localhost:8080/report?top=5&min_flap=2'
  ```
- History for an alert for a day:
  ```bash
  curl 'http://localhost:8080/history?alertname=CPUThrottlingHigh&since=2024-06-01T00:00:00'
  ```

---

## .gitignore (recommended)
```
.venv/
__pycache__/
*.pyc
*.sqlite3
.DS_Store
```

---

## Author

@VitalySemenov

![MIT License](https://img.shields.io/badge/license-MIT-green)
![Docker](https://img.shields.io/badge/docker-ready-blue)
![Helm](https://img.shields.io/badge/helm-chart-blue)
