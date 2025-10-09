# Design: LLM Alert Classifier & Recommendation System

## Архитектурное решение

### Общая архитектура (12-Factor + Horizontal Scaling)

```mermaid
graph TB
    subgraph "Kubernetes Cluster"
        subgraph "AlertHistory Namespace"
            subgraph "Application Instances (Stateless)"
                AH1[AlertHistory Pod 1<br/>+ Intelligent Proxy]
                AH2[AlertHistory Pod 2<br/>+ Intelligent Proxy]
                AH3[AlertHistory Pod 3<br/>+ Intelligent Proxy]
            end

            subgraph "Shared State & Config"
                CM[ConfigMap<br/>12-Factor Config]
                SEC[Secret<br/>Credentials]
                SVC[LoadBalancer Service]
            end

            subgraph "Persistent Layer"
                PG[(PostgreSQL<br/>Shared Database)]
                REDIS[(Redis<br/>Caching & Session)]
                PVC[(PVC<br/>Logs & Temp)]
            end
        end

        subgraph "LLM Namespace"
            LP[LLM Proxy Service]
        end

        subgraph "Monitoring"
            AM[Alertmanager]
            GR[Grafana]
        end

        subgraph "External Systems"
            RT[Rootly]
            WH1[Webhook Target 1]
            WH2[Webhook Target 2]
            PD[PagerDuty]
        end
    end

    AM -->|webhook<br/>load balanced| SVC
    SVC -->|distribute| AH1
    SVC -->|distribute| AH2
    SVC -->|distribute| AH3

    AH1 -->|classify| LP
    AH2 -->|classify| LP
    AH3 -->|classify| LP

    LP -->|OpenAI/Azure| LLM[External LLM]

    AH1 -->|store/read| PG
    AH2 -->|store/read| PG
    AH3 -->|store/read| PG

    AH1 -->|cache| REDIS
    AH2 -->|cache| REDIS
    AH3 -->|cache| REDIS

    AH1 -->|read config| CM
    AH2 -->|read config| CM
    AH3 -->|read config| CM

    AH1 -->|publish| RT
    AH2 -->|publish| WH1
    AH3 -->|publish| PD

    GR -->|metrics| SVC
```

## 12-Factor App Principles

### Factor I: Codebase
- **Одна кодовая база** для всех environments (dev, staging, prod)
- **Git repository** с branches для разных environments
- **Container image** собирается один раз, деплоится везде
- **Helm chart** управляет конфигурацией для разных сред

### Factor II: Dependencies
- **Explicit dependencies** в `requirements.txt`
- **No system-wide packages** - все зависимости изолированы
- **Dependency isolation** через Docker containers
- **Lock файлы** для reproducible builds
- **PEP8 compliance tools** в dev dependencies (black, flake8, mypy)

### Factor III: Config
- **Environment variables** для всех настроек
- **No hardcoded config** в коде
- **ConfigMaps** для Kubernetes-специфичных настроек
- **Secrets** для sensitive data (API keys, passwords)

```yaml
# Примеры environment variables
DATABASE_URL: "postgresql://user:pass@postgres:5432/alerthistory"
REDIS_URL: "redis://redis:6379/0"
LLM_PROXY_URL: "http://llm-proxy.llm-namespace:8080"
LLM_PROXY_API_KEY: "${LLM_PROXY_API_KEY}"  # From Secret
LOG_LEVEL: "INFO"
METRICS_ENABLED: "true"
PUBLISHING_ENABLED: "true"
```

### Factor IV: Backing Services
- **PostgreSQL** - основная база данных (treated as attached resource)
- **Redis** - кеширование и session storage
- **LLM Proxy** - external service через URL
- **External APIs** (Rootly, PagerDuty) - attached resources
- **All services** accessible via URLs from config

### Factor V: Build, Release, Run
- **Build stage:** Docker image creation with all dependencies
- **Release stage:** Combine build + config (Helm values)
- **Run stage:** Execute in Kubernetes environment
- **Immutable releases** with version tags
- **Rollback capability** через Kubernetes deployments

### Factor VI: Processes
- **Stateless processes** - no local state between requests
- **Session data** stored in Redis (external)
- **Database state** in PostgreSQL (external)
- **File uploads** in PVC or object storage
- **Process crash recovery** handled by Kubernetes

### Factor VII: Port Binding
- **Self-contained service** exports HTTP via port binding
- **No web server dependency** - uses FastAPI/uvicorn internally
- **Service exports** via Kubernetes Service
- **Load balancing** handled by K8s Service

### Factor VIII: Concurrency
- **Horizontal scaling** via Kubernetes replicas
- **Process types:** web (HTTP), worker (background tasks)
- **Stateless design** enables unlimited horizontal scaling
- **No shared memory** between processes

### Factor IX: Disposability
- **Fast startup** - optimized container startup time
- **Graceful shutdown** - proper signal handling (SIGTERM)
- **Robust against sudden death** - idempotent operations
- **Health checks** for liveness/readiness probes

### Factor X: Dev/Prod Parity
- **Same technology stack** across environments
- **Same backing services** (PostgreSQL, Redis)
- **Same deployment method** (Helm charts)
- **Minimal time gap** between dev and prod deployments
- **Continuous deployment** pipeline

### Factor XI: Logs
- **Logs to stdout/stderr** - no log files
- **Structured logging** (JSON format)
- **Log aggregation** handled by Kubernetes/ELK
- **No log rotation** in application
- **Centralized logging** via Fluent Bit/Fluentd

### Factor XII: Admin Processes
- **Database migrations** via init containers
- **One-time scripts** as Kubernetes Jobs
- **Admin tasks** via kubectl exec or dedicated endpoints
- **Same codebase** for admin processes
- **Same environment** for admin and regular processes

## Python Code Quality Standards (PEP8+)

### Code Formatting & Style
- **PEP8 compliance** - строгое соблюдение Python style guide
- **Black** - автоформатирование кода (line length: 88)
- **isort** - сортировка imports
- **flake8** - проверка style violations
- **mypy** - static type checking
- **pylint** - comprehensive code analysis

### Type Annotations
```python
# Обязательные type hints для всех функций
from typing import Dict, List, Optional, Union, AsyncGenerator
from dataclasses import dataclass
from enum import Enum

class AlertSeverity(Enum):
    """Alert severity levels following PEP8 enum conventions."""
    CRITICAL = "critical"
    WARNING = "warning"
    INFO = "info"
    NOISE = "noise"

@dataclass
class ClassificationResult:
    """Classification result with comprehensive type annotations."""
    severity: AlertSeverity
    confidence: float
    reasoning: str
    recommendations: List[str]
    processing_time: float
    metadata: Optional[Dict[str, Union[str, int, float]]] = None

class AlertClassifier:
    """Alert classifier following PEP8 naming conventions."""

    def __init__(self, llm_client: 'LLMProxyClient', config: Dict[str, str]) -> None:
        """Initialize classifier with proper docstring format."""
        self.llm_client = llm_client
        self.config = config
        self._cache: Dict[str, ClassificationResult] = {}

    async def classify_alert(
        self,
        alert: Dict[str, Union[str, Dict, List]],
        context: Optional[Dict[str, str]] = None
    ) -> ClassificationResult:
        """
        Classify alert using LLM with comprehensive type hints.

        Args:
            alert: Alert data from Alertmanager
            context: Optional context for classification

        Returns:
            Classification result with severity and reasoning

        Raises:
            ValueError: If alert data is invalid
            ConnectionError: If LLM service is unavailable
        """
        if not alert.get('labels', {}).get('alertname'):
            raise ValueError("Alert must contain alertname in labels")

        # Implementation following PEP8 conventions
        alert_key = self._generate_cache_key(alert)

        if alert_key in self._cache:
            return self._cache[alert_key]

        try:
            result = await self._perform_classification(alert, context)
            self._cache[alert_key] = result
            return result
        except Exception as exc:
            logger.error(f"Classification failed for alert {alert_key}: {exc}")
            raise

    def _generate_cache_key(self, alert: Dict[str, Union[str, Dict, List]]) -> str:
        """Generate cache key following PEP8 private method conventions."""
        # Private method implementation
        pass

    async def _perform_classification(
        self,
        alert: Dict[str, Union[str, Dict, List]],
        context: Optional[Dict[str, str]]
    ) -> ClassificationResult:
        """Perform actual classification - private async method."""
        # Implementation
        pass
```

### Documentation Standards
```python
"""
Module: alert_classifier

This module provides intelligent alert classification using LLM.

Classes:
    AlertClassifier: Main classification engine
    ClassificationResult: Result dataclass
    AlertSeverity: Severity enumeration

Functions:
    create_classifier: Factory function for classifier instances

Example:
    >>> classifier = create_classifier(llm_client, config)
    >>> result = await classifier.classify_alert(alert_data)
    >>> print(result.severity.value)
    'critical'
"""

import asyncio
import logging
from dataclasses import dataclass
from enum import Enum
from typing import Dict, List, Optional, Union, AsyncGenerator

# Standard library imports first
# Third-party imports second
# Local imports last (following isort conventions)
```

