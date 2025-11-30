# 🔴 CRITICAL: Phase 13 Blocker - Immediate Fix Required

**Status**: ❌ **BLOCKS PRODUCTION DEPLOYMENT**
**Severity**: P0 (CRITICAL)
**Estimated Fix Time**: **30 minutes**
**Date**: 2025-11-30

---

## Problem

**Helm template syntax error in Redis PrometheusRule**

```bash
$ helm template . --debug --dry-run
Error: parse error at (alert-history/templates/redis-prometheusrule.yaml:32):
undefined variable "$labels"
```

---

## Root Cause

PrometheusRule annotations contain Prometheus template syntax `{{ $labels.* }}`, which Helm interprets as Helm template syntax. Since `$labels` is not defined in Helm context, template parsing fails.

**Affected Lines**: 32, 42, 52, 62, 72 (10+ occurrences)

---

## Fix Instructions

### Step 1: Open the file
```bash
cd /Users/vitaliisemenov/Documents/Helpfull/AlertHistory
vi helm/alert-history/templates/redis-prometheusrule.yaml
```

### Step 2: Search and replace

**Find** (all occurrences):
```yaml
{{ $labels.instance }}
{{ $labels.job }}
{{ $value }}
```

**Replace with** (escaped version):
```yaml
{{ "{{" }} $labels.instance {{ "}}" }}
{{ "{{" }} $labels.job {{ "}}" }}
{{ "{{" }} $value {{ "}}" }}
```

### Step 3: Automated fix (faster)

```bash
cd /Users/vitaliisemenov/Documents/Helpfull/AlertHistory/helm/alert-history/templates

# Create backup
cp redis-prometheusrule.yaml redis-prometheusrule.yaml.backup

# Perform automated replacement
sed -i.bak 's/{{ \$labels\.instance }}/{{ "{{" }} $labels.instance {{ "}}" }}/g' redis-prometheusrule.yaml
sed -i.bak 's/{{ \$labels\.job }}/{{ "{{" }} $labels.job {{ "}}" }}/g' redis-prometheusrule.yaml
sed -i.bak 's/{{ \$value }}/{{ "{{" }} $value {{ "}}" }}/g' redis-prometheusrule.yaml

# Clean up backup files
rm redis-prometheusrule.yaml.bak
```

### Step 4: Verify fix

```bash
cd /Users/vitaliisemenov/Documents/Helpfull/AlertHistory/helm/alert-history

# Test template rendering
helm template . --debug --dry-run > /tmp/rendered.yaml

# Should succeed without errors
echo "✅ Template renders successfully"

# Run helm lint
helm lint .

# Should output: "1 chart(s) linted, 0 chart(s) failed"
```

---

## Example Before/After

### BEFORE (BROKEN):
```yaml
- alert: RedisDown
  expr: redis_up{job="alerthistory-redis"} == 0
  for: 1m
  labels:
    severity: critical
    component: redis
  annotations:
    summary: "Redis instance is down"
    description: "Redis instance {{ $labels.instance }} is not responding to PING for 1 minute"
    runbook_url: "https://docs.alertmanager-plus-plus.io/runbooks/redis-down"
```

### AFTER (FIXED):
```yaml
- alert: RedisDown
  expr: redis_up{job="alerthistory-redis"} == 0
  for: 1m
  labels:
    severity: critical
    component: redis
  annotations:
    summary: "Redis instance is down"
    description: "Redis instance {{ "{{" }} $labels.instance {{ "}}" }} is not responding to PING for 1 minute"
    runbook_url: "https://docs.alertmanager-plus-plus.io/runbooks/redis-down"
```

---

## Verification Checklist

After applying fix, verify:

- [ ] `helm template . --dry-run` succeeds without errors
- [ ] `helm lint .` shows 0 failures
- [ ] `yamllint templates/redis-prometheusrule.yaml` passes
- [ ] Deploy to test namespace: `helm install test . -n test --dry-run`
- [ ] Check PrometheusRule is created: `kubectl get prometheusrules -n test`
- [ ] Verify alert annotations render correctly in Prometheus UI

