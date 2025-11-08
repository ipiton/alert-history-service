# K8s Publishing System Manifests

**Purpose**: RBAC configuration for TN-046/047/048/049
**Status**: 🚀 Ready for deployment

---

## Overview

Эти манифесты создают минимальные RBAC permissions для Publishing System:
- **ServiceAccount**: `alert-history-publishing`
- **Role**: `alert-history-secrets-reader` (read-only access to secrets)
- **RoleBinding**: Binds role to ServiceAccount

---

## Quick Start

### 1. Customize Namespace

```bash
# Replace 'default' with your namespace in all files
export TARGET_NAMESPACE="production"

sed -i "s/namespace: default/namespace: $TARGET_NAMESPACE/g" *.yaml
```

### 2. Apply Manifests

```bash
# Apply in order
kubectl apply -f serviceaccount.yaml
kubectl apply -f role.yaml
kubectl apply -f rolebinding.yaml

# Verify
kubectl get serviceaccount alert-history-publishing -n $TARGET_NAMESPACE
kubectl get role alert-history-secrets-reader -n $TARGET_NAMESPACE
kubectl get rolebinding alert-history-secrets-reader-binding -n $TARGET_NAMESPACE
```

### 3. Test Permissions

```bash
# Test if ServiceAccount can list secrets
kubectl auth can-i list secrets \
  --as=system:serviceaccount:$TARGET_NAMESPACE:alert-history-publishing \
  -n $TARGET_NAMESPACE

# Should return: yes

# Test if ServiceAccount can delete secrets (should be NO)
kubectl auth can-i delete secrets \
  --as=system:serviceaccount:$TARGET_NAMESPACE:alert-history-publishing \
  -n $TARGET_NAMESPACE

# Should return: no
```

---

## Security Considerations

### ✅ What This Allows

- **Read secrets** (`get`, `list`, `watch`)
- **Read events** (`get`, `list`, `watch`) - for debugging only

### ❌ What This Does NOT Allow

- Create secrets
- Update secrets
- Delete secrets
- Access other namespaces
- Access pods, deployments, or other resources

### 🔒 Best Practices

1. **Use label selectors** in application code to filter secrets:
   ```go
   labelSelector := "publishing-target=true"
   secretList, err := k8sClient.ListSecrets(ctx, namespace, labelSelector)
   ```

2. **Use separate namespaces** for different environments:
   - `alert-history-dev` - Development
   - `alert-history-staging` - Staging
   - `alert-history-prod` - Production

3. **Rotate ServiceAccount tokens** periodically (K8s does this automatically)

4. **Audit secret access**:
   ```bash
   # Check audit logs for secret access
   kubectl logs -n kube-system -l component=kube-apiserver | grep secrets
   ```

---

## Example: Target Secret

Create a secret that will be discovered by TN-047:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: rootly-prod
  namespace: production
  labels:
    publishing-target: "true"  # Required for discovery!
    target-type: rootly
    environment: production
type: Opaque
stringData:
  config: |
    {
      "name": "rootly-prod",
      "type": "rootly",
      "url": "https://api.rootly.com/v1",
      "enabled": true,
      "headers": {
        "Authorization": "Bearer YOUR_ROOTLY_API_TOKEN"
      },
      "format": "rootly"
    }
```

Apply:
```bash
kubectl apply -f rootly-target-secret.yaml
```

Verify discovery:
```bash
kubectl get secrets -l publishing-target=true -n production
```

---

## Troubleshooting

### Problem: "Forbidden" errors when listing secrets

**Symptoms**:
```
Error: secrets is forbidden: User "system:serviceaccount:default:alert-history-publishing"
cannot list resource "secrets" in API group "" in the namespace "default"
```

**Solution**:
```bash
# Check if RoleBinding exists
kubectl get rolebinding alert-history-secrets-reader-binding

# Check if Role exists
kubectl get role alert-history-secrets-reader

# Check if ServiceAccount exists
kubectl get serviceaccount alert-history-publishing

# Re-apply manifests
kubectl apply -f rolebinding.yaml
```

---

### Problem: Wrong namespace

**Symptoms**:
- Secrets not discovered
- RBAC works in testing but not in production

**Solution**:
```bash
# Check which namespace the pod is running in
kubectl get pods -l app=alert-history -A

# Update namespace in all manifests
sed -i 's/namespace: default/namespace: production/g' *.yaml

# Re-apply
kubectl apply -f *.yaml
```

---

### Problem: ServiceAccount token not mounted

**Symptoms**:
```
Error: unable to load in-cluster configuration, KUBERNETES_SERVICE_HOST and
KUBERNETES_SERVICE_PORT must be defined
```

**Solution**:
Add `serviceAccountName` to deployment:
```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      serviceAccountName: alert-history-publishing  # Add this line
      containers:
      - name: alert-history
        ...
```

---

## Files

| File | Description | Lines |
|------|-------------|-------|
| `serviceaccount.yaml` | ServiceAccount definition | 15 |
| `role.yaml` | Role with secrets read access | 30 |
| `rolebinding.yaml` | RoleBinding | 20 |
| `README.md` | This file | 200+ |

---

## Related Documentation

- **TN-046**: [K8s Client README](../../../go-app/internal/infrastructure/k8s/README.md)
- **TN-047**: [Target Discovery README](../../../go-app/internal/business/publishing/DISCOVERY_README.md)
- **TN-049**: [Health Monitoring README](../../../go-app/internal/business/publishing/HEALTH_MONITORING_README.md)
- **Integration Guide**: [TN-049 Integration Guide](../../../tasks/go-migration-analysis/TN-049-target-health-monitoring/INTEGRATION_GUIDE.md)

---

**Created by**: TN-049 Target Health Monitoring
**Last Updated**: 2025-11-08
**Status**: 🚀 READY FOR DEPLOYMENT
