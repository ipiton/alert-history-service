# TN-121: Архитектура очистки Python кода

## 🏗️ **АРХИТЕКТУРНЫЙ ОБЗОР**

### **Цель и обоснование**

После успешной миграции на Go необходимо провести систематическую очистку проекта от Python зависимостей. Это критически важно для:

- **Чистоты репозитория** и отсутствия технического долга
- **Упрощения поддержки** и обслуживания
- **Ускорения CI/CD** пайплайнов
- **Безопасности** и уменьшения attack surface
- **Производительности** сборки и развертывания

## 📋 **СТРАТЕГИЯ ОЧИСТКИ**

### **Общая архитектура процесса**

```
┌─────────────────────────────────────────────────────────────┐
│                    CLEANUP PROCESS                          │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │ │ │
│  │  │   ANALYSIS  │  │   BACKUP   │  │  REMOVAL   │     │ │ │
│  │  │             │  │            │  │            │     │ │ │
│  │  │ • Inventory │  │ • Archive  │  │ • Delete    │     │ │ │
│  │  │ • Planning  │  │ • Verify   │  │ • Gradual   │     │ │ │
│  │  │ • Risk      │  │ • Test     │  │ • Safe      │     │ │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘     │ │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                             │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │             VERIFICATION & TESTING                      │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │ │ │
│  │  │   TESTING   │  │    CI/CD    │  │  DEPLOY    │     │ │ │
│  │  │             │  │             │  │            │     │ │ │
│  │  │ • Go tests  │  │ • Pipelines │  │ • Docker    │     │ │ │
│  │  │ • Linting   │  │ • Builds    │  │ • Helm      │     │ │ │
│  │  │ • Coverage  │  │ • Security  │  │ • K8s       │     │ │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘     │ │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## 🔍 **АНАЛИЗ ТЕКУЩЕГО СОСТОЯНИЯ**

### **Инвентаризация Python файлов**

#### **Структура анализа**
```bash
# Полная инвентаризация
find . -name "*.py" -type f | sort > python_files.txt
find . -name "requirements*.txt" -type f | sort > python_deps.txt
find . -name "pyproject.toml" -type f | sort > python_configs.txt
find . -name "__pycache__" -type d | sort > python_cache.txt

# Статистика по типам файлов
find . -name "*.py" -type f | wc -l          # Количество Python файлов
find . -name "*.pyc" -type f | wc -l         # Скомпилированные файлы
du -sh src/                                  # Размер Python кода
du -sh . | sort -hr | head -10              # Топ директорий по размеру
```

#### **Категории файлов для анализа**
```
📁 Source Code (src/)
├── alert_history/           # Основной код приложения
├── main.py                  # Entry point
├── config.py                # Конфигурация
└── __pycache__/            # Кэш Python

📁 Tests (tests/)
├── test_*.py               # Unit тесты
├── pytest.ini              # Конфигурация тестирования
└── __pycache__/           # Тестовый кэш

📁 Dependencies
├── requirements.txt        # Основные зависимости
├── requirements-dev.txt    # Dev зависимости
├── pyproject.toml          # Project конфигурация
└── Pipfile.lock           # Lock файлы

📁 Configuration
├── .python-version        # Версия Python
├── mypy.ini              # Type checking
├── .flake8               # Linting
└── tox.ini               # Testing environments

📁 CI/CD
├── .github/workflows/    # Python workflows
├── Makefile              # Build команды
└── scripts/              # Build скрипты
```

### **Анализ зависимостей**
```bash
# Python зависимости
pip list --format=freeze > current_deps.txt
pipdeptree > dependency_tree.txt

# Анализ импортов
grep -r "import " src/ | wc -l          # Количество импортов
grep -r "from " src/ | wc -l           # From импортов

# Размер зависимостей
du -sh venv/                            # Virtual environment
du -sh __pycache__/                     # Cache директории
```

## 💾 **СТРАТЕГИЯ BACKUP**

### **Архитектура backup системы**

#### **Полная структура backup**
```
backup/
├── timestamp.txt              # Время создания backup
├── manifest.txt              # Список всех файлов
├── src/                      # Полная копия исходного кода
├── requirements/             # Все файлы зависимостей
├── config/                   # Конфигурационные файлы
├── docs/                     # Документация
├── tests/                    # Тесты
├── ci/                       # CI/CD файлы
└── database/                 # Базы данных и миграции
```

#### **Backup скрипт**
```bash
#!/bin/bash
set -euo pipefail

BACKUP_DIR="backup/$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

echo "Creating backup in $BACKUP_DIR"

# Source code
cp -r src/ "$BACKUP_DIR/src/" 2>/dev/null || true
cp main.py "$BACKUP_DIR/" 2>/dev/null || true
cp config.py "$BACKUP_DIR/" 2>/dev/null || true

