# TN-24: Структура Helm Chart

## Файлы
```
helm/alert-history-go/
├── Chart.yaml
├── values.yaml
└── templates/
    ├── deployment.yaml
    ├── service.yaml
    └── configmap.yaml
```

## values.yaml
```yaml
image:
  repository: alert-history-go
  tag: latest
  pullPolicy: IfNotPresent

service:
  type: ClusterIP
  port: 8080

resources:
  limits:
    cpu: 500m
    memory: 256Mi
  requests:
    cpu: 100m
    memory: 128Mi

config:
  database_url: ""
  redis_url: ""
```