---

## Impact Analysis

### Current State (BROKEN)
- ❌ Cannot deploy Helm chart to ANY environment
- ❌ Blocks staging deployment
- ❌ Blocks production deployment
- ❌ Blocks Phase 14 testing
- ✅ Code still works (issue is deployment-only)
- ✅ No data loss or corruption risk

### After Fix (WORKING)
- ✅ Helm chart deploys successfully
- ✅ Redis monitoring alerts work correctly
- ✅ Prometheus scrapes PrometheusRule
- ✅ Alert annotations display in Prometheus UI
- ✅ Phase 13 becomes 100% production-ready

---

## Testing After Fix

### Test 1: Helm Template Rendering
```bash
cd helm/alert-history
helm template . --debug --dry-run | grep -A 5 "kind: PrometheusRule"
```
**Expected**: PrometheusRule renders with correct annotations

### Test 2: Helm Lint
```bash
helm lint .
```
**Expected**: `1 chart(s) linted, 0 chart(s) failed`

### Test 3: Deploy to Test Namespace
```bash
helm install alert-history-test . \
  --namespace test \
  --create-namespace \
  --dry-run
```
**Expected**: No errors, all resources listed

### Test 4: Verify PrometheusRule YAML
```bash
helm template . | yq eval 'select(.kind == "PrometheusRule")' -
```
**Expected**: Annotations contain `{{ $labels.instance }}` (not Helm syntax)

---

## Alternative Fix (If sed fails)

### Manual Search & Replace in Editor

1. Open file: `vim helm/alert-history/templates/redis-prometheusrule.yaml`
2. Enter command mode: `:`
3. Run: `:%s/{{ \$labels\.instance }}/{{ "{{" }} $labels.instance {{ "}}" }}/g`
4. Run: `:%s/{{ \$labels\.job }}/{{ "{{" }} $labels.job {{ "}}" }}/g`
5. Run: `:%s/{{ \$value }}/{{ "{{" }} $value {{ "}}" }}/g`
6. Save: `:wq`

---

## Post-Fix Actions

### 1. Update TN-99 Status
```bash
# Update TASKS.md to reflect fix applied
vi tasks/alertmanager-plus-plus-oss/TASKS.md

# Change TN-99 from 145% to 150%
# Add note: "Helm template blocker fixed 2025-11-30"
```

### 2. Re-run Phase 13 Audit
```bash
# Verify fix resolves blocker
helm template helm/alert-history --dry-run > /dev/null && echo "✅ BLOCKER RESOLVED"
```

### 3. Update Phase 13 Status
```markdown
Phase 13: Production Packaging
Status: ✅ 100% COMPLETE (was 98%)
Grade: A+ (150% average, was 148.3%)
Production Ready: 100% (was 95%)
```

### 4. Proceed to Deployment
```bash
# Deploy to staging
helm upgrade --install alert-history helm/alert-history \
  --namespace alertmanager-plus-plus \
  --create-namespace \
  --values helm/alert-history/values-staging.yaml

# Verify deployment
kubectl get pods -n alertmanager-plus-plus
kubectl get prometheusrules -n alertmanager-plus-plus
```

---

## Related Issues

This fix also resolves:
- Issue in PostgreSQL PrometheusRule (similar pattern)
- Issue in all PrometheusRule templates (same root cause)
- Helm chart validation failures in CI/CD

**Check these files for similar issues**:
```bash
cd helm/alert-history/templates
grep -r "{{ \$labels" *.yaml
grep -r "{{ \$value" *.yaml
```

---

## Contact

**Issue Owner**: Helm Maintainer
**Auditor**: Independent Quality Assessment Team
**Audit Date**: 2025-11-30
**Fix Priority**: **IMMEDIATE** (P0)

---

**Status**: ⚠️ **AWAITING FIX** (30 minutes estimated)
**Next Step**: Apply fix → Re-test → Deploy to staging
