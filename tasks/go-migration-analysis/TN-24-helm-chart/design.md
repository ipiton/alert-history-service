# TN-24: Архитектура Helm Chart для alert-history-go

## 🏗️ **АРХИТЕКТУРНЫЙ ОБЗОР**

### **Цель и обоснование**

Helm chart является критически важным компонентом для production deployment Go версии Alert History Service. Он обеспечивает:

- **Автоматизированное развертывание** всех компонентов
- **Конфигурационное управление** для разных сред
- **Масштабируемость** и высокую доступность
- **Мониторинг** и наблюдаемость
- **Безопасность** на уровне инфраструктуры

## 📋 **АРХИТЕКТУРА HELM CHART**

### **Общая архитектура развертывания**

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Kubernetes Cluster                           │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │                    Ingress Controller                          │ │
│  │  ┌─────────────────────────────────────────────────────────────┐ │
│  │  │                    alert-history-go                         │ │
│  │  │  ┌─────────────────────────────────────────────────────────┐ │ │
│  │  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │ │ │
│  │  │  │  │  Deployment │  │   Service   │  │  ConfigMap  │     │ │ │
│  │  │  │  │             │  │             │  │             │     │ │ │
│  │  │  │  │ • 3 replicas│  │ • ClusterIP │  │ • App config│     │ │ │
│  │  │  │  │ • Resources │  │ • Port 8080 │  │ • Env vars   │     │ │ │
│  │  │  │  │ • Probes    │  │             │  │             │     │ │ │
│  │  │  │  └─────────────┘  └─────────────┘  └─────────────┘     │ │ │
│  │  │  └─────────────────────────────────────────────────────────┘ │ │
│  │  └─────────────────────────────────────────────────────────────┘ │
│  └─────────────────────────────────────────────────────────────────┘ │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │                      PostgreSQL StatefulSet                     │ │
│  │  ┌─────────────────────────────────────────────────────────────┐ │ │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │ │ │
│  │  │  │ StatefulSet │  │   Service   │  │     PVC     │         │ │ │
│  │  │  │             │  │             │  │             │         │ │ │
│  │  │  │ • 1 replica │  │ • ClusterIP │  │ • 10Gi SSD  │         │ │ │
│  │  │  │ • PVC       │  │ • Port 5432 │  │ • Retention │         │ │ │
│  │  │  │ • Init      │  │             │  │             │         │ │ │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘         │ │ │
│  │  └─────────────────────────────────────────────────────────────┘ │ │
│  └─────────────────────────────────────────────────────────────────┘ │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │                        Redis Deployment                        │ │
│  │  ┌─────────────────────────────────────────────────────────────┐ │ │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │ │ │
│  │  │  │ Deployment  │  │   Service   │  │     PVC     │         │ │ │
│  │  │  │             │  │             │  │             │         │ │ │
│  │  │  │ • 1 replica │  │ • ClusterIP │  │ • 1Gi SSD   │         │ │ │
│  │  │  │ • PVC       │  │ • Port 6379 │  │ • Snapshots │         │ │ │
│  │  │  │ • Auth      │  │             │  │             │         │ │ │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘         │ │ │
│  │  └─────────────────────────────────────────────────────────────┘ │ │
│  └─────────────────────────────────────────────────────────────────┘ │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │                     Monitoring Stack                           │ │
│  │  ┌─────────────────────────────────────────────────────────────┐ │ │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │ │ │
│  │  │  │ServiceMonitor│  │PodMonitor │  │Prometheus   │         │ │ │
│  │  │  │             │  │            │  │Rules        │         │ │ │
│  │  │  │ • App metrics│  │ • Container│  │ • Alerts    │         │ │ │
│  │  │  │ • Endpoints │  │  metrics   │  │ • SLOs      │         │ │ │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘         │ │ │
│  │  └─────────────────────────────────────────────────────────────┘ │ │
│  └─────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

## 🔧 **СТРУКТУРА HELM CHART**

### **Корневая структура**

