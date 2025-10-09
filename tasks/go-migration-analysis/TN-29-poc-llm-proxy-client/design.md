# TN-29: Дизайн LLM Client

## Интерфейс
```go
type LLMClient interface {
    ClassifyAlert(ctx context.Context, alert *Alert) (*Classification, error)
}

type HTTPLLMClient struct {
    baseURL    string
    httpClient *http.Client
    apiKey     string
}

func (c *HTTPLLMClient) ClassifyAlert(ctx context.Context, alert *Alert) (*Classification, error) {
    payload := map[string]interface{}{
        "alert": alert,
        "model": "gpt-4",
    }

    body, _ := json.Marshal(payload)
    req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/classify", bytes.NewBuffer(body))
    req.Header.Set("Authorization", "Bearer "+c.apiKey)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result Classification
    json.NewDecoder(resp.Body).Decode(&result)
    return &result, nil
}
```
