"""
Dynamic Target Discovery для publishing алертов.

Обнаруживает и управляет publishing targets из Kubernetes Secrets:
- Автоматическое обнаружение через label selectors
- Periodic refresh конфигурации
- Support для Rootly, PagerDuty, Slack, custom webhooks
- Metrics-only mode fallback
"""

# Standard library imports
import asyncio
import base64
import time
from dataclasses import dataclass
from typing import Optional

try:
    # Third-party imports
    from kubernetes import client, config
    from kubernetes.client.rest import ApiException

    KUBERNETES_AVAILABLE = True
except ImportError:
    KUBERNETES_AVAILABLE = False

    # Mock classes for development without kubernetes
    class client:
        class CoreV1Api:
            pass

    class config:
        @staticmethod
        def load_incluster_config():
            raise Exception("Kubernetes not available")

        @staticmethod
        def load_kube_config():
            raise Exception("Kubernetes not available")


# Local imports
from ..core.interfaces import PublishingFormat, PublishingTarget
from ..logging_config import get_logger
from ..utils.common import parse_duration, validate_url

logger = get_logger(__name__)


@dataclass
class TargetDiscoveryConfig:
    """Конфигурация для discovery targets."""

    enabled: bool = True
    secret_labels: list[str] = None
    secret_namespaces: list[str] = None
    config_refresh_interval: str = "300s"  # 5 minutes

    def __post_init__(self):
        if self.secret_labels is None:
            self.secret_labels = ["publishing-target=true"]
        if self.secret_namespaces is None:
            self.secret_namespaces = ["default"]


