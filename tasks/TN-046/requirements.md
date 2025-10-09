# TN-046: Kubernetes Client для Secrets Discovery

## 1. Обоснование
Kubernetes client для обнаружения publishing targets из secrets.

## 2. Сценарий
Приложение автоматически обнаруживает новые targets из K8s secrets.

## 3. Требования
- client-go integration
- Watch для secrets changes
- Label selector filtering
- RBAC permissions

## 4. Критерии приёмки
- [ ] K8s client работает
- [ ] Watch events обрабатываются
- [ ] Label filtering функционирует
- [ ] RBAC настроен
