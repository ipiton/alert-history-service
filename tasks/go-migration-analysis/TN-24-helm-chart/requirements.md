# TN-24: Создание Helm Chart для alert-history-go

## 🎯 **Цель задачи**

Создать production-ready Helm chart для развертывания Go версии Alert History Service в Kubernetes с поддержкой всех необходимых компонентов и конфигураций.

## 📋 **Функциональные требования**

### **1. Основные компоненты**
- [ ] **Deployment**: Основное приложение alert-history-go
- [ ] **Service**: Kubernetes service для доступа к приложению
- [ ] **ConfigMap**: Конфигурационные файлы и переменные окружения
- [ ] **Secret**: Управление секретами (DB credentials, API keys)
- [ ] **Ingress**: Внешний доступ через ingress controller

### **2. База данных**
- [ ] **PostgreSQL StatefulSet**: Production-ready PostgreSQL
- [ ] **PersistentVolumeClaim**: Хранение данных PostgreSQL
- [ ] **PostgreSQL Service**: Доступ к базе данных
- [ ] **Init Container**: Инициализация базы данных и миграции

### **3. Кэширование и очереди**
- [ ] **Redis Deployment**: Redis для кэширования
- [ ] **Redis Service**: Доступ к Redis
- [ ] **PersistentVolumeClaim**: Хранение Redis данных

### **4. Мониторинг и логирование**
- [ ] **ServiceMonitor**: Интеграция с Prometheus
- [ ] **PodMonitor**: Мониторинг pods
- [ ] **ConfigMap**: Prometheus rules и alerts
- [ ] **NetworkPolicy**: Сетевые политики безопасности

### **5. Масштабируемость**
- [ ] **HorizontalPodAutoscaler**: Автоматическое масштабирование
- [ ] **PodDisruptionBudget**: Гарантии доступности
- [ ] **Resource Limits/Requests**: Управление ресурсами

## 🔧 **Технические требования**

### **Helm Chart Структура**
```
alert-history-go/
├── Chart.yaml                 # Метаданные чарта
├── values.yaml               # Значения по умолчанию
├── templates/
│   ├── deployment.yaml       # Deployment приложения
│   ├── service.yaml          # Service
│   ├── ingress.yaml          # Ingress
│   ├── configmap.yaml        # ConfigMap
│   ├── secret.yaml           # Secret
│   ├── postgresql/
│   │   ├── statefulset.yaml  # PostgreSQL StatefulSet
│   │   ├── service.yaml      # PostgreSQL Service
│   │   ├── pvc.yaml          # PersistentVolumeClaim
│   │   └── secret.yaml       # PostgreSQL credentials
│   ├── redis/
│   │   ├── deployment.yaml   # Redis Deployment
│   │   ├── service.yaml      # Redis Service
│   │   └── pvc.yaml          # Redis PVC
│   ├── monitoring/
│   │   ├── servicemonitor.yaml # Prometheus ServiceMonitor
│   │   ├── podmonitor.yaml    # Prometheus PodMonitor
│   │   └── prometheus-rules.yaml # Alert rules
│   ├── hpa.yaml             # HorizontalPodAutoscaler
│   ├── pdb.yaml             # PodDisruptionBudget
│   ├── networkpolicy.yaml   # NetworkPolicy
│   └── _helpers.tpl         # Helper templates
├── charts/                   # Зависимости (postgresql, redis)
└── README.md                 # Документация
```

### **Конфигурационные параметры**

#### **Application Settings**
```yaml
# values.yaml
image:
  repository: alert-history-go
  tag: "latest"
  pullPolicy: IfNotPresent

replicaCount: 3

env:
  - name: DB_HOST
    value: "{{ .Chart.Name }}-postgresql"
  - name: DB_PORT
    value: "5432"
  - name: DB_NAME
    value: "alerthistory"
  - name: REDIS_HOST
    value: "{{ .Chart.Name }}-redis"

service:
  type: ClusterIP
  port: 8080
  annotations: {}

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: alert-history.local
      paths:
        - path: /
          pathType: Prefix
```

#### **Database Configuration**
```yaml
postgresql:
  enabled: true
  image:
    repository: postgres
    tag: "15-alpine"
  auth:
    postgresPassword: "changeme"
    username: "alerthistory"
    password: "changeme"
    database: "alerthistory"

  persistence:
    enabled: true
    size: 10Gi
    storageClass: "standard"

  resources:
    requests:
      memory: 256Mi
      cpu: 250m
    limits:
      memory: 512Mi
      cpu: 500m
```

#### **Redis Configuration**
```yaml
redis:
  enabled: true
  image:
    repository: redis
    tag: "7-alpine"
  auth:
    password: "changeme"

  persistence:
    enabled: true
    size: 1Gi
    storageClass: "standard"

  resources:
    requests:
      memory: 64Mi
      cpu: 100m
    limits:
      memory: 128Mi
      cpu: 200m
```

## ✅ **Критерии готовности**

### **Chart Quality**
- [ ] **Helm lint**: Чарт проходит валидацию
- [ ] **Template rendering**: Все шаблоны рендерятся корректно
- [ ] **Dependencies**: Все зависимости разрешены
- [ ] **Security**: Нет уязвимостей в конфигурациях

### **Functionality**
- [ ] **Deployment**: Приложение разворачивается успешно
- [ ] **Database**: PostgreSQL инициализируется корректно
- [ ] **Redis**: Redis доступен и работает
- [ ] **Networking**: Все сервисы доступны
- [ ] **Persistence**: Данные сохраняются при перезапуске