### Error Handling Patterns
```python
class AlertClassificationError(Exception):
    """Base exception for alert classification errors."""
    pass

class LLMServiceError(AlertClassificationError):
    """LLM service unavailable or returned error."""
    pass

class InvalidAlertDataError(AlertClassificationError):
    """Alert data is invalid or incomplete."""
    pass

# Proper exception handling with context
async def classify_alert_with_error_handling(
    self,
    alert: Dict[str, Union[str, Dict, List]]
) -> ClassificationResult:
    """Classify alert with comprehensive error handling."""
    try:
        return await self.classify_alert(alert)
    except ValueError as exc:
        raise InvalidAlertDataError(f"Invalid alert data: {exc}") from exc
    except ConnectionError as exc:
        raise LLMServiceError(f"LLM service unavailable: {exc}") from exc
    except Exception as exc:
        logger.exception("Unexpected error during classification")
        raise AlertClassificationError(f"Classification failed: {exc}") from exc
```

### Async/Await Best Practices
```python
import asyncio
from contextlib import asynccontextmanager
from typing import AsyncGenerator

class AlertProcessor:
    """Alert processor with proper async patterns."""

    async def __aenter__(self) -> 'AlertProcessor':
        """Async context manager entry."""
        await self._initialize_connections()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb) -> None:
        """Async context manager exit with proper cleanup."""
        await self._cleanup_connections()

    @asynccontextmanager
    async def processing_session(self) -> AsyncGenerator['ProcessingSession', None]:
        """Async context manager for processing sessions."""
        session = ProcessingSession()
        try:
            await session.start()
            yield session
        finally:
            await session.close()

    async def process_alerts_batch(
        self,
        alerts: List[Dict[str, Union[str, Dict, List]]]
    ) -> List[ClassificationResult]:
        """Process alerts in batch with proper concurrency control."""
        semaphore = asyncio.Semaphore(10)  # Limit concurrent operations

        async def process_single_alert(alert: Dict) -> ClassificationResult:
            async with semaphore:
                return await self.classify_alert(alert)

        tasks = [process_single_alert(alert) for alert in alerts]
        return await asyncio.gather(*tasks, return_exceptions=True)
```

### Logging Standards
```python
import logging
import json
from datetime import datetime
from typing import Any, Dict

# Structured logging setup
logging.basicConfig(
    level=logging.INFO,
    format='%(message)s',  # JSON format in stdout
    handlers=[logging.StreamHandler()]
)

logger = logging.getLogger(__name__)

class StructuredLogger:
    """Structured JSON logger for 12-factor compliance."""

    @staticmethod
    def log_classification(
        alert_fingerprint: str,
        severity: AlertSeverity,
        confidence: float,
        processing_time: float,
        **kwargs: Any
    ) -> None:
        """Log classification with structured format."""
        log_entry = {
            "timestamp": datetime.utcnow().isoformat(),
            "event": "alert_classified",
            "alert_fingerprint": alert_fingerprint,
            "severity": severity.value,
            "confidence": confidence,
            "processing_time": processing_time,
            "service": "alert-history-llm",
            **kwargs
        }
        logger.info(json.dumps(log_entry))

    @staticmethod
    def log_error(
        operation: str,
        error: Exception,
        alert_fingerprint: Optional[str] = None,
        **kwargs: Any
    ) -> None:
        """Log error with structured format."""
        log_entry = {
            "timestamp": datetime.utcnow().isoformat(),
            "event": "error",
            "operation": operation,
            "error_type": type(error).__name__,
            "error_message": str(error),
            "service": "alert-history-llm",
            **kwargs
        }
        if alert_fingerprint:
            log_entry["alert_fingerprint"] = alert_fingerprint

        logger.error(json.dumps(log_entry))
```

### Testing Standards
```python
import pytest
from unittest.mock import AsyncMock, MagicMock, patch
from typing import AsyncGenerator

class TestAlertClassifier:
    """Test class following pytest conventions."""

    @pytest.fixture
    async def classifier(self) -> AsyncGenerator[AlertClassifier, None]:
        """Fixture for classifier instance."""
        mock_llm_client = AsyncMock()
        config = {"model": "gpt-4", "timeout": 30}

        classifier = AlertClassifier(mock_llm_client, config)
        yield classifier

        # Cleanup if needed
        await classifier.cleanup()

    @pytest.mark.asyncio
    async def test_classify_alert_success(self, classifier: AlertClassifier) -> None:
        """Test successful alert classification."""
        # Arrange
        alert_data = {
            "labels": {"alertname": "CPUHigh"},
            "status": "firing"
        }

        # Act
        result = await classifier.classify_alert(alert_data)

        # Assert
        assert isinstance(result, ClassificationResult)
        assert result.severity in AlertSeverity
        assert 0.0 <= result.confidence <= 1.0
        assert isinstance(result.reasoning, str)
        assert len(result.reasoning) > 0

    @pytest.mark.asyncio
    async def test_classify_alert_invalid_data(self, classifier: AlertClassifier) -> None:
        """Test classification with invalid alert data."""
        # Arrange
        invalid_alert = {"status": "firing"}  # Missing alertname

        # Act & Assert
        with pytest.raises(InvalidAlertDataError):
            await classifier.classify_alert(invalid_alert)

    @pytest.mark.parametrize("alert_name,expected_severity", [
        ("CPUHigh", AlertSeverity.WARNING),
        ("ServiceDown", AlertSeverity.CRITICAL),
        ("DiskSpaceInfo", AlertSeverity.INFO),
    ])
    async def test_classify_different_severities(
        self,
        classifier: AlertClassifier,
        alert_name: str,
        expected_severity: AlertSeverity
    ) -> None:
        """Parametrized test for different alert severities."""
        # Implementation
        pass
```

### Requirements для Code Quality
```python
# requirements-dev.txt
black==23.12.1              # Code formatting
isort==5.13.2               # Import sorting
flake8==6.1.0               # Style checking
mypy==1.8.0                 # Static type checking
pylint==3.0.3               # Comprehensive analysis
pytest==7.4.3              # Testing framework
pytest-asyncio==0.21.1     # Async testing
pytest-cov==4.1.0          # Coverage reporting
pre-commit==3.6.0          # Git hooks
bandit==1.7.5               # Security linting

# pyproject.toml configuration
[tool.black]
line-length = 88
target-version = ['py38', 'py39', 'py310', 'py311']
include = '\.pyi?$'
extend-exclude = '''
/(
  # directories
  \.eggs
  | \.git
  | \.mypy_cache
  | \.tox
  | \.venv
  | build
  | dist
)/
'''

[tool.isort]
profile = "black"
multi_line_output = 3
line_length = 88
known_first_party = ["alert_history"]

[tool.mypy]
python_version = "3.8"
warn_return_any = true
warn_unused_configs = true
disallow_untyped_defs = true
disallow_incomplete_defs = true
check_untyped_defs = true
disallow_untyped_decorators = true
no_implicit_optional = true
warn_redundant_casts = true
warn_unused_ignores = true
warn_no_return = true
warn_unreachable = true
strict_equality = true

[tool.pylint]
max-line-length = 88
disable = [
    "C0114",  # missing-module-docstring
    "R0903",  # too-few-public-methods
]

[tool.pytest.ini_options]
asyncio_mode = "auto"
testpaths = ["tests"]
python_files = ["test_*.py", "*_test.py"]
python_classes = ["Test*"]
python_functions = ["test_*"]
addopts = "--cov=alert_history --cov-report=html --cov-report=term"

[tool.coverage.run]
source = ["alert_history"]
omit = ["tests/*", "*/test_*"]
```

## Horizontal Scaling Architecture

### Stateless Application Design

```python
# Пример stateless application design
class StatelessAlertProcessor:
    def __init__(self):
        # Все конфигурации из environment variables
        self.db_url = os.environ['DATABASE_URL']
        self.redis_url = os.environ['REDIS_URL']
        self.llm_proxy_url = os.environ['LLM_PROXY_URL']

        # Shared connections (connection pooling)
        self.db_pool = create_async_pool(self.db_url)
        self.redis_pool = aioredis.from_url(self.redis_url)

        # No local state
        self.session_data = None  # Always fetch from Redis

    async def process_alert(self, alert_data: Dict) -> Dict:
        """Stateless alert processing - no instance state"""
        # Get session data from Redis (if needed)
        session_key = f"session:{alert_data.get('fingerprint')}"
        session_data = await self.redis_pool.get(session_key)

        # Process alert (stateless)
        result = await self._classify_and_publish(alert_data)

        # Store result in database (shared state)
        await self._store_to_database(result)

        # Update cache (shared state)
        await self.redis_pool.setex(
            f"classification:{alert_data.get('fingerprint')}",
            3600,
            json.dumps(result)
        )

        return result
```

### Database Architecture for Multiple Instances

```sql
-- Database schema optimized for concurrent access
CREATE TABLE alert_history (
    id BIGSERIAL PRIMARY KEY,
    fingerprint TEXT NOT NULL,
    alertname TEXT NOT NULL,
    status TEXT NOT NULL,
    labels JSONB NOT NULL,
    raw_json JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_by TEXT,  -- Instance ID for debugging

    -- Optimistic locking for concurrent updates
    version INTEGER DEFAULT 1,

    -- Indexes for performance
    CONSTRAINT unique_fingerprint_status UNIQUE (fingerprint, status, created_at)
);

-- Separate table for classifications (to avoid conflicts)
CREATE TABLE alert_classifications (
    id BIGSERIAL PRIMARY KEY,
    alert_fingerprint TEXT NOT NULL,
    severity TEXT NOT NULL,
    confidence REAL NOT NULL,
    reasoning TEXT,
    model_version TEXT,
    classified_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    classified_by TEXT,  -- Instance ID

    -- Prevent duplicate classifications
    CONSTRAINT unique_classification UNIQUE (alert_fingerprint, model_version)
);

-- Publishing log for tracking
CREATE TABLE publishing_log (
    id BIGSERIAL PRIMARY KEY,
    alert_fingerprint TEXT NOT NULL,
    target_name TEXT NOT NULL,
    target_url TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL,  -- success, failed, retrying
    response_code INTEGER,
    response_body TEXT,
    published_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    published_by TEXT,  -- Instance ID
    retry_count INTEGER DEFAULT 0
);

-- Distributed locking table
CREATE TABLE distributed_locks (
    lock_name TEXT PRIMARY KEY,
    locked_by TEXT NOT NULL,  -- Instance ID
    locked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    lock_data JSONB
);
```

