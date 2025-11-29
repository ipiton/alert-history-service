# TN-97: HPA Configuration - Implementation Tasks

**Task ID:** TN-97
**Phase:** 13 - Production Packaging
**Date:** 2025-11-29
**Quality Target:** 150% (Grade A+)
**Duration:** Estimated 8-10 hours

---

## 🎯 Implementation Roadmap

### Overview

Complete implementation of Horizontal Pod Autoscaler (HPA) for Alertmanager++ OSS Core **Standard Profile** with 150% Enterprise Quality.

**Total Phases:** 10
**Estimated Duration:** 8-10 hours
**Quality Target:** 150% (Grade A+)

---

## 📋 Phase Breakdown

| Phase | Name | Duration | Status |
|-------|------|----------|--------|
| 0 | Comprehensive Analysis | 2h | ✅ COMPLETE |
| 1 | HPA Template Implementation | 1h | ⏳ TODO |
| 2 | Template Testing & Validation | 1h | ⏳ TODO |
| 3 | Integration Testing (K8s) | 2h | ⏳ TODO |
| 4 | Load Testing & Performance | 2h | ⏳ TODO |
| 5 | Monitoring & Alerting Setup | 1h | ⏳ TODO |
| 6 | Documentation | 2h | ⏳ TODO |
| 7 | Quality Assurance | 1h | ⏳ TODO |
| 8 | Production Readiness | 1h | ⏳ TODO |
| 9 | Final Review & Certification | 1h | ⏳ TODO |

---

## ✅ Phase 0: Comprehensive Analysis (2h) [COMPLETE]

### Objectives
- [x] Read existing Helm chart configuration
- [x] Analyze TN-96, TN-200, TN-201 implementations
- [x] Understand Standard vs Lite profile architecture
- [x] Identify requirements and constraints
- [x] Create comprehensive requirements.md
- [x] Create detailed design.md
- [x] Create this tasks.md

### Deliverables
- [x] `requirements.md` (18 sections, comprehensive)
- [x] `design.md` (20+ sections, architecture + diagrams)
- [x] `tasks.md` (this file)

### Status
✅ **COMPLETE** - All analysis documents created

---

## ⏳ Phase 1: HPA Template Implementation (1h)

### Objectives
- [ ] Create `helm/alert-history/templates/hpa.yaml`
- [ ] Implement profile-aware conditional logic
- [ ] Configure resource-based metrics (CPU, Memory)
- [ ] Configure custom metrics (Pods metrics)
- [ ] Implement scaling policies (scale-up, scale-down)
- [ ] Set replica bounds (minReplicas, maxReplicas)

### Implementation Checklist

#### 1.1 Create HPA Template File
```bash
- [ ] Create file: helm/alert-history/templates/hpa.yaml
- [ ] Add file header with description
- [ ] Add conditional rendering: {{- if and .Values.autoscaling.enabled (eq .Values.profile "standard") }}
- [ ] Add closing conditional: {{- end }}
```

#### 1.2 HPA Metadata
```yaml
- [ ] apiVersion: autoscaling/v2
- [ ] kind: HorizontalPodAutoscaler
- [ ] metadata.name: {{ include "alert-history.fullname" . }}
- [ ] metadata.namespace: {{ .Values.namespace | default .Release.Namespace }}
- [ ] metadata.labels: {{- include "alert-history.labels" . | nindent 4 }}
- [ ] metadata.annotations: description, managed-by
```

#### 1.3 HPA Spec - Basic Configuration
```yaml
- [ ] spec.scaleTargetRef.apiVersion: apps/v1
- [ ] spec.scaleTargetRef.kind: Deployment
- [ ] spec.scaleTargetRef.name: {{ include "alert-history.fullname" . }}
- [ ] spec.minReplicas: {{ .Values.autoscaling.minReplicas | default 2 }}
- [ ] spec.maxReplicas: {{ .Values.autoscaling.maxReplicas | default 10 }}
```

#### 1.4 HPA Spec - Resource Metrics
```yaml
- [ ] CPU metric type: Resource
- [ ] CPU resource.name: cpu
- [ ] CPU target.type: Utilization
- [ ] CPU target.averageUtilization: {{ .Values.autoscaling.targetCPUUtilizationPercentage }}
- [ ] Conditional: {{- if .Values.autoscaling.targetCPUUtilizationPercentage }}

- [ ] Memory metric type: Resource
- [ ] Memory resource.name: memory
- [ ] Memory target.type: Utilization
- [ ] Memory target.averageUtilization: {{ .Values.autoscaling.targetMemoryUtilizationPercentage }}
- [ ] Conditional: {{- if .Values.autoscaling.targetMemoryUtilizationPercentage }}
```

