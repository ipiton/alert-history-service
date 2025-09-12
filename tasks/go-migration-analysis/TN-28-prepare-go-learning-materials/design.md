# TN-28: Структура материалов

## docs/go-for-python-devs.md
1. **Основные отличия**
   - Статическая типизация
   - Компиляция
   - Goroutines vs async/await
   - Error handling

2. **Синтаксис**
   ```python
   # Python
   def process_alert(alert):
       try:
           result = classify(alert)
           return result
       except Exception as e:
           logger.error(f"Error: {e}")
   ```

   ```go
   // Go
   func processAlert(alert Alert) (*Result, error) {
       result, err := classify(alert)
       if err != nil {
           logger.Error("Error", "error", err)
           return nil, err
       }
       return result, nil
   }
   ```

3. **Инструменты**
   - go mod (vs pip)
   - go test (vs pytest)
   - go fmt (vs black)