### Redis Caching Strategy

```python
class DistributedCache:
    """Redis-based distributed caching for multiple instances"""

    def __init__(self, redis_pool):
        self.redis = redis_pool
        self.instance_id = os.environ.get('HOSTNAME', 'unknown')

    async def get_classification(self, fingerprint: str) -> Optional[Dict]:
        """Get cached classification result"""
        cache_key = f"classification:{fingerprint}"
        cached = await self.redis.get(cache_key)
        return json.loads(cached) if cached else None

    async def set_classification(self, fingerprint: str, result: Dict, ttl: int = 3600):
        """Cache classification result with TTL"""
        cache_key = f"classification:{fingerprint}"
        await self.redis.setex(cache_key, ttl, json.dumps(result))

    async def acquire_lock(self, lock_name: str, timeout: int = 30) -> bool:
        """Distributed lock for critical sections"""
        lock_key = f"lock:{lock_name}"
        acquired = await self.redis.set(
            lock_key,
            self.instance_id,
            ex=timeout,
            nx=True
        )
        return bool(acquired)

    async def release_lock(self, lock_name: str):
        """Release distributed lock"""
        lock_key = f"lock:{lock_name}"
        # Lua script for atomic release
        script = """
        if redis.call("get", KEYS[1]) == ARGV[1] then
            return redis.call("del", KEYS[1])
        else
            return 0
        end
        """
        await self.redis.eval(script, 1, lock_key, self.instance_id)
```

### Horizontal Pod Autoscaler Configuration

```yaml
# HPA for automatic scaling based on metrics
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: alert-history-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: alert-history
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  - type: Pods
    pods:
      metric:
        name: requests_per_second
      target:
        type: AverageValue
        averageValue: "30"
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 50
        periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 100
        periodSeconds: 60
```

### Load Balancing and Session Affinity

```yaml
# Service configuration for proper load balancing
apiVersion: v1
kind: Service
metadata:
  name: alert-history
spec:
  selector:
    app: alert-history
  ports:
  - port: 8080
    targetPort: 8080
  sessionAffinity: None  # No session affinity for stateless design
  type: ClusterIP

# Ingress for external access with load balancing
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: alert-history-ingress
  annotations:
    nginx.ingress.kubernetes.io/upstream-hash-by: "$request_uri"  # Consistent hashing
    nginx.ingress.kubernetes.io/load-balance: "round_robin"
spec:
  rules:
  - host: alert-history.company.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: alert-history
            port:
              number: 8080
```

### Компонентная архитектура

#### 1. Intelligent Alert Proxy (Stateless)
**Назначение:** Прием, классификация и перенаправление алертов

**Компоненты:**
- `AlertReceiver` - прием webhook от Alertmanager
- `AlertClassifier` - классификация через LLM
- `AlertEnricher` - обогащение алертов классификацией и контекстом
- `AlertPublisher` - публикация в downstream системы
- `FilterEngine` - фильтрация по правилам и классификации

#### 2. LLM Classifier Module
**Назначение:** Классификация алертов и генерация рекомендаций

**Компоненты:**
- `AlertClassifier` - основной класс для классификации
- `RecommendationEngine` - генерация рекомендаций по настройкам
- `LLMProxyClient` - клиент для взаимодействия с LLM-proxy
- `ClassificationCache` - кеширование результатов
- `AlertAnalyzer` - анализ паттернов алертов

#### 3. Alert Publisher
**Назначение:** Публикация обработанных алертов в различные системы

**Компоненты:**
- `WebhookPublisher` - отправка в webhook endpoints
- `AlertFormatter` - форматирование в различные форматы
- `TargetManager` - управление целевыми системами
- `DeliveryTracker` - отслеживание доставки

#### 4. LLM Proxy Integration
**Назначение:** Интеграция с внутренним LLM-proxy сервисом

**Особенности:**
- HTTP клиент для вызовов LLM-proxy
- Retry механизм с exponential backoff
- Circuit breaker pattern для надежности
- Метрики и мониторинг вызовов

#### 5. Configuration Management
**Назначение:** Управление конфигурацией через Kubernetes ресурсы

**ConfigMap структура:**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: alert-history-llm-config
data:
  llm-proxy-url: "http://llm-proxy.llm-namespace:8080"
  classification-enabled: "true"
  recommendations-enabled: "true"
  publishing-enabled: "true"
  batch-size: "10"
  cache-ttl: "3600"
  models: |
    classification: "gpt-4"
    recommendations: "gpt-4"
  prompts: |
    classification: |
      Analyze this Prometheus alert and classify it into one of these categories:
      - critical: Requires immediate attention, service is down or severely degraded
      - warning: Performance issue or potential problem, needs investigation
      - info: Informational alert, good to know but not urgent
      - noise: False positive or irrelevant alert

      Alert data: {alert_data}
      Historical context: {historical_context}
    recommendations: |
      Based on this alert pattern analysis, provide recommendations for Alertmanager configuration:
      Alert: {alert_name}
      Flapping count: {flap_count}
      Average duration: {avg_duration}

      Suggest improvements for: evaluation_interval, for duration, repeat_interval
    # Publishing configuration (targets discovered dynamically from secrets)
  publishing-discovery:
    enabled: true
    secret-labels:
      - "publishing-target=true"  # Label selector for discovering target secrets
    secret-namespaces:
      - "default"  # Namespaces to search for target secrets
      - "alert-history"
    config-refresh-interval: "300s"  # How often to reload target configs

  # Default publishing configuration (fallback if no secrets found)
  publishing-defaults:
    metrics-only-mode: true  # If no targets configured, only collect metrics
    fallback-targets: |
      # Example targets that can be enabled via secrets
      # These serve as templates and documentation
      - name: "rootly-template"
        type: "webhook"
        description: "Rootly integration template"
        format: "rootly"
        enabled: false  # Disabled by default, enable via secret
        filter:
          severity: ["critical", "warning"]
          exclude_noise: true
        required_secrets:
          - "rootly-api-key"
          - "rootly-webhook-url"

      - name: "pagerduty-template"
        type: "webhook"
        description: "PagerDuty integration template"
        format: "pagerduty"
        enabled: false
        filter:
          severity: ["critical"]
          namespaces: ["production", "staging"]
        required_secrets:
          - "pagerduty-routing-key"

      - name: "slack-template"
        type: "webhook"
        description: "Slack webhook template"
        format: "slack"
        enabled: false
        filter:
          severity: ["critical"]
          exclude_noise: true
        required_secrets:
          - "slack-webhook-url"

      - name: "custom-webhook-template"
        type: "webhook"
        description: "Generic webhook template"
        format: "alertmanager"
        enabled: false
        filter:
          severity: ["critical", "warning", "info"]
        required_secrets:
          - "webhook-url"
```

**Secret структура:**
```yaml
# Main LLM secrets
apiVersion: v1
kind: Secret
metadata:
  name: alert-history-llm-secrets
type: Opaque
data:
  llm-proxy-api-key: <base64-encoded-key>
  llm-proxy-auth-token: <base64-encoded-token>

---
# Dynamic publishing targets secrets (optional)
apiVersion: v1
kind: Secret
metadata:
  name: alert-history-publishing-secrets
type: Opaque
data:
  # Rootly integration (optional)
  rootly-api-key: <base64-encoded-key>
  rootly-webhook-url: <base64-encoded-url>

  # PagerDuty integration (optional)
  pagerduty-routing-key: <base64-encoded-key>
  pagerduty-token: <base64-encoded-token>

  # Slack webhooks (optional, can be multiple)
  slack-webhook-critical: <base64-encoded-url>
  slack-webhook-general: <base64-encoded-url>

  # Custom webhook targets (optional, dynamic)
  webhook-target-1-url: <base64-encoded-url>
  webhook-target-1-auth: <base64-encoded-header>
  webhook-target-2-url: <base64-encoded-url>
  webhook-target-2-auth: <base64-encoded-header>

  # Additional targets can be added dynamically
  # Format: {target-name}-{property}: value

---
# Alternative: Multiple secrets for different teams/environments
apiVersion: v1
kind: Secret
metadata:
  name: team-a-publishing-secrets
  labels:
    publishing-target: "true"
    team: "team-a"
type: Opaque
data:
  target-name: dGVhbS1hLXdlYmhvb2s=  # team-a-webhook
  webhook-url: <base64-encoded-url>
  auth-header: <base64-encoded-auth>
  enabled: dHJ1ZQ==  # true

---
apiVersion: v1
kind: Secret
metadata:
  name: team-b-publishing-secrets
  labels:
    publishing-target: "true"
    team: "team-b"