```
alert-history-go/
├── Chart.yaml                 # Метаданные чарта
├── values.yaml               # Значения по умолчанию
├── values-dev.yaml          # Конфигурация для development
├── values-staging.yaml      # Конфигурация для staging
├── values-prod.yaml         # Конфигурация для production
├── templates/
│   ├── _helpers.tpl         # Helper функции
│   ├── deployment.yaml      # Deployment приложения
│   ├── service.yaml         # Service для приложения
│   ├── ingress.yaml         # Ingress правила
│   ├── configmap.yaml       # ConfigMap для конфигурации
│   ├── secret.yaml          # Secret для чувствительных данных
│   ├── hpa.yaml            # HorizontalPodAutoscaler
│   ├── pdb.yaml            # PodDisruptionBudget
│   ├── networkpolicy.yaml  # NetworkPolicy
│   ├── postgresql/
│   │   ├── statefulset.yaml # PostgreSQL StatefulSet
│   │   ├── service.yaml     # PostgreSQL Service
│   │   ├── pvc.yaml         # PersistentVolumeClaim
│   │   ├── secret.yaml      # PostgreSQL credentials
│   │   └── configmap.yaml   # PostgreSQL configuration
│   ├── redis/
│   │   ├── deployment.yaml  # Redis Deployment
│   │   ├── service.yaml     # Redis Service
│   │   ├── pvc.yaml         # Redis PVC
│   │   ├── secret.yaml      # Redis credentials
│   │   └── configmap.yaml   # Redis configuration
│   ├── monitoring/
│   │   ├── servicemonitor.yaml # ServiceMonitor
│   │   ├── podmonitor.yaml    # PodMonitor
│   │   ├── prometheus-rules.yaml # Alert rules
│   │   └── grafana-dashboard.yaml # Grafana dashboard
│   └── tests/
│       ├── test-connection.yaml  # Test pod для проверки connectivity
│       └── test-database.yaml    # Test pod для проверки database
├── charts/                   # Зависимости от других чартов
│   ├── postgresql-*.tgz     # PostgreSQL chart (опционально)
│   └── redis-*.tgz          # Redis chart (опционально)
└── README.md                # Документация
```

### **Chart.yaml**

```yaml
apiVersion: v2
name: alert-history-go
description: A Helm chart for Alert History Service (Go version)
type: application
version: 0.1.0
appVersion: "1.0.0"
keywords:
  - alert
  - monitoring
  - history
  - go
  - kubernetes
home: https://github.com/your-org/alert-history
sources:
  - https://github.com/your-org/alert-history
maintainers:
  - name: Your Team
    email: team@yourcompany.com
dependencies:
  - name: postgresql
    version: "12.x.x"
    repository: "https://charts.bitnami.com/bitnami"
    condition: postgresql.enabled
  - name: redis
    version: "17.x.x"
    repository: "https://charts.bitnami.com/bitnami"
    condition: redis.enabled
```

## 🔄 **КОМПОНЕНТЫ СИСТЕМЫ**

### **1. Application Deployment**