#### 1.5 HPA Spec - Custom Metrics
```yaml
- [ ] Custom metrics conditional: {{- if .Values.autoscaling.customMetrics.enabled }}

- [ ] Metric 1: alert_history_api_requests_per_second
  - [ ] type: Pods
  - [ ] metric.name: alert_history_api_requests_per_second
  - [ ] target.type: AverageValue
  - [ ] target.averageValue: {{ .Values.autoscaling.customMetrics.requestsPerSecond | quote }}
  - [ ] Conditional: {{- if .Values.autoscaling.customMetrics.requestsPerSecond }}

- [ ] Metric 2: alert_history_business_classification_queue_size
  - [ ] type: Pods
  - [ ] metric.name: alert_history_business_classification_queue_size
  - [ ] target.type: AverageValue
  - [ ] target.averageValue: {{ .Values.autoscaling.customMetrics.classificationQueueSize | quote }}
  - [ ] Conditional: {{- if .Values.autoscaling.customMetrics.classificationQueueSize }}

- [ ] Metric 3: alert_history_business_publishing_queue_size
  - [ ] type: Pods
  - [ ] metric.name: alert_history_business_publishing_queue_size
  - [ ] target.type: AverageValue
  - [ ] target.averageValue: {{ .Values.autoscaling.customMetrics.publishingQueueSize | quote }}
  - [ ] Conditional: {{- if .Values.autoscaling.customMetrics.publishingQueueSize }}
```

#### 1.6 HPA Spec - Scaling Behavior
```yaml
- [ ] Behavior conditional: {{- with .Values.autoscaling.behavior }}

- [ ] Scale-down behavior:
  - [ ] stabilizationWindowSeconds: {{ .scaleDown.stabilizationWindowSeconds | default 300 }}
  - [ ] Policy 1 (Percent): type=Percent, value={{ .scaleDown.percentPolicy }}, periodSeconds={{ .scaleDown.periodSeconds | default 60 }}
  - [ ] Policy 2 (Pods): type=Pods, value={{ .scaleDown.podsPolicy }}, periodSeconds={{ .scaleDown.periodSeconds | default 60 }}
  - [ ] selectPolicy: Min
  - [ ] Conditionals: {{- if .scaleDown }}

- [ ] Scale-up behavior:
  - [ ] stabilizationWindowSeconds: {{ .scaleUp.stabilizationWindowSeconds | default 60 }}
  - [ ] Policy 1 (Percent): type=Percent, value={{ .scaleUp.percentPolicy }}, periodSeconds={{ .scaleUp.periodSeconds | default 30 }}
  - [ ] Policy 2 (Pods): type=Pods, value={{ .scaleUp.podsPolicy }}, periodSeconds={{ .scaleUp.periodSeconds | default 30 }}
  - [ ] selectPolicy: Max
  - [ ] Conditionals: {{- if .scaleUp }}
```

### Deliverables
- [ ] `helm/alert-history/templates/hpa.yaml` (fully implemented)
- [ ] All template variables properly referenced
- [ ] All conditionals correctly implemented

### Validation
```bash
# Syntax validation
- [ ] helm template ./helm/alert-history --show-only templates/hpa.yaml
- [ ] No YAML syntax errors
- [ ] No Helm template errors
```

---

## ⏳ Phase 2: Template Testing & Validation (1h)

### Objectives
- [ ] Validate Helm template rendering
- [ ] Test profile-aware behavior
- [ ] Test configuration variations
- [ ] Run Helm lint
- [ ] Fix any issues

### Test Cases

#### 2.1 Profile-Aware Rendering Tests
```bash
# Test 1: Lite profile - No HPA
- [ ] helm template ./helm/alert-history --set profile=lite --show-only templates/hpa.yaml
- [ ] Expected: Empty output (HPA not rendered)

# Test 2: Standard profile - HPA rendered
- [ ] helm template ./helm/alert-history --set profile=standard --show-only templates/hpa.yaml
- [ ] Expected: HPA manifest with correct configuration

# Test 3: Standard profile with HPA disabled
- [ ] helm template ./helm/alert-history --set profile=standard --set autoscaling.enabled=false --show-only templates/hpa.yaml
- [ ] Expected: Empty output (HPA not rendered)
```

