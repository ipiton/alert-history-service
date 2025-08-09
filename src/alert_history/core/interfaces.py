"""
Core interfaces for Alert History Service.

Implements SOLID principles through well-defined interfaces:
- Single Responsibility: Each interface has one clear purpose
- Interface Segregation: Small, focused interfaces
- Dependency Inversion: Depend on abstractions, not concretions
"""

# Standard library imports
from abc import ABC, abstractmethod
from dataclasses import dataclass
from datetime import datetime
from enum import Enum
from typing import Any, Dict, List, Optional


class AlertSeverity(Enum):
    """Alert severity levels."""

    CRITICAL = "critical"
    WARNING = "warning"
    INFO = "info"
    NOISE = "noise"


class AlertStatus(Enum):
    """Alert status values."""

    FIRING = "firing"
    RESOLVED = "resolved"


class PublishingFormat(Enum):
    """Publishing format options."""

    ALERTMANAGER = "alertmanager"
    ROOTLY = "rootly"
    PAGERDUTY = "pagerduty"
    SLACK = "slack"
    WEBHOOK = "webhook"


@dataclass
class Alert:
    """Alert data model."""

    fingerprint: str
    alert_name: str
    status: AlertStatus
    labels: Dict[str, str]
    annotations: Dict[str, str]
    starts_at: datetime
    ends_at: Optional[datetime] = None
    generator_url: Optional[str] = None

    @property
    def namespace(self) -> Optional[str]:
        """Get alert namespace from labels."""
        return self.labels.get("namespace")

    @property
    def severity(self) -> Optional[str]:
        """Get alert severity from labels."""
        return self.labels.get("severity")


@dataclass
class ClassificationResult:
    """LLM classification result."""

    severity: AlertSeverity
    confidence: float
    reasoning: str
    recommendations: List[str]
    processing_time: float
    metadata: Optional[Dict[str, Any]] = None


@dataclass
class PublishingTarget:
    """Publishing target configuration."""

    name: str
    type: str
    url: str
    enabled: bool
    filter_config: Dict[str, Any]
    headers: Dict[str, str]
    format: PublishingFormat


@dataclass
class EnrichedAlert:
    """Alert enriched with classification data."""

    alert: Alert
    classification: Optional[ClassificationResult] = None
    enrichment_metadata: Optional[Dict[str, Any]] = None
    processing_timestamp: Optional[datetime] = None


# Storage Interfaces (Single Responsibility Principle)


class IAlertStorage(ABC):
    """Interface for alert storage operations."""

    @abstractmethod
    async def save_alert(self, alert: Alert) -> bool:
        """Save alert to storage."""
        pass

    @abstractmethod
    async def get_alert_by_fingerprint(self, fingerprint: str) -> Optional[Alert]:
        """Get alert by fingerprint."""
        pass

    @abstractmethod
    async def get_alerts(
        self, filters: Dict[str, Any], limit: int = 100, offset: int = 0
    ) -> List[Alert]:
        """Get alerts with filters."""
        pass

    @abstractmethod
    async def cleanup_old_alerts(self, retention_days: int) -> int:
        """Clean up old alerts and return count of deleted records."""
        pass


class IClassificationStorage(ABC):
    """Interface for classification storage operations."""

    @abstractmethod
    async def save_classification(
        self, fingerprint: str, result: ClassificationResult
    ) -> bool:
        """Save classification result."""
        pass

    @abstractmethod
    async def get_classification(
        self, fingerprint: str
    ) -> Optional[ClassificationResult]:
        """Get classification by fingerprint."""
        pass


class IPublishingLogStorage(ABC):
    """Interface for publishing log storage."""

    @abstractmethod
    async def log_publishing_attempt(
        self,
        fingerprint: str,
        target_name: str,
        success: bool,
        error_message: Optional[str] = None,
        processing_time: Optional[float] = None,
    ) -> bool:
        """Log publishing attempt."""
        pass


# Caching Interfaces (Interface Segregation Principle)


class ICache(ABC):
    """Generic cache interface."""

    @abstractmethod
    async def get(self, key: str) -> Optional[Any]:
        """Get value from cache."""
        pass

    @abstractmethod
    async def set(self, key: str, value: Any, ttl: Optional[int] = None) -> bool:
        """Set value in cache."""
        pass

    @abstractmethod
    async def delete(self, key: str) -> bool:
        """Delete value from cache."""
        pass

    @abstractmethod
    async def exists(self, key: str) -> bool:
        """Check if key exists in cache."""
        pass


class IDistributedLock(ABC):
    """Interface for distributed locking."""

    @abstractmethod
    async def acquire_lock(self, key: str, timeout: int = 60) -> bool:
        """Acquire distributed lock."""
        pass

    @abstractmethod
    async def release_lock(self, key: str) -> bool:
        """Release distributed lock."""
        pass


# LLM Service Interfaces (Single Responsibility Principle)