type: Opaque
data:
  target-name: dGVhbS1iLXNsYWNr  # team-b-slack
  webhook-url: <base64-encoded-url>
  format: c2xhY2s=  # slack
  enabled: dHJ1ZQ==  # true
```

## Детальный дизайн компонентов

### 1. AlertClassifier класс

```python
from dataclasses import dataclass
from typing import Dict, List, Optional
from enum import Enum

class AlertSeverity(Enum):
    CRITICAL = "critical"
    WARNING = "warning"
    INFO = "info"
    NOISE = "noise"

@dataclass
class ClassificationResult:
    severity: AlertSeverity
    confidence: float
    reasoning: str
    recommendations: List[str]
    processing_time: float

class AlertClassifier:
    def __init__(self, llm_client: LLMProxyClient, config: Dict):
        self.llm_client = llm_client
        self.config = config
        self.cache = ClassificationCache()

    async def classify_alert(self, alert: Dict, context: Dict = None) -> ClassificationResult:
        """Классифицирует алерт с использованием LLM"""
        pass

    async def batch_classify(self, alerts: List[Dict]) -> List[ClassificationResult]:
        """Пакетная классификация алертов"""
        pass

    def _prepare_prompt(self, alert: Dict, context: Dict) -> str:
        """Подготавливает промпт для LLM"""
        pass
```

### 2. LLMProxyClient класс

```python
import aiohttp
import asyncio
from typing import Dict, Any
import logging

class LLMProxyClient:
    def __init__(self, base_url: str, api_key: str, auth_token: str = None):
        self.base_url = base_url.rstrip('/')
        self.api_key = api_key
        self.auth_token = auth_token
        self.session = None
        self.circuit_breaker = CircuitBreaker()

    async def __aenter__(self):
        self.session = aiohttp.ClientSession(
            headers=self._get_auth_headers(),
            timeout=aiohttp.ClientTimeout(total=30)
        )
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        if self.session:
            await self.session.close()

    async def call_llm(self, model: str, messages: List[Dict],
                      functions: List[Dict] = None) -> Dict[str, Any]:
        """Вызов LLM через proxy с retry логикой"""

        @self.circuit_breaker
        async def _make_request():
            async with self.session.post(
                f"{self.base_url}/v1/chat/completions",
                json={
                    "model": model,
                    "messages": messages,
                    "functions": functions,
                    "function_call": "auto" if functions else None
                }
            ) as response:
                response.raise_for_status()
                return await response.json()

        return await self._retry_with_backoff(_make_request)

    def _get_auth_headers(self) -> Dict[str, str]:
        headers = {"Authorization": f"Bearer {self.api_key}"}
        if self.auth_token:
            headers["X-Auth-Token"] = self.auth_token
        return headers

    async def _retry_with_backoff(self, func, max_retries: int = 3):
        """Retry с exponential backoff"""
        for attempt in range(max_retries):
            try:
                return await func()
            except Exception as e:
                if attempt == max_retries - 1:
                    raise
                wait_time = 2 ** attempt
                await asyncio.sleep(wait_time)
```

### 3. RecommendationEngine класс

```python
@dataclass
class AlertRecommendation:
    alert_name: str
    namespace: str
    current_config: Dict
    recommendations: Dict
    impact_estimate: str
    confidence: float

class RecommendationEngine:
    def __init__(self, llm_client: LLMProxyClient, db_client):
        self.llm_client = llm_client
        self.db = db_client

    async def analyze_flapping_alerts(self, days: int = 7) -> List[AlertRecommendation]:
        """Анализ flapping алертов и генерация рекомендаций"""
        flapping_alerts = await self._get_flapping_alerts(days)
        recommendations = []

        for alert in flapping_alerts:
            context = await self._build_alert_context(alert)
            recommendation = await self._generate_recommendation(alert, context)
            recommendations.append(recommendation)

        return recommendations

    async def _generate_recommendation(self, alert: Dict, context: Dict) -> AlertRecommendation:
        """Генерирует рекомендации через LLM"""
        prompt = self._build_recommendation_prompt(alert, context)

        functions = [{
            "name": "provide_alertmanager_recommendations",
            "description": "Provide recommendations for Alertmanager configuration",
            "parameters": {
                "type": "object",
                "properties": {
                    "evaluation_interval": {"type": "string"},
                    "for_duration": {"type": "string"},
                    "repeat_interval": {"type": "string"},
                    "threshold_adjustment": {"type": "string"},
                    "reasoning": {"type": "string"},
                    "impact_estimate": {"type": "string"}
                },
                "required": ["reasoning", "impact_estimate"]
            }
        }]

        response = await self.llm_client.call_llm(
            model="gpt-4",
            messages=[{"role": "user", "content": prompt}],
            functions=functions
        )

        return self._parse_recommendation_response(alert, response)

### 4. AlertPublisher и AlertFormatter классы