#### 2.2 Configuration Variation Tests
```bash
# Test 4: Custom replica bounds
- [ ] helm template ./helm/alert-history --set profile=standard --set autoscaling.minReplicas=1 --set autoscaling.maxReplicas=20 --show-only templates/hpa.yaml
- [ ] Expected: HPA with minReplicas=1, maxReplicas=20

# Test 5: Custom CPU threshold
- [ ] helm template ./helm/alert-history --set profile=standard --set autoscaling.targetCPUUtilizationPercentage=60 --show-only templates/hpa.yaml
- [ ] Expected: HPA with CPU target 60%

# Test 6: Disable custom metrics
- [ ] helm template ./helm/alert-history --set profile=standard --set autoscaling.customMetrics.enabled=false --show-only templates/hpa.yaml
- [ ] Expected: HPA with only CPU/Memory metrics

# Test 7: Custom scaling policies
- [ ] helm template ./helm/alert-history --set profile=standard --set autoscaling.behavior.scaleDown.stabilizationWindowSeconds=600 --show-only templates/hpa.yaml
- [ ] Expected: HPA with 10-minute scale-down stabilization
```

#### 2.3 Helm Validation
```bash
# Test 8: Helm lint (all profiles)
- [ ] helm lint ./helm/alert-history --set profile=lite
- [ ] helm lint ./helm/alert-history --set profile=standard
- [ ] Expected: No errors or warnings

# Test 9: Helm template (full chart)
- [ ] helm template ./helm/alert-history --set profile=standard
- [ ] Expected: All templates render correctly, no YAML syntax errors

# Test 10: Dry-run install
- [ ] helm install test-hpa ./helm/alert-history --set profile=standard --dry-run --debug
- [ ] Expected: No errors, all manifests valid
```

### Deliverables
- [ ] All 10 test cases passing
- [ ] Zero Helm lint warnings
- [ ] Zero YAML syntax errors

---

## ⏳ Phase 3: Integration Testing (K8s) (2h)

### Objectives
- [ ] Deploy to test Kubernetes cluster
- [ ] Verify HPA creation and configuration
- [ ] Test manual scaling operations
- [ ] Verify metrics integration
- [ ] Test edge cases

### Prerequisites
```bash
- [ ] Kubernetes cluster available (kind/minikube/dev cluster)
- [ ] Metrics server installed: kubectl get deployment metrics-server -n kube-system
- [ ] kubectl configured and working
- [ ] Helm 3.x installed
```

### Test Cases

#### 3.1 Deployment Tests
```bash
# Test 1: Deploy Lite profile (no HPA expected)
- [ ] helm install test-lite ./helm/alert-history --set profile=lite --namespace test-lite --create-namespace
- [ ] kubectl get hpa -n test-lite
- [ ] Expected: No HPA resource found

# Test 2: Deploy Standard profile (HPA expected)
- [ ] helm install test-standard ./helm/alert-history --set profile=standard --namespace test-standard --create-namespace
- [ ] kubectl get hpa -n test-standard
- [ ] Expected: HPA resource exists

# Test 3: Verify HPA configuration
- [ ] kubectl describe hpa -n test-standard
- [ ] Expected: minReplicas=2, maxReplicas=10, CPU/Memory metrics configured

# Test 4: Check HPA status
- [ ] kubectl get hpa -n test-standard -o yaml
- [ ] Expected: status.currentReplicas exists, status.conditions present
```

#### 3.2 Metrics Integration Tests
```bash
# Test 5: Verify resource metrics available
- [ ] kubectl top pods -n test-standard
- [ ] Expected: CPU and Memory metrics displayed

# Test 6: Check HPA metrics
- [ ] kubectl describe hpa -n test-standard | grep -A5 "Metrics:"
- [ ] Expected: CPU and Memory current values shown

# Test 7: Custom metrics (if Prometheus Adapter available)
- [ ] kubectl get --raw /apis/custom.metrics.k8s.io/v1beta1/namespaces/test-standard/pods/*/alert_history_api_requests_per_second
- [ ] Expected: Metric values returned (or graceful error if adapter not installed)
```

#### 3.3 Scaling Tests
```bash
# Test 8: Initial replica count
- [ ] kubectl get deployment -n test-standard
- [ ] Expected: Replicas=2 (or minReplicas value)

# Test 9: Manual scale (should be reverted by HPA)
- [ ] kubectl scale deployment test-standard-alert-history --replicas=5 -n test-standard
- [ ] sleep 60
- [ ] kubectl get deployment -n test-standard
- [ ] Expected: Replicas reverted to 2 (if load is low)

# Test 10: Check HPA events
- [ ] kubectl get events -n test-standard --field-selector involvedObject.kind=HorizontalPodAutoscaler
- [ ] Expected: Scaling events visible
```

