"""
Configuration management для Alert History.
12-Factor App compliant configuration through environment variables.
"""

import os
from typing import Optional

from pydantic import BaseModel, Field


class DatabaseConfig(BaseModel):
    """Database configuration."""

    # Legacy SQLite support
    url: str = Field(default="sqlite:///alert_history.db")
    sqlite_path: str = Field(default="alert_history.db")

    # PostgreSQL configuration (T1.2)
    postgres_url: Optional[str] = Field(default=None)
    database_url: Optional[str] = Field(default=None)  # Generic database URL

    # Connection pooling
    pool_size: int = Field(default=10)
    pool_timeout: int = Field(default=30)
    pool_recycle: int = Field(default=3600)

    # PostgreSQL specific settings
    min_pool_size: int = Field(default=5)
    max_pool_size: int = Field(default=20)
    command_timeout: float = Field(default=60.0)
    query_timeout: float = Field(default=30.0)

    # Migration settings
    enable_migrations: bool = Field(default=True)
    auto_migrate: bool = Field(default=False)

    # Development settings
    echo: bool = Field(default=False)


class RedisConfig(BaseModel):
    """Redis configuration."""

    url: str = Field(default="redis://localhost:6379/0")
    pool_size: int = Field(default=10)
    pool_timeout: int = Field(default=30)
    socket_keepalive: bool = Field(default=True)
    socket_keepalive_options: dict = Field(default_factory=dict)


class LLMConfig(BaseModel):
    """LLM service configuration."""

    model_config = {"protected_namespaces": ()}

    enabled: bool = Field(default=False)
    api_key: Optional[str] = Field(default=None)
    base_url: str = Field(default="http://localhost:8000")
    timeout: int = Field(default=30)
    max_retries: int = Field(default=3)
    model_name: str = Field(default="gpt-4")


class ServerConfig(BaseModel):
    """Server configuration."""

    host: str = Field(default="0.0.0.0")
    port: int = Field(default=8080)
    debug: bool = Field(default=False)
    reload: bool = Field(default=False)
    workers: int = Field(default=1)
    log_level: str = Field(default="INFO")


class ProxyConfig(BaseModel):
    """Intelligent proxy configuration."""

    enabled: bool = Field(default=True)
    metrics_only_mode: bool = Field(default=False)
    target_discovery_enabled: bool = Field(default=True)
    target_refresh_interval: int = Field(default=300)  # seconds


class MonitoringConfig(BaseModel):
    """Monitoring and observability configuration."""

    metrics_enabled: bool = Field(default=True)
    health_check_enabled: bool = Field(default=True)
    structured_logs: bool = Field(default=True)
    log_requests: bool = Field(default=True)
    prometheus_enabled: bool = Field(default=True)


class SecurityConfig(BaseModel):
    """Security configuration."""

    cors_enabled: bool = Field(default=True)
    cors_origins: list = Field(default_factory=lambda: ["*"])
    rate_limiting_enabled: bool = Field(default=False)
    rate_limit_per_minute: int = Field(default=60)


class AlertsConfig(BaseModel):
    """Alert processing configuration."""

    retention_days: int = Field(default=30)
    batch_size: int = Field(default=100)
    max_concurrent_alerts: int = Field(default=10)
    enable_classification: bool = Field(default=False)


class Config(BaseModel):
    """Complete application configuration."""

    # Service metadata
    service_name: str = Field(default="alert-history")
    service_version: str = Field(default="1.0.0")
    environment: str = Field(default="development")

    # Component configurations
    database: DatabaseConfig = Field(default_factory=DatabaseConfig)
    redis: RedisConfig = Field(default_factory=RedisConfig)
    alerts: AlertsConfig = Field(default_factory=AlertsConfig)
    llm: LLMConfig = Field(default_factory=LLMConfig)
    server: ServerConfig = Field(default_factory=ServerConfig)
    proxy: ProxyConfig = Field(default_factory=ProxyConfig)
    monitoring: MonitoringConfig = Field(default_factory=MonitoringConfig)
    security: SecurityConfig = Field(default_factory=SecurityConfig)


