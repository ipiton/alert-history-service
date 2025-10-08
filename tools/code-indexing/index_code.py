#!/usr/bin/env python3
"""
Скрипт для индексации кода проекта с помощью Ollama и Qdrant.

Использование:
    python index_code.py --collection alert-history-code --model mxbai-embed-large:latest
"""

import argparse
import hashlib
import json
import logging
import os
import sys
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Dict, List, Optional

import requests

# Настройка логирования
logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)


@dataclass
class CodeFile:
    """Представление файла кода для индексации."""

    path: str
    content: str
    language: str
    size: int
    hash: str


class OllamaClient:
    """Клиент для работы с Ollama API."""

    def __init__(self, base_url: str = "http://localhost:11434"):
        self.base_url = base_url
        self.session = requests.Session()
        self.session.timeout = 30

    def get_embeddings(self, texts: List[str], model: str) -> List[List[float]]:
        """Получить эмбеддинги для списка текстов."""
        embeddings = []

        for text in texts:
            try:
                response = self.session.post(
                    f"{self.base_url}/api/embeddings",
                    json={"model": model, "prompt": text},
                )
                response.raise_for_status()

                data = response.json()
                if "data" in data:
                    embeddings.append(data["data"][0]["embedding"])
                else:
                    embeddings.append(data["embedding"])

            except requests.exceptions.RequestException as e:
                logger.error(f"Ошибка при получении эмбеддинга для текста: {e}")
                raise
            except Exception as e:
                logger.error(f"Неожиданная ошибка при получении эмбеддинга: {e}")
                raise

        return embeddings


class QdrantClient:
    """Клиент для работы с Qdrant API."""

    def __init__(self, base_url: str = "http://localhost:6333"):
        self.base_url = base_url
        self.session = requests.Session()
        self.session.timeout = 30

    def create_collection(self, collection_name: str, vector_size: int = 1024) -> bool:
        """Создать коллекцию в Qdrant."""
        try:
            response = self.session.put(
                f"{self.base_url}/collections/{collection_name}",
                json={"vectors": {"size": vector_size, "distance": "Cosine"}},
            )
            response.raise_for_status()
            logger.info(f"Коллекция {collection_name} создана успешно")
            return True
        except requests.exceptions.RequestException as e:
            if e.response and e.response.status_code == 409:
                logger.info(f"Коллекция {collection_name} уже существует")
                return True
            logger.error(f"Ошибка при создании коллекции: {e}")
            return False

    def upsert_points(self, collection_name: str, points: List[Dict[str, Any]]) -> bool:
        """Добавить точки в коллекцию."""
        try:
            response = self.session.put(
                f"{self.base_url}/collections/{collection_name}/points",
                json={"points": points},
            )
            response.raise_for_status()
            logger.info(f"Добавлено {len(points)} точек в коллекцию {collection_name}")
            return True
        except requests.exceptions.RequestException as e:
            logger.error(f"Ошибка при добавлении точек: {e}")
            if e.response:
                logger.error(f"Ответ сервера: {e.response.text}")
                logger.error(f"Статус код: {e.response.status_code}")
            return False


