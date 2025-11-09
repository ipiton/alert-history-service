# RBAC Configuration Guide - Publishing System

**Version**: 1.0
**Last Updated**: 2025-11-08
**Applies to**: Alert History Service Publishing System (TN-046, TN-047, TN-048, TN-049)
**Target Audience**: DevOps Engineers, SRE, Platform Engineers, Security Teams

---

## 📑 Table of Contents

1. [Quick Start](#1-quick-start)
2. [Architecture Deep Dive](#2-architecture-deep-dive)
3. [ServiceAccount Configuration](#3-serviceaccount-configuration)
4. [Role vs ClusterRole Decision](#4-role-vs-clusterrole-decision)
5. [Permissions Design](#5-permissions-design)
6. [Integration with Publishing System](#6-integration-with-publishing-system)
7. [Security Best Practices](#7-security-best-practices)
8. [Monitoring with PromQL](#8-monitoring-with-promql)
9. [Troubleshooting Quick Reference](#9-troubleshooting-quick-reference)
10. [References and Resources](#10-references-and-resources)

---

## 1. Quick Start

### 1.1 Overview

The Alert History Publishing System requires read-only access to Kubernetes Secrets for dynamic discovery of publishing targets (Rootly, PagerDuty, Slack). This guide provides comprehensive RBAC configuration for **secure, least-privilege access**.

**What you'll accomplish**:
- ✅ Create ServiceAccount with minimal permissions
- ✅ Configure namespace-scoped Role (recommended)
- ✅ Deploy Publishing System with RBAC
- ✅ Verify permissions and test access
- ✅ Monitor and audit secret access

**Time to deploy**: 5 minutes
**Difficulty**: Beginner

### 1.2 Prerequisites

Before starting, ensure you have:

- [ ] **Kubernetes cluster** (v1.20+) with RBAC enabled
- [ ] **kubectl** configured with cluster-admin privileges
- [ ] **Namespace** created (e.g., `production`, `default`)
- [ ] **Helm 3.x** (optional, for Helm deployment)
- [ ] **Publishing System components** deployed (TN-046, TN-047)

**Verify cluster RBAC**:

```bash
# Check if RBAC is enabled
kubectl api-resources | grep rbac

# Expected output:
# clusterrolebindings    rbac.authorization.k8s.io/v1
# clusterroles           rbac.authorization.k8s.io/v1
# rolebindings           rbac.authorization.k8s.io/v1
# roles                  rbac.authorization.k8s.io/v1
```

### 1.3 5-Minute Deployment

**Step 1: Create ServiceAccount** (30 seconds)

```bash
kubectl create serviceaccount alert-history-publishing -n production
```

**Step 2: Create Role** (30 seconds)

```bash
cat <<EOF | kubectl apply -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: alert-history-secrets-reader
  namespace: production
rules:
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list", "watch"]
EOF
```

**Step 3: Create RoleBinding** (30 seconds)

```bash
cat <<EOF | kubectl apply -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: alert-history-secrets-reader-binding
  namespace: production
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: alert-history-secrets-reader
subjects:
- kind: ServiceAccount
  name: alert-history-publishing
  namespace: production
EOF
```

**Step 4: Verify Permissions** (1 minute)

```bash
# Test if ServiceAccount can list secrets
kubectl auth can-i list secrets \
  --as=system:serviceaccount:production:alert-history-publishing \
  -n production

# Expected output: yes

# Test if ServiceAccount CANNOT delete secrets (should fail)
kubectl auth can-i delete secrets \
  --as=system:serviceaccount:production:alert-history-publishing \
  -n production

# Expected output: no
```

**Step 5: Deploy Application** (2 minutes)

```bash
# Update deployment to use ServiceAccount
kubectl patch deployment alert-history \
  -n production \
  -p '{"spec":{"template":{"spec":{"serviceAccountName":"alert-history-publishing"}}}}'

# Verify token is mounted
kubectl exec -n production deployment/alert-history -- \
  ls -la /var/run/secrets/kubernetes.io/serviceaccount/

# Expected output:
# ca.crt
# namespace
# token
```

**Step 6: Test Secret Discovery** (1 minute)

```bash
# Create a test secret
kubectl create secret generic test-target \
  -n production \
  --from-literal=config='{"name":"test","type":"slack","url":"https://hooks.slack.com/test"}' \
  --dry-run=client -o yaml | kubectl label -f - publishing-target=true --local -o yaml | kubectl apply -f -

# Check application logs
kubectl logs -n production deployment/alert-history --tail=20 | grep "Target.*discovered"

# Expected: Log entry showing target discovery
```

🎉 **Congratulations!** RBAC is configured and working.

### 1.4 Common Gotchas

| Issue | Symptom | Fix |
|-------|---------|-----|
| **Forbidden errors** | "User cannot list secrets" | Verify RoleBinding exists and matches ServiceAccount namespace |
| **Token not mounted** | "unable to load in-cluster configuration" | Add `serviceAccountName: alert-history-publishing` to Pod spec |
| **Wrong namespace** | Secrets not discovered | Ensure Role, RoleBinding, and ServiceAccount are in same namespace |
| **Label selector mismatch** | No targets found | Add label `publishing-target=true` to secrets |

---

## 2. Architecture Deep Dive

### 2.1 RBAC Components Overview

Kubernetes RBAC consists of four core resources:

```
┌─────────────────────────────────────────────────────────────┐
│                    RBAC Architecture                        │
│                                                             │
│  ┌──────────────────┐     ┌──────────────────┐            │
│  │  ServiceAccount  │────▶│       Pod        │            │
│  │  (Identity)      │     │  (consumes SA)   │            │
│  └──────────────────┘     └──────────────────┘            │
│           │                                                 │
│           │ bound by                                        │
│           ▼                                                 │
│  ┌──────────────────┐                                      │
│  │   RoleBinding    │                                      │
│  │  (Association)   │                                      │
│  └──────────────────┘                                      │
│           │                                                 │
│           │ references                                      │
│           ▼                                                 │
│  ┌──────────────────┐                                      │
│  │      Role        │                                      │
│  │  (Permissions)   │                                      │
│  └──────────────────┘                                      │
│           │                                                 │
│           │ grants access to                                │
│           ▼                                                 │
│  ┌──────────────────┐                                      │
│  │     Secrets      │                                      │
│  │  (Resources)     │                                      │
│  └──────────────────┘                                      │
└─────────────────────────────────────────────────────────────┘
```

**Component Descriptions**:

| Component | Scope | Purpose | Example |
|-----------|-------|---------|---------|
| **ServiceAccount** | Namespace | Provides identity to pods via JWT token | `alert-history-publishing` |
| **Role** | Namespace | Defines permissions within a namespace | `alert-history-secrets-reader` |
| **ClusterRole** | Cluster | Defines permissions across all namespaces | `alert-history-secrets-reader-cluster` |
| **RoleBinding** | Namespace | Binds Role to ServiceAccount (namespace-scoped) | `alert-history-secrets-reader-binding` |
| **ClusterRoleBinding** | Cluster | Binds ClusterRole to ServiceAccount (cluster-wide) | `alert-history-secrets-reader-cluster-binding` |

### 2.2 Authentication Flow

When a pod makes a K8s API request, the following authentication flow occurs:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Authentication Flow                          │
│                                                                 │
│  Step 1: Pod reads token from mounted volume                   │
│  ┌────────────────────────────────────────────────────────┐   │
│  │  File: /var/run/secrets/kubernetes.io/serviceaccount/  │   │
│  │        token                                           │   │
│  │  Content: JWT token (base64-encoded)                   │   │
│  └────────────────────────────────────────────────────────┘   │
│                           │                                     │
│                           ▼                                     │
│  Step 2: Pod sends HTTPS request to K8s API                    │
│  ┌────────────────────────────────────────────────────────┐   │
│  │  GET /api/v1/namespaces/production/secrets            │   │
│  │  Authorization: Bearer <JWT token>                     │   │
│  └────────────────────────────────────────────────────────┘   │
│                           │                                     │
│                           ▼                                     │
│  Step 3: K8s API verifies token signature                      │
│  ┌────────────────────────────────────────────────────────┐   │
│  │  - Verify signature with CA cert                       │   │
│  │  - Extract subject (system:serviceaccount:ns:name)     │   │
│  │  - Check token expiration                              │   │
│  └────────────────────────────────────────────────────────┘   │
│                           │                                     │
│                           ▼                                     │
│  Step 4: K8s API checks RBAC permissions                       │
│  ┌────────────────────────────────────────────────────────┐   │
│  │  - Lookup RoleBinding for ServiceAccount              │   │
│  │  - Lookup Role referenced by RoleBinding              │   │
│  │  - Check if verb "list" + resource "secrets" allowed  │   │
│  └────────────────────────────────────────────────────────┘   │
│                           │                                     │
│                           ▼                                     │
│  Step 5: K8s API returns response                              │
│  ┌────────────────────────────────────────────────────────┐   │
│  │  200 OK + list of secrets (if authorized)             │   │
│  │  403 Forbidden (if not authorized)                     │   │
│  └────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### 2.3 Authorization Flow

RBAC authorization evaluates permissions using the following logic:

```
┌──────────────────────────────────────────────────────────┐
│              RBAC Authorization Logic                    │
│                                                          │
│  Request: GET /api/v1/namespaces/production/secrets     │
│           by system:serviceaccount:production:alert-...  │
│                                                          │
│  ┌────────────────────────────────────────────────┐     │
│  │  Step 1: Find all RoleBindings/ClusterRole    │     │
│  │          Bindings for this ServiceAccount      │     │
│  └────────────────────────────────────────────────┘     │
│                       │                                  │
│                       ▼                                  │
│  ┌────────────────────────────────────────────────┐     │
│  │  Step 2: For each binding, get Role/Cluster   │     │
│  │          Role permissions                      │     │
│  └────────────────────────────────────────────────┘     │
│                       │                                  │
│                       ▼                                  │
│  ┌────────────────────────────────────────────────┐     │
│  │  Step 3: Check if ANY rule matches:           │     │
│  │          - apiGroups: [""] (core API)          │     │
│  │          - resources: ["secrets"]              │     │
│  │          - verbs: ["list"]                     │     │
│  │          - namespace: production (if Role)     │     │
│  └────────────────────────────────────────────────┘     │
│                       │                                  │
│          ┌────────────┴────────────┐                    │
│          ▼                         ▼                    │
│     ┌─────────┐              ┌─────────┐               │
│     │  ALLOW  │              │  DENY   │               │
│     │ (200 OK)│              │ (403)   │               │
│     └─────────┘              └─────────┘               │
└──────────────────────────────────────────────────────────┘
```

**Key Points**:
- ✅ **Allow by default**: If ANY rule matches, access is granted
- ✅ **Deny by default**: If NO rule matches, access is denied
- ✅ **No explicit deny**: RBAC doesn't have deny rules (only allow)
- ✅ **Namespace boundary**: Role only applies to resources in same namespace

### 2.4 Security Boundaries

The Publishing System implements **5 security layers**:

```
┌─────────────────────────────────────────────────────────────┐
│                   Security Layers (Defense in Depth)        │
│                                                             │
│  Layer 5: Secrets Encryption at Rest                        │
│  ┌────────────────────────────────────────────────────┐    │
│  │  - etcd encryption (AES-256-GCM)                   │    │
│  │  - KMS provider (AWS KMS, GCP KMS, Azure KV)      │    │
│  └────────────────────────────────────────────────────┘    │
│                       │                                     │
│  Layer 4: Audit Logging                                     │
│  ┌────────────────────────────────────────────────────┐    │
│  │  - Log all secret access (get, list, watch)       │    │
│  │  - Ship to SIEM (Elasticsearch, Splunk)           │    │
│  └────────────────────────────────────────────────────┘    │
│                       │                                     │
│  Layer 3: RBAC (ServiceAccount + Role + RoleBinding)        │
│  ┌────────────────────────────────────────────────────┐    │
│  │  - Read-only permissions (get, list, watch)       │    │
│  │  - Namespace-scoped (production only)             │    │
│  │  - Label selector filtering (app level)           │    │
│  └────────────────────────────────────────────────────┘    │
│                       │                                     │
│  Layer 2: Pod Security (PodSecurityPolicy/Standards)        │
│  ┌────────────────────────────────────────────────────┐    │
│  │  - readOnlyRootFilesystem: true                   │    │
│  │  - runAsNonRoot: true                             │    │
│  │  - allowPrivilegeEscalation: false                │    │
│  └────────────────────────────────────────────────────┘    │
│                       │                                     │
│  Layer 1: Network Isolation (NetworkPolicy)                 │
│  ┌────────────────────────────────────────────────────┐    │
│  │  - Deny all ingress/egress by default            │    │
│  │  - Allow pod → K8s API (443) only                │    │
│  │  - Allow pod → DNS (53) only                      │    │
│  └────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

---

## 3. ServiceAccount Configuration

### 3.1 Basic ServiceAccount

**Minimal ServiceAccount** (recommended for most deployments):

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: alert-history-publishing
  namespace: production
  labels:
    app: alert-history
    component: publishing
  annotations:
    description: "ServiceAccount for publishing target discovery"
automountServiceAccountToken: true  # Required!
```

**Key Properties**:
- `automountServiceAccountToken: true` - Auto-mount token in pods (required for K8s API access)
- `namespace: production` - ServiceAccount is namespace-scoped
- Token lifetime: Default 1 hour (can be configured with TokenRequest API)
- Automatic rotation: Handled automatically by Kubernetes

**Deploy**:

```bash
kubectl apply -f serviceaccount.yaml
```

**Verify**:

```bash
# Check ServiceAccount exists
kubectl get serviceaccount alert-history-publishing -n production

# Check token secret is created
kubectl get secrets -n production | grep alert-history-publishing-token
```

### 3.2 Advanced: Token Projection (K8s 1.20+)

**Bound Service Account Tokens** provide enhanced security:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: alert-history
  namespace: production
spec:
  serviceAccountName: alert-history-publishing
  containers:
  - name: alert-history
    image: alert-history:latest
    volumeMounts:
    - name: sa-token
      mountPath: /var/run/secrets/tokens
      readOnly: true
  volumes:
  - name: sa-token
    projected:
      sources:
      - serviceAccountToken:
          path: token
          expirationSeconds: 3600  # 1 hour (default: 3600)
          audience: kubernetes.default.svc  # Optional: restrict audience
```

**Benefits**:
- ✅ **Short-lived tokens**: Expire after 1 hour (configurable: 600-86400 seconds)
- ✅ **Audience restriction**: Token only valid for specific audiences
- ✅ **Automatic rotation**: K8s rotates tokens before expiration
- ✅ **Reduced blast radius**: Compromised token limited in scope and time

**Configuration Options**:

| Parameter | Default | Min | Max | Recommended |
|-----------|---------|-----|-----|-------------|
| `expirationSeconds` | 3600 (1h) | 600 (10m) | 86400 (24h) | 3600 (1h) |
| `audience` | kubernetes.default.svc | - | - | kubernetes.default.svc |

**When to use**:
- ✅ **Production environments**: Always use token projection
- ✅ **High-security environments**: Set expiration to 600-1800 seconds
- ✅ **Compliance requirements**: PCI-DSS, SOC 2, HIPAA
- ⚠️ **Development**: Default token is acceptable for dev

### 3.3 Token Lifecycle

**Token rotation workflow**:

```
┌──────────────────────────────────────────────────────────────┐
│                   Token Rotation Lifecycle                   │
│                                                              │
│  T=0h: Token created                                         │
│  ┌────────────────────────────────────────────────────┐     │
│  │  Token issued with expiration: T+1h                │     │
│  │  Token stored in: /var/run/secrets/tokens/token    │     │
│  └────────────────────────────────────────────────────┘     │
│                       │                                      │
│  T=45m: Token refresh starts (80% of lifetime)               │
│  ┌────────────────────────────────────────────────────┐     │
│  │  Kubelet requests new token from K8s API           │     │
│  │  New token issued with expiration: T+1h45m         │     │
│  │  Old token still valid until T+1h                  │     │
│  └────────────────────────────────────────────────────┘     │
│                       │                                      │
│  T=46m: Token replacement                                    │
│  ┌────────────────────────────────────────────────────┐     │
│  │  Kubelet replaces token file atomically           │     │
│  │  Application re-reads token on next API call      │     │
│  └────────────────────────────────────────────────────┘     │
│                       │                                      │
│  T=1h: Old token expires                                     │
│  ┌────────────────────────────────────────────────────┐     │
│  │  Old token no longer valid                         │     │
│  │  Application using new token (T+1h45m expiration)  │     │
│  └────────────────────────────────────────────────────┘     │
│                       │                                      │
│  Cycle repeats every 45 minutes...                           │
└──────────────────────────────────────────────────────────────┘
```

**Application considerations**:
- ✅ **No code changes required**: K8s Client (TN-046) automatically handles token refresh
- ✅ **Zero downtime**: Token replacement is atomic
- ⚠️ **Token caching**: Don't cache token in application (read on each API call)

---

## 4. Role vs ClusterRole Decision

### 4.1 Decision Tree

Use this decision tree to choose between Role and ClusterRole:

```
                           Start
                             │
                             ▼
         ┌───────────────────────────────────────────┐
         │  How many namespaces need access?         │
         └───────────────┬───────────────────────────┘
                         │
         ┌───────────────┴───────────────┐
         ▼                               ▼
   Single Namespace              Multiple Namespaces
   (e.g., "production")          (e.g., "prod", "staging", "dev")
         │                               │
         ▼                               ▼
   ┌──────────────┐              ┌──────────────────┐
   │  Use Role    │              │  Use ClusterRole │
   │  +           │              │  +               │
   │  RoleBinding │              │  ClusterRole     │
   │              │              │  Binding         │
   │  ✅ Secure   │              │                  │
   │  ✅ Simple   │              │  ⚠️ Broad        │
   │  ✅ Audit    │              │  ⚠️ Complex      │
   └──────────────┘              └──────────────────┘
         │                               │
         ▼                               ▼
   RECOMMENDED                    ADVANCED USE CASE
   for Publishing                 (multi-tenant only)
   System
```

### 4.2 Option A: Namespace-Scoped (Role + RoleBinding)

**Use when**:
- ✅ All secrets are in a single namespace (e.g., `production`)
- ✅ You want to limit blast radius
- ✅ You need simple, auditable RBAC
- ✅ **Recommended for Publishing System**

**Example**:

```yaml
---
# Role: Namespace-scoped permissions
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: alert-history-secrets-reader
  namespace: production
rules:
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list", "watch"]

---
# RoleBinding: Bind Role to ServiceAccount
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: alert-history-secrets-reader-binding
  namespace: production
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: alert-history-secrets-reader
subjects:
- kind: ServiceAccount
  name: alert-history-publishing
  namespace: production
```

**Pros**:
- ✅ **Security**: Limited to single namespace
- ✅ **Simplicity**: Easy to understand and audit
- ✅ **Compliance**: Easier to pass security reviews
- ✅ **Isolation**: Namespace boundaries enforced

**Cons**:
- ❌ **Multi-namespace**: Doesn't work for cross-namespace targets
- ❌ **Scalability**: Need to replicate Role in each namespace

### 4.3 Option B: Cluster-Wide (ClusterRole + ClusterRoleBinding)

**Use when**:
- ⚠️ Secrets are in multiple namespaces
- ⚠️ You need cross-namespace target discovery
- ⚠️ You have multi-tenant requirements
- ⚠️ **Not recommended for most deployments**

**Example**:

```yaml
---
# ClusterRole: Cluster-wide permissions
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: alert-history-secrets-reader
rules:
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get", "list"]  # Required for multi-namespace

---
# ClusterRoleBinding: Bind ClusterRole to ServiceAccount
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: alert-history-secrets-reader
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: alert-history-secrets-reader
subjects:
- kind: ServiceAccount
  name: alert-history-publishing
  namespace: production
```

**Pros**:
- ✅ **Multi-namespace**: Access secrets in all namespaces
- ✅ **Scalability**: Single RBAC configuration for entire cluster

**Cons**:
- ❌ **Security Risk**: Access to ALL secrets in ALL namespaces
- ❌ **Compliance**: Harder to justify in security reviews
- ❌ **Audit**: Difficult to track cross-namespace access
- ❌ **Blast Radius**: Compromised token = cluster-wide access

**Security mitigation** (if ClusterRole is required):

```yaml
# Restrict to specific namespaces using label selectors
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: alert-history-secrets-reader
rules:
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get", "list"]
  # Application filters namespaces by label:
  #   publishing-enabled=true
```

**Application code** (namespace filtering):

```go
// Filter namespaces with label selector
labelSelector := "publishing-enabled=true"
namespaces, err := k8sClient.Core().V1().Namespaces().List(ctx, metav1.ListOptions{
    LabelSelector: labelSelector,
})

// Only allowed namespaces: ["production", "staging"]
// Denied namespaces: ["kube-system", "kube-public", "dev"]
```

### 4.4 Option C: Hybrid (ClusterRole + RoleBinding)

**Use when**:
- ✅ You want to define permissions once (DRY)
- ✅ You want to apply to multiple namespaces selectively
- ✅ You want namespace isolation with shared definitions

**Example**:

```yaml
---
# ClusterRole: Define permissions once
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: alert-history-secrets-reader
rules:
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list", "watch"]

---
# RoleBinding: Apply to "production" namespace only
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: alert-history-secrets-reader-binding
  namespace: production  # Only affects "production"
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole  # ← References ClusterRole
  name: alert-history-secrets-reader
subjects:
- kind: ServiceAccount
  name: alert-history-publishing
  namespace: production

---
# RoleBinding: Apply to "staging" namespace
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: alert-history-secrets-reader-binding
  namespace: staging  # Only affects "staging"
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole  # ← Same ClusterRole
  name: alert-history-secrets-reader
subjects:
- kind: ServiceAccount
  name: alert-history-publishing
  namespace: staging
```

**Pros**:
- ✅ **DRY**: Define permissions once
- ✅ **Namespace isolation**: Each RoleBinding scopes to one namespace
- ✅ **Maintainability**: Update ClusterRole, all bindings inherit changes

**Cons**:
- ❌ **Complexity**: More resources to manage
- ❌ **Confusion**: Mixing ClusterRole and RoleBinding can be confusing

### 4.5 Comparison Matrix

| Feature | Role + RoleBinding | ClusterRole + ClusterRoleBinding | ClusterRole + RoleBinding |
|---------|-------------------|--------------------------------|-------------------------|
| **Scope** | Single namespace | All namespaces | Multiple namespaces (selected) |
| **Security** | ✅ High | ⚠️ Low | ✅ High |
| **Simplicity** | ✅ High | ✅ High | ⚠️ Medium |
| **Maintainability** | ⚠️ Medium | ✅ High | ✅ High |
| **Audit** | ✅ Easy | ⚠️ Hard | ✅ Easy |
| **Compliance** | ✅ Easy | ❌ Hard | ✅ Easy |
| **Recommendation** | **✅ Recommended** | ❌ Avoid | ✅ Advanced |

---

## 5. Permissions Design

### 5.1 Read-Only Permissions (Recommended)

**Minimal permissions** for Publishing System:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: alert-history-secrets-reader
  namespace: production
rules:
# Rule 1: Read secrets (publishing targets)
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list", "watch"]
  # Note: Label selector applied at application level
  # k8sClient.ListSecrets(ctx, "production", "publishing-target=true")

# Rule 2: Read events (debugging, optional)
- apiGroups: [""]
  resources: ["events"]
  verbs: ["get", "list"]
  # Useful for troubleshooting RBAC issues
  # Can be removed in production if not needed
```

**Verbs explained**:

| Verb | HTTP Method | Purpose | Example |
|------|-------------|---------|---------|
| `get` | GET | Read single resource | `kubectl get secret rootly-prod` |
| `list` | GET | Read all resources | `kubectl get secrets` |
| `watch` | GET (streaming) | Watch for changes | `kubectl get secrets --watch` |

**Why read-only?**:
- ✅ **Least privilege**: Publishing System only needs to read secrets
- ✅ **Security**: Cannot modify or delete secrets
- ✅ **Compliance**: Easier to justify in audits
- ✅ **Fail-safe**: Mistakes cannot corrupt secrets

### 5.2 Label Selector Strategy

**RBAC doesn't support label selectors** in rules. Filter at application level:

```yaml
# RBAC: Grant access to ALL secrets in namespace
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: alert-history-secrets-reader
  namespace: production
rules:
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list", "watch"]
  # ❌ Cannot restrict to specific labels here
```

**Application code** (TN-047 Target Discovery):

```go
// Application filters secrets by label
labelSelector := "publishing-target=true"
secretList, err := k8sClient.ListSecrets(ctx, "production", labelSelector)

// K8s API returns only secrets with label "publishing-target=true"
// Other secrets are not returned (filtered server-side)
```

**Secret labeling best practices**:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: rootly-prod
  namespace: production
  labels:
    # Required: Publishing target marker
    publishing-target: "true"

    # Optional: Target type
    target-type: rootly

    # Optional: Environment
    environment: production

    # Optional: Managed-by annotation
    managed-by: terraform
type: Opaque
stringData:
  config: |
    {
      "name": "rootly-prod",
      "type": "rootly",
      "url": "https://api.rootly.com/v1",
      "enabled": true,
      "headers": {
        "Authorization": "Bearer YOUR_TOKEN"
      }
    }
```

**Label naming conventions**:

| Label | Values | Purpose |
|-------|--------|---------|
| `publishing-target` | `"true"` | Mark secret for discovery (required) |
| `target-type` | `rootly`, `pagerduty`, `slack` | Target type (optional) |
| `environment` | `dev`, `staging`, `production` | Environment (optional) |
| `managed-by` | `terraform`, `helm`, `manual` | Management tool (optional) |

### 5.3 Multi-Resource Permissions (Advanced)

**Extended permissions** for advanced use cases:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: alert-history-advanced
  namespace: production
rules:
# Rule 1: Read secrets
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list", "watch"]

# Rule 2: Read configmaps (configuration discovery)
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "list", "watch"]
  # Use case: Store publishing configuration in ConfigMaps
  # Label: publishing-config=true

# Rule 3: Read events (debugging)
- apiGroups: [""]
  resources: ["events"]
  verbs: ["get", "list"]
  # Use case: Troubleshoot RBAC issues

# Rule 4: Read pods (health checking, optional)
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list"]
  # Use case: Discover other Publishing System pods
  # Not required for typical deployment
```

**When to add additional permissions**:

| Permission | Use Case | Risk | Recommendation |
|------------|----------|------|----------------|
| `configmaps` (read) | Configuration discovery | Low | ✅ OK if needed |
| `events` (read) | Debugging | Low | ✅ OK for non-prod |
| `pods` (read) | Health checking | Low | ⚠️ Only if required |
| `secrets` (write) | Dynamic secrets | **High** | ❌ Avoid |

### 5.4 Negative Test Cases

**Test that permissions are correctly restricted**:

```bash
# Test 1: Can list secrets (should pass)
kubectl auth can-i list secrets \
  --as=system:serviceaccount:production:alert-history-publishing \
  -n production
# Expected: yes

# Test 2: Cannot delete secrets (should fail)
kubectl auth can-i delete secrets \
  --as=system:serviceaccount:production:alert-history-publishing \
  -n production
# Expected: no

# Test 3: Cannot create secrets (should fail)
kubectl auth can-i create secrets \
  --as=system:serviceaccount:production:alert-history-publishing \
  -n production
# Expected: no

# Test 4: Cannot access kube-system namespace (should fail)
kubectl auth can-i list secrets \
  --as=system:serviceaccount:production:alert-history-publishing \
  -n kube-system
# Expected: no

# Test 5: Cannot list pods (should fail, unless explicitly granted)
kubectl auth can-i list pods \
  --as=system:serviceaccount:production:alert-history-publishing \
  -n production
# Expected: no
```

---

## 6. Integration with Publishing System

### 6.1 K8s Client (TN-046) Integration

**K8s Client** uses ServiceAccount token automatically:

```go
// internal/infrastructure/k8s/client.go

// NewK8sClient creates client with in-cluster configuration
func NewK8sClient(config *K8sClientConfig) (K8sClient, error) {
    // Uses ServiceAccount token from:
    // /var/run/secrets/kubernetes.io/serviceaccount/token
    restConfig, err := rest.InClusterConfig()
    if err != nil {
        return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
    }

    // Create Kubernetes clientset
    clientset, err := kubernetes.NewForConfig(restConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create clientset: %w", err)
    }

    return &DefaultK8sClient{
        clientset: clientset,
        logger:    config.Logger,
    }, nil
}

// ListSecrets lists secrets with label selector
func (c *DefaultK8sClient) ListSecrets(ctx context.Context, namespace, labelSelector string) ([]*corev1.Secret, error) {
    // RBAC check happens here (K8s API)
    secretList, err := c.clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{
        LabelSelector: labelSelector,
    })
    if err != nil {
        // Handle RBAC errors
        if statusErr, ok := err.(*errors.StatusError); ok {
            if statusErr.Status().Code == 403 {
                return nil, fmt.Errorf("forbidden: check RBAC permissions: %w", err)
            }
        }
        return nil, err
    }

    // Convert to pointers
    secrets := make([]*corev1.Secret, len(secretList.Items))
    for i := range secretList.Items {
        secrets[i] = &secretList.Items[i]
    }

    return secrets, nil
}
```

**Error handling**:

```go
// Example: Graceful degradation on RBAC errors
secrets, err := k8sClient.ListSecrets(ctx, "production", "publishing-target=true")
if err != nil {
    if strings.Contains(err.Error(), "forbidden") {
        // RBAC error: log and continue with empty list
        logger.Warn("RBAC permission denied, continuing without targets",
            "error", err,
            "namespace", "production",
            "labelSelector", "publishing-target=true")
        return []PublishingTarget{}, nil
    }
    // Other error: fail fast
    return nil, fmt.Errorf("failed to list secrets: %w", err)
}
```

### 6.2 Target Discovery (TN-047) Integration

**Target Discovery Manager** discovers secrets using label selectors:

```go
// internal/business/publishing/discovery_impl.go

func (m *DefaultTargetDiscoveryManager) DiscoverTargets(ctx context.Context) error {
    // List secrets with label selector
    secrets, err := m.k8sClient.ListSecrets(ctx, m.namespace, m.labelSelector)
    if err != nil {
        return fmt.Errorf("failed to list secrets: %w", err)
    }

    // Parse each secret into PublishingTarget
    targets := make([]PublishingTarget, 0, len(secrets))
    for _, secret := range secrets {
        target, err := m.parseSecretToTarget(secret)
        if err != nil {
            m.logger.Warn("Failed to parse secret, skipping",
                "secret", secret.Name,
                "error", err)
            continue  // Skip invalid secrets (fail-safe)
        }
        targets = append(targets, target)
    }

    // Update cache
    m.cache.UpdateTargets(targets)

    m.logger.Info("Target discovery complete",
        "total_secrets", len(secrets),
        "valid_targets", len(targets),
        "invalid_targets", len(secrets)-len(targets))

    return nil
}
```

**End-to-end flow**:

```
┌─────────────────────────────────────────────────────────────┐
│           Target Discovery Flow (TN-047)                    │
│                                                             │
│  Step 1: DiscoverTargets() called                           │
│  ┌────────────────────────────────────────────────────┐    │
│  │  discovery_impl.go:DiscoverTargets()               │    │
│  └────────────────────────────────────────────────────┘    │
│                       │                                     │
│  Step 2: List secrets with label selector                   │
│  ┌────────────────────────────────────────────────────┐    │
│  │  k8sClient.ListSecrets(ctx, "production",          │    │
│  │    "publishing-target=true")                       │    │
│  └────────────────────────────────────────────────────┘    │
│                       │                                     │
│  Step 3: K8s API authenticates + authorizes (RBAC)          │
│  ┌────────────────────────────────────────────────────┐    │
│  │  - Verify ServiceAccount token                     │    │
│  │  - Check Role permissions (list secrets)           │    │
│  │  - Filter by label selector server-side            │    │
│  └────────────────────────────────────────────────────┘    │
│                       │                                     │
│  Step 4: Return secrets matching label                      │
│  ┌────────────────────────────────────────────────────┐    │
│  │  200 OK + [secret/rootly-prod, secret/pagerduty]  │    │
│  └────────────────────────────────────────────────────┘    │
│                       │                                     │
│  Step 5: Parse secrets into PublishingTarget objects        │
│  ┌────────────────────────────────────────────────────┐    │
│  │  parseSecretToTarget(secret) for each secret       │    │
│  │  Validate target configuration                     │    │
│  └────────────────────────────────────────────────────┘    │
│                       │                                     │
│  Step 6: Update cache with discovered targets               │
│  ┌────────────────────────────────────────────────────┐    │
│  │  cache.UpdateTargets(targets)                      │    │
│  │  Metrics: targets_discovered_total++               │    │
│  └────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

### 6.3 Health Monitor (TN-049) Integration

**Health Monitor** checks target connectivity:

```go
// internal/business/publishing/health_impl.go

func (m *DefaultHealthMonitor) CheckTargetHealth(ctx context.Context, target *PublishingTarget) (*TargetHealth, error) {
    // Target already discovered via TN-047 (with RBAC)
    // Now check HTTP connectivity

    health := &TargetHealth{
        TargetName: target.Name,
        Status:     StatusHealthy,
        CheckedAt:  time.Now(),
    }

    // HTTP connectivity check
    req, err := http.NewRequestWithContext(ctx, "GET", target.URL, nil)
    if err != nil {
        health.Status = StatusUnhealthy
        health.Error = err.Error()
        return health, nil
    }

    // Add headers from secret (configured via RBAC-protected secret)
    for k, v := range target.Headers {
        req.Header.Set(k, v)
    }

    // Execute request
    resp, err := m.httpClient.Do(req)
    if err != nil {
        health.Status = StatusUnhealthy
        health.Error = err.Error()
        return health, nil
    }
    defer resp.Body.Close()

    // Check status code
    if resp.StatusCode >= 400 {
        health.Status = StatusDegraded
        health.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
    }

    return health, nil
}
```

### 6.4 Deployment Configuration

**Kubernetes Deployment** with RBAC:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: alert-history
  namespace: production
spec:
  replicas: 2
  selector:
    matchLabels:
      app: alert-history
  template:
    metadata:
      labels:
        app: alert-history
    spec:
      # Specify ServiceAccount
      serviceAccountName: alert-history-publishing

      # Security context (Pod level)
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        fsGroup: 1000

      containers:
      - name: alert-history
        image: alert-history:latest

        # Security context (Container level)
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop:
            - ALL

        # Environment variables
        env:
        - name: K8S_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: TARGET_LABEL_SELECTOR
          value: "publishing-target=true"

        # Volume mounts
        volumeMounts:
        - name: tmp
          mountPath: /tmp
        - name: cache
          mountPath: /var/cache

      volumes:
      - name: tmp
        emptyDir: {}
      - name: cache
        emptyDir: {}
```

**Verify ServiceAccount token is mounted**:

```bash
# Check token is mounted in pod
kubectl exec -n production deployment/alert-history -- \
  ls -la /var/run/secrets/kubernetes.io/serviceaccount/

# Expected output:
# ca.crt
# namespace
# token

# Read token (for debugging only, DO NOT LOG IN PRODUCTION!)
kubectl exec -n production deployment/alert-history -- \
  cat /var/run/secrets/kubernetes.io/serviceaccount/token | head -c 50

# Expected: eyJhbGciOiJSUzI1NiIsImtpZCI6Ikxxxxxxxxxxxxxxxxxxxxxxx...
```

---

## 7. Security Best Practices

### 7.1 Principle of Least Privilege

**Apply minimal permissions**:

```yaml
# ✅ GOOD: Read-only, namespace-scoped
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: alert-history-secrets-reader
  namespace: production
rules:
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list", "watch"]  # Read-only

# ❌ BAD: Write permissions, cluster-wide
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: alert-history-admin
rules:
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["*"]  # All verbs (including delete!)
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["*"]  # All resources!
```

**15 Least Privilege Examples**:

1. ✅ **Namespace-scoped Role** (not ClusterRole)
2. ✅ **Read-only verbs** (`get`, `list`, `watch` only)
3. ✅ **Specific resources** (`secrets` only, not `*`)
4. ✅ **Label selector filtering** (application level)
5. ✅ **No wildcard permissions** (no `*` in apiGroups, resources, verbs)
6. ✅ **Separate ServiceAccounts** (one per application)
7. ✅ **No default ServiceAccount** (create dedicated)
8. ✅ **Short-lived tokens** (1 hour expiration)
9. ✅ **Audience restrictions** (kubernetes.default.svc)
10. ✅ **Read-only filesystem** (readOnlyRootFilesystem: true)
11. ✅ **Non-root user** (runAsNonRoot: true)
12. ✅ **No privilege escalation** (allowPrivilegeEscalation: false)
13. ✅ **Drop all capabilities** (capabilities: drop: [ALL])
14. ✅ **NetworkPolicy enforcement** (deny all by default)
15. ✅ **Audit logging enabled** (track all secret access)

### 7.2 Token Security

**Token best practices**:

| Practice | Recommendation | Rationale |
|----------|----------------|-----------|
| **Token lifetime** | 1 hour (3600s) | Balance security and usability |
| **Token rotation** | Automatic (Kubelet) | Zero downtime, no manual intervention |
| **Token projection** | Always use in prod | Short-lived, audience-restricted |
| **Token caching** | Never cache tokens | Use fresh token on each API call |
| **Token logging** | Never log tokens | Prevent credential leakage |
| **Token storage** | Only in memory | Never persist to disk |

**Token expiration settings**:

```yaml
# Production (strict)
expirationSeconds: 3600  # 1 hour

# High-security (very strict)
expirationSeconds: 1800  # 30 minutes

# Development (relaxed, NOT for production!)
expirationSeconds: 86400  # 24 hours
```

### 7.3 Namespace Isolation

**Enforce namespace boundaries**:

```yaml
# ✅ GOOD: Namespace-scoped Role
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: alert-history-secrets-reader
  namespace: production  # Only affects "production"
rules:
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list", "watch"]

---
# RoleBinding in same namespace
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: alert-history-secrets-reader-binding
  namespace: production  # Same namespace
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: alert-history-secrets-reader
subjects:
- kind: ServiceAccount
  name: alert-history-publishing
  namespace: production  # Same namespace
```

**Test namespace isolation**:

```bash
# Can access "production" namespace
kubectl auth can-i list secrets \
  --as=system:serviceaccount:production:alert-history-publishing \
  -n production
# Expected: yes

# CANNOT access "kube-system" namespace
kubectl auth can-i list secrets \
  --as=system:serviceaccount:production:alert-history-publishing \
  -n kube-system
# Expected: no

# CANNOT access "dev" namespace
kubectl auth can-i list secrets \
  --as=system:serviceaccount:production:alert-history-publishing \
  -n dev
# Expected: no
```

### 7.4 Audit Logging

**Enable audit logging** for all secret access:

```yaml
# /etc/kubernetes/audit-policy.yaml
apiVersion: audit.k8s.io/v1
kind: Policy
rules:
# Rule 1: Log all secret access (RequestResponse level)
- level: RequestResponse
  verbs: ["get", "list", "watch"]
  resources:
  - group: ""
    resources: ["secrets"]
  namespaces: ["production", "staging"]
  # WARNING: Logs full secret data!
  # Use Metadata level in production if secrets contain sensitive data

# Rule 2: Log RBAC changes (Metadata level)
- level: Metadata
  verbs: ["create", "update", "patch", "delete"]
  resources:
  - group: "rbac.authorization.k8s.io"
    resources: ["roles", "rolebindings", "clusterroles", "clusterrolebindings"]

# Rule 3: Omit read-only requests to kube-system
- level: None
  verbs: ["get", "list", "watch"]
  namespaces: ["kube-system"]
```

**Audit log analysis** (Loki):

```promql
# Count secret access by ServiceAccount
count by (user_username) (
  {job="kube-apiserver"}
  | json
  | verb =~ "get|list|watch"
  | objectRef_resource = "secrets"
  | objectRef_namespace = "production"
)

# Alert on unusual access patterns
rate(
  {job="kube-apiserver"}
  | json
  | verb =~ "get|list|watch"
  | objectRef_resource = "secrets"
  | user_username = "system:serviceaccount:production:alert-history-publishing"
) > 100  # More than 100 requests per second
```

### 7.5 Regular Permission Reviews

**Quarterly review checklist** (20+ items):

- [ ] Review all ServiceAccounts (delete unused)
- [ ] Review all Roles/ClusterRoles (remove excessive permissions)
- [ ] Review all RoleBindings/ClusterRoleBindings (verify subjects)
- [ ] Check for wildcard permissions (`*` in apiGroups, resources, verbs)
- [ ] Check for cluster-admin bindings (should be minimal)
- [ ] Verify ServiceAccount token expiration (1 hour recommended)
- [ ] Verify audit logging is enabled
- [ ] Review audit logs for anomalies
- [ ] Check for failed authentication attempts (403 Forbidden)
- [ ] Verify NetworkPolicies are enabled
- [ ] Check Pod Security Policies/Standards
- [ ] Review Secret access frequency (is it necessary?)
- [ ] Verify label selectors are restrictive
- [ ] Check for secrets without `publishing-target=true` label
- [ ] Review RBAC compliance (CIS, PCI-DSS, SOC 2)
- [ ] Test negative scenarios (should fail)
- [ ] Verify RBAC documentation is up-to-date
- [ ] Review incident response procedures
- [ ] Check for RBAC-related incidents (past quarter)
- [ ] Update RBAC configurations based on lessons learned

**Automation**:

```bash
# Quarterly RBAC review script
#!/bin/bash

# List all ServiceAccounts
kubectl get serviceaccounts --all-namespaces

# List all Roles with wildcard permissions
kubectl get roles --all-namespaces -o json | \
  jq '.items[] | select(.rules[].verbs[] == "*")'

# List all ClusterRoles with wildcard permissions
kubectl get clusterroles -o json | \
  jq '.items[] | select(.rules[].verbs[] == "*")'

# List all cluster-admin bindings
kubectl get clusterrolebindings -o json | \
  jq '.items[] | select(.roleRef.name == "cluster-admin")'
```

---

## 8. Monitoring with PromQL

### 8.1 RBAC Metrics

**Key metrics to monitor**:

```promql
# 1. Secret access count by ServiceAccount
sum by (user_username) (
  rate(apiserver_request_total{
    resource="secrets",
    verb=~"get|list|watch"
  }[5m])
)

# 2. RBAC Forbidden errors
sum by (user_username) (
  rate(apiserver_request_total{
    code="403",
    resource="secrets"
  }[5m])
)

# 3. Secret access latency (p95)
histogram_quantile(0.95,
  sum by (le) (
    rate(apiserver_request_duration_seconds_bucket{
      resource="secrets",
      verb="list"
    }[5m])
  )
)

# 4. ServiceAccount token expiration
kubernetes_serviceaccount_token_expiration_seconds < 3600

# 5. RBAC permission check latency
histogram_quantile(0.99,
  sum by (le) (
    rate(apiserver_authorization_duration_seconds_bucket[5m])
  )
)
```

### 8.2 Publishing System Metrics

**Publishing-specific metrics**:

```promql
# 1. Target discovery success rate
sum(rate(alert_history_publishing_discovery_total{status="success"}[5m])) /
sum(rate(alert_history_publishing_discovery_total[5m]))

# 2. Targets discovered count
alert_history_publishing_targets_total

# 3. Invalid secrets count
sum(rate(alert_history_publishing_discovery_errors_total{error_type="invalid_format"}[5m]))

# 4. Target health by type
sum by (target_type, status) (
  alert_history_publishing_health_targets{
    status=~"healthy|degraded|unhealthy"
  }
)

# 5. Secret refresh frequency
rate(alert_history_publishing_refresh_total{status="success"}[10m])
```

### 8.3 Alerting Rules

**Prometheus alerting rules**:

```yaml
groups:
- name: rbac_alerts
  interval: 30s
  rules:
  # Alert 1: High RBAC Forbidden error rate
  - alert: HighRBACForbiddenRate
    expr: |
      sum(rate(apiserver_request_total{
        code="403",
        resource="secrets",
        user=~"system:serviceaccount:production:alert-history-.*"
      }[5m])) > 1
    for: 5m
    labels:
      severity: warning
      component: rbac
    annotations:
      summary: "High RBAC Forbidden error rate"
      description: "ServiceAccount {{ $labels.user }} is experiencing RBAC Forbidden errors ({{ $value }} req/s)"

  # Alert 2: Target discovery failing
  - alert: TargetDiscoveryFailing
    expr: |
      rate(alert_history_publishing_discovery_total{status="failed"}[10m]) > 0.1
    for: 15m
    labels:
      severity: critical
      component: publishing
    annotations:
      summary: "Target discovery failing"
      description: "Target discovery is failing ({{ $value }} failures/s). Check RBAC permissions."

  # Alert 3: No targets discovered
  - alert: NoTargetsDiscovered
    expr: |
      alert_history_publishing_targets_total == 0
    for: 30m
    labels:
      severity: warning
      component: publishing
    annotations:
      summary: "No publishing targets discovered"
      description: "No targets discovered in last 30 minutes. Check RBAC and secret labels."

  # Alert 4: ServiceAccount token expiring soon
  - alert: ServiceAccountTokenExpiringSoon
    expr: |
      kubernetes_serviceaccount_token_expiration_seconds{
        serviceaccount="alert-history-publishing",
        namespace="production"
      } < 3600
    for: 5m
    labels:
      severity: warning
      component: rbac
    annotations:
      summary: "ServiceAccount token expiring soon"
      description: "Token for {{ $labels.serviceaccount }} expires in {{ $value }}s"
```

### 8.4 Grafana Dashboards

**Grafana panel examples**:

**Panel 1: Secret Access Rate**

```json
{
  "title": "Secret Access Rate (by ServiceAccount)",
  "targets": [
    {
      "expr": "sum by (user_username) (rate(apiserver_request_total{resource=\"secrets\", verb=~\"get|list|watch\"}[5m]))",
      "legendFormat": "{{ user_username }}"
    }
  ],
  "type": "graph"
}
```

**Panel 2: RBAC Forbidden Errors**

```json
{
  "title": "RBAC Forbidden Errors",
  "targets": [
    {
      "expr": "sum(rate(apiserver_request_total{code=\"403\", resource=\"secrets\"}[5m]))",
      "legendFormat": "Forbidden (403)"
    }
  ],
  "type": "stat",
  "thresholds": {
    "mode": "absolute",
    "steps": [
      {"value": 0, "color": "green"},
      {"value": 0.1, "color": "yellow"},
      {"value": 1, "color": "red"}
    ]
  }
}
```

**Panel 3: Targets Discovered**

```json
{
  "title": "Publishing Targets Discovered",
  "targets": [
    {
      "expr": "alert_history_publishing_targets_total",
      "legendFormat": "Total Targets"
    }
  ],
  "type": "stat"
}
```

---

## 9. Troubleshooting Quick Reference

### 9.1 Common Issues

#### Issue 1: "Forbidden: User cannot list secrets"

**Symptoms**:
```
Error from server (Forbidden): secrets is forbidden:
User "system:serviceaccount:production:alert-history-publishing"
cannot list resource "secrets" in API group "" in the namespace "production"
```

**Root Causes**:
1. Role or ClusterRole doesn't exist
2. RoleBinding or ClusterRoleBinding doesn't exist
3. RoleBinding subject doesn't match ServiceAccount
4. Wrong namespace in RoleBinding

**Diagnostic Steps**:

```bash
# Step 1: Check if ServiceAccount exists
kubectl get serviceaccount alert-history-publishing -n production

# Step 2: Check if Role exists
kubectl get role alert-history-secrets-reader -n production

# Step 3: Check if RoleBinding exists
kubectl get rolebinding alert-history-secrets-reader-binding -n production

# Step 4: Describe RoleBinding (check subject)
kubectl describe rolebinding alert-history-secrets-reader-binding -n production
# Verify:
# - subjects[].name matches ServiceAccount name
# - subjects[].namespace matches ServiceAccount namespace

# Step 5: Test permissions
kubectl auth can-i list secrets \
  --as=system:serviceaccount:production:alert-history-publishing \
  -n production
```

**Solutions**:

```bash
# Solution 1: Create missing Role
kubectl apply -f k8s/publishing/examples/single-namespace/role.yaml

# Solution 2: Create missing RoleBinding
kubectl apply -f k8s/publishing/examples/single-namespace/rolebinding.yaml

# Solution 3: Fix RoleBinding subject namespace
kubectl edit rolebinding alert-history-secrets-reader-binding -n production
# Change:
#   subjects:
#   - kind: ServiceAccount
#     name: alert-history-publishing
#     namespace: default  # ← Wrong namespace
# To:
#   subjects:
#   - kind: ServiceAccount
#     name: alert-history-publishing
#     namespace: production  # ← Correct namespace
```

#### Issue 2: "ServiceAccount token not mounted"

**Symptoms**:
```
Error: unable to load in-cluster configuration, KUBERNETES_SERVICE_HOST and
KUBERNETES_SERVICE_PORT must be defined
```

**Root Cause**:
- ServiceAccount token not mounted in pod
- `automountServiceAccountToken: false` (explicitly disabled)
- Missing `serviceAccountName` in Pod spec

**Diagnostic Steps**:

```bash
# Step 1: Check if token is mounted
kubectl exec -n production deployment/alert-history -- \
  ls -la /var/run/secrets/kubernetes.io/serviceaccount/
# Expected output: ca.crt, namespace, token

# Step 2: Check Pod spec
kubectl get pod -n production -l app=alert-history -o yaml | \
  grep -A 2 "serviceAccountName"
# Expected: serviceAccountName: alert-history-publishing
```

**Solutions**:

```bash
# Solution 1: Add serviceAccountName to Deployment
kubectl patch deployment alert-history -n production \
  -p '{"spec":{"template":{"spec":{"serviceAccountName":"alert-history-publishing"}}}}'

# Solution 2: Enable automountServiceAccountToken
kubectl patch serviceaccount alert-history-publishing -n production \
  -p '{"automountServiceAccountToken":true}'

# Solution 3: Restart pods
kubectl rollout restart deployment/alert-history -n production
```

#### Issue 3: "Label selector not matching secrets"

**Symptoms**:
- RBAC works (no Forbidden errors)
- No targets discovered (empty list)
- Secrets exist in namespace

**Root Cause**:
- Secrets missing `publishing-target=true` label
- Label selector mismatch (typo)

**Diagnostic Steps**:

```bash
# Step 1: Check secrets without label
kubectl get secrets -n production --show-labels

# Step 2: Filter secrets with correct label
kubectl get secrets -n production -l publishing-target=true
# Expected: List of publishing target secrets

# Step 3: Check application logs
kubectl logs -n production deployment/alert-history --tail=50 | \
  grep "Target.*discovered"
```

**Solutions**:

```bash
# Solution 1: Add label to existing secret
kubectl label secret rootly-prod -n production publishing-target=true

# Solution 2: Create secret with label
kubectl create secret generic test-target -n production \
  --from-literal=config='{}' \
  --dry-run=client -o yaml | \
  kubectl label -f - publishing-target=true --local -o yaml | \
  kubectl apply -f -

# Solution 3: Verify label selector in application
kubectl get deployment alert-history -n production -o yaml | \
  grep TARGET_LABEL_SELECTOR
# Expected: TARGET_LABEL_SELECTOR=publishing-target=true
```

### 9.2 Diagnostic Commands

**Essential kubectl commands**:

```bash
# Test permissions (can-i checks)
kubectl auth can-i list secrets --as=system:serviceaccount:production:alert-history-publishing -n production
kubectl auth can-i get secret rootly-prod --as=system:serviceaccount:production:alert-history-publishing -n production
kubectl auth can-i delete secrets --as=system:serviceaccount:production:alert-history-publishing -n production

# Describe RBAC resources
kubectl describe serviceaccount alert-history-publishing -n production
kubectl describe role alert-history-secrets-reader -n production
kubectl describe rolebinding alert-history-secrets-reader-binding -n production

# View RBAC YAML
kubectl get role alert-history-secrets-reader -n production -o yaml
kubectl get rolebinding alert-history-secrets-reader-binding -n production -o yaml

# Check pod ServiceAccount
kubectl get pods -n production -l app=alert-history -o jsonpath='{.items[0].spec.serviceAccountName}'

# Check token mount
kubectl exec -n production deployment/alert-history -- ls -la /var/run/secrets/kubernetes.io/serviceaccount/

# Check application logs
kubectl logs -n production deployment/alert-history --tail=100 | grep -E "RBAC|Forbidden|secret"
```

### 9.3 Quick Fixes

| Issue | One-liner Fix |
|-------|---------------|
| **Missing ServiceAccount** | `kubectl create serviceaccount alert-history-publishing -n production` |
| **Missing Role** | `kubectl apply -f k8s/publishing/examples/single-namespace/role.yaml` |
| **Missing RoleBinding** | `kubectl apply -f k8s/publishing/examples/single-namespace/rolebinding.yaml` |
| **Wrong namespace** | `kubectl edit rolebinding alert-history-secrets-reader-binding -n production` |
| **Token not mounted** | `kubectl patch deployment alert-history -n production -p '{"spec":{"template":{"spec":{"serviceAccountName":"alert-history-publishing"}}}}'` |
| **Missing label** | `kubectl label secret rootly-prod -n production publishing-target=true` |
| **RBAC changes not applying** | `kubectl rollout restart deployment/alert-history -n production` |

---

## 10. References and Resources

### 10.1 Internal Documentation

**Publishing System (TN-046 to TN-049)**:
- [TN-046: K8s Client README](../../go-app/internal/infrastructure/k8s/README.md)
- [TN-046: K8s Client Requirements](../../tasks/go-migration-analysis/TN-046-k8s-secrets-client/requirements.md)
- [TN-047: Target Discovery README](../../go-app/internal/business/publishing/DISCOVERY_README.md)
- [TN-049: Health Monitoring README](../../go-app/internal/business/publishing/HEALTH_MONITORING_README.md)
- [TN-049: Integration Guide](../../tasks/go-migration-analysis/TN-049-target-health-monitoring/INTEGRATION_GUIDE.md)

**RBAC Documentation (TN-050)**:
- [SECURITY_COMPLIANCE.md](./SECURITY_COMPLIANCE.md) - CIS/PCI-DSS/SOC2 checklist
- [TROUBLESHOOTING_RUNBOOK.md](./TROUBLESHOOTING_RUNBOOK.md) - Detailed troubleshooting
- [Examples Directory](./examples/) - Multi-environment YAML examples

### 10.2 External Standards

**Security Compliance**:
- [CIS Kubernetes Benchmark](https://www.cisecurity.org/benchmark/kubernetes) - Industry standard security benchmark
- [PCI-DSS v4.0](https://www.pcisecuritystandards.org/) - Payment card industry security
- [SOC 2 Type II](https://www.aicpa.org/soc2) - Service organization controls
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework) - Cybersecurity framework

**Kubernetes Official Docs**:
- [RBAC Authorization](https://kubernetes.io/docs/reference/access-authn-authz/rbac/) - Official RBAC docs
- [ServiceAccounts](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/) - ServiceAccount configuration
- [Audit Logging](https://kubernetes.io/docs/tasks/debug/debug-cluster/audit/) - K8s audit logs
- [NetworkPolicies](https://kubernetes.io/docs/concepts/services-networking/network-policies/) - Network isolation

### 10.3 Security Tools

**Static Analysis**:
- [kubesec](https://kubesec.io/) - Security risk analysis for K8s resources
- [Polaris](https://www.fairwinds.com/polaris) - Best practices validation
- [kube-bench](https://github.com/aquasecurity/kube-bench) - CIS benchmark compliance

**Runtime Security**:
- [Falco](https://falco.org/) - Runtime security monitoring
- [OPA Gatekeeper](https://open-policy-agent.github.io/gatekeeper/) - Policy enforcement
- [Trivy](https://aquasecurity.github.io/trivy/) - Vulnerability scanning

**GitOps**:
- [ArgoCD](https://argo-cd.readthedocs.io/) - Declarative GitOps CD
- [Flux](https://fluxcd.io/) - GitOps toolkit

### 10.4 Community Resources

**Kubernetes RBAC Guides**:
- [Kubernetes RBAC Good Practices](https://kubernetes.io/docs/concepts/security/rbac-good-practices/)
- [RBAC Manager](https://github.com/FairwindsOps/rbac-manager) - Simplified RBAC management
- [kubectl-who-can](https://github.com/aquasecurity/kubectl-who-can) - RBAC debugging tool

**Training and Certification**:
- [CKA (Certified Kubernetes Administrator)](https://www.cncf.io/certification/cka/) - Official K8s certification
- [CKS (Certified Kubernetes Security Specialist)](https://www.cncf.io/certification/cks/) - Security certification

### 10.5 Support and Escalation

**Internal Support**:
- **L1 Support (Application Team)**: RBAC configuration questions, deployment issues
- **L2 Support (Platform Team)**: Cluster-level RBAC, advanced troubleshooting
- **L3 Support (Security Team)**: Security reviews, compliance audits, incident response

**External Support**:
- [Kubernetes Slack](https://slack.k8s.io/) - #kubernetes-security channel
- [Stack Overflow](https://stackoverflow.com/questions/tagged/kubernetes-rbac) - RBAC questions
- [GitHub Issues](https://github.com/kubernetes/kubernetes/issues) - Bug reports

---

## Document Metadata

**Version**: 1.0
**Created**: 2025-11-08
**Author**: Platform Team (TN-050 Implementation)
**Last Updated**: 2025-11-08
**Status**: ✅ PRODUCTION-READY
**Review Date**: 2026-02-08 (Quarterly)

**Change Log**:
- 2025-11-08: Initial version (TN-050 Phase 4)

**Maintenance**:
- Review quarterly (every 3 months)
- Update when K8s version changes
- Update when RBAC requirements change
- Update after security incidents

---

**🎉 End of RBAC Configuration Guide**

For detailed troubleshooting, see [TROUBLESHOOTING_RUNBOOK.md](./TROUBLESHOOTING_RUNBOOK.md).

For security compliance, see [SECURITY_COMPLIANCE.md](./SECURITY_COMPLIANCE.md).

For GitOps integration, see [docs/rbac/GITOPS_WORKFLOW.md](../../docs/rbac/GITOPS_WORKFLOW.md).
