# 📊 TN-056: Publishing Queue Grafana Dashboard

## 📝 Overview

Comprehensive Grafana dashboard for monitoring the **Publishing Queue** (TN-056) in the Alert History Service. This dashboard provides real-time insights into queue performance, job processing, error tracking, and Dead Letter Queue (DLQ) management.

## 🎯 Dashboard Panels (8 Total)

### 1. Queue Size by Priority
- **Type**: Time Series Graph
- **Metrics**:
  - `alert_history_publishing_queue_size{priority="high"}`
  - `alert_history_publishing_queue_size{priority="medium"}`
  - `alert_history_publishing_queue_size{priority="low"}`
- **Purpose**: Track queue depth across 3 priority tiers
- **Use Case**: Identify bottlenecks, monitor queue saturation

### 2. Job Success Rate
- **Type**: Gauge
- **Metrics**:
  - Success rate: `(completed / (completed + failed)) * 100`
- **Thresholds**:
  - 🔴 Red: < 80%
  - 🟠 Orange: 80-95%
  - 🟢 Green: > 95%
- **Purpose**: Overall publishing health indicator
- **Use Case**: SLA monitoring, alerting trigger

### 3. Active Workers
- **Type**: Stat
- **Metrics**: `alert_history_publishing_active_workers`
- **Purpose**: Track worker pool utilization
- **Use Case**: Capacity planning, horizontal scaling decisions

### 4. Jobs Processed (1h) by Target
- **Type**: Pie Chart
- **Metrics**: `increase(alert_history_publishing_job_completed_total[1h])`
- **Purpose**: Distribution of jobs across publishing targets
- **Use Case**: Identify target-specific issues, load balancing

### 5. Dead Letter Queue Size
- **Type**: Stat with Graph
- **Metrics**: `alert_history_publishing_dlq_size`
- **Thresholds**:
  - 🟢 Green: 0-9 entries
  - 🟠 Orange: 10-49 entries
  - 🔴 Red: ≥50 entries
- **Purpose**: Monitor failed job accumulation
- **Use Case**: Trigger DLQ purge, investigate persistent failures

### 6. Processing Duration Distribution
- **Type**: Heatmap
- **Metrics**: `alert_history_publishing_job_duration_seconds_bucket`
- **Purpose**: Visualize latency distribution over time
- **Use Case**: Identify performance degradation, p95/p99 analysis

### 7. Error Breakdown (1h) by Type
- **Type**: Pie Chart
- **Metrics**: `increase(alert_history_publishing_job_failed_total[1h])`
- **Labels**: `error_type` (transient, permanent, unknown)
- **Purpose**: Categorize failure modes
- **Use Case**: Root cause analysis, prioritize fixes

### 8. Recent Failed Jobs (Top 20 in DLQ)
- **Type**: Table
- **Metrics**: `topk(20, alert_history_publishing_dlq_entries{replayed="false"})`
- **Columns**: Failed At, Target, Error Type, Fingerprint, Retry Count
- **Purpose**: Drill-down into specific failures
- **Use Case**: Manual investigation, replay decisions

## 📦 Installation

### Option 1: Grafana UI Import

1. Open Grafana UI → Dashboards → New → Import
2. Upload `publishing-queue-tn056.json`
3. Select Prometheus data source
4. Click "Import"

### Option 2: Grafana Provisioning (Recommended for Production)

Add to your Grafana provisioning config:

```yaml
# grafana/provisioning/dashboards/alert-history.yml
apiVersion: 1

providers:
  - name: 'Alert History'
    orgId: 1
    folder: 'Alert History'
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /etc/grafana/provisioning/dashboards/alert-history
```

Copy dashboard file:

```bash
cp grafana/dashboards/publishing-queue-tn056.json \
   /etc/grafana/provisioning/dashboards/alert-history/
```

## ⚙️ Configuration

### Required Prometheus Metrics

Ensure these metrics are exposed by your Alert History Service:

```
# Queue size by priority
alert_history_publishing_queue_size{priority="high|medium|low"}

# Job counters
alert_history_publishing_job_completed_total{target="...",priority="..."}
alert_history_publishing_job_failed_total{target="...",error_type="..."}

# Active workers
alert_history_publishing_active_workers

# DLQ size
alert_history_publishing_dlq_size

# Job duration histogram
alert_history_publishing_job_duration_seconds_bucket{le="..."}

# DLQ entries (for table panel)
alert_history_publishing_dlq_entries{replayed="true|false",target="...",error_type="...",fingerprint="..."}
```