class CodeIndexer:
    """Основной класс для индексации кода."""

    def __init__(
        self,
        ollama_url: str = "http://localhost:11434",
        qdrant_url: str = "http://localhost:6333",
    ):
        self.ollama = OllamaClient(ollama_url)
        self.qdrant = QdrantClient(qdrant_url)

        # Поддерживаемые расширения файлов
        self.supported_extensions = {
            ".py",
            ".go",
            ".js",
            ".ts",
            ".jsx",
            ".tsx",
            ".java",
            ".cpp",
            ".c",
            ".h",
            ".hpp",
            ".cs",
            ".php",
            ".rb",
            ".rs",
            ".swift",
            ".kt",
            ".scala",
            ".sh",
            ".yaml",
            ".yml",
            ".json",
            ".xml",
            ".html",
            ".css",
            ".scss",
            ".sql",
            ".md",
            ".txt",
            ".dockerfile",
            ".makefile",
            ".cmake",
            ".gradle",
        }

        # Игнорируемые директории
        self.ignore_dirs = {
            ".git",
            ".venv",
            "venv",
            "__pycache__",
            "node_modules",
            ".pytest_cache",
            ".mypy_cache",
            "target",
            "build",
            "dist",
            ".idea",
            ".vscode",
            ".cursor",
            "vendor",
            "coverage",
            ".coverage",
            "htmlcov",
            ".tox",
            ".eggs",
        }

    def should_ignore_file(self, file_path: Path) -> bool:
        """Проверить, нужно ли игнорировать файл."""
        # Проверить расширение
        if file_path.suffix.lower() not in self.supported_extensions:
            return True

        # Проверить, находится ли файл в игнорируемой директории
        for part in file_path.parts:
            if part in self.ignore_dirs:
                return True

        # Проверить размер файла (больше 1MB)
        if file_path.stat().st_size > 1024 * 1024:
            return True

        return False

    def read_file_content(self, file_path: Path) -> Optional[str]:
        """Прочитать содержимое файла."""
        try:
            with open(file_path, encoding="utf-8") as f:
                return f.read()
        except UnicodeDecodeError:
            try:
                with open(file_path, encoding="latin-1") as f:
                    return f.read()
            except Exception as e:
                logger.warning(f"Не удалось прочитать файл {file_path}: {e}")
                return None
        except Exception as e:
            logger.warning(f"Ошибка при чтении файла {file_path}: {e}")
            return None

    def get_language_from_extension(self, file_path: Path) -> str:
        """Определить язык программирования по расширению файла."""
        ext = file_path.suffix.lower()
        language_map = {
            ".py": "python",
            ".go": "go",
            ".js": "javascript",
            ".ts": "typescript",
            ".jsx": "javascript",
            ".tsx": "typescript",
            ".java": "java",
            ".cpp": "cpp",
            ".c": "c",
            ".h": "c",
            ".hpp": "cpp",
            ".cs": "csharp",
            ".php": "php",
            ".rb": "ruby",
            ".rs": "rust",
            ".swift": "swift",
            ".kt": "kotlin",
            ".scala": "scala",
            ".sh": "bash",
            ".yaml": "yaml",
            ".yml": "yaml",
            ".json": "json",
            ".xml": "xml",
            ".html": "html",
            ".css": "css",
            ".scss": "scss",
            ".sql": "sql",
            ".md": "markdown",
            ".txt": "text",
            ".dockerfile": "dockerfile",
            ".makefile": "makefile",
            ".cmake": "cmake",
            ".gradle": "gradle",
        }
        return language_map.get(ext, "unknown")

    def scan_directory(self, directory: Path) -> List[CodeFile]:
        """Сканировать директорию и собрать файлы для индексации."""
        files = []

        for file_path in directory.rglob("*"):
            if file_path.is_file() and not self.should_ignore_file(file_path):
                content = self.read_file_content(file_path)
                if content:
                    file_hash = hashlib.md5(content.encode("utf-8")).hexdigest()
                    files.append(
                        CodeFile(
                            path=str(file_path.relative_to(directory)),
                            content=content,
                            language=self.get_language_from_extension(file_path),
                            size=len(content),
                            hash=file_hash,
                        )
                    )

        logger.info(f"Найдено {len(files)} файлов для индексации")
        return files

    def create_chunks(
        self, file: CodeFile, chunk_size: int = 1000
    ) -> List[Dict[str, Any]]:
        """Разбить файл на чанки для индексации."""
        chunks = []
        content = file.content

        # Разбиваем на строки и группируем в чанки
        lines = content.split("\n")
        current_chunk = []
        current_size = 0

        for i, line in enumerate(lines):
            current_chunk.append(line)
            current_size += len(line) + 1  # +1 для символа новой строки

            if current_size >= chunk_size or i == len(lines) - 1:
                chunk_content = "\n".join(current_chunk)
                chunk_id = f"{file.hash}_{len(chunks)}"

                chunks.append(
                    {
                        "id": chunk_id,
                        "content": chunk_content,
                        "metadata": {
                            "file_path": file.path,
                            "language": file.language,
                            "chunk_index": len(chunks),
                            "total_chunks": 0,  # Будет обновлено позже
                            "file_size": file.size,
                            "file_hash": file.hash,
                        },
                    }
                )

                current_chunk = []
                current_size = 0

        # Обновляем total_chunks для всех чанков
        for chunk in chunks:
            chunk["metadata"]["total_chunks"] = len(chunks)

        return chunks

    def index_files(
        self,
        files: List[CodeFile],
        collection_name: str,
        model: str,
        batch_size: int = 10,
    ) -> bool:
        """Индексировать файлы в Qdrant."""
        try:
            # Создаем коллекцию
            if not self.qdrant.create_collection(collection_name):
                return False

            # Создаем чанки из всех файлов
            all_chunks = []
            for file in files:
                chunks = self.create_chunks(file)
                all_chunks.extend(chunks)

            logger.info(f"Создано {len(all_chunks)} чанков для индексации")

            # Обрабатываем чанки батчами
            for i in range(0, len(all_chunks), batch_size):
                batch = all_chunks[i : i + batch_size]
                logger.info(
                    f"Обрабатываем батч {i//batch_size + 1}/{(len(all_chunks) + batch_size - 1)//batch_size}"
                )

                # Получаем эмбеддинги для батча
                texts = [chunk["content"] for chunk in batch]
                embeddings = self.ollama.get_embeddings(texts, model)

                # Создаем точки для Qdrant
                points = []
                for j, (chunk, embedding) in enumerate(zip(batch, embeddings)):
                    # Создаем payload без content
                    payload = chunk["metadata"].copy()
                    payload["content"] = chunk["content"]

                    points.append(
                        {"id": chunk["id"], "vector": embedding, "payload": payload}
                    )

                # Отладочная информация для первого батча
                if i == 0:
                    logger.info(
                        f"Пример точки: {json.dumps(points[0], indent=2, ensure_ascii=False)}"
                    )

                # Добавляем точки в Qdrant
                if not self.qdrant.upsert_points(collection_name, points):
                    logger.error(f"Ошибка при добавлении батча {i//batch_size + 1}")
                    return False

                # Небольшая пауза между батчами
                time.sleep(0.1)

            logger.info(
                f"Индексация завершена успешно. Добавлено {len(all_chunks)} чанков"
            )
            return True

        except Exception as e:
            logger.error(f"Ошибка при индексации: {e}")
            return False


