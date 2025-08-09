"""
Graceful shutdown handler for Alert History Service.

Implements proper shutdown sequence to ensure:
- No data loss
- Ongoing requests complete
- Resources are cleaned up
- Connections are properly closed
"""

# Standard library imports
import asyncio
import time
from collections.abc import Awaitable
from typing import Callable, List

# Local imports
from ..logging_config import get_logger

logger = get_logger(__name__)


class GracefulShutdownHandler:
    """
    Handles graceful shutdown of the service.

    Ensures all ongoing operations complete before termination
    and resources are properly cleaned up.
    """

    def __init__(self, shutdown_timeout: int = 30):
        """Initialize graceful shutdown handler."""
        self.shutdown_timeout = shutdown_timeout
        self.shutdown_requested = False
        self.shutdown_callbacks: List[Callable[[], Awaitable[None]]] = []
        self._shutdown_event = asyncio.Event()

    def register_shutdown_callback(
        self, callback: Callable[[], Awaitable[None]]
    ) -> None:
        """Register callback to be called during shutdown."""
        self.shutdown_callbacks.append(callback)

    def initiate_shutdown(self) -> None:
        """Initiate graceful shutdown sequence."""
        if self.shutdown_requested:
            return

        self.shutdown_requested = True
        logger.info("Graceful shutdown initiated")
        self._shutdown_event.set()

    async def wait_for_shutdown(self) -> None:
        """Wait for shutdown to be initiated."""
        await self._shutdown_event.wait()

    async def perform_shutdown(self) -> None:
        """Perform the actual shutdown sequence."""
        logger.info("Starting graceful shutdown sequence")
        start_time = time.time()

        try:
            # Execute shutdown callbacks
            for i, callback in enumerate(self.shutdown_callbacks):
                try:
                    logger.info(
                        f"Executing shutdown callback {i+1}/{len(self.shutdown_callbacks)}"
                    )
                    await asyncio.wait_for(callback(), timeout=10)
                    logger.info(f"Shutdown callback {i+1} completed")
                except asyncio.TimeoutError:
                    logger.warning(f"Shutdown callback {i+1} timed out")
                except Exception as e:
                    logger.error(f"Error in shutdown callback {i+1}: {e}")

            # Wait for any remaining tasks
            pending_tasks = [
                task
                for task in asyncio.all_tasks()
                if not task.done() and task != asyncio.current_task()
            ]

            if pending_tasks:
                logger.info(
                    f"Waiting for {len(pending_tasks)} pending tasks to complete"
                )
                done, pending = await asyncio.wait(
                    pending_tasks,
                    timeout=self.shutdown_timeout - (time.time() - start_time),
                    return_when=asyncio.ALL_COMPLETED,
                )

                if pending:
                    logger.warning(
                        f"Forcefully cancelling {len(pending)} remaining tasks"
                    )
                    for task in pending:
                        task.cancel()

            elapsed = time.time() - start_time
            logger.info(f"Graceful shutdown completed in {elapsed:.2f} seconds")

        except Exception as e:
            logger.error(f"Error during graceful shutdown: {e}")
            raise