```python
from dataclasses import dataclass
from typing import Dict, List, Optional, Any
from enum import Enum
import aiohttp
import asyncio

class PublishingFormat(Enum):
    ALERTMANAGER = "alertmanager"
    ROOTLY = "rootly"
    PAGERDUTY = "pagerduty"
    SLACK = "slack"
    WEBHOOK = "webhook"

@dataclass
class PublishingTarget:
    name: str
    type: str
    url: str
    enabled: bool
    filter_config: Dict
    headers: Dict
    format: PublishingFormat
    retry_config: Optional[Dict] = None

@dataclass
class EnrichedAlert:
    """Алерт обогащенный классификацией и контекстом"""
    original_alert: Dict
    classification: ClassificationResult
    enriched_labels: Dict
    enriched_annotations: Dict
    processing_metadata: Dict

class AlertFormatter:
    """Форматирование алертов для различных систем"""

    def format_for_alertmanager(self, alert: EnrichedAlert) -> Dict:
        """Стандартный Alertmanager формат с дополнительными метками"""
        formatted = alert.original_alert.copy()

        # Добавляем классификацию в labels
        if 'labels' not in formatted:
            formatted['labels'] = {}

        formatted['labels'].update({
            'llm_severity': alert.classification.severity.value,
            'llm_confidence': str(alert.classification.confidence),
            'processed_by': 'alert-history-llm'
        })

        # Добавляем reasoning в annotations
        if 'annotations' not in formatted:
            formatted['annotations'] = {}

        formatted['annotations'].update({
            'llm_reasoning': alert.classification.reasoning,
            'llm_recommendations': ', '.join(alert.classification.recommendations),
            'classification_timestamp': alert.processing_metadata.get('classified_at')
        })

        return formatted

    def format_for_rootly(self, alert: EnrichedAlert) -> Dict:
        """Формат для Rootly API"""
        return {
            "data": {
                "type": "alerts",
                "attributes": {
                    "title": alert.original_alert.get('labels', {}).get('alertname', 'Unknown Alert'),
                    "description": alert.original_alert.get('annotations', {}).get('description', ''),
                    "severity": self._map_severity_to_rootly(alert.classification.severity),
                    "source": "alertmanager",
                    "labels": alert.original_alert.get('labels', {}),
                    "annotations": alert.original_alert.get('annotations', {}),
                    "classification": {
                        "severity": alert.classification.severity.value,
                        "confidence": alert.classification.confidence,
                        "reasoning": alert.classification.reasoning
                    },
                    "fingerprint": alert.original_alert.get('fingerprint'),
                    "starts_at": alert.original_alert.get('startsAt'),
                    "ends_at": alert.original_alert.get('endsAt')
                }
            }
        }

    def format_for_pagerduty(self, alert: EnrichedAlert) -> Dict:
        """Формат для PagerDuty Events API v2"""
        severity_map = {
            AlertSeverity.CRITICAL: "critical",
            AlertSeverity.WARNING: "warning",
            AlertSeverity.INFO: "info",
            AlertSeverity.NOISE: "info"
        }

        return {
            "routing_key": "${PAGERDUTY_ROUTING_KEY}",  # Заменяется из конфига
            "event_action": "trigger" if alert.original_alert.get('status') == 'firing' else "resolve",
            "dedup_key": alert.original_alert.get('fingerprint'),
            "payload": {
                "summary": alert.original_alert.get('labels', {}).get('alertname', 'Unknown Alert'),
                "source": alert.original_alert.get('generatorURL', 'alertmanager'),
                "severity": severity_map.get(alert.classification.severity, "warning"),
                "component": alert.original_alert.get('labels', {}).get('job', 'unknown'),
                "group": alert.original_alert.get('labels', {}).get('namespace', 'default'),
                "class": alert.classification.severity.value,
                "custom_details": {
                    "labels": alert.original_alert.get('labels', {}),
                    "annotations": alert.original_alert.get('annotations', {}),
                    "classification": {
                        "confidence": alert.classification.confidence,
                        "reasoning": alert.classification.reasoning
                    }
                }
            }
        }

    def format_for_slack(self, alert: EnrichedAlert) -> Dict:
        """Формат для Slack webhook"""
        color_map = {
            AlertSeverity.CRITICAL: "danger",
            AlertSeverity.WARNING: "warning",
            AlertSeverity.INFO: "good",
            AlertSeverity.NOISE: "#D3D3D3"
        }

        return {
            "attachments": [{
                "color": color_map.get(alert.classification.severity, "warning"),
                "title": f"🚨 {alert.original_alert.get('labels', {}).get('alertname', 'Alert')}",
                "text": alert.original_alert.get('annotations', {}).get('description', ''),
                "fields": [
                    {
                        "title": "Severity",
                        "value": alert.classification.severity.value.upper(),
                        "short": True
                    },
                    {
                        "title": "Confidence",
                        "value": f"{alert.classification.confidence:.2f}",
                        "short": True
                    },
                    {
                        "title": "Namespace",
                        "value": alert.original_alert.get('labels', {}).get('namespace', 'unknown'),
                        "short": True
                    },
                    {
                        "title": "LLM Reasoning",
                        "value": alert.classification.reasoning[:200] + "..." if len(alert.classification.reasoning) > 200 else alert.classification.reasoning,
                        "short": False
                    }
                ],
                "footer": "Alert History LLM Classifier",
                "ts": int(time.time())
            }]
        }

    def _map_severity_to_rootly(self, severity: AlertSeverity) -> str:
        mapping = {
            AlertSeverity.CRITICAL: "critical",
            AlertSeverity.WARNING: "major",
            AlertSeverity.INFO: "minor",
            AlertSeverity.NOISE: "low"
        }
        return mapping.get(severity, "major")

class FilterEngine:
    """Фильтрация алертов по правилам"""

    def should_publish(self, alert: EnrichedAlert, target: PublishingTarget) -> bool:
        """Проверяет, нужно ли публиковать алерт в цель"""
        filter_config = target.filter_config

        # Фильтр по severity
        if 'severity' in filter_config:
            allowed_severities = filter_config['severity']
            if alert.classification.severity.value not in allowed_severities:
                return False

        # Исключение noise алертов
        if filter_config.get('exclude_noise', False):
            if alert.classification.severity == AlertSeverity.NOISE:
                return False

        # Фильтр по namespace
        if 'namespaces' in filter_config:
            alert_namespace = alert.original_alert.get('labels', {}).get('namespace')
            if alert_namespace not in filter_config['namespaces']:
                return False

        # Фильтр по alertname patterns
        if 'alertname_patterns' in filter_config:
            import re
            alertname = alert.original_alert.get('labels', {}).get('alertname', '')
            patterns = filter_config['alertname_patterns']
            if not any(re.match(pattern, alertname) for pattern in patterns):
                return False

        # Минимальный уровень confidence
        if 'min_confidence' in filter_config:
            min_confidence = filter_config['min_confidence']
            if alert.classification.confidence < min_confidence:
                return False

        return True

class AlertPublisher:
    """Публикация алертов в различные системы"""

    def __init__(self, targets: List[PublishingTarget], formatter: AlertFormatter):
        self.targets = targets
        self.formatter = formatter
        self.filter_engine = FilterEngine()
        self.session = None

    async def __aenter__(self):
        self.session = aiohttp.ClientSession(
            timeout=aiohttp.ClientTimeout(total=30)
        )
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        if self.session:
            await self.session.close()

    async def publish_alert(self, alert: EnrichedAlert) -> Dict[str, bool]:
        """Публикует алерт во все активные цели"""
        results = {}

        # Публикуем параллельно во все цели
        tasks = []
        for target in self.targets:
            if target.enabled and self.filter_engine.should_publish(alert, target):
                task = self._publish_to_target(alert, target)
                tasks.append((target.name, task))

        if tasks:
            task_results = await asyncio.gather(
                *[task for _, task in tasks],
                return_exceptions=True
            )

            for (target_name, _), result in zip(tasks, task_results):
                results[target_name] = not isinstance(result, Exception)
                if isinstance(result, Exception):
                    logging.error(f"Failed to publish to {target_name}: {result}")

        return results

    async def _publish_to_target(self, alert: EnrichedAlert, target: PublishingTarget) -> bool:
        """Публикует алерт в конкретную цель"""
        try:
            # Форматируем алерт в нужный формат
            if target.format == PublishingFormat.ALERTMANAGER:
                payload = self.formatter.format_for_alertmanager(alert)
            elif target.format == PublishingFormat.ROOTLY:
                payload = self.formatter.format_for_rootly(alert)
            elif target.format == PublishingFormat.PAGERDUTY:
                payload = self.formatter.format_for_pagerduty(alert)
            elif target.format == PublishingFormat.SLACK:
                payload = self.formatter.format_for_slack(alert)
            else:
                payload = self.formatter.format_for_alertmanager(alert)

            # Отправляем HTTP запрос
            headers = target.headers.copy()
            # Заменяем переменные окружения в headers
            for key, value in headers.items():
                if isinstance(value, str) and '${' in value:
                    headers[key] = self._substitute_env_vars(value)

            async with self.session.post(
                target.url,
                json=payload,
                headers=headers
            ) as response:
                response.raise_for_status()

                # Метрики
                ALERT_PUBLISHER_SUCCESS.labels(target=target.name, format=target.format.value).inc()
                return True

        except Exception as e:
            ALERT_PUBLISHER_ERRORS.labels(target=target.name, format=target.format.value).inc()
            logging.error(f"Failed to publish to {target.name}: {e}")
            raise

    def _substitute_env_vars(self, text: str) -> str:
        """Заменяет ${VAR_NAME} на значения из переменных окружения"""
        import re
        import os

        def replace_var(match):
            var_name = match.group(1)
            return os.environ.get(var_name, match.group(0))

        return re.sub(r'\$\{([^}]+)\}', replace_var, text)

### 5. DynamicTargetManager класс

```python
import asyncio
from kubernetes import client, config, watch
from typing import Dict, List, Optional, Set
import base64
import yaml
import logging

class DynamicTargetManager:
    """Управление publishing targets через Kubernetes secrets"""

    def __init__(self, config_manager):
        self.config = config_manager
        self.k8s_client = None
        self.current_targets: Dict[str, PublishingTarget] = {}
        self.watched_secrets: Set[str] = set()
        self.refresh_task = None

        # Load Kubernetes config
        try:
            config.load_incluster_config()  # Running in cluster
        except:
            config.load_kube_config()  # Development

        self.k8s_client = client.CoreV1Api()

    async def start_monitoring(self):
        """Запуск мониторинга secrets для автоматического обновления targets"""
        self.refresh_task = asyncio.create_task(self._refresh_loop())
        await self._discover_and_load_targets()

    async def stop_monitoring(self):
        """Остановка мониторинга"""
        if self.refresh_task:
            self.refresh_task.cancel()

    async def _refresh_loop(self):
        """Периодическое обновление конфигурации targets"""
        interval = self.config.get('publishing-discovery.config-refresh-interval', '300s')
        interval_seconds = self._parse_duration(interval)

        while True:
            try:
                await asyncio.sleep(interval_seconds)
                await self._discover_and_load_targets()
            except asyncio.CancelledError:
                break
            except Exception as e:
                logging.error(f"Error in target refresh loop: {e}")

    async def _discover_and_load_targets(self):
        """Обнаружение и загрузка publishing targets из secrets"""
        try:
            discovered_targets = await self._discover_targets_from_secrets()

            # Сравниваем с текущими targets
            if discovered_targets != self.current_targets:
                old_count = len(self.current_targets)
                self.current_targets = discovered_targets
                new_count = len(self.current_targets)

                logging.info(f"Publishing targets updated: {old_count} -> {new_count}")

                # Метрики
                PUBLISHING_TARGETS_DISCOVERED.set(new_count)

                # Логируем изменения
                for target_name, target in self.current_targets.items():
                    logging.info(f"Active publishing target: {target_name} -> {target.url}")

        except Exception as e:
            logging.error(f"Failed to discover publishing targets: {e}")

    async def _discover_targets_from_secrets(self) -> Dict[str, PublishingTarget]:
        """Обнаружение targets из Kubernetes secrets"""
        targets = {}

        # Get discovery configuration
        discovery_config = self.config.get('publishing-discovery', {})
        if not discovery_config.get('enabled', False):
            logging.info("Publishing target discovery disabled")
            return targets

        label_selectors = discovery_config.get('secret-labels', [])
        namespaces = discovery_config.get('secret-namespaces', ['default'])

        for namespace in namespaces:
            for label_selector in label_selectors:
                try:
                    secrets = self.k8s_client.list_namespaced_secret(
                        namespace=namespace,
                        label_selector=label_selector
                    )

                    for secret in secrets.items:
                        target = await self._create_target_from_secret(secret, namespace)
                        if target:
                            targets[target.name] = target

                except Exception as e:
                    logging.error(f"Error listing secrets in {namespace}: {e}")

        return targets

    async def _create_target_from_secret(self, secret, namespace: str) -> Optional[PublishingTarget]:
        """Создание PublishingTarget из Kubernetes secret"""
        try:
            secret_data = secret.data or {}

            # Decode base64 values
            decoded_data = {}
            for key, value in secret_data.items():
                try:
                    decoded_data[key] = base64.b64decode(value).decode('utf-8')
                except Exception as e:
                    logging.warning(f"Failed to decode secret key {key}: {e}")
                    continue

            # Extract target configuration
            target_name = decoded_data.get('target-name', secret.metadata.name)
            webhook_url = decoded_data.get('webhook-url') or decoded_data.get('url')

            if not webhook_url:
                logging.warning(f"No webhook URL found in secret {secret.metadata.name}")
                return None

            # Check if target is enabled
            enabled = decoded_data.get('enabled', 'true').lower() == 'true'
            if not enabled:
                logging.info(f"Target {target_name} is disabled")
                return None

            # Build headers
            headers = {'Content-Type': 'application/json'}

            # Authentication
            if 'auth-header' in decoded_data:
                auth_parts = decoded_data['auth-header'].split(' ', 1)
                if len(auth_parts) == 2:
                    headers['Authorization'] = decoded_data['auth-header']
            elif 'api-key' in decoded_data:
                headers['Authorization'] = f"Bearer {decoded_data['api-key']}"
            elif 'token' in decoded_data:
                headers['Authorization'] = f"Token {decoded_data['token']}"

            # Additional headers
            for key, value in decoded_data.items():
                if key.startswith('header-'):
                    header_name = key[7:].replace('-', '-').title()
                    headers[header_name] = value

            # Extract format
            format_str = decoded_data.get('format', 'alertmanager')
            try:
                target_format = PublishingFormat(format_str)
            except ValueError:
                target_format = PublishingFormat.ALERTMANAGER

            # Extract filter configuration
            filter_config = {}

            # Severity filter
            if 'filter-severity' in decoded_data:
                filter_config['severity'] = decoded_data['filter-severity'].split(',')
            elif 'severity' in decoded_data:
                filter_config['severity'] = decoded_data['severity'].split(',')

            # Namespace filter
            if 'filter-namespaces' in decoded_data:
                filter_config['namespaces'] = decoded_data['filter-namespaces'].split(',')
            elif 'namespaces' in decoded_data:
                filter_config['namespaces'] = decoded_data['namespaces'].split(',')

            # Exclude noise
            exclude_noise = decoded_data.get('exclude-noise', 'true').lower() == 'true'
            filter_config['exclude_noise'] = exclude_noise

            # Min confidence
            if 'min-confidence' in decoded_data:
                try:
                    filter_config['min_confidence'] = float(decoded_data['min-confidence'])
                except ValueError:
                    pass

            # Create target
            target = PublishingTarget(
                name=target_name,
                type="webhook",
                url=webhook_url,
                enabled=True,
                filter_config=filter_config,
                headers=headers,
                format=target_format
            )

            logging.info(f"Created publishing target from secret: {target_name}")
            return target

        except Exception as e:
            logging.error(f"Failed to create target from secret {secret.metadata.name}: {e}")
            return None

    def get_active_targets(self) -> List[PublishingTarget]:
        """Получить все активные publishing targets"""
        return list(self.current_targets.values())

    def get_target_by_name(self, name: str) -> Optional[PublishingTarget]:
        """Получить target по имени"""
        return self.current_targets.get(name)

    def get_targets_count(self) -> int:
        """Получить количество активных targets"""
        return len(self.current_targets)

    def is_metrics_only_mode(self) -> bool:
        """Проверить, работает ли сервис в режиме только метрик"""
        return len(self.current_targets) == 0

    def _parse_duration(self, duration: str) -> int:
        """Парсинг строки времени в секунды"""
        if duration.endswith('s'):
            return int(duration[:-1])
        elif duration.endswith('m'):
            return int(duration[:-1]) * 60
        elif duration.endswith('h'):
            return int(duration[:-1]) * 3600
        else:
            return int(duration)  # Assume seconds

