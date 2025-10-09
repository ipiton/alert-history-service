#!/usr/bin/env python3
"""
Скрипт для поиска в индексированном коде с помощью Ollama и Qdrant.

Использование:
    python search_code.py "функция для работы с базой данных" --collection alert-history-code
"""

import argparse
import json
import logging
import sys
from typing import Any, Dict, List

import requests

# Настройка логирования
logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)


class CodeSearcher:
    """Класс для поиска в индексированном коде."""

    def __init__(
        self,
        ollama_url: str = "http://localhost:11434",
        qdrant_url: str = "http://localhost:6333",
    ):
        self.ollama_url = ollama_url
        self.qdrant_url = qdrant_url
        self.session = requests.Session()
        self.session.timeout = 30

    def get_query_embedding(self, query: str, model: str) -> List[float]:
        """Получить эмбеддинг для поискового запроса."""
        try:
            response = self.session.post(
                f"{self.ollama_url}/api/embeddings",
                json={"model": model, "prompt": query},
            )
            response.raise_for_status()

            data = response.json()
            if "data" in data:
                return data["data"][0]["embedding"]
            else:
                return data["embedding"]

        except requests.exceptions.RequestException as e:
            logger.error(f"Ошибка при получении эмбеддинга запроса: {e}")
            raise
        except Exception as e:
            logger.error(f"Неожиданная ошибка при получении эмбеддинга: {e}")
            raise

    def search_in_collection(
        self,
        collection_name: str,
        query_vector: List[float],
        limit: int = 10,
        score_threshold: float = 0.7,
    ) -> List[Dict[str, Any]]:
        """Поиск в коллекции Qdrant."""
        try:
            response = self.session.post(
                f"{self.qdrant_url}/collections/{collection_name}/points/search",
                json={
                    "vector": query_vector,
                    "limit": limit,
                    "score_threshold": score_threshold,
                    "with_payload": True,
                },
            )
            response.raise_for_status()

            data = response.json()
            return data.get("result", [])

        except requests.exceptions.RequestException as e:
            logger.error(f"Ошибка при поиске в коллекции: {e}")
            raise
        except Exception as e:
            logger.error(f"Неожиданная ошибка при поиске: {e}")
            raise

    def search(
        self,
        query: str,
        collection_name: str,
        model: str,
        limit: int = 10,
        score_threshold: float = 0.7,
    ) -> List[Dict[str, Any]]:
        """Выполнить поиск по запросу."""
        logger.info(f"Поиск: '{query}' в коллекции {collection_name}")

        # Получаем эмбеддинг запроса
        query_vector = self.get_query_embedding(query, model)

        # Ищем в коллекции
        results = self.search_in_collection(
            collection_name, query_vector, limit, score_threshold
        )

        logger.info(f"Найдено {len(results)} результатов")
        return results

    def format_result(self, result: Dict[str, Any]) -> str:
        """Форматировать результат поиска для вывода."""
        payload = result.get("payload", {})
        score = result.get("score", 0)

        file_path = payload.get("file_path", "unknown")
        language = payload.get("language", "unknown")
        chunk_index = payload.get("chunk_index", 0)
        total_chunks = payload.get("total_chunks", 1)

        # Обрезаем содержимое для вывода
        content = payload.get("content", "")
        if len(content) > 200:
            content = content[:200] + "..."

        return f"""
Файл: {file_path} ({language})
Чанк: {chunk_index + 1}/{total_chunks}
Релевантность: {score:.3f}
---
{content}
---"""


def main():
    parser = argparse.ArgumentParser(description="Поиск в индексированном коде")
    parser.add_argument("query", help="Поисковый запрос")
    parser.add_argument(
        "--collection", default="alert-history-code", help="Название коллекции в Qdrant"
    )
    parser.add_argument(
        "--model",
        default="mxbai-embed-large:latest",
        help="Модель для эмбеддингов в Ollama",
    )
    parser.add_argument(
        "--limit", type=int, default=10, help="Максимальное количество результатов"
    )
    parser.add_argument(
        "--score-threshold",
        type=float,
        default=0.7,
        help="Минимальный порог релевантности",
    )
    parser.add_argument(
        "--ollama-url", default="http://localhost:11434", help="URL Ollama сервера"
    )
    parser.add_argument(
        "--qdrant-url", default="http://localhost:6333", help="URL Qdrant сервера"
    )
    parser.add_argument(
        "--json", action="store_true", help="Вывести результаты в JSON формате"
    )

    args = parser.parse_args()

    # Создаем поисковик
    searcher = CodeSearcher(args.ollama_url, args.qdrant_url)

    try:
        # Выполняем поиск
        results = searcher.search(
            args.query, args.collection, args.model, args.limit, args.score_threshold
        )

        if not results:
            print("Результаты не найдены")
            sys.exit(0)

        if args.json:
            # Выводим в JSON формате
            print(json.dumps(results, indent=2, ensure_ascii=False))
        else:
            # Выводим в читаемом формате
            for i, result in enumerate(results, 1):
                print(f"\n=== Результат {i} ===")
                print(searcher.format_result(result))

    except Exception as e:
        logger.error(f"Ошибка при поиске: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main()
