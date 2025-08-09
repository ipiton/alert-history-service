"""
Configuration module for Alert History Service.

Implements 12-Factor App configuration through environment variables.
All settings are externalized from code and can be overridden by environment.
"""

# Standard library imports
import os
from dataclasses import dataclass
from typing import Optional


@dataclass
class DatabaseConfig:
    """Database configuration settings."""

    # SQLite (legacy, for backward compatibility)
    sqlite_path: str

    # PostgreSQL (for horizontal scaling)
    postgres_host: str
    postgres_port: int
    postgres_database: str
    postgres_username: str
    postgres_password: str
    postgres_ssl_mode: str
    postgres_pool_size: int
    postgres_max_overflow: int
    postgres_pool_timeout: int

    @property
    def postgres_url(self) -> str:
        """Construct PostgreSQL connection URL."""
        # Support both direct URL and individual components
        database_url = os.getenv("DATABASE_URL")
        if database_url:
            return database_url

        return (
            f"postgresql://{self.postgres_username}:{self.postgres_password}"
            f"@{self.postgres_host}:{self.postgres_port}/{self.postgres_database}"
            f"?sslmode={self.postgres_ssl_mode}"
        )


@dataclass
class RedisConfig:
    """Redis/KeyDB configuration for caching and distributed locking."""

    host: str
    port: int
    password: Optional[str]
    database: int
    ssl: bool
    pool_size: int
    retry_attempts: int
    timeout: int

    @property
    def redis_url(self) -> str:
        """Construct Redis/KeyDB connection URL."""
        # Support both direct URL and individual components
        redis_url = os.getenv("REDIS_URL")
        if redis_url:
            return redis_url

        scheme = "rediss" if self.ssl else "redis"
        auth = f":{self.password}@" if self.password else ""
        return f"{scheme}://{auth}{self.host}:{self.port}/{self.database}"


@dataclass
class LLMConfig:
    """LLM Proxy configuration settings."""

    enabled: bool = False
    proxy_url: str = ""
    api_key: str = ""
    model: str = "gpt-4"
    timeout: int = 30
    max_retries: int = 3
    retry_delay: float = 1.0
    batch_size: int = 10
    cache_ttl: int = 3600


@dataclass
class ServerConfig:
    """Server configuration settings."""

    host: str
    port: int
    workers: int
    log_level: str
    reload: bool
    debug: bool
    cors_origins: list[str]
    request_timeout: int
    graceful_timeout: int


@dataclass
class AlertConfig:
    """Alert processing configuration."""

    retention_days: int
    batch_size: int
    processing_timeout: int
    max_concurrent_alerts: int
    enable_classification: bool


@dataclass
class MigrationConfig:
    """Database migration configuration."""

    enabled: bool
    auto_migrate: bool
    backup_before_migration: bool
    batch_size: int
    verify_data: bool
    cleanup_after_migration: bool
    enable_publishing: bool


@dataclass
class MonitoringConfig:
    """Monitoring and metrics configuration."""

    prometheus_enabled: bool
    metrics_port: int
    health_check_interval: int
    log_structured: bool
    log_format: str


