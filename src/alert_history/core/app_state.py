"""
Application state management для dependency injection.
"""

from typing import Any, Dict

from ..logging_config import get_logger

logger = get_logger(__name__)


class AppState:
    """Глобальное состояние приложения для dependency injection."""

    def __init__(self):
        self._state: Dict[str, Any] = {}

    def set(self, key: str, value: Any) -> None:
        """Установить значение в state."""
        self._state[key] = value
        logger.debug(f"App state set: {key} = {type(value).__name__}")

    def get(self, key: str, default: Any = None) -> Any:
        """Получить значение из state."""
        return self._state.get(key, default)

    def has(self, key: str) -> bool:
        """Проверить наличие ключа в state."""
        return key in self._state

    def __getattr__(self, name: str) -> Any:
        """Позволяет обращаться к state как к атрибутам."""
        if name.startswith("_"):
            raise AttributeError(
                f"'{self.__class__.__name__}' object has no attribute '{name}'"
            )
        return self._state.get(name)

    def __setattr__(self, name: str, value: Any) -> None:
        """Позволяет устанавливать state как атрибуты."""
        if name.startswith("_"):
            super().__setattr__(name, value)
        else:
            if not hasattr(self, "_state"):
                super().__setattr__("_state", {})
            self._state[name] = value
            logger.debug(f"App state set: {name} = {type(value).__name__}")


# Global app state instance
app_state = AppState()