#### 3.4 Edge Case Tests
```bash
# Test 11: HPA with minReplicas=1 (cost-saving mode)
- [ ] helm upgrade test-standard ./helm/alert-history --set profile=standard --set autoscaling.minReplicas=1 -n test-standard
- [ ] kubectl get hpa -n test-standard
- [ ] Expected: minReplicas updated to 1

# Test 12: HPA with very high maxReplicas
- [ ] helm upgrade test-standard ./helm/alert-history --set profile=standard --set autoscaling.maxReplicas=50 -n test-standard
- [ ] kubectl get hpa -n test-standard
- [ ] Expected: maxReplicas updated to 50

# Test 13: Disable HPA (should delete resource)
- [ ] helm upgrade test-standard ./helm/alert-history --set profile=standard --set autoscaling.enabled=false -n test-standard
- [ ] kubectl get hpa -n test-standard
- [ ] Expected: HPA resource deleted, deployment manages replicas directly
```

### Cleanup
```bash
- [ ] helm uninstall test-lite -n test-lite
- [ ] helm uninstall test-standard -n test-standard
- [ ] kubectl delete namespace test-lite test-standard
```

### Deliverables
- [ ] All 13 integration tests passing
- [ ] HPA creates successfully
- [ ] HPA responds to metrics
- [ ] Edge cases handled correctly

---

## ⏳ Phase 4: Load Testing & Performance (2h)

### Objectives
- [ ] Simulate real-world load patterns
- [ ] Verify scale-up behavior (response to load spikes)
- [ ] Verify scale-down behavior (stabilization)
- [ ] Test flapping prevention
- [ ] Measure scaling latency

### Prerequisites
```bash
- [ ] k6 load testing tool installed
- [ ] Test cluster with HPA deployed
- [ ] Monitoring dashboard available (Grafana optional)
```

### Test Scenarios

#### 4.1 Sustained Load Test (Scale-Up)
```bash
# Test 1: Baseline (low load)
- [ ] Deploy with minReplicas=2
- [ ] Run k6: k6 run --vus 10 --duration 5m k6/baseline.js
- [ ] Observe: Replicas stay at 2 (CPU <70%)

# Test 2: Moderate load (should scale up)
- [ ] Run k6: k6 run --vus 100 --duration 10m k6/moderate-load.js
- [ ] Observe: CPU rises above 70%, HPA scales to 3-4 replicas
- [ ] Measure: Time from load spike to new pods ready
- [ ] Expected: <2 minutes

# Test 3: High load (should scale to max)
- [ ] Run k6: k6 run --vus 500 --duration 10m k6/high-load.js
- [ ] Observe: CPU rises above 80%, HPA scales to maxReplicas
- [ ] Measure: Time from load spike to max replicas
- [ ] Expected: <5 minutes
```

#### 4.2 Load Spike Test (Rapid Scale-Up)
```bash
# Test 4: Sudden spike
- [ ] Run k6 spike: k6 run --stage 1m:10 --stage 1m:500 --stage 5m:500 --stage 1m:10 k6/spike.js
- [ ] Observe: HPA responds to spike within 60-90 seconds
- [ ] Observe: Service remains healthy during spike (error rate <1%)
- [ ] Measure: Scale-up latency
- [ ] Expected: <2 minutes from spike to sufficient replicas
```

#### 4.3 Scale-Down Test (Conservative)
```bash
# Test 5: Load decrease
- [ ] Run k6 high load for 10 minutes (scale to 8-10 replicas)
- [ ] Stop load generator
- [ ] Observe: HPA waits 5 minutes (stabilization window)
- [ ] Observe: Gradual scale-down (50% or 2 pods per minute)
- [ ] Measure: Time from load drop to minReplicas
- [ ] Expected: 10-15 minutes (conservative by design)
```

#### 4.4 Flapping Prevention Test
```bash
# Test 6: Oscillating load
- [ ] Run k6: k6 run --stage 2m:100 --stage 2m:50 --stage 2m:100 --stage 2m:50 k6/oscillating.js
- [ ] Observe: Replicas do not fluctuate rapidly
- [ ] Observe: Scale-down is delayed (300s stabilization)
- [ ] Count: Number of scaling events in 10 minutes
- [ ] Expected: <3 scale-up, <2 scale-down (no thrashing)
```

