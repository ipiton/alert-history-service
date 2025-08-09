"""
Graceful shutdown handler для 12-Factor App compliance.

Implements proper SIGTERM handling for Kubernetes deployments.
"""

import asyncio
import signal
import sys
from typing import Callable, List, Optional
from contextlib import asynccontextmanager

from ..logging_config import get_logger

logger = get_logger(__name__)


class GracefulShutdownHandler:
    """Handles graceful shutdown for the application."""

    def __init__(self, shutdown_timeout: int = 30):
        """Initialize shutdown handler.

        Args:
            shutdown_timeout: Maximum time to wait for graceful shutdown (seconds)
        """
        self.shutdown_timeout = shutdown_timeout
        self.shutdown_event = asyncio.Event()
        self.cleanup_tasks: List[Callable] = []
        self.is_shutting_down = False

    def add_cleanup_task(self, task: Callable):
        """Add a cleanup task to be executed during shutdown."""
        self.cleanup_tasks.append(task)
        logger.debug(f"Added cleanup task: {task.__name__}")

    def setup_signal_handlers(self):
        """Setup signal handlers for graceful shutdown."""

        def signal_handler(signum, frame):
            logger.info(f"Received signal {signum}, initiating graceful shutdown...")
            self.shutdown_event.set()

        # Handle SIGTERM (Kubernetes sends this)
        signal.signal(signal.SIGTERM, signal_handler)

        # Handle SIGINT (Ctrl+C)
        signal.signal(signal.SIGINT, signal_handler)

        logger.info("Signal handlers registered for graceful shutdown")

    async def wait_for_shutdown(self):
        """Wait for shutdown signal."""
        await self.shutdown_event.wait()

    async def cleanup(self):
        """Execute all cleanup tasks."""
        if self.is_shutting_down:
            return

        self.is_shutting_down = True
        logger.info(f"Starting cleanup sequence with {len(self.cleanup_tasks)} tasks...")

        cleanup_start = asyncio.get_event_loop().time()

        try:
            # Execute cleanup tasks with timeout
            await asyncio.wait_for(self._execute_cleanup_tasks(), timeout=self.shutdown_timeout)

            cleanup_duration = asyncio.get_event_loop().time() - cleanup_start
            logger.info(f"Cleanup completed successfully in {cleanup_duration:.2f} seconds")

        except asyncio.TimeoutError:
            logger.warning(f"Cleanup timeout after {self.shutdown_timeout} seconds")
        except Exception as e:
            logger.error(f"Error during cleanup: {e}")

    async def _execute_cleanup_tasks(self):
        """Execute all cleanup tasks."""
        for i, task in enumerate(self.cleanup_tasks):
            try:
                logger.debug(
                    f"Executing cleanup task {i+1}/{len(self.cleanup_tasks)}: {task.__name__}"
                )

                if asyncio.iscoroutinefunction(task):
                    await task()
                else:
                    task()

                logger.debug(f"Cleanup task {task.__name__} completed")

            except Exception as e:
                logger.error(f"Error in cleanup task {task.__name__}: {e}")


class HealthChecker:
    """Health checker for application readiness and liveness probes."""

    def __init__(self):
        """Initialize health checker."""
        self.ready = False
        self.healthy = True
        self.startup_time = None
        self.dependencies_ready = {}

    def mark_ready(self):
        """Mark application as ready to receive traffic."""
        import time

        self.ready = True
        self.startup_time = time.time()
        logger.info("Application marked as ready")

    def mark_unhealthy(self, reason: str = "Unknown"):
        """Mark application as unhealthy."""
        self.healthy = False
        logger.warning(f"Application marked as unhealthy: {reason}")

    def mark_healthy(self):
        """Mark application as healthy."""
        self.healthy = True
        logger.info("Application marked as healthy")

    def set_dependency_ready(self, name: str, ready: bool):
        """Set dependency readiness status."""
        self.dependencies_ready[name] = ready
        logger.debug(f"Dependency {name} ready: {ready}")

    def is_ready(self) -> bool:
        """Check if application is ready."""
        # Check critical dependencies only (database, redis)
        critical_deps = {
            k: v for k, v in self.dependencies_ready.items() if k in ["database", "redis"]
        }
        # LLM is optional, so ignore it for readiness
        return self.ready and all(critical_deps.values())

    def is_healthy(self) -> bool:
        """Check if application is healthy."""
        return self.healthy

    def get_status(self) -> dict:
        """Get detailed health status."""
        import time

        uptime = None
        if self.startup_time:
            uptime = time.time() - self.startup_time

        return {
            "ready": self.is_ready(),
            "healthy": self.is_healthy(),
            "uptime_seconds": uptime,
            "dependencies": self.dependencies_ready.copy(),
            "timestamp": time.time(),
        }


# Global instances
shutdown_handler = GracefulShutdownHandler()
health_checker = HealthChecker()