def main():
    parser = argparse.ArgumentParser(
        description="Индексация кода с помощью Ollama и Qdrant"
    )
    parser.add_argument(
        "--collection", default="alert-history-code", help="Название коллекции в Qdrant"
    )
    parser.add_argument(
        "--model",
        default="mxbai-embed-large:latest",
        help="Модель для эмбеддингов в Ollama",
    )
    parser.add_argument("--directory", default=".", help="Директория для индексации")
    parser.add_argument(
        "--batch-size", type=int, default=10, help="Размер батча для обработки"
    )
    parser.add_argument(
        "--chunk-size", type=int, default=1000, help="Размер чанка в символах"
    )
    parser.add_argument(
        "--ollama-url", default="http://localhost:11434", help="URL Ollama сервера"
    )
    parser.add_argument(
        "--qdrant-url", default="http://localhost:6333", help="URL Qdrant сервера"
    )

    args = parser.parse_args()

    # Проверяем доступность сервисов
    try:
        ollama_client = OllamaClient(args.ollama_url)
        qdrant_client = QdrantClient(args.qdrant_url)

        # Тестируем подключение к Ollama
        logger.info("Проверяем подключение к Ollama...")
        response = requests.get(f"{args.ollama_url}/api/tags", timeout=5)
        response.raise_for_status()
        logger.info("Ollama доступен")

        # Тестируем подключение к Qdrant
        logger.info("Проверяем подключение к Qdrant...")
        response = requests.get(f"{args.qdrant_url}/collections", timeout=5)
        response.raise_for_status()
        logger.info("Qdrant доступен")

    except Exception as e:
        logger.error(f"Ошибка подключения к сервисам: {e}")
        sys.exit(1)

    # Создаем индексатор
    indexer = CodeIndexer(args.ollama_url, args.qdrant_url)

    # Сканируем директорию
    directory = Path(args.directory)
    if not directory.exists():
        logger.error(f"Директория {directory} не существует")
        sys.exit(1)

    logger.info(f"Сканируем директорию {directory}")
    files = indexer.scan_directory(directory)

    if not files:
        logger.warning("Не найдено файлов для индексации")
        sys.exit(0)

    # Индексируем файлы
    logger.info(
        f"Начинаем индексацию {len(files)} файлов в коллекцию {args.collection}"
    )
    success = indexer.index_files(files, args.collection, args.model, args.batch_size)

    if success:
        logger.info("Индексация завершена успешно!")
        sys.exit(0)
    else:
        logger.error("Индексация завершилась с ошибками")
        sys.exit(1)


if __name__ == "__main__":
    main()