class MetricsOnlyMode:
    """Режим работы только с метриками (без publishing)"""

    @staticmethod
    def create_enriched_alert(alert: Dict, classification: ClassificationResult) -> EnrichedAlert:
        """Создать обогащенный алерт для метрик"""
        return EnrichedAlert(
            original_alert=alert,
            classification=classification,
            enriched_labels={
                'llm_severity': classification.severity.value,
                'llm_confidence': str(classification.confidence),
                'processed_by': 'alert-history-llm'
            },
            enriched_annotations={
                'llm_reasoning': classification.reasoning,
                'classification_timestamp': datetime.utcnow().isoformat()
            },
            processing_metadata={
                'mode': 'metrics-only',
                'targets_available': 0
            }
        )

    @staticmethod
    def record_metrics(alert: EnrichedAlert):
        """Записать метрики для алерта"""
        # Основные метрики
        ALERTS_ENRICHED.labels(severity=alert.classification.severity.value).inc()
        PROXY_PROCESSING_TIME.observe(0.1)  # Minimal processing time

        # Метрика режима работы
        METRICS_ONLY_MODE.set(1)

    @staticmethod
    def get_dashboard_data() -> Dict:
        """Данные для dashboard в режиме только метрик"""
        return {
            "mode": "metrics-only",
            "publishing_enabled": False,
            "active_targets": 0,
            "message": "Service running in metrics-only mode. No publishing targets configured."
        }
```
```

### 4. Database Schema Extensions

**Новые таблицы:**

```sql
-- Классификации алертов
CREATE TABLE alert_classifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    alert_id INTEGER REFERENCES alert_history(id),
    severity TEXT NOT NULL, -- critical, warning, info, noise
    confidence REAL NOT NULL,
    reasoning TEXT,
    model_version TEXT,
    classified_at TEXT NOT NULL,
    FOREIGN KEY (alert_id) REFERENCES alert_history(id)
);

-- Рекомендации по настройкам
CREATE TABLE alert_recommendations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    alert_name TEXT NOT NULL,
    namespace TEXT,
    recommendation_type TEXT, -- flapping, threshold, duration
    current_config TEXT, -- JSON
    recommended_config TEXT, -- JSON
    reasoning TEXT,
    confidence REAL,
    status TEXT DEFAULT 'pending', -- pending, applied, rejected
    created_at TEXT NOT NULL,
    applied_at TEXT
);

-- Кеш классификаций
CREATE TABLE classification_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    alert_hash TEXT UNIQUE NOT NULL, -- hash алерта для кеширования
    classification_result TEXT NOT NULL, -- JSON результат
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);

-- Метрики LLM вызовов
CREATE TABLE llm_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    operation_type TEXT NOT NULL, -- classification, recommendation
    model_name TEXT NOT NULL,
    tokens_used INTEGER,
    processing_time REAL,
    success BOOLEAN,
    error_message TEXT,
    created_at TEXT NOT NULL
);
```

### 5. API Endpoints

**Новые эндпоинты:**