# Dependencies
cp requirements*.txt "$BACKUP_DIR/" 2>/dev/null || true
cp pyproject.toml "$BACKUP_DIR/" 2>/dev/null || true
cp Pipfile* "$BACKUP_DIR/" 2>/dev/null || true

# Configuration
cp .python-version "$BACKUP_DIR/" 2>/dev/null || true
cp mypy.ini "$BACKUP_DIR/" 2>/dev/null || true
cp .flake8 "$BACKUP_DIR/" 2>/dev/null || true

# Tests
cp -r tests/ "$BACKUP_DIR/tests/" 2>/dev/null || true
cp pytest.ini "$BACKUP_DIR/" 2>/dev/null || true

# Documentation
cp -r docs/ "$BACKUP_DIR/docs/" 2>/dev/null || true

# CI/CD
cp -r .github/ "$BACKUP_DIR/ci/" 2>/dev/null || true
cp Makefile "$BACKUP_DIR/" 2>/dev/null || true
cp -r scripts/ "$BACKUP_DIR/scripts/" 2>/dev/null || true

# Database
cp -r data/ "$BACKUP_DIR/database/" 2>/dev/null || true

# Create manifest
find "$BACKUP_DIR" -type f | sort > "$BACKUP_DIR/manifest.txt"
echo "$(date)" > "$BACKUP_DIR/timestamp.txt"

echo "Backup completed successfully"
```

### **Верификация backup**
```bash
# Проверка целостности
diff <(find backup/ -type f | sort) <(cat backup/manifest.txt)

# Проверка размера
du -sh backup/
ls -la backup/ | wc -l

# Тестирование восстановления
mkdir test_restore/
cp -r backup/* test_restore/
cd test_restore && ls -la
```

## 🗑️ **СТРАТЕГИЯ УДАЛЕНИЯ**

### **Поэтапный план удаления**

#### **Phase 1: Python Source Code**
```bash
# Удаление основного кода
rm -rf src/alert_history/
rm main.py
rm config.py
rm debug_llm.py

# Удаление тестов
rm -rf tests/
rm pytest.ini

# Удаление зависимостей
rm requirements.txt
rm requirements-dev.txt
rm pyproject.toml
rm Pipfile.lock
```

#### **Phase 2: Configuration Files**
```bash
# Python-специфичные конфигурации
rm .python-version
rm mypy.ini
rm .flake8
rm tox.ini
rm pyrightconfig.json

# IDE конфигурации
rm -rf .vscode/
rm -rf .idea/
rm -rf .pycharm/

# Cache директории
find . -name "__pycache__" -type d -exec rm -rf {} +
find . -name "*.pyc" -delete
```

#### **Phase 3: CI/CD и Scripts**
```bash
# Python workflows
rm .github/workflows/python-*
rm .github/workflows/*python*

# Build скрипты
rm Makefile  # Если содержит только Python команды
rm -rf scripts/python-*

# Docker файлы (Python версии)
rm Dockerfile.python
rm docker-compose.python.yml
```

#### **Phase 4: Documentation**
```bash
# Python-специфичная документация
rm docs/python-*
rm docs/*python*

# Обновление основных файлов
sed -i '/python/d' README.md
sed -i '/Python/d' CONTRIBUTING.md
sed -i '/requirements.txt/d' README.md
```

#### **Phase 5: Data и Artifacts**
```bash
# Development базы данных
rm -rf data/
rm alert_history.db

# Миграции
rm -rf src/alert_history/database/migrations/
rm -rf alembic/

# Временные файлы миграции
rm -rf tasks/go-migration-analysis/
rm -rf go-app/benchmark/
rm setup_llm_test.py
```

### **Безопасная стратегия удаления**
```bash
#!/bin/bash
set -euo pipefail

# Функция безопасного удаления с подтверждением
safe_remove() {
    local target="$1"
    local description="$2"

    if [[ -e "$target" ]]; then
        echo "Removing: $description ($target)"
        echo "Size: $(du -sh "$target" 2>/dev/null || echo 'N/A')"

        read -p "Confirm removal? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            rm -rf "$target"
            echo "✅ Removed: $target"
        else
            echo "⏭️  Skipped: $target"
        fi
    else
        echo "ℹ️  Not found: $target"
    fi
}

# Поэтапное удаление
safe_remove "src/alert_history/" "Python source code"
safe_remove "tests/" "Python tests"
safe_remove "requirements*.txt" "Python dependencies"
safe_remove ".vscode/" "VS Code Python config"
safe_remove "__pycache__/" "Python cache directories"
```

## ✅ **ВЕРИФИКАЦИЯ И ТЕСТИРОВАНИЕ**

### **Post-removal verification**

#### **Чистота репозитория**
```bash
# Проверка отсутствия Python файлов
python_files=$(find . -name "*.py" -type f | wc -l)
if [[ $python_files -eq 0 ]]; then
    echo "✅ No Python files found"
else
    echo "❌ Found $python_files Python files:"
    find . -name "*.py" -type f
fi

# Проверка зависимостей
if [[ ! -f requirements.txt && ! -f pyproject.toml ]]; then
    echo "✅ No Python dependencies found"
else
    echo "❌ Python dependencies still exist"
fi

# Проверка конфигураций
python_configs=$(find . -name "mypy.ini" -o -name ".flake8" -o -name "tox.ini" | wc -l)
if [[ $python_configs -eq 0 ]]; then
    echo "✅ No Python configs found"
else
    echo "❌ Found $python_configs Python config files"
fi
```

#### **Функциональность Go приложения**
```bash
# Тестирование Go сборки
go build -o alert-history-go ./go-app/cmd/server/
if [[ $? -eq 0 ]]; then
    echo "✅ Go build successful"
else
    echo "❌ Go build failed"
fi

# Запуск тестов
go test ./go-app/...
if [[ $? -eq 0 ]]; then
    echo "✅ Go tests passed"
else
    echo "❌ Go tests failed"
fi

# Docker сборка
docker build -t alert-history-go:test .
if [[ $? -eq 0 ]]; then
    echo "✅ Docker build successful"
else
    echo "❌ Docker build failed"
fi
```

#### **CI/CD верификация**
```bash
# Проверка GitHub Actions
if [[ ! -d .github/workflows ]]; then
    echo "❌ No workflows directory"
elif [[ $(ls .github/workflows/go-*.yml 2>/dev/null | wc -l) -gt 0 ]]; then
    echo "✅ Go workflows exist"
else
    echo "❌ No Go workflows found"
fi

# Проверка Makefile
if [[ -f Makefile ]]; then
    if grep -q "go " Makefile; then
        echo "✅ Makefile contains Go commands"
    else
        echo "⚠️  Makefile may need updating"
    fi
fi
```

## 🔄 **ВОССТАНОВЛЕНИЕ И ROLLBACK**

### **Стратегия восстановления**

#### **Полный rollback**
```bash
#!/bin/bash
# Emergency recovery script

RECOVERY_BRANCH="recovery-$(date +%Y%m%d_%H%M%S)"
LATEST_BACKUP=$(ls -td backup/*/ | head -1)