#### **Deployment Template**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "alert-history-go.fullname" . }}
  labels:
    {{- include "alert-history-go.labels" . | nindent 4 }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      {{- include "alert-history-go.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "alert-history-go.selectorLabels" . | nindent 8 }}
    spec:
      containers:
        - name: {{ .Chart.Name }}
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          ports:
            - name: http
              containerPort: 8080
              protocol: TCP
          env:
            {{- range .Values.env }}
            - name: {{ .name }}
              value: {{ .value | quote }}
            {{- end }}
            - name: DB_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: {{ include "alert-history-go.fullname" . }}-db
                  key: password
          livenessProbe:
            httpGet:
              path: /health
              port: http
            initialDelaySeconds: 30
            periodSeconds: 10
            timeoutSeconds: 5
            failureThreshold: 3
          readinessProbe:
            httpGet:
              path: /ready
              port: http
            initialDelaySeconds: 5
            periodSeconds: 5
            timeoutSeconds: 3
            failureThreshold: 3
          resources:
            {{- toYaml .Values.resources | nindent 12 }}
          securityContext:
            {{- toYaml .Values.securityContext | nindent 12 }}
      {{- with .Values.nodeSelector }}
      nodeSelector:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.tolerations }}
      tolerations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
```

#### **Service Template**
```yaml
apiVersion: v1
kind: Service
metadata:
  name: {{ include "alert-history-go.fullname" . }}
  labels:
    {{- include "alert-history-go.labels" . | nindent 4 }}
spec:
  type: {{ .Values.service.type }}
  ports:
    - port: {{ .Values.service.port }}
      targetPort: http
      protocol: TCP
      name: http
  selector:
    {{- include "alert-history-go.selectorLabels" . | nindent 4 }}
```

### **2. PostgreSQL StatefulSet**

#### **StatefulSet Template**
```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: {{ include "alert-history-go.fullname" . }}-postgresql
  labels:
    {{- include "alert-history-go.labels" . | nindent 4 }}
    app.kubernetes.io/component: postgresql
spec:
  serviceName: {{ include "alert-history-go.fullname" . }}-postgresql
  replicas: 1
  selector:
    matchLabels:
      {{- include "alert-history-go.selectorLabels" . | nindent 6 }}
      app.kubernetes.io/component: postgresql
  template:
    metadata:
      labels:
        {{- include "alert-history-go.selectorLabels" . | nindent 8 }}
        app.kubernetes.io/component: postgresql
    spec:
      containers:
        - name: postgresql
          image: "{{ .Values.postgresql.image.repository }}:{{ .Values.postgresql.image.tag }}"
          ports:
            - name: postgresql
              containerPort: 5432
              protocol: TCP
          env:
            - name: POSTGRESQL_DATABASE
              value: {{ .Values.postgresql.auth.database | quote }}
            - name: POSTGRESQL_USERNAME
              value: {{ .Values.postgresql.auth.username | quote }}
            - name: POSTGRESQL_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: {{ include "alert-history-go.fullname" . }}-postgresql
                  key: password
          volumeMounts:
            - name: data
              mountPath: /bitnami/postgresql
          livenessProbe:
            exec:
              command:
                - /bin/sh
                - -c
                - exec pg_isready -U {{ .Values.postgresql.auth.username }} -h 127.0.0.1 -p 5432
            initialDelaySeconds: 30
            periodSeconds: 10
            timeoutSeconds: 5
            failureThreshold: 6
          readinessProbe:
            exec:
              command:
                - /bin/sh
                - -c
                - exec pg_isready -U {{ .Values.postgresql.auth.username }} -h 127.0.0.1 -p 5432
            initialDelaySeconds: 5
            periodSeconds: 5
            timeoutSeconds: 3
            failureThreshold: 3
          resources:
            {{- toYaml .Values.postgresql.resources | nindent 12 }}
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: {{ include "alert-history-go.fullname" . }}-postgresql
  volumeClaimTemplates:
    - metadata:
        name: data
      spec:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: {{ .Values.postgresql.persistence.size }}
        {{- if .Values.postgresql.persistence.storageClass }}
        storageClassName: {{ .Values.postgresql.persistence.storageClass }}
        {{- end }}
```

### **3. Redis Deployment**

#### **Redis Deployment Template**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "alert-history-go.fullname" . }}-redis
  labels:
    {{- include "alert-history-go.labels" . | nindent 4 }}
    app.kubernetes.io/component: redis
spec:
  replicas: 1
  selector:
    matchLabels:
      {{- include "alert-history-go.selectorLabels" . | nindent 6 }}
      app.kubernetes.io/component: redis
  template:
    metadata:
      labels:
        {{- include "alert-history-go.selectorLabels" . | nindent 8 }}
        app.kubernetes.io/component: redis
    spec:
      containers:
        - name: redis
          image: "{{ .Values.redis.image.repository }}:{{ .Values.redis.image.tag }}"
          ports:
            - name: redis
              containerPort: 6379
              protocol: TCP
          env:
            - name: REDIS_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: {{ include "alert-history-go.fullname" . }}-redis
                  key: password
          command:
            - redis-server
            - --requirepass $(REDIS_PASSWORD)
            - --appendonly yes
          volumeMounts:
            - name: data
              mountPath: /data
          livenessProbe:
            tcpSocket:
              port: redis
            initialDelaySeconds: 30
            periodSeconds: 10
            timeoutSeconds: 5
          readinessProbe:
            tcpSocket:
              port: redis
            initialDelaySeconds: 5
            periodSeconds: 5
            timeoutSeconds: 3
          resources:
            {{- toYaml .Values.redis.resources | nindent 12 }}
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: {{ include "alert-history-go.fullname" . }}-redis
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: {{ include "alert-history-go.fullname" . }}-redis
  labels:
    {{- include "alert-history-go.labels" . | nindent 4 }}
    app.kubernetes.io/component: redis
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: {{ .Values.redis.persistence.size }}
  {{- if .Values.redis.persistence.storageClass }}
  storageClassName: {{ .Values.redis.persistence.storageClass }}
  {{- end }}
```

### **4. Monitoring & Observability**

#### **ServiceMonitor Template**
```yaml
{{- if .Values.monitoring.enabled }}
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: {{ include "alert-history-go.fullname" . }}
  labels:
    {{- include "alert-history-go.labels" . | nindent 4 }}
    app.kubernetes.io/component: monitoring
spec:
  selector:
    matchLabels:
      {{- include "alert-history-go.selectorLabels" . | nindent 6 }}
  endpoints:
    - port: http
      path: /metrics
      interval: 30s
      scrapeTimeout: 10s
  namespaceSelector:
    matchNames:
      - {{ .Release.Namespace }}
{{- end }}
```

#### **Prometheus Rules**
```yaml
{{- if .Values.monitoring.enabled }}
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: {{ include "alert-history-go.fullname" . }}-alerts
  labels:
    {{- include "alert-history-go.labels" . | nindent 4 }}
    app.kubernetes.io/component: monitoring
spec:
  groups:
    - name: alert-history-go
      rules:
        - alert: AlertHistoryGoDown
          expr: up{job="{{ include "alert-history-go.fullname" . }}"} == 0
          for: 5m
          labels:
            severity: critical
          annotations:
            summary: "Alert History Go is down"
            description: "Alert History Go has been down for more than 5 minutes."

        - alert: AlertHistoryGoHighCPU
          expr: rate(container_cpu_usage_seconds_total{pod=~"{{ include "alert-history-go.fullname" . }}-.*"}[5m]) > 0.8
          for: 10m
          labels:
            severity: warning
          annotations:
            summary: "High CPU usage on Alert History Go"
            description: "CPU usage is above 80% for more than 10 minutes."
{{- end }}
```

### **5. Scaling & High Availability**

#### **HorizontalPodAutoscaler**
```yaml
{{- if .Values.hpa.enabled }}
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: {{ include "alert-history-go.fullname" . }}
  labels:
    {{- include "alert-history-go.labels" . | nindent 4 }}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: {{ include "alert-history-go.fullname" . }}
  minReplicas: {{ .Values.hpa.minReplicas }}
  maxReplicas: {{ .Values.hpa.maxReplicas }}
  metrics:
    {{- if .Values.hpa.targetCPUUtilizationPercentage }}
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: {{ .Values.hpa.targetCPUUtilizationPercentage }}
    {{- end }}
    {{- if .Values.hpa.targetMemoryUtilizationPercentage }}
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: {{ .Values.hpa.targetMemoryUtilizationPercentage }}
    {{- end }}
{{- end }}
```

#### **PodDisruptionBudget**
```yaml
{{- if .Values.pdb.enabled }}
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: {{ include "alert-history-go.fullname" . }}
  labels:
    {{- include "alert-history-go.labels" . | nindent 4 }}
spec:
  {{- if .Values.pdb.minAvailable }}
  minAvailable: {{ .Values.pdb.minAvailable }}
  {{- end }}
  {{- if .Values.pdb.maxUnavailable }}
  maxUnavailable: {{ .Values.pdb.maxUnavailable }}
  {{- end }}
  selector:
    matchLabels:
      {{- include "alert-history-go.selectorLabels" . | nindent 4 }}
{{- end }}
```

## 🔒 **БЕЗОПАСНОСТЬ**

### **Network Policies**
```yaml
{{- if .Values.networkPolicy.enabled }}
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: {{ include "alert-history-go.fullname" . }}
  labels:
    {{- include "alert-history-go.labels" . | nindent 4 }}
spec:
  podSelector:
    matchLabels:
      {{- include "alert-history-go.selectorLabels" . | nindent 4 }}
  policyTypes:
    - Ingress
    - Egress
  ingress:
    # Allow traffic from ingress controller
    - from:
        - namespaceSelector:
            matchLabels:
              name: ingress-nginx
      ports:
        - protocol: TCP
          port: 8080
    # Allow traffic from Prometheus
    - from:
        - namespaceSelector:
            matchLabels:
              name: monitoring
      ports:
        - protocol: TCP
          port: 8080
  egress:
    # Allow DNS resolution
    - to: []
      ports:
        - protocol: UDP
          port: 53
    # Allow traffic to PostgreSQL
    - to:
        - podSelector:
            matchLabels:
              app.kubernetes.io/name: postgresql
      ports:
        - protocol: TCP
          port: 5432
    # Allow traffic to Redis
    - to:
        - podSelector:
            matchLabels:
              app.kubernetes.io/name: redis
      ports:
        - protocol: TCP
          port: 6379
{{- end }}
```

### **Security Contexts**
```yaml
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 1000
    runAsGroup: 1000
    fsGroup: 1000
  containers:
    - securityContext:
        allowPrivilegeEscalation: false
        readOnlyRootFilesystem: true
        runAsNonRoot: true
        runAsUser: 1000
        capabilities:
          drop:
            - ALL
```

## 📊 **КОНФИГУРАЦИИ ДЛЯ РАЗНЫХ СРЕД**

### **Development Environment**
```yaml
# values-dev.yaml
replicaCount: 1

image:
  tag: "dev"
  pullPolicy: Always

postgresql:
  persistence:
    enabled: false  # Use emptyDir for dev

redis:
  persistence:
    enabled: false  # Use emptyDir for dev

ingress:
  enabled: false  # No ingress in dev

monitoring:
  enabled: false  # No monitoring in dev

hpa:
  enabled: false  # No autoscaling in dev

resources:
  requests:
    memory: 128Mi
    cpu: 100m
  limits:
    memory: 256Mi
    cpu: 200m
```

### **Staging Environment**
```yaml
# values-staging.yaml
replicaCount: 2

ingress:
  enabled: true
  hosts:
    - host: alert-history.staging.yourcompany.com
  tls:
    - secretName: alert-history-staging-tls
      hosts:
        - alert-history.staging.yourcompany.com

postgresql:
  persistence:
    size: 20Gi
    storageClass: "standard"

redis:
  persistence:
    size: 2Gi
    storageClass: "standard"

monitoring:
  enabled: true

hpa:
  enabled: true
  minReplicas: 2
  maxReplicas: 5
```

### **Production Environment**
```yaml
# values-prod.yaml
replicaCount: 5

ingress:
  enabled: true
  hosts:
    - host: alert-history.yourcompany.com
  tls:
    - secretName: alert-history-prod-tls
      hosts:
        - alert-history.yourcompany.com

postgresql:
  persistence:
    size: 100Gi
    storageClass: "fast-ssd"

redis:
  persistence:
    size: 10Gi
    storageClass: "fast-ssd"

monitoring:
  enabled: true

hpa:
  enabled: true
  minReplicas: 5
  maxReplicas: 20

pdb:
  enabled: true
  minAvailable: 3

networkPolicy:
  enabled: true

securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  runAsGroup: 1000

resources:
  requests:
    memory: 512Mi
    cpu: 500m
  limits:
    memory: 1Gi
    cpu: 1000m
```

## 🔧 **HELPER FUNCTIONS**

### **_helpers.tpl**
```go
{{/*
Expand the name of the chart.
*/}}
{{- define "alert-history-go.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "alert-history-go.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "alert-history-go.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "alert-history-go.labels" -}}
helm.sh/chart: {{ include "alert-history-go.chart" . }}
{{ include "alert-history-go.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "alert-history-go.selectorLabels" -}}
app.kubernetes.io/name: {{ include "alert-history-go.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
```

## 🧪 **ТЕСТИРОВАНИЕ**

### **Test Templates**
```yaml
# templates/tests/test-connection.yaml
apiVersion: v1
kind: Pod
metadata:
  name: "{{ include "alert-history-go.fullname" . }}-test-connection"
  labels:
    {{- include "alert-history-go.labels" . | nindent 4 }}
  annotations:
    "helm.sh/hook": test
spec:
  restartPolicy: Never
  containers:
    - name: wget
      image: busybox
      command: ['wget']
      args:  ['{{ include "alert-history-go.fullname" . }}:{{ .Values.service.port }}/health']
```

### **Database Connectivity Test**
```yaml
# templates/tests/test-database.yaml
apiVersion: v1
kind: Pod
metadata:
  name: "{{ include "alert-history-go.fullname" . }}-test-database"
  labels:
    {{- include "alert-history-go.labels" . | nindent 4 }}
  annotations:
    "helm.sh/hook": test
spec:
  restartPolicy: Never
  containers:
    - name: postgresql-client
      image: postgres:15-alpine
      env:
        - name: PGPASSWORD
          valueFrom:
            secretKeyRef:
              name: {{ include "alert-history-go.fullname" . }}-postgresql
              key: password
      command:
        - psql
        - -h
        - {{ include "alert-history-go.fullname" . }}-postgresql
        - -U
        - {{ .Values.postgresql.auth.username }}
        - -d
        - {{ .Values.postgresql.auth.database }}
        - -c
        - "SELECT 1;"
```

## 🚀 **DEPLOYMENT PROCESS**

### **Installation**
```bash
# Add Helm repository (if needed)
helm repo add your-repo https://charts.yourcompany.com
helm repo update

# Install in development
helm install alert-history-dev ./alert-history-go \
  -f values-dev.yaml \
  -n development

# Install in staging
helm install alert-history-staging ./alert-history-go \
  -f values-staging.yaml \
  -n staging

# Install in production
helm install alert-history-prod ./alert-history-go \
  -f values-prod.yaml \
  -n production
```

### **Upgrade**
```bash
# Upgrade with new values
helm upgrade alert-history-prod ./alert-history-go \
  -f values-prod.yaml \
  --set image.tag=v1.2.0

# Rollback if needed
helm rollback alert-history-prod 1
```

### **Uninstallation**
```bash
# Uninstall (keeps PVCs)
helm uninstall alert-history-prod

# Remove PVCs if needed
kubectl delete pvc -l app.kubernetes.io/instance=alert-history-prod
```

## 📈 **MONITORING & ALERTING**

### **Key Metrics to Monitor**
- **Application Health**: HTTP 200 responses, latency < 100ms
- **Database Performance**: Connection pool utilization < 80%
- **Resource Usage**: CPU < 70%, Memory < 80%
- **Error Rates**: < 0.1% 5xx errors
- **Pod Status**: All pods ready, no restarts

### **Grafana Dashboard**
```json
{
  "dashboard": {
    "title": "Alert History Go",
    "panels": [
      {
        "title": "HTTP Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total{job=\"alert-history-go\"}[5m])",
            "legendFormat": "Requests/sec"
          }
        ]
      },
      {
        "title": "HTTP Request Duration",
        "type": "heatmap",
        "targets": [
          {
            "expr": "rate(http_request_duration_seconds_bucket{job=\"alert-history-go\"}[5m])",
            "legendFormat": "{{le}}"
          }
        ]
      },
      {
        "title": "Database Connections",
        "type": "graph",
        "targets": [
          {
            "expr": "pg_stat_activity_count{datname=\"alerthistory\"}",
            "legendFormat": "Active connections"
          }
        ]
      }
    ]
  }
}
```

## 🎯 **SUCCESS CRITERIA**

### **Functional Requirements**
- ✅ **Deployment Success**: All pods start and pass health checks
- ✅ **Service Connectivity**: All services are accessible
- ✅ **Database Access**: Application can connect to PostgreSQL
- ✅ **Cache Access**: Application can connect to Redis
- ✅ **External Access**: Ingress routes traffic correctly

### **Performance Requirements**
- ✅ **Startup Time**: < 60 seconds for full deployment
- ✅ **Resource Usage**: Within defined limits
- ✅ **Scalability**: HPA works correctly
- ✅ **High Availability**: PDB prevents disruptions

### **Operational Requirements**
- ✅ **Monitoring**: Metrics collected and alerts configured
- ✅ **Security**: Network policies and security contexts applied
- ✅ **Backups**: PVCs configured for persistence
- ✅ **Updates**: Rolling updates work without downtime

### **Quality Requirements**
- ✅ **Helm Validation**: Chart passes `helm lint`
- ✅ **Template Rendering**: All templates render correctly
- ✅ **Documentation**: Complete README and usage examples
- ✅ **Testing**: Test pods validate functionality

This comprehensive Helm chart design provides a production-ready deployment solution for Alert History Service with full support for scalability, monitoring, security, and operational excellence.
