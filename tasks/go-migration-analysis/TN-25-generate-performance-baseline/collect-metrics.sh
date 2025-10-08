#!/bin/bash
set -euo pipefail

echo "=== Collecting Performance Metrics ==="

# Проверяем, что Go приложение запущено
if ! curl -s http://localhost:8080/health > /dev/null; then
    echo "Go application is not running on port 8080"
    exit 1
fi

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULTS_DIR="results"
PROFILES_DIR="profiles"

# Создаем директории если их нет
mkdir -p "$RESULTS_DIR" "$PROFILES_DIR"

echo "=== Running History API Load Test ==="
k6 run --out json="$RESULTS_DIR/history_test_$TIMESTAMP.json" k6-history-test.js

echo "=== Collecting CPU Profile ==="
curl -s "http://localhost:8080/debug/pprof/profile?seconds=30" > "$PROFILES_DIR/cpu_profile_$TIMESTAMP.prof"

echo "=== Collecting Memory Profile ==="
curl -s "http://localhost:8080/debug/pprof/heap" > "$PROFILES_DIR/memory_profile_$TIMESTAMP.prof"

echo "=== Collecting Goroutine Profile ==="
curl -s "http://localhost:8080/debug/pprof/goroutine" > "$PROFILES_DIR/goroutine_profile_$TIMESTAMP.prof"

echo "=== Performance metrics collection completed ==="
echo "Results saved to: $RESULTS_DIR/"
echo "Profiles saved to: $PROFILES_DIR/"

# Показываем размеры файлов
echo "=== File sizes ==="
ls -lh "$RESULTS_DIR"/ "$PROFILES_DIR"/
