# TN-046: Kubernetes Client Design

```go
type KubernetesClient interface {
    ListSecrets(ctx context.Context, namespace string, labelSelector string) (*v1.SecretList, error)
    WatchSecrets(ctx context.Context, namespace string, labelSelector string) (watch.Interface, error)
    GetSecret(ctx context.Context, namespace, name string) (*v1.Secret, error)
}

type kubernetesClient struct {
    clientset kubernetes.Interface
    config    *rest.Config
    logger    *slog.Logger
}

func NewKubernetesClient(kubeconfig string) (KubernetesClient, error) {
    config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
    if err != nil {
        // Try in-cluster config
        config, err = rest.InClusterConfig()
        if err != nil {
            return nil, err
        }
    }

    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        return nil, err
    }

    return &kubernetesClient{
        clientset: clientset,
        config:    config,
        logger:    slog.Default(),
    }, nil
}
```