def get_config() -> Config:
    """Get application configuration from environment variables (12-Factor App)."""

    # Service metadata from environment
    service_name = os.getenv("SERVICE_NAME", "alert-history")
    service_version = os.getenv("SERVICE_VERSION", "1.0.0")
    environment = os.getenv("ENVIRONMENT", "development")

    # Database configuration (T1.2: PostgreSQL support)
    database_config = DatabaseConfig(
        # Legacy support
        url=os.getenv("DATABASE_URL", "sqlite:///alert_history.db"),
        sqlite_path=os.getenv("SQLITE_PATH", "alert_history.db"),
        # PostgreSQL configuration
        postgres_url=os.getenv("DATABASE_POSTGRES_URL"),
        database_url=os.getenv("DATABASE_URL"),  # Generic database URL
        # Connection pooling
        pool_size=int(os.getenv("DATABASE_POOL_SIZE", "10")),
        pool_timeout=int(os.getenv("DATABASE_POOL_TIMEOUT", "30")),
        pool_recycle=int(os.getenv("DATABASE_POOL_RECYCLE", "3600")),
        # PostgreSQL specific
        min_pool_size=int(os.getenv("DATABASE_MIN_POOL_SIZE", "5")),
        max_pool_size=int(os.getenv("DATABASE_MAX_POOL_SIZE", "20")),
        command_timeout=float(os.getenv("DATABASE_COMMAND_TIMEOUT", "60.0")),
        query_timeout=float(os.getenv("DATABASE_QUERY_TIMEOUT", "30.0")),
        # Migration settings
        enable_migrations=os.getenv("DATABASE_ENABLE_MIGRATIONS", "true").lower()
        == "true",
        auto_migrate=os.getenv("DATABASE_AUTO_MIGRATE", "false").lower() == "true",
        # Development
        echo=os.getenv("DATABASE_ECHO", "false").lower() == "true",
    )

    # Redis configuration
    redis_config = RedisConfig(
        url=os.getenv("REDIS_URL", "redis://localhost:6379/0"),
        pool_size=int(os.getenv("REDIS_POOL_SIZE", "10")),
        pool_timeout=int(os.getenv("REDIS_POOL_TIMEOUT", "30")),
        socket_keepalive=os.getenv("REDIS_KEEPALIVE", "true").lower() == "true",
    )

    # Alerts configuration
    alerts_config = AlertsConfig(
        retention_days=int(os.getenv("RETENTION_DAYS", "30")),
        batch_size=int(os.getenv("ALERT_BATCH_SIZE", "100")),
        max_concurrent_alerts=int(os.getenv("MAX_CONCURRENT_ALERTS", "10")),
        enable_classification=os.getenv("ENABLE_CLASSIFICATION", "false").lower()
        == "true",
    )

    # LLM configuration
    llm_config = LLMConfig(
        enabled=os.getenv("LLM_ENABLED", "false").lower() == "true",
        api_key=os.getenv("LLM_API_KEY"),
        base_url=os.getenv(
            "LLM_PROXY_URL", "http://localhost:8000"
        ),  # Use LLM_PROXY_URL from Helm
        timeout=int(os.getenv("LLM_TIMEOUT", "30")),
        max_retries=int(os.getenv("LLM_MAX_RETRIES", "3")),
        model_name=os.getenv("LLM_MODEL", "gpt-4"),  # Use LLM_MODEL from Helm
    )

    # Server configuration
    server_config = ServerConfig(
        host=os.getenv("HOST", "0.0.0.0"),
        port=int(os.getenv("PORT", "8080")),
        debug=os.getenv("DEBUG", "false").lower() == "true",
        reload=os.getenv("RELOAD", "false").lower() == "true",
        workers=int(os.getenv("WORKERS", "1")),
        log_level=os.getenv("LOG_LEVEL", "INFO"),
    )

    # Proxy configuration
    proxy_config = ProxyConfig(
        enabled=os.getenv("PROXY_ENABLED", "true").lower() == "true",
        metrics_only_mode=os.getenv("METRICS_ONLY_MODE", "false").lower() == "true",
        target_discovery_enabled=os.getenv("TARGET_DISCOVERY_ENABLED", "true").lower()
        == "true",
        target_refresh_interval=int(os.getenv("TARGET_REFRESH_INTERVAL", "300")),
    )

    # Monitoring configuration
    monitoring_config = MonitoringConfig(
        metrics_enabled=os.getenv("METRICS_ENABLED", "true").lower() == "true",
        health_check_enabled=os.getenv("HEALTH_CHECK_ENABLED", "true").lower()
        == "true",
        structured_logs=os.getenv("STRUCTURED_LOGS", "true").lower() == "true",
        log_requests=os.getenv("LOG_REQUESTS", "true").lower() == "true",
        prometheus_enabled=os.getenv("PROMETHEUS_ENABLED", "true").lower() == "true",
    )

    # Security configuration
    cors_origins = (
        os.getenv("CORS_ORIGINS", "*").split(",")
        if os.getenv("CORS_ORIGINS")
        else ["*"]
    )
    security_config = SecurityConfig(
        cors_enabled=os.getenv("CORS_ENABLED", "true").lower() == "true",
        cors_origins=cors_origins,
        rate_limiting_enabled=os.getenv("RATE_LIMITING_ENABLED", "false").lower()
        == "true",
        rate_limit_per_minute=int(os.getenv("RATE_LIMIT_PER_MINUTE", "60")),
    )

    # Create complete configuration
    config = Config(
        service_name=service_name,
        service_version=service_version,
        environment=environment,
        database=database_config,
        redis=redis_config,
        alerts=alerts_config,
        llm=llm_config,
        server=server_config,
        proxy=proxy_config,
        monitoring=monitoring_config,
        security=security_config,
    )

    return config


def validate_config(config: Config) -> bool:
    """Validate configuration."""

    try:
        # Basic validation
        if config.server.port < 1 or config.server.port > 65535:
            print(f"Invalid port number: {config.server.port}")
            return False

        if config.database.pool_size < 1:
            print(f"Invalid database pool size: {config.database.pool_size}")
            return False

        if config.redis.pool_size < 1:
            print(f"Invalid redis pool size: {config.redis.pool_size}")
            return False

        return True

    except Exception as e:
        print(f"Configuration validation failed: {e}")
        return False