### Data Source

- **Default**: `${DS_PROMETHEUS}` variable
- **Fallback**: Configure in Grafana UI → Settings → Data Sources

## 🔧 Customization

### Adjust Refresh Rate

Default: 30 seconds

Change in dashboard settings or query:
```json
"refresh": "30s"  // Options: 10s, 30s, 1m, 5m, 15m, 30m, 1h
```

### Modify Time Range

Default: Last 6 hours

Change in dashboard UI or JSON:
```json
"time": {
  "from": "now-6h",
  "to": "now"
}
```

### Alert Rules (Optional)

Add alerts to panels:

#### Panel 2: Job Success Rate < 90%

```yaml
alert:
  name: "Publishing Queue Success Rate Low"
  for: 5m
  annotations:
    summary: "Publishing success rate below 90% for 5 minutes"
  labels:
    severity: warning
```

#### Panel 5: DLQ Size > 100

```yaml
alert:
  name: "DLQ Size Critical"
  for: 10m
  annotations:
    summary: "Dead Letter Queue has >100 failed jobs"
  labels:
    severity: critical
```

## 📊 Dashboard Features

- **Auto-refresh**: Updates every 30 seconds
- **Time range**: Last 6 hours (configurable)
- **Theme**: Dark mode (Grafana default)
- **Tooltip**: Unified hover (mode: multi)
- **Legend**: Table format with last/max values
- **Tags**: `alert-history`, `publishing`, `TN-056`, `queue`

## 🎯 Use Cases

### 1. Production Monitoring

- Monitor queue health in real-time
- Detect anomalies (spikes, drops, flatlines)
- Track SLA compliance (success rate)

### 2. Performance Tuning

- Analyze processing duration (p50, p95, p99)
- Identify bottlenecks (queue saturation)
- Optimize worker pool size

### 3. Incident Response

- Drill down into recent failures (Panel 8)
- Correlate errors by type (Panel 7)
- Check DLQ for persistent issues (Panel 5)

### 4. Capacity Planning

- Track worker utilization (Panel 3)
- Forecast queue growth
- Plan horizontal scaling

## 🔍 Troubleshooting

### No Data Displayed

1. **Check Prometheus data source**: Settings → Data Sources → Test
2. **Verify metrics**: Prometheus UI → `alert_history_publishing_*`
3. **Check time range**: Expand to wider range (24h)
4. **Review logs**: Alert History Service Prometheus metrics endpoint

### Incomplete Panels

- **Panel 8 (Recent Failed Jobs)**: Requires `alert_history_publishing_dlq_entries` metric
  - This may need custom exporter or scraping DLQ repository
  - Alternative: Use Prometheus Alertmanager API

### Performance Issues

- **Slow queries**: Reduce time range (6h → 1h)
- **High cardinality**: Add label filters (e.g., `target="specific-target"`)
- **Heavy panels**: Disable auto-refresh on complex panels (Panel 6 heatmap)

## 📚 Related Documentation

- [TN-056 Requirements](../../tasks/go-migration-analysis/tasks.md)
- [TN-056 Design Document](../../tasks/go-migration-analysis/TN-056/design.md)
- [TN-056 API Guide](../../tasks/go-migration-analysis/TN-056/API_GUIDE.md)
- [TN-056 Troubleshooting](../../tasks/go-migration-analysis/TN-056/TROUBLESHOOTING.md)
- [Prometheus Metrics Reference](../../go-app/internal/infrastructure/publishing/metrics.go)

## 🚀 Future Enhancements

- [ ] Add variables for filtering by `target`, `priority`, `error_type`
- [ ] Integrate with Alertmanager for panel 8 (DLQ entries)
- [ ] Add trend prediction (ML-based queue growth forecast)
- [ ] Create companion dashboard for individual target deep-dive
- [ ] Add cost tracking (if cloud-based metrics available)

## 📄 License

Part of Alert History Service (TN-056)

---

**Version**: 1.0.0
**Created**: 2025-11-12
**Last Updated**: 2025-11-12
**Author**: TN-056 Implementation Team



