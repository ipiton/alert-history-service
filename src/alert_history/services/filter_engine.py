"""
Filter Engine для умной фильтрации алертов перед публикацией.

Поддерживает:
- Severity-based фильтрация
- Namespace и label фильтры
- LLM confidence thresholds
- Custom regex patterns
- Time-based фильтры
- Alert deduplication
"""

# Standard library imports
import re
import time
from dataclasses import dataclass
from enum import Enum
from typing import Any, Optional

# Local imports
from ..core.interfaces import (
    Alert,
    AlertSeverity,
    EnrichedAlert,
    IFilterEngine,
    PublishingTarget,
)
from ..logging_config import get_logger

logger = get_logger(__name__)


class FilterAction(Enum):
    """Возможные действия фильтра."""

    ALLOW = "allow"
    DENY = "deny"
    DELAY = "delay"


@dataclass
class FilterRule:
    """Правило фильтрации."""

    name: str
    action: FilterAction
    conditions: dict[str, Any]
    priority: int = 100  # Lower number = higher priority
    enabled: bool = True

    def __post_init__(self):
        """Validate filter rule."""
        if not self.name:
            raise ValueError("Filter rule name cannot be empty")

        if not isinstance(self.conditions, dict):
            raise ValueError("Filter conditions must be a dictionary")