#### 4.5 Custom Metrics Test (if Prometheus Adapter available)
```bash
# Test 7: Custom metric (requests/sec per pod)
- [ ] Configure custom metric: alert_history_api_requests_per_second target=50
- [ ] Run k6: k6 run --rps 300 --duration 5m k6/high-rps.js
- [ ] Observe: 300 RPS / 2 pods = 150 RPS per pod (above 50 target)
- [ ] Observe: HPA scales to ~6 replicas (300 / 50)
- [ ] Expected: Custom metrics trigger scaling
```

### Performance Metrics to Collect

```bash
# Metric 1: Scale-up latency
- [ ] Measure: Time from CPU >70% to new pod Ready
- [ ] Target: <2 minutes

# Metric 2: Scale-down latency
- [ ] Measure: Time from CPU <50% to pod Terminating
- [ ] Target: 5-10 minutes (by design)

# Metric 3: Service availability during scaling
- [ ] Measure: Error rate during scale-up/scale-down
- [ ] Target: <1% errors

# Metric 4: Resource utilization
- [ ] Measure: Average CPU/Memory utilization across all replicas
- [ ] Target: 60-70% (efficient utilization)

# Metric 5: Scaling event frequency
- [ ] Measure: Number of scaling events per hour
- [ ] Target: <6 events/hour (stable)
```

### Deliverables
- [ ] All 7 load tests executed
- [ ] Performance metrics collected
- [ ] Scale-up latency <2 minutes ✅
- [ ] No flapping observed ✅
- [ ] Service availability >99% ✅
- [ ] Load test report (summary + charts)

---

## ⏳ Phase 5: Monitoring & Alerting Setup (1h)

### Objectives
- [ ] Create Grafana dashboard for HPA
- [ ] Define Prometheus alerting rules
- [ ] Document PromQL queries
- [ ] Test alerts

### Implementation

#### 5.1 Grafana Dashboard
```bash
# Create dashboard: HPA Monitoring
- [ ] Panel 1: Current vs Desired Replicas (graph)
  - [ ] Query: kube_hpa_status_current_replicas{hpa="alert-history"}
  - [ ] Query: kube_hpa_status_desired_replicas{hpa="alert-history"}
  - [ ] Query: kube_hpa_spec_min_replicas{hpa="alert-history"}
  - [ ] Query: kube_hpa_spec_max_replicas{hpa="alert-history"}

- [ ] Panel 2: CPU Utilization (%) per pod (graph)
  - [ ] Query: 100 * sum(rate(container_cpu_usage_seconds_total{pod=~"alert-history-.*"}[5m])) by (pod) / sum(kube_pod_container_resource_requests{pod=~"alert-history-.*", resource="cpu"}) by (pod)
  - [ ] Threshold line: 70

- [ ] Panel 3: Memory Utilization (%) per pod (graph)
  - [ ] Query: 100 * sum(container_memory_working_set_bytes{pod=~"alert-history-.*"}) by (pod) / sum(kube_pod_container_resource_requests{pod=~"alert-history-.*", resource="memory"}) by (pod)
  - [ ] Threshold line: 80

- [ ] Panel 4: API Requests per Second per pod (graph)
  - [ ] Query: sum(rate(alert_history_infra_http_requests_total[1m])) by (pod)
  - [ ] Threshold line: 50

- [ ] Panel 5: Scaling Events (stat)
  - [ ] Query: changes(kube_hpa_status_desired_replicas{hpa="alert-history"}[1h])

- [ ] Panel 6: HPA Conditions (stat)
  - [ ] Query: kube_hpa_status_condition{hpa="alert-history"}

- [ ] Export dashboard JSON to: docs/grafana-hpa-dashboard.json
```

#### 5.2 Prometheus Alerting Rules
```yaml
# Create file: prometheus/alerts/hpa-alerts.yaml

- [ ] Alert 1: HPAAtMaxCapacity
  - [ ] Condition: current_replicas >= max_replicas for 5m
  - [ ] Severity: warning
  - [ ] Action: Consider increasing maxReplicas

- [ ] Alert 2: HPAUnableToScale
  - [ ] Condition: condition=AbleToScale, status=false for 5m
  - [ ] Severity: critical
  - [ ] Action: Check metrics-server and HPA config

- [ ] Alert 3: HPAFlapping
  - [ ] Condition: >6 replica changes in 15m
  - [ ] Severity: warning
  - [ ] Action: Check stabilization windows and metrics

- [ ] Alert 4: HPAMetricsUnavailable
  - [ ] Condition: HPA unable to get metrics for 10m
  - [ ] Severity: critical
  - [ ] Action: Check metrics-server availability

- [ ] Alert 5: HPAHighCPUUtilization
  - [ ] Condition: avg CPU >90% across all pods for 10m
  - [ ] Severity: warning
  - [ ] Action: HPA may be at capacity, investigate load
```