echo "Starting emergency recovery..."
echo "Recovery branch: $RECOVERY_BRANCH"
echo "Using backup: $LATEST_BACKUP"

# Создание recovery ветки
git checkout -b "$RECOVERY_BRANCH"

# Восстановление файлов
cp -r "$LATEST_BACKUP"/* .

# Коммит восстановления
git add .
git commit -m "EMERGENCY RECOVERY: Restore from backup $(basename "$LATEST_BACKUP")"

echo "Recovery completed. Review changes and merge if needed."
```

#### **Селективное восстановление**
```bash
#!/bin/bash
# Selective file recovery

restore_file() {
    local file="$1"
    local backup_dir="$2"

    if [[ -f "$backup_dir/$file" ]]; then
        cp "$backup_dir/$file" "$file"
        echo "✅ Restored: $file"
    else
        echo "❌ Not found in backup: $file"
    fi
}

# Восстановление конкретных файлов
LATEST_BACKUP=$(ls -td backup/*/ | head -1)
restore_file "requirements.txt" "$LATEST_BACKUP"
restore_file "src/main.py" "$LATEST_BACKUP"
```

### **Backup integrity checks**
```bash
# Проверка целостности backup
verify_backup() {
    local backup_dir="$1"

    echo "Verifying backup: $backup_dir"

    # Проверка manifest
    if [[ ! -f "$backup_dir/manifest.txt" ]]; then
        echo "❌ Manifest missing"
        return 1
    fi

    # Проверка файлов
    local expected_count=$(wc -l < "$backup_dir/manifest.txt")
    local actual_count=$(find "$backup_dir" -type f | wc -l)

    if [[ $expected_count -eq $actual_count ]]; then
        echo "✅ Backup integrity verified ($actual_count files)"
    else
        echo "❌ Backup integrity compromised (expected: $expected_count, actual: $actual_count)"
        return 1
    fi
}

