"""
Structured logging configuration for Alert History Service.

Implements 12-Factor App logging principles:
- Logs to stdout (not files)
- Structured JSON format
- Configurable log levels
- Request tracing
- Performance monitoring
"""

# Standard library imports
import json
import logging
import sys
import traceback
from datetime import datetime
from typing import Any, Dict, Optional
from uuid import uuid4

# First-party imports
import os


class JSONFormatter(logging.Formatter):
    """JSON formatter for structured logging."""

    def __init__(self, service_name: str, version: str, environment: str):
        """Initialize JSON formatter with service metadata."""
        super().__init__()
        self.service_name = service_name
        self.version = version
        self.environment = environment

    def format(self, record: logging.LogRecord) -> str:
        """Format log record as JSON."""
        log_entry = {
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "level": record.levelname,
            "service": self.service_name,
            "version": self.version,
            "environment": self.environment,
            "logger": record.name,
            "message": record.getMessage(),
            "module": record.module,
            "function": record.funcName,
            "line": record.lineno,
        }

        # Add thread/process info for debugging
        if record.thread:
            log_entry["thread_id"] = record.thread
        if record.process:
            log_entry["process_id"] = record.process

        # Add exception information if present
        if record.exc_info:
            log_entry["exception"] = {
                "type": record.exc_info[0].__name__,
                "message": str(record.exc_info[1]),
                "traceback": traceback.format_exception(*record.exc_info),
            }

        # Add extra fields from LogRecord
        extra_fields = {}
        for key, value in record.__dict__.items():
            if key not in {
                "name",
                "msg",
                "args",
                "levelname",
                "levelno",
                "pathname",
                "filename",
                "module",
                "lineno",
                "funcName",
                "created",
                "msecs",
                "relativeCreated",
                "thread",
                "threadName",
                "processName",
                "process",
                "getMessage",
                "exc_info",
                "exc_text",
                "stack_info",
            }:
                extra_fields[key] = value

        if extra_fields:
            log_entry["extra"] = extra_fields

        return json.dumps(log_entry, default=str, ensure_ascii=False)


class StructuredLogger:
    """Structured logger with convenience methods for common operations."""

    def __init__(self, name: str):
        """Initialize structured logger."""
        self.logger = logging.getLogger(name)
        self._request_id: Optional[str] = None

    def set_request_id(self, request_id: str) -> None:
        """Set request ID for request tracing."""
        self._request_id = request_id

    def clear_request_id(self) -> None:
        """Clear request ID."""
        self._request_id = None

    def _log(self, level: int, message: str, **kwargs: Any) -> None:
        """Internal logging method with request ID injection."""
        extra = kwargs.copy()
        if self._request_id:
            extra["request_id"] = self._request_id

        self.logger.log(level, message, extra=extra)

    def debug(self, message: str, **kwargs: Any) -> None:
        """Log debug message."""
        self._log(logging.DEBUG, message, **kwargs)

    def info(self, message: str, **kwargs: Any) -> None:
        """Log info message."""
        self._log(logging.INFO, message, **kwargs)

    def warning(self, message: str, **kwargs: Any) -> None:
        """Log warning message."""
        self._log(logging.WARNING, message, **kwargs)

    def error(self, message: str, **kwargs: Any) -> None:
        """Log error message."""
        self._log(logging.ERROR, message, **kwargs)

    def critical(self, message: str, **kwargs: Any) -> None:
        """Log critical message."""
        self._log(logging.CRITICAL, message, **kwargs)

    def exception(self, message: str, **kwargs: Any) -> None:
        """Log exception with traceback."""
        extra = kwargs.copy()
        if self._request_id:
            extra["request_id"] = self._request_id
        self.logger.exception(message, extra=extra)


class AlertLogger(StructuredLogger):
    """Specialized logger for alert operations."""

    def alert_received(
        self,
        alert_name: str,
        fingerprint: str,
        status: str,
        processing_time: Optional[float] = None,
        **kwargs: Any,
    ) -> None:
        """Log alert received event."""
        self.info(
            "Alert received",
            event="alert_received",
            alert_name=alert_name,
            fingerprint=fingerprint,
            status=status,
            processing_time=processing_time,
            **kwargs,
        )

    def alert_classified(
        self,
        alert_name: str,
        fingerprint: str,
        severity: str,
        confidence: float,
        processing_time: float,
        **kwargs: Any,
    ) -> None:
        """Log alert classification event."""
        self.info(
            "Alert classified",
            event="alert_classified",
            alert_name=alert_name,
            fingerprint=fingerprint,
            severity=severity,
            confidence=confidence,
            processing_time=processing_time,
            **kwargs,
        )

    def alert_published(
        self,
        alert_name: str,
        fingerprint: str,
        target: str,
        success: bool,
        processing_time: float,
        error: Optional[str] = None,
        **kwargs: Any,
    ) -> None:
        """Log alert publishing event."""
        level = self.info if success else self.error
        level(
            "Alert published" if success else "Alert publishing failed",
            event="alert_published",
            alert_name=alert_name,
            fingerprint=fingerprint,
            target=target,
            success=success,
            processing_time=processing_time,
            error=error,
            **kwargs,
        )

    def database_operation(
        self,
        operation: str,
        table: str,
        affected_rows: int,
        processing_time: float,
        **kwargs: Any,
    ) -> None:
        """Log database operation."""
        self.debug(
            f"Database {operation}",
            event="database_operation",
            operation=operation,
            table=table,
            affected_rows=affected_rows,
            processing_time=processing_time,
            **kwargs,
        )


