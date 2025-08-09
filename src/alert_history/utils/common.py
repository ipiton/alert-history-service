"""
Common utility functions implementing DRY principle.

Reusable functions to avoid code duplication across the application:
- Data validation
- Formatting and parsing
- Common algorithms
- Helper functions
"""

# Standard library imports
import hashlib
import json
import re
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional, Union
from urllib.parse import urlparse


def generate_fingerprint(data: Union[Dict[str, Any], str]) -> str:
    """
    Generate consistent fingerprint for data.

    Args:
        data: Data to generate fingerprint for

    Returns:
        SHA256 fingerprint as hex string
    """
    if isinstance(data, dict):
        # Sort keys for consistent hashing
        sorted_data = json.dumps(data, sort_keys=True, separators=(",", ":"))
    else:
        sorted_data = str(data)

    return hashlib.sha256(sorted_data.encode("utf-8")).hexdigest()


def sanitize_metric_name(name: str) -> str:
    """
    Sanitize metric name for Prometheus compliance.

    Args:
        name: Original metric name

    Returns:
        Sanitized metric name
    """
    # Replace invalid characters with underscores
    sanitized = re.sub(r"[^a-zA-Z0-9_:]", "_", name)

    # Ensure it starts with a letter or underscore
    if sanitized and not sanitized[0].isalpha() and sanitized[0] != "_":
        sanitized = f"_{sanitized}"

    # Remove consecutive underscores
    sanitized = re.sub(r"_+", "_", sanitized)

    # Remove trailing underscores
    sanitized = sanitized.rstrip("_")

    return sanitized or "unnamed_metric"


def sanitize_label_value(value: str) -> str:
    """
    Sanitize label value for Prometheus compliance.

    Args:
        value: Original label value

    Returns:
        Sanitized label value
    """
    # Remove or replace problematic characters
    sanitized = re.sub(r'["\n\r\t]', "_", str(value))

    # Truncate if too long
    max_length = 100
    if len(sanitized) > max_length:
        sanitized = sanitized[: max_length - 3] + "..."

    return sanitized


def normalize_alert_labels(labels: Dict[str, str]) -> Dict[str, str]:
    """
    Normalize alert labels for consistent processing.

    Args:
        labels: Original labels

    Returns:
        Normalized labels
    """
    normalized = {}

    for key, value in labels.items():
        # Normalize key
        normalized_key = key.lower().strip()

        # Normalize value
        normalized_value = str(value).strip()

        # Skip empty values
        if normalized_value:
            normalized[normalized_key] = normalized_value

    return normalized


def parse_duration(duration_str: str) -> int:
    """
    Parse duration string to seconds.

    Supports formats: "30s", "5m", "2h", "1d"

    Args:
        duration_str: Duration string

    Returns:
        Duration in seconds

    Raises:
        ValueError: If format is invalid
    """
    if not duration_str:
        raise ValueError("Duration string cannot be empty")

    duration_str = duration_str.strip().lower()

    # Extract number and unit
    match = re.match(r"^(\d+\.?\d*)([smhd]?)$", duration_str)
    if not match:
        raise ValueError(f"Invalid duration format: {duration_str}")

    value = float(match.group(1))
    unit = match.group(2) or "s"  # Default to seconds

    multipliers = {"s": 1, "m": 60, "h": 3600, "d": 86400}

    if unit not in multipliers:
        raise ValueError(f"Invalid duration unit: {unit}")

    return int(value * multipliers[unit])


def format_duration(seconds: int) -> str:
    """
    Format seconds to human-readable duration.

    Args:
        seconds: Duration in seconds

    Returns:
        Human-readable duration string
    """
    if seconds < 60:
        return f"{seconds}s"
    elif seconds < 3600:
        return f"{seconds // 60}m"
    elif seconds < 86400:
        return f"{seconds // 3600}h"
    else:
        return f"{seconds // 86400}d"


def parse_timestamp(timestamp: Union[str, int, float, datetime]) -> datetime:
    """
    Parse various timestamp formats to datetime.

    Args:
        timestamp: Timestamp in various formats

    Returns:
        UTC datetime object

    Raises:
        ValueError: If timestamp format is invalid
    """
    if isinstance(timestamp, datetime):
        # Ensure timezone aware
        if timestamp.tzinfo is None:
            return timestamp.replace(tzinfo=timezone.utc)
        return timestamp.astimezone(timezone.utc)

    if isinstance(timestamp, (int, float)):
        # Unix timestamp
        return datetime.fromtimestamp(timestamp, tz=timezone.utc)

    if isinstance(timestamp, str):
        # Try common ISO formats
        formats = [
            "%Y-%m-%dT%H:%M:%S.%fZ",
            "%Y-%m-%dT%H:%M:%SZ",
            "%Y-%m-%dT%H:%M:%S.%f%z",
            "%Y-%m-%dT%H:%M:%S%z",
            "%Y-%m-%d %H:%M:%S",
            "%Y-%m-%d",
        ]

        for fmt in formats:
            try:
                dt = datetime.strptime(timestamp, fmt)
                if dt.tzinfo is None:
                    dt = dt.replace(tzinfo=timezone.utc)
                return dt.astimezone(timezone.utc)
            except ValueError:
                continue

        # Try parsing as Unix timestamp string
        try:
            unix_timestamp = float(timestamp)
            return datetime.fromtimestamp(unix_timestamp, tz=timezone.utc)
        except ValueError:
            pass

        raise ValueError(f"Unable to parse timestamp: {timestamp}")

    raise ValueError(f"Unsupported timestamp type: {type(timestamp)}")


