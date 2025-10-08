# Makefile для утилиты индексации кода

.PHONY: help install index search clean test

# Переменные
COLLECTION ?= alert-history-code
MODEL ?= mxbai-embed-large:latest
DIRECTORY ?= ../..
BATCH_SIZE ?= 10
CHUNK_SIZE ?= 1000
OLLAMA_URL ?= http://localhost:11434
QDRANT_URL ?= http://localhost:6333

help: ## Показать справку
	@echo "Доступные команды:"
	@echo "  install     - Установить зависимости"
	@echo "  index       - Индексировать код"
	@echo "  search      - Поиск в индексированном коде"
	@echo "  clean       - Очистить коллекцию"
	@echo "  test        - Тестировать подключения"
	@echo ""
	@echo "Переменные:"
	@echo "  COLLECTION  - Название коллекции (по умолчанию: $(COLLECTION))"
	@echo "  MODEL       - Модель Ollama (по умолчанию: $(MODEL))"
	@echo "  DIRECTORY   - Директория для индексации (по умолчанию: $(DIRECTORY))"
	@echo "  BATCH_SIZE  - Размер батча (по умолчанию: $(BATCH_SIZE))"
	@echo "  CHUNK_SIZE  - Размер чанка (по умолчанию: $(CHUNK_SIZE))"
	@echo "  OLLAMA_URL  - URL Ollama (по умолчанию: $(OLLAMA_URL))"
	@echo "  QDRANT_URL  - URL Qdrant (по умолчанию: $(QDRANT_URL))"

install: ## Установить зависимости
	pip install -r requirements.txt

index: ## Индексировать код
	python index_code.py \
		--collection $(COLLECTION) \
		--model $(MODEL) \
		--directory $(DIRECTORY) \
		--batch-size $(BATCH_SIZE) \
		--chunk-size $(CHUNK_SIZE) \
		--ollama-url $(OLLAMA_URL) \
		--qdrant-url $(QDRANT_URL)

search: ## Поиск в индексированном коде (использование: make search QUERY="ваш запрос")
	@if [ -z "$(QUERY)" ]; then \
		echo "Ошибка: укажите QUERY для поиска"; \
		echo "Использование: make search QUERY=\"ваш запрос\""; \
		exit 1; \
	fi
	python search_code.py "$(QUERY)" \
		--collection $(COLLECTION) \
		--model $(MODEL) \
		--ollama-url $(OLLAMA_URL) \
		--qdrant-url $(QDRANT_URL)

clean: ## Очистить коллекцию
	@echo "Удаляем коллекцию $(COLLECTION)..."
	curl -X DELETE $(QDRANT_URL)/collections/$(COLLECTION) || echo "Коллекция не найдена"

test: ## Тестировать подключения
	@echo "Тестируем подключение к Ollama..."
	curl -s $(OLLAMA_URL)/api/tags | jq '.models[].name' || echo "Ollama недоступен"
	@echo ""
	@echo "Тестируем подключение к Qdrant..."
	curl -s $(QDRANT_URL)/collections | jq '.result.collections[].name' || echo "Qdrant недоступен"

# Примеры использования
index-go: ## Индексировать только Go код
	$(MAKE) index DIRECTORY=../../go-app COLLECTION=alert-history-go

index-python: ## Индексировать только Python код
	$(MAKE) index DIRECTORY=../../src COLLECTION=alert-history-python

search-go: ## Поиск в Go коде
	$(MAKE) search COLLECTION=alert-history-go QUERY="$(QUERY)"

search-python: ## Поиск в Python коде
	$(MAKE) search COLLECTION=alert-history-python QUERY="$(QUERY)"