class Config:
    """Main configuration class with 12-Factor App compliance."""

    def __init__(self):
        """Initialize configuration from environment variables."""
        self.database = self._load_database_config()
        self.redis = self._load_redis_config()
        self.llm = self._load_llm_config()
        self.server = self._load_server_config()
        self.alerts = self._load_alert_config()
        self.migration = self._load_migration_config()
        self.monitoring = self._load_monitoring_config()

        # Application environment
        self.environment = os.getenv("ENVIRONMENT", "development")
        self.service_name = os.getenv("SERVICE_NAME", "alert-history-llm")
        self.version = os.getenv("SERVICE_VERSION", "1.0.0")

    def _load_database_config(self) -> DatabaseConfig:
        """Load database configuration from environment."""
        return DatabaseConfig(
            # SQLite (legacy)
            sqlite_path=os.getenv("ALERT_HISTORY_DB", "/data/alert_history.sqlite3"),
            # PostgreSQL (CloudNativePG compatible)
            postgres_host=os.getenv("POSTGRES_HOST", "localhost"),
            postgres_port=int(os.getenv("POSTGRES_PORT", "5432")),
            postgres_database=os.getenv("POSTGRES_DATABASE", "alert_history"),
            postgres_username=os.getenv("POSTGRES_USERNAME", "alert_history"),
            postgres_password=os.getenv("POSTGRES_PASSWORD", ""),
            postgres_ssl_mode=os.getenv("POSTGRES_SSL_MODE", "prefer"),
            postgres_pool_size=int(os.getenv("POSTGRES_POOL_SIZE", "10")),
            postgres_max_overflow=int(os.getenv("POSTGRES_MAX_OVERFLOW", "20")),
            postgres_pool_timeout=int(os.getenv("POSTGRES_POOL_TIMEOUT", "30")),
        )

    def _load_redis_config(self) -> RedisConfig:
        """Load Redis configuration from environment (KeyDB compatible)."""
        return RedisConfig(
            host=os.getenv("REDIS_HOST", "localhost"),
            port=int(os.getenv("REDIS_PORT", "6379")),
            password=os.getenv("REDIS_PASSWORD"),
            database=int(os.getenv("REDIS_DATABASE", "0")),
            ssl=os.getenv("REDIS_SSL", "false").lower() == "true",
            pool_size=int(os.getenv("REDIS_POOL_SIZE", "10")),
            retry_attempts=int(os.getenv("REDIS_RETRY_ATTEMPTS", "3")),
            timeout=int(os.getenv("REDIS_TIMEOUT", "5")),
        )

    def _load_llm_config(self) -> LLMConfig:
        """Load LLM configuration from environment."""
        # LLM enabled if API key is provided and enabled flag is set
        enabled = (
            os.getenv("LLM_ENABLED", "false").lower() == "true" and
            bool(os.getenv("LLM_API_KEY", ""))
        )

        return LLMConfig(
            enabled=enabled,
            proxy_url=os.getenv("LLM_PROXY_URL", "http://llm-proxy:8080"),
            api_key=os.getenv("LLM_API_KEY", ""),
            model=os.getenv("LLM_MODEL", "gpt-4"),
            timeout=int(os.getenv("LLM_TIMEOUT", "30")),
            max_retries=int(os.getenv("LLM_MAX_RETRIES", "3")),
            retry_delay=float(os.getenv("LLM_RETRY_DELAY", "1.0")),
            batch_size=int(os.getenv("LLM_BATCH_SIZE", "10")),
            cache_ttl=int(os.getenv("LLM_CACHE_TTL", "3600")),
        )

    def _load_server_config(self) -> ServerConfig:
        """Load server configuration from environment."""
        cors_origins_str = os.getenv("CORS_ORIGINS", "*")
        cors_origins = [origin.strip() for origin in cors_origins_str.split(",")]

        return ServerConfig(
            host=os.getenv("SERVER_HOST", "0.0.0.0"),
            port=int(os.getenv("SERVER_PORT", "8080")),
            workers=int(os.getenv("SERVER_WORKERS", "1")),
            log_level=os.getenv("LOG_LEVEL", "INFO"),
            reload=os.getenv("SERVER_RELOAD", "false").lower() == "true",
            debug=os.getenv("DEBUG", "false").lower() == "true",
            cors_origins=cors_origins,
            request_timeout=int(os.getenv("REQUEST_TIMEOUT", "30")),
            graceful_timeout=int(os.getenv("GRACEFUL_TIMEOUT", "30")),
        )

    def _load_alert_config(self) -> AlertConfig:
        """Load alert processing configuration from environment."""
        return AlertConfig(
            retention_days=int(os.getenv("RETENTION_DAYS", "30")),
            batch_size=int(os.getenv("ALERT_BATCH_SIZE", "100")),
            processing_timeout=int(os.getenv("ALERT_PROCESSING_TIMEOUT", "60")),
            max_concurrent_alerts=int(os.getenv("MAX_CONCURRENT_ALERTS", "50")),
            enable_classification=os.getenv("ENABLE_CLASSIFICATION", "true").lower()
            == "true",
        )

    def _load_migration_config(self) -> MigrationConfig:
        """Load database migration configuration from environment."""
        return MigrationConfig(
            enabled=os.getenv("MIGRATION_ENABLED", "false").lower() == "true",
            auto_migrate=os.getenv("AUTO_MIGRATE", "false").lower() == "true",
            backup_before_migration=os.getenv("BACKUP_BEFORE_MIGRATION", "true").lower()
            == "true",
            batch_size=int(os.getenv("MIGRATION_BATCH_SIZE", "1000")),
            verify_data=os.getenv("VERIFY_MIGRATION_DATA", "true").lower() == "true",
            cleanup_after_migration=os.getenv(
                "CLEANUP_AFTER_MIGRATION", "false"
            ).lower()
            == "true",
            enable_publishing=os.getenv("ENABLE_PUBLISHING", "true").lower() == "true",
        )

    def _load_monitoring_config(self) -> MonitoringConfig:
        """Load monitoring configuration from environment."""
        return MonitoringConfig(
            prometheus_enabled=os.getenv("PROMETHEUS_ENABLED", "true").lower()
            == "true",
            metrics_port=int(os.getenv("METRICS_PORT", "9090")),
            health_check_interval=int(os.getenv("HEALTH_CHECK_INTERVAL", "30")),
            log_structured=os.getenv("LOG_STRUCTURED", "true").lower() == "true",
            log_format=os.getenv("LOG_FORMAT", "json"),
        )

    def is_production(self) -> bool:
        """Check if running in production environment."""
        return self.environment.lower() in ["production", "prod"]

    def is_development(self) -> bool:
        """Check if running in development environment."""
        return self.environment.lower() in ["development", "dev"]

    def validate(self) -> list[str]:
        """Validate configuration and return list of errors."""
        errors = []

        # Required in production
        if self.is_production():
            if not self.database.postgres_password:
                errors.append("POSTGRES_PASSWORD is required in production")
            if not self.llm.api_key:
                errors.append("LLM_API_KEY is required in production")

        # Database validation
        if self.database.postgres_port <= 0 or self.database.postgres_port > 65535:
            errors.append("POSTGRES_PORT must be between 1 and 65535")

        # Redis validation
        if self.redis.port <= 0 or self.redis.port > 65535:
            errors.append("REDIS_PORT must be between 1 and 65535")

        # Server validation
        if self.server.port <= 0 or self.server.port > 65535:
            errors.append("SERVER_PORT must be between 1 and 65535")
        if self.server.workers <= 0:
            errors.append("SERVER_WORKERS must be positive")

        # Alert validation
        if self.alerts.retention_days <= 0:
            errors.append("RETENTION_DAYS must be positive")
        if self.alerts.batch_size <= 0:
            errors.append("ALERT_BATCH_SIZE must be positive")

        return errors


# Global configuration instance
config = Config()


def get_config() -> Config:
    """Get global configuration instance."""
    return config


def validate_config() -> None:
    """Validate configuration and raise exception if invalid."""
    errors = config.validate()
    if errors:
        error_msg = "Configuration validation failed:\n" + "\n".join(
            f"- {error}" for error in errors
        )
        raise ValueError(error_msg)
