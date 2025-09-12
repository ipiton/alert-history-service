# TN-30: Coverage Setup

## Makefile targets
```makefile
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	go tool cover -func=coverage.out

coverage-badge:
	go-acc ./... -o coverage.out
	goveralls -coverprofile=coverage.out -service=github
```

## GitHub Actions
```yaml
- name: Test with Coverage
  run: |
    go test -v -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out

- name: Upload Coverage
  uses: codecov/codecov-action@v3
  with:
    file: ./coverage.out

- name: Check Coverage Threshold
  run: |
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    if (( $(echo "$COVERAGE < 80" | bc -l) )); then
      echo "Coverage is below 80%"
      exit 1
    fi
```