```python
# --- INTELLIGENT PROXY ENDPOINTS ---

@app.post("/webhook/proxy")
async def intelligent_webhook_proxy(request: Request):
    """
    Intelligent proxy для алертов от Alertmanager:
    1. Принимает алерты от Alertmanager
    2. Классифицирует через LLM
    3. Обогащает метаданными
    4. Публикует в downstream системы
    5. Сохраняет в историю
    """
    start = time.time()
    try:
        payload = await request.json()
        alerts = payload.get("alerts", [])

        processed_alerts = []
        publishing_results = {}

        async with AlertPublisher(publishing_targets, AlertFormatter()) as publisher:
            for alert in alerts:
                # 1. Классифицируем алерт
                classification = await classifier.classify_alert(alert)

                # 2. Обогащаем алерт
                enriched_alert = EnrichedAlert(
                    original_alert=alert,
                    classification=classification,
                    enriched_labels={},
                    enriched_annotations={},
                    processing_metadata={
                        'classified_at': datetime.utcnow().isoformat(),
                        'processing_time': time.time() - start
                    }
                )

                # 3. Сохраняем в историю (как обычно)
                await save_to_history(alert, classification)

                # 4. Публикуем в downstream системы
                publish_results = await publisher.publish_alert(enriched_alert)
                publishing_results[alert.get('fingerprint', 'unknown')] = publish_results

                processed_alerts.append({
                    'fingerprint': alert.get('fingerprint'),
                    'classification': classification.severity.value,
                    'confidence': classification.confidence,
                    'published_to': list(publish_results.keys())
                })

        return {
            "result": "ok",
            "processed": len(processed_alerts),
            "alerts": processed_alerts,
            "publishing_results": publishing_results
        }

    except Exception as e:
        logging.error(f"Intelligent proxy error: {e}")
        # Fallback - сохраняем как обычные алерты без классификации
        await webhook(request)  # Существующий handler
        raise HTTPException(status_code=500, detail=f"Proxy processing failed: {e}")

# --- DYNAMIC PUBLISHING MANAGEMENT ---

@app.get("/publishing/targets")
async def get_publishing_targets() -> Dict:
    """Получает список целей для публикации алертов"""
    targets = target_manager.get_active_targets()

    return {
        "targets": [
            {
                "name": target.name,
                "type": target.type,
                "url": target.url[:50] + "..." if len(target.url) > 50 else target.url,  # Truncate for security
                "format": target.format.value,
                "enabled": target.enabled,
                "filter_config": target.filter_config
            }
            for target in targets
        ],
        "total_count": len(targets),
        "metrics_only_mode": target_manager.is_metrics_only_mode(),
        "discovery_enabled": target_manager.config.get('publishing-discovery.enabled', False)
    }

@app.post("/publishing/targets/refresh")
async def refresh_publishing_targets() -> Dict:
    """Принудительное обновление publishing targets из secrets"""
    old_count = target_manager.get_targets_count()
    await target_manager._discover_and_load_targets()
    new_count = target_manager.get_targets_count()

    return {
        "result": "ok",
        "targets_before": old_count,
        "targets_after": new_count,
        "refreshed_at": datetime.utcnow().isoformat()
    }

@app.get("/publishing/mode")
async def get_publishing_mode() -> Dict:
    """Получает текущий режим работы publishing"""
    targets_count = target_manager.get_targets_count()

    return {
        "mode": "metrics-only" if targets_count == 0 else "publishing",
        "targets_count": targets_count,
        "metrics_only": targets_count == 0,
        "available_targets": [t.name for t in target_manager.get_active_targets()]
    }

@app.get("/publishing/stats")
async def get_publishing_stats(
    since: Optional[str] = None,
    until: Optional[str] = None,
    target_name: Optional[str] = None
) -> Dict:
    """Статистика публикации алертов по целям"""
    # Query publishing log from database
    query = "SELECT target_name, status, COUNT(*) as count FROM publishing_log WHERE 1=1"
    params = []

    if since:
        query += " AND published_at >= ?"
        params.append(since)
    if until:
        query += " AND published_at <= ?"
        params.append(until)
    if target_name:
        query += " AND target_name = ?"
        params.append(target_name)

    query += " GROUP BY target_name, status ORDER BY target_name, status"

    with get_db() as conn:
        c = conn.cursor()
        c.execute(query, params)
        rows = c.fetchall()

    # Aggregate stats
    stats_by_target = {}
    total_stats = {"success": 0, "failed": 0, "total": 0}

    for target, status, count in rows:
        if target not in stats_by_target:
            stats_by_target[target] = {"success": 0, "failed": 0, "total": 0}

        stats_by_target[target][status] = count
        stats_by_target[target]["total"] += count
        total_stats[status] += count
        total_stats["total"] += count

    return {
        "period": {"since": since, "until": until},
        "total_stats": total_stats,
        "by_target": stats_by_target,
        "active_targets": target_manager.get_targets_count(),
        "metrics_only_mode": target_manager.is_metrics_only_mode()
    }

@app.post("/publishing/test/{target_name}")
async def test_publishing_target(target_name: str, test_payload: Optional[Dict] = None) -> Dict:
    """Тестирует публикацию в конкретную цель"""
    target = target_manager.get_target_by_name(target_name)
    if not target:
        raise HTTPException(status_code=404, detail=f"Publishing target {target_name} not found")

    # Create test alert if not provided
    if not test_payload:
        test_payload = {
            "alertname": "TestAlert",
            "status": "firing",
            "labels": {
                "alertname": "TestAlert",
                "namespace": "test",
                "severity": "info"
            },
            "annotations": {
                "description": "Test alert for publishing target validation"
            },
            "fingerprint": "test-fingerprint",
            "startsAt": datetime.utcnow().isoformat(),
            "generatorURL": "http://test"
        }

    # Create mock classification
    test_classification = ClassificationResult(
        severity=AlertSeverity.INFO,
        confidence=0.95,
        reasoning="Test classification for publishing validation",
        recommendations=["Test recommendation"],
        processing_time=0.1
    )

    # Create enriched alert
    enriched_alert = EnrichedAlert(
        original_alert=test_payload,
        classification=test_classification,
        enriched_labels={},
        enriched_annotations={},
        processing_metadata={"test_mode": True}
    )

    # Test publishing
    try:
        async with AlertPublisher([target], AlertFormatter()) as publisher:
            results = await publisher.publish_alert(enriched_alert)

        return {
            "target_name": target_name,
            "test_result": "success" if results.get(target_name, False) else "failed",
            "target_url": target.url,
            "target_format": target.format.value,
            "tested_at": datetime.utcnow().isoformat(),
            "details": results
        }

    except Exception as e:
        return {
            "target_name": target_name,
            "test_result": "failed",
            "error": str(e),
            "tested_at": datetime.utcnow().isoformat()
        }

@app.get("/publishing/secrets/template")
async def get_secret_template(format: str = "rootly") -> Dict:
    """Получает template для создания secret для publishing target"""
    templates = {
        "rootly": {
            "apiVersion": "v1",
            "kind": "Secret",
            "metadata": {
                "name": "rootly-publishing-target",
                "labels": {
                    "publishing-target": "true",
                    "team": "your-team"
                }
            },
            "type": "Opaque",
            "stringData": {
                "target-name": "rootly-production",
                "webhook-url": "https://api.rootly.com/v1/alerts",
                "api-key": "your-rootly-api-key",
                "format": "rootly",
                "enabled": "true",
                "filter-severity": "critical,warning",
                "exclude-noise": "true"
            }
        },
        "pagerduty": {
            "apiVersion": "v1",
            "kind": "Secret",
            "metadata": {
                "name": "pagerduty-publishing-target",
                "labels": {
                    "publishing-target": "true",
                    "team": "your-team"
                }
            },
            "type": "Opaque",
            "stringData": {
                "target-name": "pagerduty-critical",
                "webhook-url": "https://events.pagerduty.com/v2/enqueue",
                "routing-key": "your-pagerduty-routing-key",
                "format": "pagerduty",
                "enabled": "true",
                "filter-severity": "critical",
                "filter-namespaces": "production,staging"
            }
        },
        "slack": {
            "apiVersion": "v1",
            "kind": "Secret",
            "metadata": {
                "name": "slack-publishing-target",
                "labels": {
                    "publishing-target": "true",
                    "team": "your-team"
                }
            },
            "type": "Opaque",
            "stringData": {
                "target-name": "slack-alerts",
                "webhook-url": "https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK",
                "format": "slack",
                "enabled": "true",
                "filter-severity": "critical,warning",
                "exclude-noise": "true"
            }
        },
        "webhook": {
            "apiVersion": "v1",
            "kind": "Secret",
            "metadata": {
                "name": "custom-webhook-target",
                "labels": {
                    "publishing-target": "true",
                    "team": "your-team"
                }
            },
            "type": "Opaque",
            "stringData": {
                "target-name": "custom-webhook",
                "webhook-url": "https://your-webhook-endpoint.com/alerts",
                "auth-header": "Bearer your-auth-token",
                "format": "alertmanager",
                "enabled": "true",
                "filter-severity": "critical,warning,info",
                "min-confidence": "0.8"
            }
        }
    }

    template = templates.get(format)
    if not template:
        raise HTTPException(status_code=400, detail=f"Unknown template format: {format}")

    return {
        "format": format,
        "template": template,
        "available_formats": list(templates.keys()),
        "usage": f"kubectl apply -f {format}-secret.yaml"
    }

# --- CLASSIFICATION ENDPOINTS ---

@app.post("/classify")
async def classify_alert(alert_data: Dict) -> ClassificationResult:
    """Классифицирует алерт в реальном времени"""
    pass

@app.get("/classification/{alert_id}")
async def get_classification(alert_id: int) -> ClassificationResult:
    """Получает классификацию для существующего алерта"""
    pass

@app.post("/classify/batch")
async def batch_classify(alert_ids: List[int]) -> List[ClassificationResult]:
    """Пакетная классификация исторических алертов"""
    pass

# --- RECOMMENDATIONS ENDPOINTS ---

@app.get("/recommendations")
async def get_recommendations(
    alert_name: Optional[str] = None,
    namespace: Optional[str] = None,
    status: Optional[str] = None
) -> List[AlertRecommendation]:
    """Получает рекомендации по настройкам"""
    pass

@app.post("/recommendations/generate")
async def generate_recommendations() -> Dict[str, int]:
    """Генерирует новые рекомендации на основе анализа"""
    pass

@app.patch("/recommendations/{rec_id}/status")
async def update_recommendation_status(rec_id: int, status: str) -> Dict:
    """Обновляет статус рекомендации (applied/rejected)"""
    pass

# --- ENRICHED HISTORY ---

@app.get("/history/enriched")
async def get_enriched_history(
    alertname: Optional[str] = Query(None),
    severity: Optional[str] = Query(None),  # critical, warning, info, noise
    min_confidence: Optional[float] = Query(None),
    namespace: Optional[str] = Query(None),
    since: Optional[str] = Query(None),
    until: Optional[str] = Query(None),
    limit: int = Query(100, ge=1, le=1000),
    offset: int = Query(0, ge=0)
) -> List[Dict]:
    """Получает историю алертов с классификацией и обогащением"""
    pass
```

## Helm Chart расширения

### ConfigMap template

```yaml
# templates/configmap-llm.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ include "alert-history.fullname" . }}-llm-config
  labels:
    {{- include "alert-history.labels" . | nindent 4 }}
data:
  llm-proxy-url: {{ .Values.llm.proxyUrl | quote }}
  classification-enabled: {{ .Values.llm.classification.enabled | quote }}
  recommendations-enabled: {{ .Values.llm.recommendations.enabled | quote }}
  batch-size: {{ .Values.llm.batchSize | quote }}
  cache-ttl: {{ .Values.llm.cacheTtl | quote }}
  models: |
    classification: {{ .Values.llm.models.classification | quote }}
    recommendations: {{ .Values.llm.models.recommendations | quote }}
  prompts: |
    {{- .Values.llm.prompts | toYaml | nindent 4 }}
  retry-config: |
    max-retries: {{ .Values.llm.retry.maxRetries }}
    backoff-multiplier: {{ .Values.llm.retry.backoffMultiplier }}
    max-wait: {{ .Values.llm.retry.maxWait }}
```

### Secret template

```yaml
# templates/secret-llm.yaml
{{- if .Values.llm.enabled }}
apiVersion: v1
kind: Secret
metadata:
  name: {{ include "alert-history.fullname" . }}-llm-secrets
  labels:
    {{- include "alert-history.labels" . | nindent 4 }}
type: Opaque
data:
  {{- if .Values.llm.auth.apiKey }}
  llm-proxy-api-key: {{ .Values.llm.auth.apiKey | b64enc }}
  {{- end }}
  {{- if .Values.llm.auth.authToken }}
  llm-proxy-auth-token: {{ .Values.llm.auth.authToken | b64enc }}
  {{- end }}
{{- end }}
```