class DynamicTargetManager:
    """
    Управление publishing targets через Kubernetes secrets.

    Supports:
    - Automatic discovery через label selectors
    - Multiple secret formats (single secret, multiple secrets)
    - Hot reload конфигурации
    - Fallback to metrics-only mode
    """

    def __init__(self, discovery_config: TargetDiscoveryConfig):
        """Initialize target manager."""
        self.config = discovery_config
        self.k8s_client: Optional[client.CoreV1Api] = None
        self.current_targets: dict[str, PublishingTarget] = {}
        self.watched_secrets: set[str] = set()
        self.refresh_task: Optional[asyncio.Task] = None
        self._last_refresh_time = 0.0

        # Initialize Kubernetes client
        self._init_kubernetes_client()

    def _init_kubernetes_client(self) -> None:
        """Initialize Kubernetes API client."""
        if not KUBERNETES_AVAILABLE:
            logger.warning(
                "Kubernetes client not available, running in development mode"
            )
            return

        try:
            # Try in-cluster config first
            config.load_incluster_config()
            logger.info("Loaded Kubernetes in-cluster configuration")
        except Exception:
            try:
                # Fallback to local kubeconfig
                config.load_kube_config()
                logger.info("Loaded Kubernetes configuration from kubeconfig")
            except Exception as e:
                logger.error(f"Failed to load Kubernetes configuration: {e}")
                return

        self.k8s_client = client.CoreV1Api()

    async def start_monitoring(self) -> None:
        """Запуск мониторинга secrets для автоматического обновления targets."""
        if not self.config.enabled:
            logger.info("Target discovery disabled")
            return

        if not self.k8s_client:
            logger.warning(
                "Kubernetes client not available, targets discovery disabled"
            )
            return

        # Initial discovery
        await self._discover_and_load_targets()

        # Start periodic refresh
        self.refresh_task = asyncio.create_task(self._refresh_loop())

        logger.info(
            "Target discovery started",
            refresh_interval=self.config.config_refresh_interval,
            namespaces=self.config.secret_namespaces,
            label_selectors=self.config.secret_labels,
        )

    async def stop_monitoring(self) -> None:
        """Остановка мониторинга."""
        if self.refresh_task:
            self.refresh_task.cancel()
            try:
                await self.refresh_task
            except asyncio.CancelledError:
                pass

        logger.info("Target discovery stopped")

    async def _refresh_loop(self) -> None:
        """Периодическое обновление конфигурации targets."""
        interval_seconds = parse_duration(self.config.config_refresh_interval)

        while True:
            try:
                await asyncio.sleep(interval_seconds)
                await self._discover_and_load_targets()
            except asyncio.CancelledError:
                break
            except Exception as e:
                logger.error(f"Error in target refresh loop: {e}")
                # Continue loop even after errors

    async def _discover_and_load_targets(self) -> None:
        """Обнаружение и загрузка publishing targets из secrets."""
        if not self.k8s_client:
            return

        start_time = time.time()

        try:
            discovered_targets = await self._discover_targets_from_secrets()

            # Compare with current targets
            if discovered_targets != self.current_targets:
                old_count = len(self.current_targets)
                self.current_targets = discovered_targets
                new_count = len(self.current_targets)

                logger.info(
                    "Publishing targets updated",
                    old_count=old_count,
                    new_count=new_count,
                    targets=list(self.current_targets.keys()),
                )

                # Log each target
                for target_name, target in self.current_targets.items():
                    logger.info(
                        "Active publishing target",
                        name=target_name,
                        type=target.type,
                        format=target.format.value,
                        url=target.url,
                    )

            self._last_refresh_time = time.time()
            refresh_duration = self._last_refresh_time - start_time

            logger.debug(
                "Target discovery completed",
                targets_count=len(self.current_targets),
                refresh_duration=refresh_duration,
            )

        except Exception as e:
            logger.error(f"Failed to discover publishing targets: {e}")

    async def _discover_targets_from_secrets(self) -> dict[str, PublishingTarget]:
        """Обнаружение targets из Kubernetes secrets."""
        targets = {}

        if not self.config.enabled:
            return targets

        for namespace in self.config.secret_namespaces:
            for label_selector in self.config.secret_labels:
                try:
                    secrets = self.k8s_client.list_namespaced_secret(
                        namespace=namespace, label_selector=label_selector
                    )

                    for secret in secrets.items:
                        target = await self._create_target_from_secret(
                            secret, namespace
                        )
                        if target:
                            targets[target.name] = target

                except ApiException as e:
                    if e.status == 403:
                        logger.warning(
                            f"No permissions to list secrets in namespace {namespace}"
                        )
                    else:
                        logger.error(
                            f"Kubernetes API error listing secrets in {namespace}: {e}"
                        )
                except Exception as e:
                    logger.error(f"Error listing secrets in {namespace}: {e}")

        return targets

    async def _create_target_from_secret(
        self, secret, namespace: str
    ) -> Optional[PublishingTarget]:
        """Создание PublishingTarget из Kubernetes secret."""
        try:
            secret_data = secret.data or {}

            # Decode base64 values
            decoded_data = {}
            for key, value in secret_data.items():
                try:
                    decoded_data[key] = base64.b64decode(value).decode("utf-8")
                except Exception as e:
                    logger.warning(f"Failed to decode secret key {key}: {e}")
                    continue

            # Extract target configuration
            target_name = decoded_data.get("target-name", secret.metadata.name)
            webhook_url = decoded_data.get("webhook-url") or decoded_data.get("url")

            if not webhook_url:
                logger.warning(f"No webhook URL found in secret {secret.metadata.name}")
                return None

            if not validate_url(webhook_url):
                logger.warning(
                    f"Invalid webhook URL in secret {secret.metadata.name}: {webhook_url}"
                )
                return None

            # Check if target is enabled
            enabled = decoded_data.get("enabled", "true").lower() == "true"
            if not enabled:
                logger.debug(f"Target {target_name} is disabled")
                return None

            # Build headers
            headers = {"Content-Type": "application/json"}

            # Authentication headers
            if "auth-header" in decoded_data:
                headers["Authorization"] = decoded_data["auth-header"]
            elif "api-key" in decoded_data:
                headers["Authorization"] = f"Bearer {decoded_data['api-key']}"
            elif "token" in decoded_data:
                headers["Authorization"] = f"Token {decoded_data['token']}"

            # Additional custom headers
            for key, value in decoded_data.items():
                if key.startswith("header-"):
                    header_name = key[7:].replace("-", "-").title()
                    headers[header_name] = value

            # Extract and validate format
            format_str = decoded_data.get("format", "alertmanager").lower()
            try:
                target_format = PublishingFormat(format_str)
            except ValueError:
                logger.warning(
                    f"Unknown format '{format_str}' in secret {secret.metadata.name}, "
                    f"defaulting to alertmanager"
                )
                target_format = PublishingFormat.ALERTMANAGER

            # Extract filter configuration
            filter_config = self._extract_filter_config(decoded_data)

            target = PublishingTarget(
                name=target_name,
                type="webhook",
                url=webhook_url,
                enabled=True,
                filter_config=filter_config,
                headers=headers,
                format=target_format,
            )

            logger.debug(
                "Created publishing target from secret",
                target_name=target_name,
                namespace=namespace,
                secret_name=secret.metadata.name,
                format=target_format.value,
            )

            return target

        except Exception as e:
            logger.error(
                f"Failed to create target from secret {secret.metadata.name}: {e}"
            )
            return None

    def _extract_filter_config(self, data: dict[str, str]) -> dict[str, any]:
        """Извлечение конфигурации фильтров из secret data."""
        filter_config = {}

        # Severity filter
        if "filter-severity" in data:
            severities = [s.strip().lower() for s in data["filter-severity"].split(",")]
            filter_config["severity"] = severities
        elif "severity" in data:
            severities = [s.strip().lower() for s in data["severity"].split(",")]
            filter_config["severity"] = severities

        # Namespace filter
        if "filter-namespaces" in data:
            namespaces = [ns.strip() for ns in data["filter-namespaces"].split(",")]
            filter_config["namespaces"] = namespaces
        elif "namespaces" in data:
            namespaces = [ns.strip() for ns in data["namespaces"].split(",")]
            filter_config["namespaces"] = namespaces

        # Exclude noise filter
        exclude_noise = data.get("exclude-noise", "true").lower() == "true"
        filter_config["exclude_noise"] = exclude_noise

        # Minimum confidence threshold
        if "min-confidence" in data:
            try:
                filter_config["min_confidence"] = float(data["min-confidence"])
            except ValueError:
                logger.warning(
                    f"Invalid min-confidence value: {data['min-confidence']}"
                )

        # Alert name patterns (regex)
        if "alert-name-pattern" in data:
            filter_config["alert_name_pattern"] = data["alert-name-pattern"]

        return filter_config

    def get_active_targets(self) -> list[PublishingTarget]:
        """Получить все активные publishing targets."""
        return list(self.current_targets.values())

    def get_target_by_name(self, name: str) -> Optional[PublishingTarget]:
        """Получить target по имени."""
        return self.current_targets.get(name)

    def get_targets_count(self) -> int:
        """Получить количество активных targets."""
        return len(self.current_targets)

    def is_metrics_only_mode(self) -> bool:
        """Проверить, работает ли сервис в режиме только метрик."""
        return len(self.current_targets) == 0

    def get_discovery_stats(self) -> dict[str, any]:
        """Получить статистику discovery."""
        return {
            "enabled": self.config.enabled,
            "kubernetes_available": self.k8s_client is not None,
            "targets_count": len(self.current_targets),
            "last_refresh_time": self._last_refresh_time,
            "refresh_interval": self.config.config_refresh_interval,
            "watched_namespaces": self.config.secret_namespaces,
            "label_selectors": self.config.secret_labels,
            "metrics_only_mode": self.is_metrics_only_mode(),
        }

    async def refresh_targets(self) -> bool:
        """Принудительно обновить targets."""
        try:
            await self._discover_and_load_targets()
            return True
        except Exception as e:
            logger.error(f"Failed to refresh targets: {e}")
            return False
