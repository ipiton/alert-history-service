"""
LLM Proxy Client for alert classification.

Implements communication with internal LLM-proxy service:
- OpenAI Function Calling for structured responses
- Retry logic with exponential backoff
- Rate limiting and error handling
- Caching integration
"""

# Standard library imports
import asyncio
import json
import time
from typing import Any, Optional

# Third-party imports
import aiohttp

# Local imports
from ..core.interfaces import Alert, AlertSeverity, ClassificationResult, ILLMClient
from ..logging_config import get_logger
from ..utils.decorators import measure_time, retry

logger = get_logger(__name__)


class LLMProxyClient(ILLMClient):
    """
    Client for communicating with internal LLM-proxy service.

    Provides alert classification using OpenAI models through proxy.
    """

    def __init__(
        self,
        proxy_url: str,
        api_key: str,
        model: str = "gpt-4",
        timeout: int = 30,
        max_retries: int = 3,
        retry_delay: float = 1.0,
    ):
        """Initialize LLM proxy client."""
        self.proxy_url = proxy_url.rstrip("/")
        self.api_key = api_key
        self.model = model
        self.timeout = timeout
        self.max_retries = max_retries
        self.retry_delay = retry_delay

        # Session for connection pooling
        self._session: Optional[aiohttp.ClientSession] = None

        # Classification prompt template
        self.classification_prompt = self._build_classification_prompt()

        # OpenAI function schema for structured responses
        self.function_schema = self._build_function_schema()

    async def __aenter__(self) -> "LLMProxyClient":
        """Async context manager entry."""
        await self._init_session()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb) -> None:
        """Async context manager exit."""
        await self._close_session()

    async def _init_session(self) -> None:
        """Initialize HTTP session."""
        connector = aiohttp.TCPConnector(
            limit=100,
            limit_per_host=10,
            keepalive_timeout=30,
            enable_cleanup_closed=True,
        )

        timeout = aiohttp.ClientTimeout(total=self.timeout)

        headers = {
            "x-litellm-api-key": self.api_key,
            "Content-Type": "application/json",
            "User-Agent": "AlertHistory-LLM/1.0",
        }

        self._session = aiohttp.ClientSession(
            connector=connector, timeout=timeout, headers=headers
        )

    async def _close_session(self) -> None:
        """Close HTTP session."""
        if self._session:
            await self._session.close()
            self._session = None

    @retry(max_attempts=3, delay=1.0, backoff_factor=2.0)
    @measure_time()
    async def classify_alert(
        self, alert: Alert, context: Optional[dict[str, Any]] = None
    ) -> ClassificationResult:
        """
        Classify alert using LLM via proxy.

        Args:
            alert: Alert to classify
            context: Optional context information

        Returns:
            Classification result with severity and recommendations
        """
        start_time = time.time()

        try:
            # Build request payload
            payload = self._build_classification_payload(alert, context)

            # Make request to LLM proxy
            response_data = await self._make_llm_request(payload)

            # Parse structured response
            result = self._parse_classification_response(response_data, start_time)

            logger.info(
                "Alert classified successfully",
                alert_name=alert.alert_name,
                fingerprint=alert.fingerprint,
                severity=result.severity.value,
                confidence=result.confidence,
                processing_time=result.processing_time,
            )

            return result

        except Exception as e:
            logger.error(
                "Alert classification failed",
                alert_name=alert.alert_name,
                fingerprint=alert.fingerprint,
                error=str(e),
            )
            raise

    @retry(max_attempts=3, delay=1.0, backoff_factor=2.0)
    @measure_time()
    async def generate_recommendations(
        self, alert: Alert, classification: ClassificationResult
    ) -> list[str]:
        """
        Generate configuration recommendations based on classification.

        Args:
            alert: Original alert
            classification: Previous classification result

        Returns:
            List of actionable recommendations
        """
        try:
            # Build recommendation request
            payload = self._build_recommendation_payload(alert, classification)

            # Make request to LLM proxy
            response_data = await self._make_llm_request(payload)

            # Parse recommendations
            recommendations = self._parse_recommendation_response(response_data)

            logger.info(
                "Recommendations generated",
                alert_name=alert.alert_name,
                fingerprint=alert.fingerprint,
                recommendation_count=len(recommendations),
            )

            return recommendations

        except Exception as e:
            logger.error(
                "Recommendation generation failed",
                alert_name=alert.alert_name,
                fingerprint=alert.fingerprint,
                error=str(e),
            )
            # Return empty list on failure rather than raising
            return []

    async def _make_llm_request(self, payload: dict[str, Any]) -> dict[str, Any]:
        """Make request to LLM proxy service."""
        if not self._session:
            await self._init_session()

        url = f"{self.proxy_url}/v1/chat/completions"

        async with self._session.post(url, json=payload) as response:
            if response.status == 429:
                # Rate limited - add extra delay
                await asyncio.sleep(5)
                raise aiohttp.ClientResponseError(
                    request_info=response.request_info,
                    history=response.history,
                    status=response.status,
                    message="Rate limited by LLM proxy",
                )

            response.raise_for_status()
            return await response.json()

    def _build_classification_payload(
        self, alert: Alert, context: Optional[dict[str, Any]]
    ) -> dict[str, Any]:
        """Build LLM request payload for classification."""

        # Prepare alert context
        alert_context = {
            "alert_name": alert.alert_name,
            "status": alert.status.value,
            "labels": alert.labels,
            "annotations": alert.annotations,
            "namespace": alert.namespace or "unknown",
            "severity": alert.severity or "unknown",
        }

        if context:
            alert_context.update(context)

        # Build system message
        system_message = self.classification_prompt

        # Build user message with alert details
        user_message = f"""
Analyze this Kubernetes/Prometheus alert and classify it:

Alert Details:
- Name: {alert.alert_name}
- Status: {alert.status.value}
- Namespace: {alert.namespace or 'unknown'}
- Current Severity: {alert.severity or 'unknown'}

Labels:
{json.dumps(alert.labels, indent=2)}

Annotations:
{json.dumps(alert.annotations, indent=2)}

Please classify this alert and provide recommendations.
"""

        return {
            "model": self.model,
            "messages": [
                {"role": "system", "content": system_message},
                {"role": "user", "content": user_message},
            ],
            "functions": [self.function_schema],
            "function_call": {"name": "classify_alert"},
            "temperature": 0.1,  # Low temperature for consistent results
            "max_tokens": 1000,
        }

    def _build_recommendation_payload(
        self, alert: Alert, classification: ClassificationResult
    ) -> dict[str, Any]:
        """Build LLM request payload for recommendations."""

        system_message = """
You are an expert DevOps engineer specializing in Kubernetes monitoring and Alertmanager configuration.
Generate specific, actionable recommendations for improving alert configuration between Alertmanager and incident management tools like Rootly.

Focus on:
1. Reducing alert noise and false positives
2. Improving alert routing and escalation
3. Better labeling and annotation strategies
4. Threshold adjustments
5. Notification optimization
"""

        user_message = f"""
Based on this alert classification, provide specific configuration recommendations:

Alert: {alert.alert_name}
Classified Severity: {classification.severity.value}
Confidence: {classification.confidence:.2f}
Reasoning: {classification.reasoning}

Current Labels: {json.dumps(alert.labels, indent=2)}
Current Annotations: {json.dumps(alert.annotations, indent=2)}

Provide 3-5 specific, actionable recommendations for improving this alert's configuration.
"""

        return {
            "model": self.model,
            "messages": [
                {"role": "system", "content": system_message},
                {"role": "user", "content": user_message},
            ],
            "temperature": 0.3,  # Slightly higher for creativity in recommendations
            "max_tokens": 800,
        }

    def _build_classification_prompt(self) -> str:
        """Build system prompt for alert classification."""
        return """
You are an expert Kubernetes monitoring specialist with deep knowledge of Prometheus alerts and incident management.

Your task is to classify Kubernetes/Prometheus alerts into severity levels and provide reasoning.

Severity Levels:
- CRITICAL: Service outages, data loss, security breaches requiring immediate response
- WARNING: Performance degradation, approaching thresholds, potential issues
- INFO: Informational alerts, configuration changes, non-critical notifications
- NOISE: False positives, overly sensitive alerts, duplicate notifications

Consider these factors:
1. Impact on end users and business operations
2. Urgency of response required
3. Historical context and alert patterns
4. Kubernetes namespace and resource criticality
5. Current alert status (firing vs resolved)

Provide a confidence score (0.0-1.0) and clear reasoning for your classification.
"""

    def _build_function_schema(self) -> dict[str, Any]:
        """Build OpenAI function schema for structured responses."""
        return {
            "name": "classify_alert",
            "description": "Classify a Kubernetes alert by severity level",
            "parameters": {
                "type": "object",
                "properties": {
                    "severity": {
                        "type": "string",
                        "enum": ["critical", "warning", "info", "noise"],
                        "description": "Alert severity classification",
                    },
                    "confidence": {
                        "type": "number",
                        "minimum": 0.0,
                        "maximum": 1.0,
                        "description": "Confidence score for classification (0.0-1.0)",
                    },
                    "reasoning": {
                        "type": "string",
                        "description": "Detailed reasoning for the classification decision",
                    },
                    "recommendations": {
                        "type": "array",
                        "items": {"type": "string"},
                        "description": "List of actionable recommendations for improving alert configuration",
                    },
                    "tags": {
                        "type": "array",
                        "items": {"type": "string"},
                        "description": "Additional tags or categories for the alert",
                    },
                },
                "required": ["severity", "confidence", "reasoning", "recommendations"],
            },
        }

    def _parse_classification_response(
        self, response_data: dict[str, Any], start_time: float
    ) -> ClassificationResult:
        """Parse LLM response into ClassificationResult."""
        try:
            # Extract function call result
            message = response_data["choices"][0]["message"]

            if "function_call" not in message:
                raise ValueError("No function call in LLM response")

            function_result = json.loads(message["function_call"]["arguments"])

            # Parse severity
            severity_str = function_result["severity"].lower()
            severity = AlertSeverity(severity_str)

            # Extract other fields
            confidence = float(function_result["confidence"])
            reasoning = function_result["reasoning"]
            recommendations = function_result.get("recommendations", [])

            # Calculate processing time
            processing_time = time.time() - start_time

            # Build metadata
            metadata = {
                "model": self.model,
                "tags": function_result.get("tags", []),
                "llm_usage": response_data.get("usage", {}),
                "proxy_url": self.proxy_url,
            }

            return ClassificationResult(
                severity=severity,
                confidence=confidence,
                reasoning=reasoning,
                recommendations=recommendations,
                processing_time=processing_time,
                metadata=metadata,
            )

        except Exception as e:
            logger.error(f"Failed to parse LLM classification response: {e}")
            # Return fallback result
            return ClassificationResult(
                severity=AlertSeverity.WARNING,
                confidence=0.1,
                reasoning=f"Failed to parse LLM response: {e}",
                recommendations=["Review alert configuration manually"],
                processing_time=time.time() - start_time,
                metadata={"error": str(e)},
            )

    def _parse_recommendation_response(
        self, response_data: dict[str, Any]
    ) -> list[str]:
        """Parse recommendation response from LLM."""
        try:
            content = response_data["choices"][0]["message"]["content"]

            # Simple parsing - extract numbered/bulleted recommendations
            lines = content.strip().split("\n")
            recommendations = []

            for line in lines:
                line = line.strip()
                # Match numbered or bulleted items
                if line.startswith(("1.", "2.", "3.", "4.", "5.")) or line.startswith(
                    ("•", "-", "*")
                ):
                    # Clean up the recommendation text
                    clean_line = line.lstrip("123456789.- •*").strip()
                    if clean_line:
                        recommendations.append(clean_line)

            return recommendations[:5]  # Limit to 5 recommendations

        except Exception as e:
            logger.error(f"Failed to parse recommendation response: {e}")
            return []