class PerformanceLogger(StructuredLogger):
    """Logger for performance monitoring."""

    def request_started(
        self, method: str, path: str, user_agent: Optional[str] = None, **kwargs: Any
    ) -> None:
        """Log request start."""
        self.info(
            "Request started",
            event="request_started",
            method=method,
            path=path,
            user_agent=user_agent,
            **kwargs,
        )

    def request_completed(
        self,
        method: str,
        path: str,
        status_code: int,
        processing_time: float,
        **kwargs: Any,
    ) -> None:
        """Log request completion."""
        level = self.info if status_code < 400 else self.error
        level(
            "Request completed",
            event="request_completed",
            method=method,
            path=path,
            status_code=status_code,
            processing_time=processing_time,
            **kwargs,
        )

    def llm_request(
        self,
        model: str,
        operation: str,
        success: bool,
        processing_time: float,
        tokens_used: Optional[int] = None,
        error: Optional[str] = None,
        **kwargs: Any,
    ) -> None:
        """Log LLM request."""
        level = self.info if success else self.error
        level(
            "LLM request completed" if success else "LLM request failed",
            event="llm_request",
            model=model,
            operation=operation,
            success=success,
            processing_time=processing_time,
            tokens_used=tokens_used,
            error=error,
            **kwargs,
        )


def setup_logging() -> None:
    """Setup structured logging configuration with 12-Factor App compliance."""

    # Get configuration from environment variables (12-Factor principle)
    service_name = os.getenv("SERVICE_NAME", "alert-history")
    version = os.getenv("SERVICE_VERSION", "1.0.0")
    environment = os.getenv("ENVIRONMENT", "development")
    log_level = os.getenv("LOG_LEVEL", "INFO").upper()
    enable_json = os.getenv("LOG_JSON", "true").lower() == "true"
    enable_correlation = os.getenv("LOG_CORRELATION", "true").lower() == "true"

    # Configure root logger
    root_logger = logging.getLogger()
    root_logger.setLevel(getattr(logging, log_level))

    # Remove existing handlers
    for handler in root_logger.handlers[:]:
        root_logger.removeHandler(handler)

    # Create stdout handler (12-Factor principle: logs to stdout)
    handler = logging.StreamHandler(sys.stdout)

    if enable_json:
        # Use JSON formatter for structured logging
        formatter = JSONFormatter(
            service_name=service_name,
            version=version,
            environment=environment,
        )
    else:
        # Use simple formatter for development
        formatter = logging.Formatter("%(asctime)s - %(name)s - %(levelname)s - %(message)s")

    handler.setFormatter(formatter)
    root_logger.addHandler(handler)

    # Silence noisy loggers in development
    if environment == "development":
        logging.getLogger("uvicorn.access").setLevel(logging.WARNING)
        logging.getLogger("asyncio").setLevel(logging.WARNING)


def get_logger(name: str) -> StructuredLogger:
    """Get structured logger instance."""
    return StructuredLogger(name)


def get_alert_logger() -> AlertLogger:
    """Get alert-specific logger instance."""
    return AlertLogger("alert_history.alerts")


def get_performance_logger() -> PerformanceLogger:
    """Get performance logger instance."""
    return PerformanceLogger("alert_history.performance")


def generate_request_id() -> str:
    """Generate unique request ID for tracing."""
    return str(uuid4())


# Context manager for request tracing
class RequestContext:
    """Context manager for request tracing."""

    def __init__(self, logger: StructuredLogger, request_id: Optional[str] = None):
        """Initialize request context."""
        self.logger = logger
        self.request_id = request_id or generate_request_id()

    def __enter__(self) -> str:
        """Enter request context."""
        self.logger.set_request_id(self.request_id)
        return self.request_id

    def __exit__(self, exc_type, exc_val, exc_tb) -> None:
        """Exit request context."""
        self.logger.clear_request_id()


# Module-level loggers
logger = get_logger(__name__)
alert_logger = get_alert_logger()
performance_logger = get_performance_logger()
