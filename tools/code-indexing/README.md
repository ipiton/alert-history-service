# Утилита индексации кода

Этот инструмент позволяет индексировать код проекта с помощью Ollama и Qdrant для семантического поиска.

## Требования

- Python 3.9+
- Ollama сервер с моделью `mxbai-embed-large:latest`
- Qdrant сервер

## Установка

```bash
cd tools/code-indexing
pip install -r requirements.txt
```

## Использование

### Базовое использование

```bash
python index_code.py
```

### Расширенные параметры

```bash
python index_code.py \
    --collection alert-history-code \
    --model mxbai-embed-large:latest \
    --directory /path/to/project \
    --batch-size 10 \
    --chunk-size 1000 \
    --ollama-url http://localhost:11434 \
    --qdrant-url http://localhost:6333
```

## Параметры

- `--collection` - название коллекции в Qdrant (по умолчанию: alert-history-code)
- `--model` - модель для эмбеддингов в Ollama (по умолчанию: mxbai-embed-large:latest)
- `--directory` - директория для индексации (по умолчанию: текущая)
- `--batch-size` - размер батча для обработки (по умолчанию: 10)
- `--chunk-size` - размер чанка в символах (по умолчанию: 1000)
- `--ollama-url` - URL Ollama сервера (по умолчанию: http://localhost:11434)
- `--qdrant-url` - URL Qdrant сервера (по умолчанию: http://localhost:6333)

## Поддерживаемые форматы файлов

- Python (.py)
- Go (.go)
- JavaScript (.js, .jsx)
- TypeScript (.ts, .tsx)
- Java (.java)
- C/C++ (.c, .cpp, .h, .hpp)
- C# (.cs)
- PHP (.php)
- Ruby (.rb)
- Rust (.rs)
- Swift (.swift)
- Kotlin (.kt)
- Scala (.scala)
- Bash (.sh)
- YAML (.yaml, .yml)
- JSON (.json)
- XML (.xml)
- HTML (.html)
- CSS (.css, .scss)
- SQL (.sql)
- Markdown (.md)
- Text (.txt)
- Dockerfile
- Makefile
- CMake (.cmake)
- Gradle (.gradle)

## Игнорируемые директории

- .git
- .venv, venv
- __pycache__
- node_modules
- .pytest_cache
- .mypy_cache
- target
- build
- dist
- .idea
- .vscode
- .cursor
- vendor
- coverage
- .coverage
- htmlcov
- .tox
- .eggs

## Примеры использования

### Индексация всего проекта

```bash
python index_code.py --collection my-project-code
```

### Индексация только Go кода

```bash
python index_code.py --collection go-code --directory go-app
```

### Индексация с большими чанками

```bash
python index_code.py --chunk-size 2000 --batch-size 5
```

## Устранение неполадок

### Ошибка "Bad Request"

1. Проверьте, что Ollama сервер запущен:
   ```bash
   curl http://localhost:11434/api/tags
   ```

2. Проверьте, что модель загружена:
   ```bash
   curl http://localhost:11434/api/tags | jq '.models[].name'
   ```

3. Проверьте, что Qdrant сервер запущен:
   ```bash
   curl http://localhost:6333/collections
   ```

### Ошибка подключения

- Убедитесь, что порты 11434 (Ollama) и 6333 (Qdrant) доступны
- Проверьте настройки файрвола
- Убедитесь, что сервисы запущены

### Ошибка памяти

- Уменьшите размер батча: `--batch-size 5`
- Уменьшите размер чанка: `--chunk-size 500`
- Исключите большие файлы из индексации