class ILLMClient(ABC):
    """Interface for LLM communication."""

    @abstractmethod
    async def classify_alert(
        self, alert: Alert, context: Optional[Dict[str, Any]] = None
    ) -> ClassificationResult:
        """Classify alert using LLM."""
        pass

    @abstractmethod
    async def generate_recommendations(
        self, alert: Alert, classification: ClassificationResult
    ) -> List[str]:
        """Generate configuration recommendations."""
        pass


class IAlertClassifier(ABC):
    """Interface for alert classification service."""

    @abstractmethod
    async def classify(self, alert: Alert) -> ClassificationResult:
        """Classify alert with caching and error handling."""
        pass


# Publishing Interfaces (Interface Segregation Principle)


class IAlertFormatter(ABC):
    """Interface for alert formatting."""

    @abstractmethod
    async def format_alert(
        self, enriched_alert: EnrichedAlert, target_format: PublishingFormat
    ) -> Dict[str, Any]:
        """Format alert for specific target."""
        pass


class IAlertPublisher(ABC):
    """Interface for alert publishing."""

    @abstractmethod
    async def publish_alert(
        self, enriched_alert: EnrichedAlert, target: PublishingTarget
    ) -> bool:
        """Publish alert to target."""
        pass


class IFilterEngine(ABC):
    """Interface for alert filtering."""

    @abstractmethod
    async def should_publish(
        self, enriched_alert: EnrichedAlert, target: PublishingTarget
    ) -> bool:
        """Check if alert should be published to target."""
        pass


# Configuration Management Interfaces (Dependency Inversion Principle)


class IConfigurationManager(ABC):
    """Interface for configuration management."""

    @abstractmethod
    async def get_config(self, key: str, default: Any = None) -> Any:
        """Get configuration value."""
        pass

    @abstractmethod
    async def get_all_configs(self) -> Dict[str, Any]:
        """Get all configuration values."""
        pass

    @abstractmethod
    async def reload_config(self) -> bool:
        """Reload configuration from source."""
        pass


class ISecretsManager(ABC):
    """Interface for secrets management."""

    @abstractmethod
    async def get_secret(self, key: str) -> Optional[str]:
        """Get secret value."""
        pass

    @abstractmethod
    async def list_secrets(self, label_selector: str) -> Dict[str, Dict[str, str]]:
        """List secrets matching label selector."""
        pass


# Target Discovery Interface (Open/Closed Principle)


class ITargetDiscovery(ABC):
    """Interface for dynamic target discovery."""

    @abstractmethod
    async def discover_targets(self) -> List[PublishingTarget]:
        """Discover available publishing targets."""
        pass

    @abstractmethod
    async def refresh_targets(self) -> bool:
        """Refresh target configuration."""
        pass


# Health Check Interface (Single Responsibility Principle)


class IHealthChecker(ABC):
    """Interface for health checking."""

    @abstractmethod
    async def check_health(self) -> Dict[str, Any]:
        """Perform health check."""
        pass

    @abstractmethod
    async def check_readiness(self) -> Dict[str, Any]:
        """Perform readiness check."""
        pass


# Metrics Interface (Interface Segregation Principle)


class IMetricsCollector(ABC):
    """Interface for metrics collection."""

    @abstractmethod
    def increment_counter(
        self, name: str, labels: Optional[Dict[str, str]] = None
    ) -> None:
        """Increment counter metric."""
        pass

    @abstractmethod
    def set_gauge(
        self, name: str, value: float, labels: Optional[Dict[str, str]] = None
    ) -> None:
        """Set gauge metric."""
        pass

    @abstractmethod
    def observe_histogram(
        self, name: str, value: float, labels: Optional[Dict[str, str]] = None
    ) -> None:
        """Observe histogram metric."""
        pass


# Event Processing Interface (Strategy Pattern for extensibility)


class IEventProcessor(ABC):
    """Interface for event processing strategies."""

    @abstractmethod
    async def process_event(self, event_data: Dict[str, Any]) -> bool:
        """Process incoming event."""
        pass

    @abstractmethod
    def can_handle(self, event_type: str) -> bool:
        """Check if processor can handle event type."""
        pass


# Repository Pattern Interfaces (DRY principle)


class IRepository(ABC):
    """Generic repository interface."""

    @abstractmethod
    async def create(self, entity: Any) -> bool:
        """Create entity."""
        pass

    @abstractmethod
    async def get_by_id(self, entity_id: str) -> Optional[Any]:
        """Get entity by ID."""
        pass

    @abstractmethod
    async def update(self, entity: Any) -> bool:
        """Update entity."""
        pass

    @abstractmethod
    async def delete(self, entity_id: str) -> bool:
        """Delete entity."""
        pass

    @abstractmethod
    async def list(
        self,
        filters: Optional[Dict[str, Any]] = None,
        limit: int = 100,
        offset: int = 0,
    ) -> List[Any]:
        """List entities with optional filters."""
        pass
