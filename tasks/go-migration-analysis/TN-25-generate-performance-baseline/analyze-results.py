#!/usr/bin/env python3
"""
Performance Test Results Analyzer for Alert History Go Service
Analyzes k6 JSON output and generates performance baseline report.
"""

import json
import os
import statistics
import sys
from collections import Counter, defaultdict
from datetime import datetime


def parse_k6_results(filename):
    """Parse k6 JSON results file and extract metrics."""
    metrics = defaultdict(list)
    scenarios = set()

    print(f"📊 Analyzing results from: {filename}")
    file_size = os.path.getsize(filename) / (1024 * 1024)  # MB
    print(f"📁 File size: {file_size:.1f} MB")

    line_count = 0
    with open(filename) as f:
        for line in f:
            line_count += 1
            if line_count % 100000 == 0:
                print(f"   Processed {line_count:,} lines...")

            try:
                data = json.loads(line.strip())
                if data.get("type") == "Point":
                    metric_name = data.get("metric")
                    point_data = data.get("data", {})
                    tags = point_data.get("tags", {})
                    value = point_data.get("value")
                    timestamp = point_data.get("time")

                    if metric_name and value is not None:
                        metrics[metric_name].append(
                            {"value": value, "timestamp": timestamp, "tags": tags}
                        )

                        # Track scenarios
                        if "scenario" in tags:
                            scenarios.add(tags["scenario"])

            except json.JSONDecodeError:
                continue

    print(f"✅ Processed {line_count:,} total lines")
    print(f"🎭 Found scenarios: {', '.join(sorted(scenarios))}")
    print(f"📈 Collected metrics: {', '.join(sorted(metrics.keys()))}")

    return metrics, scenarios


def calculate_percentiles(values, percentiles=[50, 90, 95, 99]):
    """Calculate percentiles for a list of values."""
    if not values:
        return {}

    sorted_values = sorted(values)
    result = {}

    for p in percentiles:
        index = int((p / 100.0) * len(sorted_values))
        if index >= len(sorted_values):
            index = len(sorted_values) - 1
        result[f"p{p}"] = sorted_values[index]

    result["min"] = min(values)
    result["max"] = max(values)
    result["avg"] = statistics.mean(values)
    result["count"] = len(values)

    return result


def analyze_http_metrics(metrics):
    """Analyze HTTP-related metrics."""
    print("\n🌐 HTTP METRICS ANALYSIS")
    print("=" * 50)

    # Request duration analysis
    if "http_req_duration" in metrics:
        durations = [point["value"] for point in metrics["http_req_duration"]]
        duration_stats = calculate_percentiles(durations)

        print("⏱️  Request Duration (ms):")
        print(f"   Average: {duration_stats['avg']:.2f}ms")
        print(f"   Median (p50): {duration_stats['p50']:.2f}ms")
        print(f"   p90: {duration_stats['p90']:.2f}ms")
        print(f"   p95: {duration_stats['p95']:.2f}ms")
        print(f"   p99: {duration_stats['p99']:.2f}ms")
        print(f"   Min: {duration_stats['min']:.2f}ms")
        print(f"   Max: {duration_stats['max']:.2f}ms")
        print(f"   Total requests: {duration_stats['count']:,}")

        # Check thresholds
        if duration_stats["p95"] > 100:
            print(
                f"   ⚠️  WARNING: p95 ({duration_stats['p95']:.2f}ms) exceeds 100ms threshold"
            )
        else:
            print("   ✅ p95 within 100ms threshold")

    # Request rate analysis
    if "http_reqs" in metrics:
        # Group by time windows to calculate RPS
        time_windows = defaultdict(int)
        for point in metrics["http_reqs"]:
            timestamp = point["timestamp"][:19]  # Group by second
            time_windows[timestamp] += point["value"]

        rps_values = list(time_windows.values())
        if rps_values:
            rps_stats = calculate_percentiles(rps_values)
            print("\n📊 Request Rate (RPS):")
            print(f"   Average: {rps_stats['avg']:.1f} RPS")
            print(f"   Peak: {rps_stats['max']} RPS")
            print(f"   Min: {rps_stats['min']} RPS")

    # Error rate analysis
    if "http_req_failed" in metrics:
        failed_requests = [point["value"] for point in metrics["http_req_failed"]]
        total_requests = len(failed_requests)
        failed_count = sum(failed_requests)
        error_rate = (failed_count / total_requests * 100) if total_requests > 0 else 0

        print("\n❌ Error Analysis:")
        print(f"   Total requests: {total_requests:,}")
        print(f"   Failed requests: {failed_count:,}")
        print(f"   Error rate: {error_rate:.3f}%")

        if error_rate > 1.0:
            print(f"   ⚠️  WARNING: Error rate ({error_rate:.3f}%) exceeds 1% threshold")
        else:
            print("   ✅ Error rate within 1% threshold")


