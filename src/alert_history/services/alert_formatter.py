"""
Alert Formatter для различных publishing форматов.

Поддерживаемые форматы:
- Alertmanager (стандартный webhook format)
- Rootly (incident management)
- PagerDuty (alerting platform)
- Slack (notifications)
- Generic webhook (custom format)
"""

# Standard library imports
import json
import time
from datetime import datetime
from enum import Enum
from typing import Any, Dict, List, Optional

# Local imports
from ..core.interfaces import (
    Alert,
    AlertSeverity,
    ClassificationResult,
    EnrichedAlert,
    IAlertFormatter,
    PublishingFormat,
)
from ..logging_config import get_logger
from ..utils.common import safe_json_dumps, truncate_string

logger = get_logger(__name__)


class AlertFormatter(IAlertFormatter):
    """
    Универсальный форматтер алертов для различных систем.

    Implements strategy pattern для различных форматов публикации.
    """

    def __init__(self):
        """Initialize formatter."""
        self._formatters = {
            PublishingFormat.ALERTMANAGER: self._format_alertmanager,
            PublishingFormat.ROOTLY: self._format_rootly,
            PublishingFormat.PAGERDUTY: self._format_pagerduty,
            PublishingFormat.SLACK: self._format_slack,
            PublishingFormat.WEBHOOK: self._format_webhook,
        }

    async def format_alert(
        self, enriched_alert: EnrichedAlert, target_format: PublishingFormat
    ) -> Dict[str, Any]:
        """
        Форматировать алерт для конкретного target.

        Args:
            enriched_alert: Обогащенный алерт с классификацией
            target_format: Целевой формат

        Returns:
            Отформатированные данные для отправки
        """
        formatter = self._formatters.get(target_format)
        if not formatter:
            logger.warning(f"Unknown format {target_format}, using webhook format")
            formatter = self._format_webhook

        try:
            result = formatter(enriched_alert)

            logger.debug(
                "Alert formatted successfully",
                alert_name=enriched_alert.alert.alert_name,
                fingerprint=enriched_alert.alert.fingerprint,
                format=target_format.value,
            )

            return result

        except Exception as e:
            logger.error(
                "Alert formatting failed",
                alert_name=enriched_alert.alert.alert_name,
                fingerprint=enriched_alert.alert.fingerprint,
                format=target_format.value,
                error=str(e),
            )
            raise

    def _format_alertmanager(self, enriched_alert: EnrichedAlert) -> Dict[str, Any]:
        """Форматирование в стандартный Alertmanager webhook format."""
        alert = enriched_alert.alert

        # Build alertmanager-compatible alert
        alertmanager_alert = {
            "labels": alert.labels.copy(),
            "annotations": alert.annotations.copy(),
            "startsAt": alert.starts_at.isoformat() if alert.starts_at else None,
            "endsAt": alert.ends_at.isoformat() if alert.ends_at else None,
            "generatorURL": alert.generator_url,
            "fingerprint": alert.fingerprint,
            "status": alert.status.value,
        }

        # Add classification data as annotations if available
        if enriched_alert.classification:
            classification = enriched_alert.classification
            alertmanager_alert["annotations"].update(
                {
                    "llm_severity": classification.severity.value,
                    "llm_confidence": str(classification.confidence),
                    "llm_reasoning": truncate_string(classification.reasoning, 500),
                    "llm_classification_time": str(classification.processing_time),
                }
            )

            # Add top recommendations as annotation
            if classification.recommendations:
                top_recommendations = classification.recommendations[:3]
                alertmanager_alert["annotations"]["llm_recommendations"] = "; ".join(
                    top_recommendations
                )

        # Add enrichment metadata
        if enriched_alert.enrichment_metadata:
            alertmanager_alert["annotations"]["enrichment_metadata"] = safe_json_dumps(
                enriched_alert.enrichment_metadata
            )

        return {
            "receiver": "alert-history-proxy",
            "status": "firing" if alert.status.value == "firing" else "resolved",
            "alerts": [alertmanager_alert],
            "groupLabels": {},
            "commonLabels": alert.labels,
            "commonAnnotations": alert.annotations,
            "externalURL": "",
            "version": "4",
            "groupKey": f"group:{alert.fingerprint}",
            "truncatedAlerts": 0,
        }

    def _format_rootly(self, enriched_alert: EnrichedAlert) -> Dict[str, Any]:
        """Форматирование для Rootly incident management."""
        alert = enriched_alert.alert
        classification = enriched_alert.classification

        # Map severity to Rootly severity levels
        severity_mapping = {
            AlertSeverity.CRITICAL: "critical",
            AlertSeverity.WARNING: "major",
            AlertSeverity.INFO: "minor",
            AlertSeverity.NOISE: "low",
        }

        # Determine severity (LLM classification takes precedence)
        if classification:
            rootly_severity = severity_mapping.get(classification.severity, "major")
        else:
            # Fallback to label-based severity
            label_severity = alert.labels.get("severity", "warning").lower()
            severity_map = {"critical": "critical", "warning": "major", "info": "minor"}
            rootly_severity = severity_map.get(label_severity, "major")

        # Build incident title
        title = f"[{alert.alert_name}] Alert in {alert.namespace or 'unknown'}"
        if classification:
            title += f" (LLM: {classification.severity.value}, {classification.confidence:.0%} confidence)"

        # Build description
        description_parts = []

        # Basic alert info
        description_parts.append(f"**Alert:** {alert.alert_name}")
        description_parts.append(f"**Status:** {alert.status.value}")
        description_parts.append(f"**Namespace:** {alert.namespace or 'unknown'}")
        description_parts.append(
            f"**Started:** {alert.starts_at.isoformat() if alert.starts_at else 'unknown'}"
        )

        # LLM classification info
        if classification:
            description_parts.append(f"\n**AI Classification:**")
            description_parts.append(f"- **Severity:** {classification.severity.value}")
            description_parts.append(f"- **Confidence:** {classification.confidence:.0%}")
            description_parts.append(f"- **Reasoning:** {classification.reasoning}")

            if classification.recommendations:
                description_parts.append(f"\n**Recommendations:**")
                for i, rec in enumerate(classification.recommendations[:5], 1):
                    description_parts.append(f"{i}. {rec}")

        # Alert labels and annotations
        if alert.labels:
            description_parts.append(f"\n**Labels:**")
            for key, value in alert.labels.items():
                description_parts.append(f"- {key}: {value}")

        if alert.annotations:
            description_parts.append(f"\n**Annotations:**")
            for key, value in alert.annotations.items():
                if key not in ["description", "summary"]:  # Avoid duplication
                    description_parts.append(f"- {key}: {truncate_string(value, 100)}")

        description = "\n".join(description_parts)

        # Build Rootly payload
        rootly_payload = {
            "incident": {
                "title": title,
                "description": description,
                "severity": rootly_severity,
                "status": "started" if alert.status.value == "firing" else "resolved",
                "source": "alert-history-llm",
                "alert_fingerprint": alert.fingerprint,
                "tags": self._build_rootly_tags(alert, classification),
                "custom_fields": {
                    "alert_name": alert.alert_name,
                    "namespace": alert.namespace or "unknown",
                    "generator_url": alert.generator_url or "",
                    "llm_severity": (classification.severity.value if classification else None),
                    "llm_confidence": (classification.confidence if classification else None),
                },
            }
        }

        return rootly_payload

    def _format_pagerduty(self, enriched_alert: EnrichedAlert) -> Dict[str, Any]:
        """Форматирование для PagerDuty."""
        alert = enriched_alert.alert
        classification = enriched_alert.classification

        # Map severity to PagerDuty severity
        severity_mapping = {
            AlertSeverity.CRITICAL: "critical",
            AlertSeverity.WARNING: "warning",
            AlertSeverity.INFO: "info",
            AlertSeverity.NOISE: "info",
        }

        pd_severity = "warning"  # Default
        if classification:
            pd_severity = severity_mapping.get(classification.severity, "warning")

        # Determine event action
        event_action = "trigger" if alert.status.value == "firing" else "resolve"

        # Build summary
        summary = f"{alert.alert_name}"
        if alert.namespace:
            summary += f" in {alert.namespace}"
        if classification:
            summary += f" (AI: {classification.severity.value})"

        # Build custom details
        custom_details = {
            "alert_name": alert.alert_name,
            "namespace": alert.namespace or "unknown",
            "status": alert.status.value,
            "fingerprint": alert.fingerprint,
            "labels": alert.labels,
            "annotations": alert.annotations,
        }

        if classification:
            custom_details.update(
                {
                    "llm_classification": {
                        "severity": classification.severity.value,
                        "confidence": classification.confidence,
                        "reasoning": classification.reasoning,
                        "recommendations": classification.recommendations[:3],  # Top 3
                    }
                }
            )

        # Build PagerDuty event
        pagerduty_event = {
            "routing_key": "",  # Will be set by target configuration
            "event_action": event_action,
            "dedup_key": alert.fingerprint,
            "payload": {
                "summary": summary,
                "source": alert.namespace or "unknown",
                "severity": pd_severity,
                "timestamp": (
                    alert.starts_at.isoformat()
                    if alert.starts_at
                    else datetime.utcnow().isoformat()
                ),
                "component": alert.labels.get("job", "unknown"),
                "group": alert.labels.get("alertname", "unknown"),
                "class": "alert",
                "custom_details": custom_details,
            },
            "links": [],
            "images": [],
        }

        # Add generator URL as link
        if alert.generator_url:
            pagerduty_event["links"].append(
                {"href": alert.generator_url, "text": "View in Prometheus"}
            )

        return pagerduty_event

    def _format_slack(self, enriched_alert: EnrichedAlert) -> Dict[str, Any]:
        """Форматирование для Slack."""
        alert = enriched_alert.alert
        classification = enriched_alert.classification

        # Color mapping for severity
        color_mapping = {
            AlertSeverity.CRITICAL: "#FF0000",  # Red
            AlertSeverity.WARNING: "#FFA500",  # Orange
            AlertSeverity.INFO: "#0000FF",  # Blue
            AlertSeverity.NOISE: "#808080",  # Gray
        }

        # Determine color
        if classification:
            color = color_mapping.get(classification.severity, "#FFA500")
            severity_text = classification.severity.value.upper()
            confidence_text = f" ({classification.confidence:.0%} confidence)"
        else:
            color = "#FFA500"  # Default orange
            severity_text = alert.labels.get("severity", "UNKNOWN").upper()
            confidence_text = ""

        # Status emoji
        status_emoji = "🔥" if alert.status.value == "firing" else "✅"

        # Build main text
        main_text = f"{status_emoji} *{alert.alert_name}*{confidence_text}"
        if alert.namespace:
            main_text += f" in `{alert.namespace}`"

        # Build Slack attachment
        attachment = {
            "color": color,
            "title": main_text,
            "title_link": alert.generator_url,
            "fields": [
                {"title": "Status", "value": alert.status.value.upper(), "short": True},
                {"title": "Severity", "value": severity_text, "short": True},
            ],
            "ts": (int(alert.starts_at.timestamp()) if alert.starts_at else int(time.time())),
        }

        # Add namespace field
        if alert.namespace:
            attachment["fields"].append(
                {"title": "Namespace", "value": alert.namespace, "short": True}
            )

        # Add classification info
        if classification:
            attachment["fields"].append(
                {
                    "title": "AI Classification",
                    "value": f"{classification.severity.value} ({classification.confidence:.0%})",
                    "short": True,
                }
            )

            if classification.reasoning:
                attachment["fields"].append(
                    {
                        "title": "Reasoning",
                        "value": truncate_string(classification.reasoning, 300),
                        "short": False,
                    }
                )

            if classification.recommendations:
                recommendations_text = "\n".join(
                    [f"• {rec}" for rec in classification.recommendations[:3]]
                )
                attachment["fields"].append(
                    {
                        "title": "Recommendations",
                        "value": recommendations_text,
                        "short": False,
                    }
                )

        # Add important labels as fields
        important_labels = ["instance", "job", "service"]
        for label in important_labels:
            if label in alert.labels:
                attachment["fields"].append(
                    {
                        "title": label.title(),
                        "value": f"`{alert.labels[label]}`",
                        "short": True,
                    }
                )

        # Build Slack message
        slack_message = {
            "text": f"Alert: {alert.alert_name}",
            "attachments": [attachment],
            "username": "Alert History Bot",
            "icon_emoji": ":warning:",
        }

        return slack_message

    def _format_webhook(self, enriched_alert: EnrichedAlert) -> Dict[str, Any]:
        """Форматирование для generic webhook."""
        alert = enriched_alert.alert
        classification = enriched_alert.classification

        # Build comprehensive webhook payload
        webhook_payload = {
            "alert": {
                "fingerprint": alert.fingerprint,
                "name": alert.alert_name,
                "status": alert.status.value,
                "namespace": alert.namespace,
                "starts_at": alert.starts_at.isoformat() if alert.starts_at else None,
                "ends_at": alert.ends_at.isoformat() if alert.ends_at else None,
                "generator_url": alert.generator_url,
                "labels": alert.labels,
                "annotations": alert.annotations,
            },
            "timestamp": datetime.utcnow().isoformat(),
            "source": "alert-history-llm",
            "version": "1.0",
        }

        # Add classification data if available
        if classification:
            webhook_payload["classification"] = {
                "severity": classification.severity.value,
                "confidence": classification.confidence,
                "reasoning": classification.reasoning,
                "recommendations": classification.recommendations,
                "processing_time": classification.processing_time,
                "metadata": classification.metadata,
            }

        # Add enrichment metadata
        if enriched_alert.enrichment_metadata:
            webhook_payload["enrichment"] = enriched_alert.enrichment_metadata

        if enriched_alert.processing_timestamp:
            webhook_payload["processing_timestamp"] = (
                enriched_alert.processing_timestamp.isoformat()
            )

        return webhook_payload

    def _build_rootly_tags(
        self, alert: Alert, classification: Optional[ClassificationResult]
    ) -> List[str]:
        """Построить теги для Rootly."""
        tags = ["alert-history", "automated"]

        # Add alert name as tag
        tags.append(f"alert:{alert.alert_name}")

        # Add namespace as tag
        if alert.namespace:
            tags.append(f"namespace:{alert.namespace}")

        # Add severity tags
        if classification:
            tags.append(f"llm-severity:{classification.severity.value}")
            # Add confidence range tag
            if classification.confidence >= 0.8:
                tags.append("high-confidence")
            elif classification.confidence >= 0.5:
                tags.append("medium-confidence")
            else:
                tags.append("low-confidence")

        # Add important labels as tags
        important_labels = ["service", "job", "instance"]
        for label in important_labels:
            if label in alert.labels:
                tag_value = alert.labels[label].replace(" ", "-").lower()
                tags.append(f"{label}:{tag_value}")

        return tags