#### 5.3 Documentation
```bash
- [ ] Create: docs/HPA_MONITORING.md
  - [ ] Section 1: Grafana dashboard overview
  - [ ] Section 2: PromQL queries (all panels)
  - [ ] Section 3: Alerting rules (all alerts)
  - [ ] Section 4: Interpreting metrics
  - [ ] Section 5: Common scenarios
```

### Deliverables
- [ ] Grafana dashboard JSON exported
- [ ] 5 alerting rules defined
- [ ] HPA_MONITORING.md created
- [ ] All alerts tested (trigger conditions)

---

## ⏳ Phase 6: Documentation (2h)

### Objectives
- [ ] Create operator configuration guide
- [ ] Create troubleshooting guide
- [ ] Create README overview
- [ ] Update main project documentation

### Documents to Create

#### 6.1 Configuration Guide
```bash
- [ ] Create: tasks/TN-97-hpa-configuration/CONFIGURATION_GUIDE.md
  - [ ] Section 1: Overview (what is HPA, why use it)
  - [ ] Section 2: Prerequisites (K8s version, metrics-server)
  - [ ] Section 3: Profile Configuration (Lite vs Standard)
  - [ ] Section 4: Basic Configuration (CPU, Memory thresholds)
  - [ ] Section 5: Custom Metrics Configuration (Prometheus Adapter setup)
  - [ ] Section 6: Scaling Policies (scale-up, scale-down behavior)
  - [ ] Section 7: Replica Bounds (minReplicas, maxReplicas)
  - [ ] Section 8: Configuration Examples (5-6 real-world scenarios)
  - [ ] Section 9: Best Practices (production recommendations)
  - [ ] Section 10: Testing Your Configuration
```

#### 6.2 Troubleshooting Guide
```bash
- [ ] Create: tasks/TN-97-hpa-configuration/TROUBLESHOOTING.md
  - [ ] Section 1: Common Issues
    - [ ] HPA not created (profile check)
    - [ ] HPA shows "unable to get metrics"
    - [ ] HPA shows "unknown" for metrics
    - [ ] HPA scaling too aggressively
    - [ ] HPA not scaling up despite high CPU
    - [ ] HPA flapping (rapid scale-up/scale-down)
  - [ ] Section 2: Debugging Commands
    - [ ] kubectl get hpa
    - [ ] kubectl describe hpa
    - [ ] kubectl get --raw /apis/metrics.k8s.io/v1beta1/nodes
    - [ ] kubectl top pods
    - [ ] kubectl logs deployment/metrics-server -n kube-system
  - [ ] Section 3: Metrics Server Issues
    - [ ] How to check if metrics-server is installed
    - [ ] How to install metrics-server
    - [ ] Common metrics-server errors
  - [ ] Section 4: Custom Metrics Issues
    - [ ] Prometheus Adapter not installed
    - [ ] Custom metrics not available
    - [ ] Incorrect metric names
  - [ ] Section 5: Solutions & Fixes (step-by-step)
```

#### 6.3 README Overview
```bash
- [ ] Create: tasks/TN-97-hpa-configuration/README.md
  - [ ] Section 1: Overview (elevator pitch)
  - [ ] Section 2: Features (what's included)
  - [ ] Section 3: Quick Start (5-minute setup)
  - [ ] Section 4: Architecture (high-level diagram)
  - [ ] Section 5: Configuration (link to CONFIGURATION_GUIDE.md)
  - [ ] Section 6: Monitoring (link to HPA_MONITORING.md)
  - [ ] Section 7: Troubleshooting (link to TROUBLESHOOTING.md)
  - [ ] Section 8: Performance (benchmarks, latency targets)
  - [ ] Section 9: References (links to K8s docs, internal docs)
```

#### 6.4 Update Project Documentation
```bash
- [ ] Update: helm/alert-history/README.md
  - [ ] Add section: HPA Configuration (overview + link to TN-97 docs)
  - [ ] Add example: Enabling HPA for Standard profile
  - [ ] Add warning: HPA requires metrics-server

- [ ] Update: CHANGELOG.md
  - [ ] Add entry: TN-97 HPA Configuration (date, features, quality)

- [ ] Update: tasks/alertmanager-plus-plus-oss/TASKS.md
  - [ ] Mark TN-97 as complete with quality score
  - [ ] Update Phase 13 progress (40% → 60%)
```

