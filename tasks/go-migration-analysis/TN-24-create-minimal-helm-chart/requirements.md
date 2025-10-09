# TN-24: Минимальный Helm Chart

## 1. Обоснование
Helm chart для деплоя Go-версии в Kubernetes.

## 2. Сценарий
`helm install` разворачивает приложение в кластере.

## 3. Требования
- Deployment с Go образом.
- Service для доступа.
- ConfigMap для конфигурации.
- Health probes.
- Resource limits.

## 4. Критерии приёмки
- [ ] Helm chart создан.
- [ ] Deployment работает.
- [ ] Service доступен.
- [ ] Health checks проходят.
