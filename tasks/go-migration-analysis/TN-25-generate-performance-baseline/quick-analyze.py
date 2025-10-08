#!/usr/bin/env python3
"""
Quick Performance Test Results Analyzer
Analyzes a sample of k6 JSON output for faster processing.
"""

import json
import os
import statistics
import sys
from collections import defaultdict
from datetime import datetime


def quick_analyze_k6_results(filename, sample_lines=50000):
    """Quick analysis of k6 results using sampling."""
    metrics = defaultdict(list)
    scenarios = set()

    print(f"📊 Quick analysis of: {filename}")
    file_size = os.path.getsize(filename) / (1024 * 1024)  # MB
    print(f"📁 File size: {file_size:.1f} MB")
    print(f"🔬 Sampling first {sample_lines:,} lines for quick analysis")

    line_count = 0
    with open(filename) as f:
        for line in f:
            line_count += 1
            if line_count > sample_lines:
                break

            try:
                data = json.loads(line.strip())
                if data.get("type") == "Point":
                    metric_name = data.get("metric")
                    point_data = data.get("data", {})
                    tags = point_data.get("tags", {})
                    value = point_data.get("value")

                    if metric_name and value is not None:
                        metrics[metric_name].append({"value": value, "tags": tags})

                        if "scenario" in tags:
                            scenarios.add(tags["scenario"])

            except json.JSONDecodeError:
                continue

    print(f"✅ Analyzed {line_count:,} lines")
    print(f"🎭 Found scenarios: {', '.join(sorted(scenarios))}")
    print(f"📈 Metrics found: {', '.join(sorted(metrics.keys()))}")

    return metrics, scenarios


def calculate_stats(values):
    """Calculate basic statistics."""
    if not values:
        return {}

    sorted_values = sorted(values)
    n = len(sorted_values)

    return {
        "count": n,
        "min": min(values),
        "max": max(values),
        "avg": statistics.mean(values),
        "median": statistics.median(values),
        "p90": sorted_values[int(0.9 * n)] if n > 0 else 0,
        "p95": sorted_values[int(0.95 * n)] if n > 0 else 0,
        "p99": sorted_values[int(0.99 * n)] if n > 0 else 0,
    }


def main():
    if len(sys.argv) != 2:
        print("Usage: python3 quick-analyze.py <k6-results.json>")
        sys.exit(1)

    results_file = sys.argv[1]

    if not os.path.exists(results_file):
        print(f"❌ Error: File {results_file} not found")
        sys.exit(1)

    print("🚀 Alert History Go Service - Quick Performance Analysis")
    print("=" * 65)

    # Quick analysis
    metrics, scenarios = quick_analyze_k6_results(results_file)

    if not metrics:
        print("❌ No metrics found in sample")
        sys.exit(1)

    # Analyze HTTP request duration
    if "http_req_duration" in metrics:
        durations = [point["value"] for point in metrics["http_req_duration"]]
        stats = calculate_stats(durations)

        print("\n⏱️  HTTP REQUEST DURATION ANALYSIS")
        print("=" * 40)
        print(f"Sample size: {stats['count']:,} requests")
        print(f"Average: {stats['avg']:.2f}ms")
        print(f"Median: {stats['median']:.2f}ms")
        print(f"p90: {stats['p90']:.2f}ms")
        print(f"p95: {stats['p95']:.2f}ms")
        print(f"p99: {stats['p99']:.2f}ms")
        print(f"Min: {stats['min']:.2f}ms")
        print(f"Max: {stats['max']:.2f}ms")

        # Threshold check
        if stats["p95"] > 100:
            print(f"⚠️  WARNING: p95 ({stats['p95']:.2f}ms) > 100ms threshold")
        else:
            print("✅ p95 within 100ms threshold")

    # Analyze error rate
    if "http_req_failed" in metrics:
        failed_requests = [point["value"] for point in metrics["http_req_failed"]]
        total_requests = len(failed_requests)
        failed_count = sum(failed_requests)
        error_rate = (failed_count / total_requests * 100) if total_requests > 0 else 0

        print("\n❌ ERROR RATE ANALYSIS")
        print("=" * 25)
        print(f"Total requests: {total_requests:,}")
        print(f"Failed requests: {failed_count:,}")
        print(f"Error rate: {error_rate:.3f}%")

        if error_rate > 1.0:
            print(f"⚠️  WARNING: Error rate ({error_rate:.3f}%) > 1% threshold")
        else:
            print("✅ Error rate within 1% threshold")

    # Analyze by scenario
    if scenarios:
        print("\n🎭 SCENARIO BREAKDOWN")
        print("=" * 25)

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
                stats = calculate_stats(scenario_durations)
                error_rate = (
                    (sum(scenario_errors) / len(scenario_errors) * 100)
                    if scenario_errors
                    else 0
                )

                print(f"\n📋 {scenario}:")
                print(f"   Requests: {stats['count']:,}")
                print(f"   Avg duration: {stats['avg']:.2f}ms")
                print(f"   p95 duration: {stats['p95']:.2f}ms")
                print(f"   Error rate: {error_rate:.3f}%")

    # Custom metrics
    if "webhook_duration" in metrics:
        durations = [point["value"] for point in metrics["webhook_duration"]]
        stats = calculate_stats(durations)

        print("\n🪝 WEBHOOK PROCESSING")
        print("=" * 25)
        print(f"Average: {stats['avg']:.2f}ms")
        print(f"p95: {stats['p95']:.2f}ms")
        print(f"Max: {stats['max']:.2f}ms")

    # Generate quick baseline report
    output_file = results_file.replace(".json", "_quick_baseline.md")

    with open(output_file, "w") as f:
        f.write("# Alert History Go Service - Quick Performance Baseline\n\n")
        f.write(f"**Generated:** {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n")
        f.write("**Sample Size:** First 50,000 lines of results\n\n")

        if "http_req_duration" in metrics:
            durations = [point["value"] for point in metrics["http_req_duration"]]
            stats = calculate_stats(durations)

            f.write("## 📊 Performance Summary\n\n")
            f.write(f"- **Sample Requests:** {stats['count']:,}\n")
            f.write(f"- **Average Response Time:** {stats['avg']:.2f}ms\n")
            f.write(f"- **95th Percentile:** {stats['p95']:.2f}ms\n")
            f.write(f"- **99th Percentile:** {stats['p99']:.2f}ms\n")

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

        f.write("\n## 🎯 Baseline Recommendations\n\n")

        if "http_req_duration" in metrics:
            durations = [point["value"] for point in metrics["http_req_duration"]]
            stats = calculate_stats(durations)

            f.write(f"- **Target p95:** {stats['p95']:.0f}ms (current baseline)\n")
            f.write(f"- **Target p99:** {stats['p99']:.0f}ms (current baseline)\n")
            f.write(f"- **Monitoring Alert:** p95 > {stats['p95'] * 1.5:.0f}ms\n")
            f.write(f"- **Critical Alert:** p95 > {stats['p95'] * 2:.0f}ms\n")

        f.write(
            "\n*Note: This is a quick analysis based on sample data. Run full analysis for complete results.*\n"
        )

    print("\n✅ Quick analysis complete!")
    print(f"📄 Quick baseline report: {output_file}")


if __name__ == "__main__":
    main()