def validate_url(url: str) -> bool:
    """
    Validate URL format.

    Args:
        url: URL to validate

    Returns:
        True if URL is valid
    """
    try:
        result = urlparse(url)
        return all([result.scheme, result.netloc])
    except Exception:
        return False


def deep_merge_dicts(dict1: Dict[str, Any], dict2: Dict[str, Any]) -> Dict[str, Any]:
    """
    Deep merge two dictionaries.

    Args:
        dict1: First dictionary
        dict2: Second dictionary (takes precedence)

    Returns:
        Merged dictionary
    """
    result = dict1.copy()

    for key, value in dict2.items():
        if key in result and isinstance(result[key], dict) and isinstance(value, dict):
            result[key] = deep_merge_dicts(result[key], value)
        else:
            result[key] = value

    return result


def extract_namespace_from_labels(labels: Dict[str, str]) -> Optional[str]:
    """
    Extract namespace from alert labels.

    Args:
        labels: Alert labels

    Returns:
        Namespace if found
    """
    # Common namespace label names
    namespace_keys = ["namespace", "k8s_namespace", "kubernetes_namespace"]

    for key in namespace_keys:
        if key in labels:
            return labels[key]

    return None


def extract_service_from_labels(labels: Dict[str, str]) -> Optional[str]:
    """
    Extract service name from alert labels.

    Args:
        labels: Alert labels

    Returns:
        Service name if found
    """
    # Common service label names
    service_keys = ["service", "k8s_service", "kubernetes_service", "job"]

    for key in service_keys:
        if key in labels:
            return labels[key]

    return None


def calculate_confidence_score(
    classification_factors: Dict[str, float], weights: Optional[Dict[str, float]] = None
) -> float:
    """
    Calculate confidence score from multiple factors.

    Args:
        classification_factors: Dict of factor_name -> score (0.0 to 1.0)
        weights: Optional weights for factors

    Returns:
        Weighted confidence score (0.0 to 1.0)
    """
    if not classification_factors:
        return 0.0

    if weights is None:
        # Equal weights
        weights = {factor: 1.0 for factor in classification_factors.keys()}

    total_weighted_score = 0.0
    total_weight = 0.0

    for factor, score in classification_factors.items():
        weight = weights.get(factor, 1.0)
        total_weighted_score += score * weight
        total_weight += weight

    if total_weight == 0:
        return 0.0

    return min(1.0, max(0.0, total_weighted_score / total_weight))


def truncate_string(text: str, max_length: int, suffix: str = "...") -> str:
    """
    Truncate string to maximum length.

    Args:
        text: Text to truncate
        max_length: Maximum length
        suffix: Suffix to add when truncating

    Returns:
        Truncated string
    """
    if len(text) <= max_length:
        return text

    return text[: max_length - len(suffix)] + suffix


def safe_json_dumps(data: Any, default_value: str = "{}") -> str:
    """
    Safely serialize data to JSON.

    Args:
        data: Data to serialize
        default_value: Default value if serialization fails

    Returns:
        JSON string
    """
    try:
        return json.dumps(data, default=str, ensure_ascii=False)
    except Exception:
        return default_value


def safe_json_loads(json_str: str, default_value: Any = None) -> Any:
    """
    Safely parse JSON string.

    Args:
        json_str: JSON string to parse
        default_value: Default value if parsing fails

    Returns:
        Parsed data or default value
    """
    try:
        return json.loads(json_str)
    except Exception:
        return default_value


def batch_items(items: List[Any], batch_size: int) -> List[List[Any]]:
    """
    Split items into batches.

    Args:
        items: List of items to batch
        batch_size: Size of each batch

    Returns:
        List of batches
    """
    if batch_size <= 0:
        raise ValueError("Batch size must be positive")

    batches = []
    for i in range(0, len(items), batch_size):
        batches.append(items[i : i + batch_size])

    return batches


def flatten_dict(data: Dict[str, Any], separator: str = ".", prefix: str = "") -> Dict[str, Any]:
    """
    Flatten nested dictionary.

    Args:
        data: Dictionary to flatten
        separator: Separator for nested keys
        prefix: Prefix for keys

    Returns:
        Flattened dictionary
    """
    result = {}

    for key, value in data.items():
        new_key = f"{prefix}{separator}{key}" if prefix else key

        if isinstance(value, dict):
            result.update(flatten_dict(value, separator, new_key))
        else:
            result[new_key] = value

    return result


def is_valid_severity(severity: str) -> bool:
    """
    Check if severity value is valid.

    Args:
        severity: Severity value to check

    Returns:
        True if valid
    """
    valid_severities = {"critical", "warning", "info", "noise"}
    return severity.lower() in valid_severities


def normalize_severity(severity: str) -> str:
    """
    Normalize severity value.

    Args:
        severity: Original severity

    Returns:
        Normalized severity
    """
    severity_mapping = {
        "crit": "critical",
        "warn": "warning",
        "error": "critical",
        "err": "critical",
        "information": "info",
        "inf": "info",
        "debug": "info",
        "trace": "noise",
    }

    normalized = severity.lower().strip()
    return severity_mapping.get(normalized, normalized)