def analyze_by_scenario(metrics, scenarios):
    """Analyze metrics broken down by scenario."""
    print("\n🎭 SCENARIO BREAKDOWN")
    print("=" * 50)

    for scenario in sorted(scenarios):
        print(f"\n📋 Scenario: {scenario}")
        print("-" * 30)

        # Filter metrics for this scenario
        scenario_durations = []
        scenario_errors = []

        for point in metrics.get("http_req_duration", []):
            if point["tags"].get("scenario") == scenario:
                scenario_durations.append(point["value"])

        for point in metrics.get("http_req_failed", []):
            if point["tags"].get("scenario") == scenario:
                scenario_errors.append(point["value"])

        if scenario_durations:
            stats = calculate_percentiles(scenario_durations)
            print(f"   Requests: {stats['count']:,}")
            print(f"   Avg duration: {stats['avg']:.2f}ms")
            print(f"   p95 duration: {stats['p95']:.2f}ms")

            if scenario_errors:
                error_rate = sum(scenario_errors) / len(scenario_errors) * 100
                print(f"   Error rate: {error_rate:.3f}%")


def analyze_custom_metrics(metrics):
    """Analyze custom application metrics."""
    print("\n🔧 CUSTOM METRICS ANALYSIS")
    print("=" * 50)

    # Webhook duration (custom metric)
    if "webhook_duration" in metrics:
        durations = [point["value"] for point in metrics["webhook_duration"]]
        stats = calculate_percentiles(durations)

        print("🪝 Webhook Processing Duration:")
        print(f"   Average: {stats['avg']:.2f}ms")
        print(f"   p95: {stats['p95']:.2f}ms")
        print(f"   Max: {stats['max']:.2f}ms")

    # History duration (custom metric)
    if "history_duration" in metrics:
        durations = [point["value"] for point in metrics["history_duration"]]
        stats = calculate_percentiles(durations)

        print("📚 History API Duration:")
        print(f"   Average: {stats['avg']:.2f}ms")
        print(f"   p95: {stats['p95']:.2f}ms")
        print(f"   Max: {stats['max']:.2f}ms")

    # Error rates (custom metrics)
    if "errors" in metrics:
        errors = [point["value"] for point in metrics["errors"]]
        if errors:
            error_rate = sum(errors) / len(errors) * 100
            print(f"❌ Custom Error Rate: {error_rate:.3f}%")