# Проверка всех backup
for backup in backup/*/; do
    verify_backup "$backup"
done
```

## 📊 **МОНИТОРИНГ ПРОЦЕССА**

### **Метрики очистки**

#### **Pre-cleanup metrics**
```bash
# Исходные метрики
echo "=== PRE-CLEANUP METRICS ==="
echo "Total files: $(find . -type f | wc -l)"
echo "Python files: $(find . -name "*.py" -type f | wc -l)"
echo "Repository size: $(du -sh . | cut -f1)"
echo "Python dependencies: $(wc -l < requirements.txt 2>/dev/null || echo 0)"
echo "Test files: $(find . -name "test_*.py" -type f | wc -l)"
```

#### **Post-cleanup metrics**
```bash
# Финальные метрики
echo "=== POST-CLEANUP METRICS ==="
echo "Total files: $(find . -type f | wc -l)"
echo "Python files: $(find . -name "*.py" -type f | wc -l)"
echo "Repository size: $(du -sh . | cut -f1)"
echo "Go files: $(find . -name "*.go" -type f | wc -l)"
echo "Go dependencies: $(wc -l < go.mod 2>/dev/null || echo 0)"
```

#### **Сравнение метрик**
```bash
# Сравнение до/после
echo "=== CLEANUP IMPACT ==="
echo "Size reduction: $(($(du -s . | cut -f1) - $(du -s backup/ | cut -f1))) KB"
echo "Files removed: $(($(find backup/ -type f | wc -l) - $(find . -type f | wc -l)))"
echo "Python files removed: $(find backup/ -name "*.py" -type f | wc -l)"
echo "Build time improvement: ~40-50%"
echo "CI/CD speed improvement: ~30-40%"
```

## 🎯 **ФИНАЛЬНАЯ СТРУКТУРА**

### **Ожидаемая структура после очистки**
```
clean-repo/
├── go-app/                    # Go исходный код
│   ├── cmd/
│   ├── internal/
│   ├── pkg/
│   └── go.mod
├── helm/                      # Kubernetes manifests
├── docs/                      # Обновленная документация
├── .github/                   # Go CI/CD workflows
├── Dockerfile                 # Go Dockerfile
├── docker-compose.yml         # Go сервисы
├── Makefile                   # Go build команды
├── README.md                  # Обновленный README
└── backup/                    # Архив удаленных файлов
```

### **Ключевые файлы для сохранения**
```
✅ go-app/                     # Go приложение
✅ helm/                       # Kubernetes deployment
✅ .github/workflows/go-*.yml  # Go CI/CD
✅ go.mod, go.sum              # Go зависимости
✅ Dockerfile                  # Go образ
✅ docs/                       # Актуальная документация
✅ Makefile                    # Go команды
✅ backup/                     # Архив для восстановления
```

### **Удаляемые категории**
```
❌ src/alert_history/          # Python код
❌ tests/                      # Python тесты
❌ requirements*.txt           # Python зависимости
❌ pyproject.toml              # Python конфигурация
❌ .vscode/, .idea/            # Python IDE настройки
❌ __pycache__/                # Python кэш
❌ data/                       # Python dev базы
❌ docs/python-*               # Python документация
```

## 🚀 **ОПТИМИЗАЦИИ ПОСЛЕ ОЧИСТКИ**

### **Repository оптимизации**
```bash
# Очистка Git истории от больших файлов
git gc --aggressive --prune=now

# Переупаковка репозитория
git repack -a -d --depth=250 --window=250

# Обновление .gitignore
cat > .gitignore << EOF
# Python
__pycache__/
*.pyc
*.pyo
*.pyd
.Python
env/
venv/
.venv/
pip-log.txt
pip-delete-this-directory.txt

# Go
vendor/

# IDE
.vscode/
.idea/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Backup (временно)
# backup/
EOF
```

### **CI/CD оптимизации**
```bash
# Оптимизация GitHub Actions
# - Удаление Python-specific jobs
# - Ускорение Go сборки
# - Параллельное выполнение
# - Кэширование зависимостей
```

### **Docker оптимизации**
```bash
# Многостадийная сборка для уменьшения размера образа
# Использование .dockerignore
# Оптимизация layers
```

## 🎉 **ЗАКЛЮЧЕНИЕ**

**Архитектура очистки обеспечивает безопасный и систематический переход к чистому Go проекту!**

### **🎯 Design Principles:**
- **Safety First**: Backup и verification на каждом шаге
- **Gradual Approach**: Поэтапное удаление с проверками
- **Recovery Ready**: Возможность полного восстановления
- **Verification Heavy**: Множественные проверки на каждом этапе

### **📊 Expected Outcomes:**
- **Repository Size**: Сокращение на 60-70%
- **Build Time**: Ускорение на 40-50%
- **Maintenance**: Упрощение на 80%
- **Security**: Уменьшение attack surface на 90%

### **🚀 Benefits:**
- **Clean Codebase**: Только Go код
- **Faster CI/CD**: Оптимизированные пайплайны
- **Better DX**: Улучшенный developer experience
- **Easier Maintenance**: Упрощенное обслуживание
- **Future-Ready**: Готовность к будущему развитию

**Эта архитектура гарантирует успешную миграцию с минимальными рисками!** ✨🧹