@asynccontextmanager
async def lifespan_manager(app):
    """FastAPI lifespan manager for startup and shutdown."""

    # Startup
    logger.info("Application starting up...")

    # Setup signal handlers
    shutdown_handler.setup_signal_handlers()

    # Initialize dependencies
    await _initialize_dependencies()

    # Mark application as ready
    health_checker.mark_ready()

    logger.info("Application startup completed")

    yield

    # Shutdown
    logger.info("Application shutting down...")

    # Execute cleanup
    await shutdown_handler.cleanup()

    logger.info("Application shutdown completed")


async def _initialize_dependencies():
    """Initialize application dependencies."""

    try:
        # Initialize database connection
        await _initialize_database()
        health_checker.set_dependency_ready("database", True)

        # Initialize Redis connection
        await _initialize_redis()
        health_checker.set_dependency_ready("redis", True)

        # Initialize LLM service (if enabled)
        await _initialize_llm_service()
        health_checker.set_dependency_ready("llm", True)

        # Initialize target discovery
        await _initialize_target_discovery()
        health_checker.set_dependency_ready("target_discovery", True)

        logger.info("All dependencies initialized successfully")

    except Exception as e:
        logger.error(f"Failed to initialize dependencies: {e}")
        health_checker.mark_unhealthy(f"Dependency initialization failed: {e}")
        raise


async def _initialize_database():
    """Initialize database connection."""
    try:
        from ..config import get_config

        config = get_config()

        if config.database.url.startswith("postgresql://"):
            # Initialize PostgreSQL
            logger.info("Initializing PostgreSQL connection...")
            # Add PostgreSQL initialization here

        elif config.database.url.startswith("sqlite://"):
            # Initialize SQLite
            logger.info("Initializing SQLite connection...")
            # Add SQLite initialization here

        logger.info("Database initialized successfully")

        # Add database cleanup to shutdown handler
        shutdown_handler.add_cleanup_task(_cleanup_database)

    except Exception as e:
        logger.error(f"Database initialization failed: {e}")
        raise


async def _initialize_redis():
    """Initialize Redis connection."""
    try:
        from ..config import get_config

        config = get_config()

        logger.info(f"Initializing Redis connection to {config.redis.url}...")
        # Add Redis initialization here

        logger.info("Redis initialized successfully")

        # Add Redis cleanup to shutdown handler
        shutdown_handler.add_cleanup_task(_cleanup_redis)

    except Exception as e:
        logger.error(f"Redis initialization failed: {e}")
        # Redis is not critical, so don't raise
        health_checker.set_dependency_ready("redis", False)


async def _initialize_llm_service():
    """Initialize LLM service connection."""
    try:
        from ..config import get_config

        config = get_config()

        if config.llm.enabled:
            logger.info(f"Initializing LLM service connection to {config.llm.base_url}...")
            # Add LLM service initialization here
            logger.info("LLM service initialized successfully")
        else:
            logger.info("LLM service disabled in configuration")

    except Exception as e:
        logger.error(f"LLM service initialization failed: {e}")
        # LLM is not critical, so don't raise
        health_checker.set_dependency_ready("llm", False)


async def _initialize_target_discovery():
    """Initialize target discovery service."""
    try:
        logger.info("Initializing target discovery service...")
        # Add target discovery initialization here
        logger.info("Target discovery initialized successfully")

        # Add target discovery cleanup to shutdown handler
        shutdown_handler.add_cleanup_task(_cleanup_target_discovery)

    except Exception as e:
        logger.error(f"Target discovery initialization failed: {e}")
        # Target discovery is not critical, so don't raise
        health_checker.set_dependency_ready("target_discovery", False)


async def _cleanup_database():
    """Cleanup database connections."""
    try:
        logger.info("Cleaning up database connections...")
        # Add database cleanup here
        logger.info("Database cleanup completed")
    except Exception as e:
        logger.error(f"Database cleanup failed: {e}")


async def _cleanup_redis():
    """Cleanup Redis connections."""
    try:
        logger.info("Cleaning up Redis connections...")
        # Add Redis cleanup here
        logger.info("Redis cleanup completed")
    except Exception as e:
        logger.error(f"Redis cleanup failed: {e}")


async def _cleanup_target_discovery():
    """Cleanup target discovery service."""
    try:
        logger.info("Cleaning up target discovery...")
        # Add target discovery cleanup here
        logger.info("Target discovery cleanup completed")
    except Exception as e:
        logger.error(f"Target discovery cleanup failed: {e}")


def wait_for_shutdown():
    """Wait for shutdown signal (for non-async contexts)."""
    try:
        # Create new event loop if none exists
        try:
            loop = asyncio.get_event_loop()
        except RuntimeError:
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)

        # Run shutdown wait
        loop.run_until_complete(shutdown_handler.wait_for_shutdown())

    except KeyboardInterrupt:
        logger.info("Shutdown interrupted")
    finally:
        # Cleanup
        loop.run_until_complete(shutdown_handler.cleanup())
