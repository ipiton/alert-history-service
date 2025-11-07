# Publishing System RBAC & Kubernetes Manifests

## Overview

This directory contains Kubernetes RBAC manifests for Alert History Publishing System that implements least-privilege access for secrets discovery.

## Architecture

```
┌─────────────────────────────────────────────────────┐
│            Alert History Pod                        │
│                                                     │
│  ┌──────────────────────────────────────────────┐ │
│  │  Publishing System                           │ │
│  │                                              │ │
│  │  ┌────────────────────────────────────────┐ │ │
│  │  │  K8s Client (Secrets Discovery)        │ │ │
│  │  │  - List/Get Secrets with label filters│ │ │
│  │  │  - Watch for changes (optional)        │ │ │
│  │  └────────────────────────────────────────┘ │ │
│  │                     ↓                        │ │
│  │  ┌────────────────────────────────────────┐ │ │
│  │  │  Target Discovery Manager              │ │ │
│  │  │  - Parse secret data → PublishingTarget│ │ │
│  │  └────────────────────────────────────────┘ │ │
│  │                     ↓                        │ │
│  │  ┌────────────────────────────────────────┐ │ │
│  │  │  Publishers (Rootly, PagerDuty, etc)   │ │ │
│  │  └────────────────────────────────────────┘ │ │
│  └──────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────┘
                    ↓ (uses ServiceAccount)
┌─────────────────────────────────────────────────────┐
│            Kubernetes API Server                    │
│                                                     │
│  RBAC: ServiceAccount → Role → Secrets             │
└─────────────────────────────────────────────────────┘
```

## Files

- `rbac.yaml` - Complete RBAC setup (ServiceAccount, Role, RoleBinding)
- `secret-example.yaml` - Example publishing target secret
- `README.md` - This documentation

## RBAC Permissions

### Required Permissions

The publishing system requires **read-only** access to Kubernetes Secrets:

```yaml
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list", "watch"]
```

### Principle of Least Privilege

1. **Read-Only Access**: No write/delete permissions on secrets
2. **Resource-Specific**: Only secrets, no other resources
3. **Namespace-Scoped**: Prefer namespace-scoped Role over ClusterRole
4. **Label Filtering**: Application-level filtering via `publishing-target=true` label

## Deployment Options

### Option 1: Namespace-Scoped (Recommended)

**Use Case**: Single-namespace deployment

```bash
# Deploy namespace-scoped RBAC
kubectl apply -f rbac.yaml --namespace alert-history
```

**Benefits**:
- Minimal permissions (only one namespace)
- Easier security auditing
- Complies with zero-trust principles

**Permissions**:
- Read secrets in `alert-history` namespace only

### Option 2: Cluster-Scoped

**Use Case**: Multi-namespace target discovery

```bash
# Deploy cluster-scoped RBAC (requires cluster-admin)
kubectl apply -f rbac.yaml
```

**Benefits**:
- Discover targets across all namespaces
- Centralized publishing configuration

**Permissions**:
- Read secrets cluster-wide
- Read namespaces (for cross-namespace discovery)

**⚠️ Warning**: Requires cluster-admin to create ClusterRole

## Secret Format

Publishing targets are stored as Kubernetes Secrets with specific labels and data format:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: publishing-target-pagerduty
  namespace: alert-history
  labels:
    publishing-target: "true"  # Required for discovery
    target-type: "pagerduty"
type: Opaque
stringData:
  type: "pagerduty"
  url: "https://events.pagerduty.com/v2/enqueue"
  enabled: "true"
  format: "pagerduty"
  # PagerDuty-specific
  routing-key: "your-routing-key-here"
```

See `secret-example.yaml` for complete examples.

## Security Best Practices

### 1. Namespace Isolation

```bash
# Create dedicated namespace
kubectl create namespace alert-history

# Deploy with namespace isolation
kubectl apply -f rbac.yaml -n alert-history
```

### 2. Secret Labeling

Use labels to restrict discovery scope:

```yaml
labels:
  publishing-target: "true"  # Discovery filter
  environment: "production"  # Additional filtering
  managed-by: "alert-history"
```

### 3. Secret Encryption at Rest

Ensure cluster has encryption at rest enabled:

```bash
# Check encryption
kubectl get encryptionconfig -n kube-system
```

### 4. Audit Logging

Enable audit logging for secret access:

```yaml
# audit-policy.yaml
apiVersion: audit.k8s.io/v1
kind: Policy
rules:
  - level: Metadata
    resources:
      - group: ""
        resources: ["secrets"]
    namespaces: ["alert-history"]
```

### 5. Network Policies

Restrict pod network access:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: alert-history-publisher-netpol
spec:
  podSelector:
    matchLabels:
      app: alert-history
      component: publisher
  policyTypes:
    - Egress
  egress:
    # Allow K8s API
    - to:
        - namespaceSelector: {}
      ports:
        - protocol: TCP
          port: 443  # K8s API
    # Allow external publishing endpoints
    - to:
        - podSelector: {}
      ports:
        - protocol: TCP
          port: 443  # HTTPS
```

