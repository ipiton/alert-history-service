# ConfigMaps & Secrets Management Guide

## 📚 Overview

Alertmanager++ uses a combination of **Kubernetes Secrets**, **ConfigMaps**, and optional **External Secrets Operator (ESO)** for secure configuration management.

## 🔐 Secret Management

### Option 1: Kubernetes Secrets (Development)
Default behavior. Secrets are managed directly in Helm values.

```yaml
# values.yaml
postgresql:
  password: "my-secure-password"

llm:
  apiKey: "sk-xxxxx"
```

⚠️ **Security Risk**: Secrets in values.yaml can be committed to Git.

### Option 2: External Secrets Operator (Production)
Recommended for production. Secrets are managed in AWS Secrets Manager, GCP, Azure, or Vault.

```yaml
# values.yaml
externalSecrets:
  enabled: true
  secretStore: "aws-secretsmanager"
  keyPath: "alertmanager-plus-plus"
```

**Setup:**
1. Install ESO: `helm install external-secrets external-secrets/external-secrets`
2. Create SecretStore: `kubectl apply -f secretstore.yaml`
3. Enable ESO in values: `externalSecrets.enabled=true`

**ESO Benefits:**
- ✅ Secrets stored in external vault (AWS, GCP, Azure, Vault)
- ✅ Automatic rotation
- ✅ Audit logging
- ✅ No secrets in Git

### Option 3: Sealed Secrets (Alternative)
For GitOps workflows. Secrets encrypted at rest in Git.

```bash
kubeseal --format yaml < secret.yaml > sealed-secret.yaml
kubectl apply -f sealed-secret.yaml
```

## 🔄 Auto-Reload on Config Changes

Alertmanager++ automatically restarts pods when ConfigMaps or Secrets change.

**Mechanism:** Checksum annotations on Deployment
```yaml
annotations:
  checksum/config: {{ include "configmap.yaml" . | sha256sum }}
  checksum/secret: {{ include "secret.yaml" . | sha256sum }}
```

**Zero Downtime:** Rolling restart with readinessProbe ensures no downtime.

## 📦 ConfigMaps & Secrets Inventory

### Secrets (6 templates)
1. **secret.yaml** - Application secrets (PostgreSQL, Redis, LLM, JWT)
2. **llm-secret.yaml** - LLM API credentials
3. **rootly-secrets.yaml** - Rootly publishing target
4. **postgresql-secret.yaml** - Database credentials
5. **Publishing targets** - Dynamic secrets for each target
6. **externalsecret.yaml** - ESO integration (optional)

### ConfigMaps (2 templates)
1. **configmap.yaml** - Application configuration
2. **postgresql-configmap.yaml** - Database settings

## 🎯 Production Checklist

- [ ] Enable External Secrets Operator (`externalSecrets.enabled=true`)
- [ ] Remove hardcoded secrets from values.yaml
- [ ] Set up SecretStore in cluster
- [ ] Verify auto-reload works (change ConfigMap, watch pod restart)
- [ ] Enable RBAC for secret access
- [ ] Configure secret rotation policy
- [ ] Set up audit logging for secret access
- [ ] Test disaster recovery (restore from secret manager)

## 🔧 Common Operations

### Create Secret Manually
```bash
kubectl create secret generic alert-history-secrets \
  --from-literal=postgres-password=mysecret \
  --from-literal=llm-api-key=sk-xxxxx
```

### Update Secret
```bash
kubectl edit secret alert-history-secrets
# Pods will restart automatically due to checksum
```

### View Secret
```bash
kubectl get secret alert-history-secrets -o jsonpath='{.data.postgres-password}' | base64 -d
```

### Rotate Secret (with ESO)
```bash
# Update in AWS Secrets Manager
aws secretsmanager update-secret --secret-id alertmanager-plus-plus \
  --secret-string '{"postgres_password":"new-password"}'

# ESO syncs automatically within 1h (or force with kubectl delete secret)
```

## 🚨 Security Best Practices

1. **Never commit secrets to Git** - Use ESO or Sealed Secrets
2. **Use RBAC** - Limit secret access to service accounts
3. **Enable audit logging** - Track secret access
4. **Rotate regularly** - Automate with ESO
5. **Encrypt at rest** - Enable Kubernetes encryption
6. **Least privilege** - Separate secrets per component

## 🆘 Troubleshooting

### Secret not found
```bash
# Check if secret exists
kubectl get secrets -n alertmanager-plus-plus

# Check ESO status (if enabled)
kubectl get externalsecrets -n alertmanager-plus-plus
kubectl describe externalsecret alert-history-external-secrets
```

### Pod not restarting after config change
```bash
# Verify checksum annotation
kubectl get deployment alert-history -o yaml | grep checksum

# Force restart
kubectl rollout restart deployment alert-history
```

### ESO not syncing
```bash
# Check ESO logs
kubectl logs -n external-secrets-system deployment/external-secrets

# Verify SecretStore
kubectl get secretstore -n alertmanager-plus-plus
```

## 📖 References

- [External Secrets Operator](https://external-secrets.io/)
- [Kubernetes Secrets](https://kubernetes.io/docs/concepts/configuration/secret/)
- [Sealed Secrets](https://github.com/bitnami-labs/sealed-secrets)