### Values.yaml расширения

```yaml
# Дополнения к values.yaml для 12-Factor + Horizontal Scaling
replicaCount: 3  # Minimum replicas for HA

image:
  repository: alert-history
  pullPolicy: IfNotPresent
  tag: "latest"

# 12-Factor Configuration
config:
  # Factor III: Config via environment variables
  logLevel: "INFO"
  metricsEnabled: true
  publishingEnabled: true

  # Database configuration (Factor IV: Backing Services)
  database:
    enabled: true
    type: "postgresql"  # Changed from SQLite
    host: "postgresql"
    port: 5432
    name: "alerthistory"
    username: "alerthistory"
    # Password from secret

  # Redis configuration for caching and sessions
  redis:
    enabled: true
    host: "redis"
    port: 6379
    database: 0
    # Password from secret if needed

# Horizontal Pod Autoscaler
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
  targetMemoryUtilizationPercentage: 80
  # Custom metrics
  customMetrics:
    - type: Pods
      pods:
        metric:
          name: requests_per_second
        target:
          type: AverageValue
          averageValue: "30"

# Resource limits for proper HPA
resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 200m
    memory: 256Mi

# LLM Configuration
llm:
  enabled: true
  proxyUrl: "http://llm-proxy.llm-namespace:8080"

  classification:
    enabled: true
    autoClassify: true

  recommendations:
    enabled: true
    analyzeFlapping: true

  models:
    classification: "gpt-4"
    recommendations: "gpt-4"

  batchSize: 10
  cacheTtl: 3600

  auth:
    apiKey: ""  # Set via --set or external secret
    authToken: ""  # Optional additional auth

  retry:
    maxRetries: 3
    backoffMultiplier: 2
    maxWait: 30

# PostgreSQL dependency
postgresql:
  enabled: true
  auth:
    postgresPassword: "postgres"
    username: "alerthistory"
    password: "alerthistory"
    database: "alerthistory"
  primary:
    persistence:
      enabled: true
      size: 8Gi
    resources:
      requests:
        memory: 256Mi
        cpu: 250m

# Redis dependency
redis:
  enabled: true
  auth:
    enabled: false  # Enable if needed
  master:
    persistence:
      enabled: true
      size: 2Gi
    resources:
      requests:
        memory: 128Mi
        cpu: 100m

# Health checks for Factor IX: Disposability
livenessProbe:
  httpGet:
    path: /healthz
    port: http
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5

readinessProbe:
  httpGet:
    path: /healthz
    port: http
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 3

# Graceful shutdown
terminationGracePeriodSeconds: 30

# Service configuration
service:
  type: ClusterIP
  port: 8080
  sessionAffinity: None  # Stateless design

# Ingress configuration
ingress:
  enabled: false
  annotations:
    nginx.ingress.kubernetes.io/load-balance: "round_robin"
  hosts:
    - host: alert-history.local
      paths:
        - path: /
          pathType: Prefix

  prompts:
    classification: |
      Analyze this Prometheus alert and classify it into one of these categories:
      - critical: Requires immediate attention, service is down or severely degraded
      - warning: Performance issue or potential problem, needs investigation
      - info: Informational alert, good to know but not urgent
      - noise: False positive or irrelevant alert

      Alert data: {alert_data}
      Historical context: {historical_context}

      Respond with your classification and reasoning.

    recommendations: |
      Based on this alert pattern analysis, provide recommendations for Alertmanager configuration:
      Alert: {alert_name}
      Flapping count: {flap_count}
      Average duration: {avg_duration}

      Suggest improvements for: evaluation_interval, for duration, repeat_interval, thresholds
```

### Deployment изменения

```yaml
# В templates/deployment.yaml добавляем:
spec:
  template:
    spec:
      containers:
      - name: {{ .Chart.Name }}
        env:
        {{- if .Values.llm.enabled }}
        - name: LLM_ENABLED
          value: "true"
        - name: LLM_PROXY_URL
          valueFrom:
            configMapKeyRef:
              name: {{ include "alert-history.fullname" . }}-llm-config
              key: llm-proxy-url
        - name: LLM_API_KEY
          valueFrom:
            secretKeyRef:
              name: {{ include "alert-history.fullname" . }}-llm-secrets
              key: llm-proxy-api-key
        {{- if .Values.llm.auth.authToken }}
        - name: LLM_AUTH_TOKEN
          valueFrom:
            secretKeyRef:
              name: {{ include "alert-history.fullname" . }}-llm-secrets
              key: llm-proxy-auth-token
        {{- end }}
        {{- end }}
        volumeMounts:
        {{- if .Values.llm.enabled }}
        - name: llm-config
          mountPath: /app/config/llm
          readOnly: true
        {{- end }}
      volumes:
      {{- if .Values.llm.enabled }}
      - name: llm-config
        configMap:
          name: {{ include "alert-history.fullname" . }}-llm-config
      {{- end }}
```

## Мониторинг и метрики

### Prometheus метрики

```python
# Дополнительные метрики для LLM функциональности
LLM_REQUESTS = Counter('alert_history_llm_requests_total',
                      'Total LLM requests', ['operation', 'model', 'status'])
LLM_LATENCY = Histogram('alert_history_llm_latency_seconds',
                       'LLM request latency', ['operation', 'model'])
LLM_TOKENS = Counter('alert_history_llm_tokens_total',
                    'Total tokens used', ['operation', 'model'])
CLASSIFICATION_ACCURACY = Gauge('alert_history_classification_accuracy',
                              'Classification accuracy rate')
CACHE_HITS = Counter('alert_history_cache_hits_total',
                    'Cache hits', ['type'])

# Метрики для publishing функциональности
ALERT_PUBLISHER_SUCCESS = Counter('alert_history_publisher_success_total',
                                 'Successful alert publications', ['target', 'format'])
ALERT_PUBLISHER_ERRORS = Counter('alert_history_publisher_errors_total',
                                'Failed alert publications', ['target', 'format'])
ALERT_PUBLISHER_LATENCY = Histogram('alert_history_publisher_latency_seconds',
                                   'Alert publishing latency', ['target', 'format'])
ALERTS_FILTERED = Counter('alert_history_alerts_filtered_total',
                         'Alerts filtered out', ['target', 'reason'])
ALERTS_ENRICHED = Counter('alert_history_alerts_enriched_total',
                         'Alerts enriched with classification', ['severity'])
PROXY_PROCESSING_TIME = Histogram('alert_history_proxy_processing_seconds',
                                 'Total proxy processing time per alert')

# Метрики для динамических publishing targets
PUBLISHING_TARGETS_DISCOVERED = Gauge('alert_history_publishing_targets_discovered',
                                     'Number of discovered publishing targets')
METRICS_ONLY_MODE = Gauge('alert_history_metrics_only_mode',
                         'Whether service is running in metrics-only mode (1) or with publishing (0)')
TARGET_DISCOVERY_ERRORS = Counter('alert_history_target_discovery_errors_total',
                                 'Errors during target discovery from secrets')
SECRET_REFRESH_DURATION = Histogram('alert_history_secret_refresh_duration_seconds',
                                   'Time spent refreshing publishing targets from secrets')
PUBLISHING_TARGET_STATUS = Gauge('alert_history_publishing_target_status',
                                'Status of individual publishing targets (1=active, 0=inactive)',
                                ['target_name', 'target_format'])
```

## Сценарии ошибок и восстановления

### 1. LLM-proxy недоступен
- **Fallback:** Использовать последние кешированные классификации
- **Behavior:** Логировать ошибки, продолжать базовую функциональность
- **Recovery:** Auto-retry с exponential backoff

### 2. Неверные классификации
- **Detection:** Метрики точности, user feedback
- **Mitigation:** A/B тестирование промптов, fine-tuning
- **Recovery:** Откат к предыдущей версии промптов

## Режимы обогащения алертов (Transparent vs Enriched)

### Обзор
- Режим глобальный для сервиса, консистентный между репликами.
- Источник истины — Redis ключ `enrichment:mode`. Fallback — in-memory `app_state.enrichment_mode` и `ENRICHMENT_MODE` (default=`enriched`).

### API
- `GET /enrichment/mode` → `{ mode: "transparent"|"enriched", source: "redis"|"memory"|"default" }`
- `POST /enrichment/mode` body `{ mode: "transparent"|"enriched" }` → сохраняет в Redis или память

### Интеграция в пайплайн `/webhook/proxy`
- Если `transparent`: во время шага сохранения алертов отключаем `enable_auto_classification` у `WebhookProcessor` (без LLM), остальная логика (фильтры/публикации) сохраняется.
- Если `enriched`: текущий behavior — классификация включена.

### Дашборд
- Секция Publishing показывает текущий режим и предоставляет кнопки для переключения через API.

### Безопасность
- В прод-окружениях — защитить `/enrichment/mode` аутентификацией, RBAC и аудитом. В демо — доступно без аутентификации.

### 3. Превышение лимитов
- **Prevention:** Rate limiting, batch optimization
- **Handling:** Queue система для отложенной обработки
- **Monitoring:** Алерты по превышению лимитов

---

**Архитектор:** DevOps Team
**Дата создания:** 2024-12-28
**Версия:** 1.0
**Статус:** Draft → Technical Review → Implementation Ready