class AlertFilterEngine(IFilterEngine):
    """
    Умная система фильтрации алертов.

    Supports multiple filter types:
    - Severity filtering
    - Namespace/label filtering
    - LLM confidence thresholds
    - Custom patterns
    - Rate limiting and deduplication
    """

    def __init__(self):
        """Initialize filter engine."""
        self.global_rules: list[FilterRule] = []
        self.target_specific_rules: dict[str, list[FilterRule]] = {}

        # Deduplication tracking
        self._recent_alerts: dict[str, float] = {}  # fingerprint -> last_seen_time
        self._dedup_window = 300.0  # 5 minutes default

        # Rate limiting tracking
        self._rate_limit_counters: dict[str, dict[str, int]] = (
            {}
        )  # target -> {window: count}
        self._rate_limit_windows: dict[str, float] = {}  # target -> window_start_time

        self._setup_default_rules()

    def _setup_default_rules(self) -> None:
        """Настройка стандартных правил фильтрации."""

        # Rule 1: Block noise alerts by default
        self.global_rules.append(
            FilterRule(
                name="block_noise_alerts",
                action=FilterAction.DENY,
                conditions={"llm_severity": "noise"},
                priority=10,
                enabled=True,
            )
        )

        # Rule 2: Always allow critical alerts
        self.global_rules.append(
            FilterRule(
                name="allow_critical_alerts",
                action=FilterAction.ALLOW,
                conditions={"llm_severity": "critical"},
                priority=1,
                enabled=True,
            )
        )

        # Rule 3: Require minimum confidence for LLM classifications
        self.global_rules.append(
            FilterRule(
                name="min_confidence_threshold",
                action=FilterAction.DENY,
                conditions={
                    "llm_confidence_below": 0.3,
                    "has_llm_classification": True,
                },
                priority=20,
                enabled=True,
            )
        )

        # Rule 4: Block test namespaces by default
        self.global_rules.append(
            FilterRule(
                name="block_test_namespaces",
                action=FilterAction.DENY,
                conditions={"namespace_pattern": r".*test.*|.*dev.*|.*staging.*"},
                priority=30,
                enabled=False,  # Disabled by default, can be enabled per target
            )
        )

        # Sort rules by priority
        self.global_rules.sort(key=lambda r: r.priority)

    async def should_publish(
        self, enriched_alert: EnrichedAlert, target: PublishingTarget
    ) -> bool:
        """
        Определить должен ли алерт быть опубликован в target.

        Args:
            enriched_alert: Обогащенный алерт с классификацией
            target: Целевая система публикации

        Returns:
            True если алерт должен быть опубликован
        """
        alert = enriched_alert.alert

        try:
            # Check deduplication first
            if self._is_duplicate_alert(alert):
                logger.debug(
                    "Alert filtered out due to deduplication",
                    fingerprint=alert.fingerprint,
                    target=target.name,
                )
                return False

            # Check rate limiting
            if self._is_rate_limited(target):
                logger.debug(
                    "Alert filtered out due to rate limiting", target=target.name
                )
                return False

            # Apply target-specific configuration filters
            if not self._check_target_config_filters(enriched_alert, target):
                logger.debug(
                    "Alert filtered out by target configuration",
                    fingerprint=alert.fingerprint,
                    target=target.name,
                )
                return False

            # Apply global filter rules
            global_decision = self._apply_rules(enriched_alert, self.global_rules)
            if global_decision == FilterAction.DENY:
                logger.debug(
                    "Alert filtered out by global rules",
                    fingerprint=alert.fingerprint,
                    target=target.name,
                )
                return False

            # Apply target-specific rules
            target_rules = self.target_specific_rules.get(target.name, [])
            if target_rules:
                target_decision = self._apply_rules(enriched_alert, target_rules)
                if target_decision == FilterAction.DENY:
                    logger.debug(
                        "Alert filtered out by target-specific rules",
                        fingerprint=alert.fingerprint,
                        target=target.name,
                    )
                    return False

            # Update tracking
            self._update_dedup_tracking(alert)
            self._update_rate_limit_tracking(target)

            logger.debug(
                "Alert passed all filters",
                fingerprint=alert.fingerprint,
                target=target.name,
            )

            return True

        except Exception as e:
            logger.error(
                "Error in filter engine, allowing alert by default",
                fingerprint=alert.fingerprint,
                target=target.name,
                error=str(e),
            )
            # Fail open - allow alert if filtering fails
            return True

    def _check_target_config_filters(
        self, enriched_alert: EnrichedAlert, target: PublishingTarget
    ) -> bool:
        """Проверить фильтры из конфигурации target."""
        alert = enriched_alert.alert
        # classification = enriched_alert.classification  # Available for future use
        filter_config = target.filter_config or {}

        # Severity filter
        if "severity" in filter_config:
            allowed_severities = filter_config["severity"]

            # Check LLM severity first
            if classification:
                llm_severity = classification.severity.value.lower()
                if llm_severity not in allowed_severities:
                    return False
            else:
                # Fallback to label-based severity
                label_severity = alert.labels.get("severity", "unknown").lower()
                if label_severity not in allowed_severities:
                    return False

        # Namespace filter
        if "namespaces" in filter_config:
            allowed_namespaces = filter_config["namespaces"]
            if alert.namespace and alert.namespace not in allowed_namespaces:
                return False

        # Exclude noise filter
        if filter_config.get("exclude_noise", True):
            if classification and classification.severity == AlertSeverity.NOISE:
                return False

        # Minimum confidence threshold
        if "min_confidence" in filter_config:
            min_confidence = filter_config["min_confidence"]
            if classification and classification.confidence < min_confidence:
                return False

        # Alert name pattern filter
        if "alert_name_pattern" in filter_config:
            pattern = filter_config["alert_name_pattern"]
            try:
                if not re.match(pattern, alert.alert_name):
                    return False
            except re.error as e:
                logger.warning(f"Invalid regex pattern '{pattern}': {e}")

        return True

    def _apply_rules(
        self, enriched_alert: EnrichedAlert, rules: list[FilterRule]
    ) -> FilterAction:
        """Применить список правил к алерту."""
        alert = enriched_alert.alert
        # classification = enriched_alert.classification  # Available for future use

        for rule in rules:
            if not rule.enabled:
                continue

            if self._evaluate_rule_conditions(enriched_alert, rule.conditions):
                logger.debug(
                    "Filter rule matched",
                    rule_name=rule.name,
                    action=rule.action.value,
                    fingerprint=alert.fingerprint,
                )
                return rule.action

        # Default action is ALLOW if no rules match
        return FilterAction.ALLOW

    def _evaluate_rule_conditions(
        self, enriched_alert: EnrichedAlert, conditions: dict[str, Any]
    ) -> bool:
        """Оценить условия правила."""
        alert = enriched_alert.alert
        # classification = enriched_alert.classification  # Available for future use

        for condition_key, condition_value in conditions.items():

            if condition_key == "llm_severity":
                if not classification:
                    return False
                if classification.severity.value.lower() != condition_value.lower():
                    return False

            elif condition_key == "llm_confidence_below":
                if not classification:
                    return False
                if classification.confidence >= condition_value:
                    return False

            elif condition_key == "llm_confidence_above":
                if not classification:
                    return False
                if classification.confidence <= condition_value:
                    return False

            elif condition_key == "has_llm_classification":
                has_classification = classification is not None
                if has_classification != condition_value:
                    return False

            elif condition_key == "namespace":
                if alert.namespace != condition_value:
                    return False

            elif condition_key == "namespace_pattern":
                if not alert.namespace:
                    return False
                try:
                    if not re.match(condition_value, alert.namespace):
                        return False
                except re.error:
                    logger.warning(f"Invalid regex pattern: {condition_value}")
                    return False

            elif condition_key == "alert_name":
                if alert.alert_name != condition_value:
                    return False

            elif condition_key == "alert_name_pattern":
                try:
                    if not re.match(condition_value, alert.alert_name):
                        return False
                except re.error:
                    logger.warning(f"Invalid regex pattern: {condition_value}")
                    return False

            elif condition_key == "label_exists":
                if condition_value not in alert.labels:
                    return False

            elif condition_key == "label_equals":
                for label_key, label_value in condition_value.items():
                    if alert.labels.get(label_key) != label_value:
                        return False

            elif condition_key == "status":
                if alert.status.value != condition_value:
                    return False

            else:
                logger.warning(f"Unknown filter condition: {condition_key}")

        # All conditions passed
        return True

    def _is_duplicate_alert(self, alert: Alert) -> bool:
        """Проверить является ли алерт дубликатом."""
        current_time = time.time()
        fingerprint = alert.fingerprint

        # Cleanup old entries
        self._cleanup_dedup_tracking(current_time)

        # Check if we've seen this alert recently
        if fingerprint in self._recent_alerts:
            last_seen = self._recent_alerts[fingerprint]
            if current_time - last_seen < self._dedup_window:
                return True

        return False

    def _update_dedup_tracking(self, alert: Alert) -> None:
        """Обновить трекинг дедупликации."""
        self._recent_alerts[alert.fingerprint] = time.time()

    def _cleanup_dedup_tracking(self, current_time: float) -> None:
        """Очистить устаревшие записи дедупликации."""
        cutoff_time = current_time - self._dedup_window

        expired_fingerprints = [
            fp
            for fp, timestamp in self._recent_alerts.items()
            if timestamp < cutoff_time
        ]

        for fp in expired_fingerprints:
            del self._recent_alerts[fp]

    def _is_rate_limited(self, target: PublishingTarget) -> bool:
        """Проверить превышен ли rate limit для target."""
        # Simple rate limiting implementation
        # In production, this could be more sophisticated

        target_name = target.name
        current_time = time.time()

        # Get or initialize rate limit data
        if target_name not in self._rate_limit_counters:
            self._rate_limit_counters[target_name] = {}
            self._rate_limit_windows[target_name] = current_time
            return False

        # Check if window expired (5 minute windows)
        window_start = self._rate_limit_windows[target_name]
        if current_time - window_start > 300:  # 5 minutes
            # Reset counter for new window
            self._rate_limit_counters[target_name] = {}
            self._rate_limit_windows[target_name] = current_time
            return False

        # Simple rate limit: max 100 alerts per 5 minutes per target
        current_count = sum(self._rate_limit_counters[target_name].values())
        return current_count >= 100

    def _update_rate_limit_tracking(self, target: PublishingTarget) -> None:
        """Обновить счетчик rate limiting."""
        target_name = target.name
        current_time = time.time()

        if target_name not in self._rate_limit_counters:
            self._rate_limit_counters[target_name] = {}

        # Use minute as granularity
        minute_key = int(current_time // 60)

        if minute_key not in self._rate_limit_counters[target_name]:
            self._rate_limit_counters[target_name][minute_key] = 0

        self._rate_limit_counters[target_name][minute_key] += 1

    def add_global_rule(self, rule: FilterRule) -> None:
        """Добавить глобальное правило."""
        self.global_rules.append(rule)
        self.global_rules.sort(key=lambda r: r.priority)

        logger.info(
            "Added global filter rule",
            rule_name=rule.name,
            action=rule.action.value,
            priority=rule.priority,
        )

    def add_target_rule(self, target_name: str, rule: FilterRule) -> None:
        """Добавить правило для конкретного target."""
        if target_name not in self.target_specific_rules:
            self.target_specific_rules[target_name] = []

        self.target_specific_rules[target_name].append(rule)
        self.target_specific_rules[target_name].sort(key=lambda r: r.priority)

        logger.info(
            "Added target-specific filter rule",
            target=target_name,
            rule_name=rule.name,
            action=rule.action.value,
            priority=rule.priority,
        )

    def remove_rule(self, rule_name: str, target_name: Optional[str] = None) -> bool:
        """Удалить правило."""
        if target_name:
            # Remove target-specific rule
            if target_name in self.target_specific_rules:
                original_count = len(self.target_specific_rules[target_name])
                self.target_specific_rules[target_name] = [
                    r
                    for r in self.target_specific_rules[target_name]
                    if r.name != rule_name
                ]
                removed = len(self.target_specific_rules[target_name]) < original_count
                if removed:
                    logger.info(
                        f"Removed target-specific rule {rule_name} for {target_name}"
                    )
                return removed
        else:
            # Remove global rule
            original_count = len(self.global_rules)
            self.global_rules = [r for r in self.global_rules if r.name != rule_name]
            removed = len(self.global_rules) < original_count
            if removed:
                logger.info(f"Removed global rule {rule_name}")
            return removed

        return False

    def get_filter_stats(self) -> dict[str, Any]:
        """Получить статистику фильтрации."""
        current_time = time.time()

        # Calculate dedup stats
        active_fingerprints = sum(
            1
            for timestamp in self._recent_alerts.values()
            if current_time - timestamp < self._dedup_window
        )

        # Calculate rate limit stats
        rate_limit_stats = {}
        for target_name, counters in self._rate_limit_counters.items():
            window_start = self._rate_limit_windows.get(target_name, current_time)
            if current_time - window_start < 300:  # Active window
                current_count = sum(counters.values())
                rate_limit_stats[target_name] = {
                    "current_count": current_count,
                    "limit": 100,
                    "window_remaining": 300 - (current_time - window_start),
                }

        return {
            "global_rules_count": len(self.global_rules),
            "target_specific_rules": {
                target: len(rules)
                for target, rules in self.target_specific_rules.items()
            },
            "deduplication": {
                "window_seconds": self._dedup_window,
                "active_fingerprints": active_fingerprints,
                "total_tracked": len(self._recent_alerts),
            },
            "rate_limiting": rate_limit_stats,
        }