### **Production Readiness**
- [ ] **Health checks**: Readiness и liveness probes
- [ ] **Resource limits**: Установлены лимиты ресурсов
- [ ] **Security**: Network policies, RBAC
- [ ] **Monitoring**: ServiceMonitor, PodMonitor
- [ ] **Scaling**: HPA настроен корректно

### **Documentation**
- [ ] **README**: Полная документация по установке
- [ ] **Values**: Описание всех параметров
- [ ] **Examples**: Примеры конфигураций для разных сред
- [ ] **Troubleshooting**: Руководство по устранению проблем

## 🚀 **Тестирование**

### **Unit Tests**
- [ ] **Template tests**: Шаблоны рендерятся корректно
- [ ] **Value validation**: Валидация входных значений
- [ ] **Dependency checks**: Проверка зависимостей

### **Integration Tests**
- [ ] **Local deployment**: Развертывание в локальном кластере
- [ ] **Database connectivity**: Проверка подключения к БД
- [ ] **Service discovery**: Проверка сетевых подключений
- [ ] **Scaling tests**: Тестирование HPA

### **End-to-End Tests**
- [ ] **Full deployment**: Полное развертывание всех компонентов
- [ ] **Application functionality**: Проверка работы API
- [ ] **Data persistence**: Проверка сохранения данных
- [ ] **Failover tests**: Тестирование отказоустойчивости

## 📊 **Performance & Scalability**

### **Resource Requirements**
- **Application**: 256Mi RAM, 0.2 CPU cores (per replica)
- **PostgreSQL**: 512Mi RAM, 0.5 CPU cores
- **Redis**: 128Mi RAM, 0.2 CPU cores

### **Scaling Configuration**
```yaml
hpa:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
  targetMemoryUtilizationPercentage: 80
```

### **Pod Disruption Budget**
```yaml
pdb:
  enabled: true
  minAvailable: 2
```

## 🔒 **Безопасность**

### **Network Policies**
- [ ] **Default deny**: Запрет всех входящих соединений
- [ ] **Application access**: Разрешение доступа к приложению
- [ ] **Database access**: Ограничение доступа к PostgreSQL
- [ ] **Monitoring access**: Доступ для Prometheus

### **RBAC**
- [ ] **Service accounts**: Отдельные SA для компонентов
- [ ] **Roles**: Минимально необходимые права
- [ ] **Pod security**: Security contexts для pods

### **Secrets Management**
- [ ] **External secrets**: Интеграция с external secret manager
- [ ] **Certificate management**: TLS сертификаты
- [ ] **Password rotation**: Автоматическая ротация паролей

## 📈 **Мониторинг**

### **Metrics Collection**
- [ ] **Application metrics**: HTTP requests, latency, errors
- [ ] **Database metrics**: Connections, query performance
- [ ] **Infrastructure metrics**: CPU, memory, network

### **Alerting Rules**
```yaml
groups:
  - name: alert-history
    rules:
      - alert: AlertHistoryDown
        expr: up{job="alert-history"} == 0
        for: 5m
        labels:
          severity: critical
      - alert: AlertHistoryHighCPU
        expr: rate(container_cpu_usage_seconds_total{pod=~"alert-history-.*"}[5m]) > 0.8
        for: 10m
        labels:
          severity: warning
```

### **Dashboards**
- [ ] **Application dashboard**: HTTP metrics, errors, latency
- [ ] **Database dashboard**: Connections, queries, performance
- [ ] **Infrastructure dashboard**: Resources, networking

## 🎯 **Deployment Environments**

### **Development**
```yaml
# values-dev.yaml
replicaCount: 1
image:
  tag: "dev"
postgresql:
  persistence:
    enabled: false
redis:
  persistence:
    enabled: false
```

### **Staging**
```yaml
# values-staging.yaml
replicaCount: 2
ingress:
  hosts:
    - host: alert-history.staging.company.com
postgresql:
  persistence:
    size: 20Gi
```

### **Production**
```yaml
# values-prod.yaml
replicaCount: 5
ingress:
  hosts:
    - host: alert-history.company.com
  tls:
    - secretName: alert-history-tls
      hosts:
        - alert-history.company.com
postgresql:
  persistence:
    size: 100Gi
    storageClass: "fast-ssd"
redis:
  persistence:
    size: 10Gi
    storageClass: "fast-ssd"
```

## 📋 **Implementation Plan**

### **Phase 1: Core Components (1 неделя)**
1. Создание базовой структуры Helm chart
2. Реализация Deployment и Service для приложения
3. Добавление ConfigMap и Secret
4. Настройка Ingress

### **Phase 2: Database Integration (1 неделя)**
1. Добавление PostgreSQL StatefulSet
2. Конфигурация PersistentVolumeClaim
3. Настройка init containers для миграций
4. Тестирование database connectivity

### **Phase 3: Additional Services (3 дня)**
1. Добавление Redis Deployment
2. Настройка ServiceMonitor для Prometheus
3. Создание NetworkPolicy
4. Настройка PodDisruptionBudget

### **Phase 4: Production Readiness (3 дня)**
1. Добавление HPA и resource limits
2. Настройка security contexts
3. Создание comprehensive documentation
4. End-to-end testing

### **Phase 5: Testing & Validation (2 дня)**
1. Тестирование в разных environments
2. Performance testing
3. Security auditing
4. Documentation finalization

## 🎉 **Ожидаемый результат**

Production-ready Helm chart, который:
- ✅ **Полностью автоматизирует** развертывание alert-history-go
- ✅ **Включает все компоненты**: app, PostgreSQL, Redis, monitoring
- ✅ **Production-ready**: security, scaling, monitoring
- ✅ **Environment-flexible**: dev/staging/production configs
- ✅ **Well-documented**: полная документация и examples

**Helm chart станет основным способом развертывания Go версии Alert History Service в production!** 🚀
