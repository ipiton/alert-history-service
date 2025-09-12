# TN-22: Дизайн Graceful Shutdown

## Реализация
```go
func main() {
    app := fiber.New()
    // ... setup routes

    // Start server
    go func() {
        if err := app.Listen(":8080"); err != nil {
            log.Fatal(err)
        }
    }()

    // Wait for interrupt
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("Shutting down server...")

    // Graceful shutdown with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := app.ShutdownWithContext(ctx); err != nil {
        log.Fatal("Server forced to shutdown:", err)
    }

    // Close resources
    db.Close()
    redis.Close()

    log.Println("Server shutdown complete")
}
```