### Quality Targets (150%+)
```bash
- [ ] CONFIGURATION_GUIDE.md: >2,000 lines (comprehensive examples)
- [ ] TROUBLESHOOTING.md: >1,500 lines (cover all scenarios)
- [ ] README.md: >500 lines (clear overview)
- [ ] Total documentation: >4,000 lines (150% of baseline)
```

### Deliverables
- [ ] CONFIGURATION_GUIDE.md (2,000+ lines)
- [ ] TROUBLESHOOTING.md (1,500+ lines)
- [ ] README.md (500+ lines)
- [ ] Project docs updated (CHANGELOG, TASKS, helm README)

---

## ⏳ Phase 7: Quality Assurance (1h)

### Objectives
- [ ] Final code review
- [ ] Security audit
- [ ] Performance validation
- [ ] Documentation review

### Checklists

#### 7.1 Code Quality
```bash
- [ ] Helm template syntax correct
- [ ] All variables properly referenced
- [ ] No hardcoded values (use .Values)
- [ ] Proper conditional logic
- [ ] Consistent indentation (2 spaces)
- [ ] Comments for complex logic
```

#### 7.2 Security Review
```bash
- [ ] No sensitive data in templates
- [ ] RBAC permissions minimal (HPA uses built-in)
- [ ] No privilege escalation
- [ ] PodSecurityContext enforced
- [ ] Resource limits defined
```

#### 7.3 Performance Validation
```bash
- [ ] Scale-up latency <2 minutes ✅
- [ ] Scale-down latency 5-10 minutes ✅
- [ ] No flapping observed ✅
- [ ] Service availability >99% ✅
- [ ] Resource utilization 60-70% ✅
```

#### 7.4 Documentation Quality
```bash
- [ ] All code examples tested
- [ ] All kubectl commands tested
- [ ] No typos (run spell-check)
- [ ] All links working
- [ ] Diagrams clear and accurate
- [ ] PromQL queries tested
```

#### 7.5 Test Coverage
```bash
- [ ] Unit tests: 10+ (Helm template tests)
- [ ] Integration tests: 13+ (K8s tests)
- [ ] Load tests: 7+ (performance tests)
- [ ] Total tests: 30+ ✅
- [ ] Test pass rate: 100% ✅
```

### Deliverables
- [ ] QA checklist completed
- [ ] All tests passing
- [ ] Zero linter warnings
- [ ] Documentation spell-checked

---

## ⏳ Phase 8: Production Readiness (1h)

### Objectives
- [ ] Create deployment runbook
- [ ] Define rollback procedure
- [ ] Create production checklist
- [ ] Verify backward compatibility

### Implementation

#### 8.1 Deployment Runbook
```bash
- [ ] Create: tasks/TN-97-hpa-configuration/DEPLOYMENT_RUNBOOK.md
  - [ ] Section 1: Pre-Deployment Checklist
    - [ ] Verify K8s version ≥1.23
    - [ ] Verify metrics-server installed
    - [ ] Verify resource requests defined
    - [ ] Run helm lint and helm template
  - [ ] Section 2: Staging Deployment
    - [ ] Deploy to staging namespace
    - [ ] Verify HPA created
    - [ ] Run smoke tests
    - [ ] Monitor for 24 hours
  - [ ] Section 3: Production Deployment
    - [ ] Deploy during low-traffic window
    - [ ] Monitor HPA for first hour
    - [ ] Verify scaling behavior
    - [ ] Collect baseline metrics
  - [ ] Section 4: Post-Deployment
    - [ ] Monitor for 7 days
    - [ ] Analyze scaling patterns
    - [ ] Tune thresholds if needed
    - [ ] Document learnings
```

#### 8.2 Rollback Procedure
```bash
- [ ] Document rollback steps:
  1. [ ] Disable HPA: helm upgrade ... --set autoscaling.enabled=false
  2. [ ] Set fixed replicas: helm upgrade ... --set replicaCount=5
  3. [ ] Wait for rollout: kubectl rollout status deployment/...
  4. [ ] Verify service health
  5. [ ] Investigate HPA issues
```