## Verification

### 1. Check ServiceAccount Creation

```bash
kubectl get serviceaccount alert-history-publisher -n alert-history
```

Expected output:
```
NAME                       SECRETS   AGE
alert-history-publisher    1         5m
```

### 2. Verify Role/ClusterRole

```bash
# For namespace-scoped
kubectl get role alert-history-publisher-secrets-reader -n alert-history

# For cluster-scoped
kubectl get clusterrole alert-history-publisher-secrets-reader
```

### 3. Check RoleBinding/ClusterRoleBinding

```bash
# For namespace-scoped
kubectl get rolebinding alert-history-publisher-secrets-reader -n alert-history

# For cluster-scoped
kubectl get clusterrolebinding alert-history-publisher-secrets-reader
```

### 4. Test Permissions

```bash
# Test secret access
kubectl auth can-i list secrets \
  --as=system:serviceaccount:alert-history:alert-history-publisher \
  -n alert-history
# Expected: yes

# Test write permissions (should fail)
kubectl auth can-i create secrets \
  --as=system:serviceaccount:alert-history:alert-history-publisher \
  -n alert-history
# Expected: no
```

### 5. Validate Secret Discovery

Deploy a test pod with the ServiceAccount:

```bash
kubectl run test-publisher \
  --image=bitnami/kubectl:latest \
  --serviceaccount=alert-history-publisher \
  --namespace=alert-history \
  --rm -it -- bash

# Inside the pod
kubectl get secrets -l publishing-target=true
```

## Troubleshooting

### Issue: Permission Denied

**Symptom**:
```
Error: secrets is forbidden: User "system:serviceaccount:alert-history:alert-history-publisher"
cannot list resource "secrets" in API group ""
```

**Solution**:
```bash
# Check RoleBinding
kubectl describe rolebinding alert-history-publisher-secrets-reader -n alert-history

# Verify ServiceAccount in pod spec
kubectl get pod <pod-name> -n alert-history -o yaml | grep serviceAccount
```

### Issue: Secrets Not Discovered

**Symptom**: No targets found, logs show 0 secrets

**Solution**:
```bash
# Check label on secrets
kubectl get secrets -n alert-history --show-labels

# Verify label filter in code matches secret labels
# Code expects: "publishing-target=true"
```

### Issue: Cross-Namespace Access Denied

**Symptom**: Cannot access secrets in other namespaces

**Solution**:
```bash
# Switch to ClusterRole instead of Role
kubectl delete role alert-history-publisher-secrets-reader -n alert-history
kubectl delete rolebinding alert-history-publisher-secrets-reader -n alert-history

# Apply ClusterRole section from rbac.yaml
kubectl apply -f rbac.yaml
```

## Migration Guide

### From Cluster-Scoped to Namespace-Scoped

```bash
# 1. Delete cluster-scoped resources
kubectl delete clusterrolebinding alert-history-publisher-secrets-reader
kubectl delete clusterrole alert-history-publisher-secrets-reader

# 2. Apply namespace-scoped
kubectl apply -f rbac.yaml -n alert-history

# 3. Update secret discovery config to single namespace
# In application config:
# secret_namespaces: ["alert-history"]  # Instead of ["*"]
```

### From Namespace-Scoped to Cluster-Scoped

```bash
# 1. Ensure you have cluster-admin rights
kubectl auth can-i create clusterroles
# Should return: yes

# 2. Delete namespace-scoped
kubectl delete rolebinding alert-history-publisher-secrets-reader -n alert-history
kubectl delete role alert-history-publisher-secrets-reader -n alert-history

# 3. Apply cluster-scoped
kubectl apply -f rbac.yaml
```

## Monitoring & Auditing

### Metrics to Track

1. Secret access patterns
2. Failed permission attempts
3. Secret discovery latency
4. Target refresh frequency

### Audit Queries

```bash
# View recent secret access by ServiceAccount
kubectl get events -n alert-history \
  --field-selector involvedObject.kind=Secret

# Check RBAC permissions history
kubectl logs -n kube-system -l component=kube-apiserver \
  | grep "alert-history-publisher"
```

## References

- [Kubernetes RBAC Documentation](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)
- [Secrets Security Best Practices](https://kubernetes.io/docs/concepts/security/secrets-good-practices/)
- [Principle of Least Privilege](https://kubernetes.io/docs/concepts/security/rbac-good-practices/)

## Support

For issues or questions:
1. Check logs: `kubectl logs -n alert-history -l app=alert-history`
2. Verify RBAC: `kubectl auth can-i list secrets --as=system:serviceaccount:alert-history:alert-history-publisher`
3. Review audit logs for permission denials
