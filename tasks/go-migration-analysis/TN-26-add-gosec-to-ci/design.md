# TN-26: GoSec интеграция

## GitHub Actions
```yaml
- name: Run GoSec
  uses: securego/gosec@master
  with:
    args: '-fmt sarif -out gosec.sarif ./...'

- name: Upload SARIF
  uses: github/codeql-action/upload-sarif@v2
  with:
    sarif_file: gosec.sarif
```

## Конфигурация
```yaml
# .gosec.yaml
global:
  nosec: false
  audit: true

rules:
  - G101 # Hardcoded credentials
  - G201 # SQL injection
  - G401 # Weak crypto
```
