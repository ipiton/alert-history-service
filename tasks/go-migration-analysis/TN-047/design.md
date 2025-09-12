# TN-047: Target Discovery Design

```go
type TargetDiscoveryManager interface {
    Start(ctx context.Context) error
    Stop() error
    GetTargets() []*domain.PublishingTarget
    RefreshTargets(ctx context.Context) error
    GetStats() *DiscoveryStats
}

type targetDiscoveryManager struct {
    k8sClient     KubernetesClient
    namespace     string
    labelSelector string
    targets       map[string]*domain.PublishingTarget
    mutex         sync.RWMutex
    logger        *slog.Logger
    metrics       *prometheus.GaugeVec
}

func (m *targetDiscoveryManager) Start(ctx context.Context) error {
    // Initial discovery
    if err := m.RefreshTargets(ctx); err != nil {
        return err
    }

    // Start watching for changes
    watcher, err := m.k8sClient.WatchSecrets(ctx, m.namespace, m.labelSelector)
    if err != nil {
        return err
    }

    go m.handleSecretEvents(ctx, watcher)

    return nil
}

func (m *targetDiscoveryManager) handleSecretEvents(ctx context.Context, watcher watch.Interface) {
    defer watcher.Stop()

    for event := range watcher.ResultChan() {
        secret, ok := event.Object.(*v1.Secret)
        if !ok {
            continue
        }

        switch event.Type {
        case watch.Added, watch.Modified:
            target := m.convertSecretToTarget(secret)
            if target != nil {
                m.addOrUpdateTarget(target)
            }
        case watch.Deleted:
            m.removeTarget(secret.Name)
        }
    }
}
```