#### 8.3 Production Checklist
```bash
- [ ] Create: Production Readiness Checklist (40+ items)
  - [ ] Infrastructure
    - [ ] Kubernetes 1.23+ ✅
    - [ ] Metrics server installed ✅
    - [ ] Prometheus available ✅
    - [ ] Grafana available (optional) ✅
  - [ ] Configuration
    - [ ] Profile set to "standard" ✅
    - [ ] autoscaling.enabled=true ✅
    - [ ] Resource requests defined ✅
    - [ ] Replica bounds configured ✅
    - [ ] Scaling policies tuned ✅
  - [ ] Testing
    - [ ] Helm lint passed ✅
    - [ ] Integration tests passed ✅
    - [ ] Load tests passed ✅
    - [ ] Performance targets met ✅
  - [ ] Monitoring
    - [ ] Grafana dashboard deployed ✅
    - [ ] Alerting rules configured ✅
    - [ ] Runbook accessible ✅
  - [ ] Documentation
    - [ ] Configuration guide complete ✅
    - [ ] Troubleshooting guide complete ✅
    - [ ] Team trained on HPA operations ✅
```

#### 8.4 Backward Compatibility
```bash
- [ ] Test: Existing deployments unchanged
- [ ] Test: Lite profile unaffected
- [ ] Test: autoscaling.enabled=false works
- [ ] Test: Helm upgrade from previous version
- [ ] Test: Zero downtime during upgrade
```

### Deliverables
- [ ] DEPLOYMENT_RUNBOOK.md created
- [ ] Rollback procedure documented
- [ ] Production checklist (40+ items)
- [ ] Backward compatibility verified

---

## ⏳ Phase 9: Final Review & Certification (1h)

### Objectives
- [ ] Comprehensive final review
- [ ] Calculate quality score
- [ ] Create completion report
- [ ] Obtain sign-off

### Quality Calculation

#### Implementation Quality (40 points)
```bash
- [ ] HPA template complete (10)
- [ ] Profile-aware logic (10)
- [ ] All metrics configured (10)
- [ ] Scaling policies defined (10)
- [ ] Total: /40
```

#### Testing Quality (30 points)
```bash
- [ ] Unit tests (10)
- [ ] Integration tests (10)
- [ ] Load tests (10)
- [ ] Total: /30
```

#### Documentation Quality (20 points)
```bash
- [ ] Configuration guide (7)
- [ ] Troubleshooting guide (7)
- [ ] README & monitoring docs (6)
- [ ] Total: /20
```

#### Production Readiness (10 points)
```bash
- [ ] Deployment runbook (3)
- [ ] Rollback procedure (3)
- [ ] Production checklist (4)
- [ ] Total: /10
```

**Total Quality Score:** /100 points
**Target:** 150+ points (with bonus achievements)

#### Bonus Achievements (+50 points possible)
```bash
- [ ] Advanced scaling policies (+10)
- [ ] Custom metrics support (+10)
- [ ] Comprehensive load testing (+10)
- [ ] Grafana dashboard (+10)
- [ ] Alerting rules (+10)
```

### Completion Report
```bash
- [ ] Create: tasks/TN-97-hpa-configuration/COMPLETION_REPORT.md
  - [ ] Executive Summary
  - [ ] Implementation Details
  - [ ] Test Results
  - [ ] Performance Benchmarks
  - [ ] Quality Score (150%+ target)
  - [ ] Production Readiness Assessment
  - [ ] Lessons Learned
  - [ ] Next Steps
```

### Final Checklist
```bash
- [ ] All 9 phases complete
- [ ] All deliverables created
- [ ] Quality score ≥150%
- [ ] Zero blocking issues
- [ ] Team sign-off obtained
```

### Deliverables
- [ ] COMPLETION_REPORT.md (comprehensive)
- [ ] Quality score: 150%+ (Grade A+)
- [ ] Certification approved

---

## 📊 Summary

### Total Effort
- **Estimated:** 8-10 hours
- **Actual:** _TBD_
- **Efficiency:** _TBD%_

### Deliverables Summary
- **Code:** 1 file (hpa.yaml)
- **Tests:** 30+ test cases
- **Documentation:** 6-7 comprehensive docs (6,000+ lines)
- **Monitoring:** Grafana dashboard + 5 alerts
- **Quality:** 150%+ (Grade A+)

### Next Tasks (Phase 13 continuation)
- **TN-98:** PostgreSQL StatefulSet (parallel with TN-97)
- **TN-99:** Redis StatefulSet (parallel with TN-97)
- **TN-100:** ConfigMaps & Secrets Management

---

**Created:** 2025-11-29
**Author:** AI Assistant
**Status:** In Progress
**Target Completion:** 2025-11-29 EOD