def generate_baseline_report(metrics, scenarios, output_file):
    """Generate performance baseline report."""
    print(f"\n📝 Generating baseline report: {output_file}")

    with open(output_file, "w") as f:
        f.write("# Alert History Go Service - Performance Baseline Report\n\n")
        f.write(f"**Generated:** {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n\n")

        # Summary
        f.write("## 📊 Executive Summary\n\n")

        if "http_req_duration" in metrics:
            durations = [point["value"] for point in metrics["http_req_duration"]]
            stats = calculate_percentiles(durations)

            f.write(f"- **Total Requests:** {stats['count']:,}\n")
            f.write(f"- **Average Response Time:** {stats['avg']:.2f}ms\n")
            f.write(f"- **95th Percentile:** {stats['p95']:.2f}ms\n")
            f.write(f"- **99th Percentile:** {stats['p99']:.2f}ms\n")

            # Threshold compliance
            if stats["p95"] <= 100:
                f.write(
                    f"- **p95 Threshold:** ✅ PASS ({stats['p95']:.2f}ms ≤ 100ms)\n"
                )
            else:
                f.write(
                    f"- **p95 Threshold:** ❌ FAIL ({stats['p95']:.2f}ms > 100ms)\n"
                )

        if "http_req_failed" in metrics:
            failed_requests = [point["value"] for point in metrics["http_req_failed"]]
            total_requests = len(failed_requests)
            failed_count = sum(failed_requests)
            error_rate = (
                (failed_count / total_requests * 100) if total_requests > 0 else 0
            )

            f.write(f"- **Error Rate:** {error_rate:.3f}%\n")

            if error_rate <= 1.0:
                f.write(f"- **Error Threshold:** ✅ PASS ({error_rate:.3f}% ≤ 1%)\n")
            else:
                f.write(f"- **Error Threshold:** ❌ FAIL ({error_rate:.3f}% > 1%)\n")

        # Detailed metrics
        f.write("\n## 📈 Detailed Metrics\n\n")

        # HTTP metrics table
        if "http_req_duration" in metrics:
            durations = [point["value"] for point in metrics["http_req_duration"]]
            stats = calculate_percentiles(durations)

            f.write("### Response Time Distribution\n\n")
            f.write("| Metric | Value |\n")
            f.write("|--------|-------|\n")
            f.write(f"| Min | {stats['min']:.2f}ms |\n")
            f.write(f"| Average | {stats['avg']:.2f}ms |\n")
            f.write(f"| Median (p50) | {stats['p50']:.2f}ms |\n")
            f.write(f"| p90 | {stats['p90']:.2f}ms |\n")
            f.write(f"| p95 | {stats['p95']:.2f}ms |\n")
            f.write(f"| p99 | {stats['p99']:.2f}ms |\n")
            f.write(f"| Max | {stats['max']:.2f}ms |\n")
            f.write(f"| Total Requests | {stats['count']:,} |\n\n")

        # Scenario breakdown
        f.write("### Scenario Performance\n\n")
        f.write("| Scenario | Requests | Avg Duration | p95 Duration | Error Rate |\n")
        f.write("|----------|----------|--------------|--------------|------------|\n")

        for scenario in sorted(scenarios):
            scenario_durations = []
            scenario_errors = []

            for point in metrics.get("http_req_duration", []):
                if point["tags"].get("scenario") == scenario:
                    scenario_durations.append(point["value"])

            for point in metrics.get("http_req_failed", []):
                if point["tags"].get("scenario") == scenario:
                    scenario_errors.append(point["value"])

            if scenario_durations:
                stats = calculate_percentiles(scenario_durations)
                error_rate = (
                    (sum(scenario_errors) / len(scenario_errors) * 100)
                    if scenario_errors
                    else 0
                )

                f.write(
                    f"| {scenario} | {stats['count']:,} | {stats['avg']:.2f}ms | {stats['p95']:.2f}ms | {error_rate:.3f}% |\n"
                )

        # Recommendations
        f.write("\n## 🎯 Performance Baseline & Recommendations\n\n")

        if "http_req_duration" in metrics:
            durations = [point["value"] for point in metrics["http_req_duration"]]
            stats = calculate_percentiles(durations)

            f.write("### Established Baselines\n\n")
            f.write(
                f"- **Target p95 Response Time:** {stats['p95']:.0f}ms (current baseline)\n"
            )
            f.write(
                f"- **Target p99 Response Time:** {stats['p99']:.0f}ms (current baseline)\n"
            )
            f.write("- **Target Error Rate:** < 1%\n")
            f.write("- **Sustainable RPS:** Based on test scenarios\n\n")

            f.write("### Recommendations\n\n")

            if stats["p95"] > 100:
                f.write(
                    "- ⚠️ **Performance Issue:** p95 response time exceeds 100ms threshold\n"
                )
                f.write("  - Consider optimizing webhook processing logic\n")
                f.write("  - Review database query performance\n")
                f.write("  - Add connection pooling optimization\n")
            else:
                f.write(
                    "- ✅ **Good Performance:** Response times within acceptable limits\n"
                )

            if stats["p99"] > 500:
                f.write("- ⚠️ **Tail Latency:** p99 response time is high\n")
                f.write("  - Investigate outlier requests\n")
                f.write("  - Consider request timeout optimization\n")

            f.write(
                "- 🔧 **Monitoring:** Set up alerts for p95 > 150ms and error rate > 2%\n"
            )
            f.write(
                "- 📊 **Regular Testing:** Run performance tests weekly during development\n"
            )

        f.write("\n---\n")
        f.write("*Report generated by Alert History Performance Analyzer*\n")


def main():
    if len(sys.argv) != 2:
        print("Usage: python3 analyze-results.py <k6-results.json>")
        sys.exit(1)

    results_file = sys.argv[1]

    if not os.path.exists(results_file):
        print(f"❌ Error: File {results_file} not found")
        sys.exit(1)

    print("🚀 Alert History Go Service - Performance Analysis")
    print("=" * 60)

    # Parse results
    metrics, scenarios = parse_k6_results(results_file)

    if not metrics:
        print("❌ No metrics found in results file")
        sys.exit(1)

    # Analyze metrics
    analyze_http_metrics(metrics)
    analyze_by_scenario(metrics, scenarios)
    analyze_custom_metrics(metrics)

    # Generate baseline report
    output_file = results_file.replace(".json", "_baseline_report.md")
    generate_baseline_report(metrics, scenarios, output_file)

    print("\n✅ Analysis complete!")
    print(f"📄 Baseline report saved to: {output_file}")


if __name__ == "__main__":
    main()
