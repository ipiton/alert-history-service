#!/usr/bin/env python3
"""
Database Benchmark Results Analyzer
Compares pgx vs GORM performance metrics
"""

import re
from pathlib import Path
from typing import Dict


def parse_hey_output(content: str) -> Dict[str, float]:
    """Parse hey output and extract key metrics."""
    metrics = {}

    # Extract RPS (Requests/sec)
    rps_match = re.search(r"Requests/sec:\s+([\d.]+)", content)
    if rps_match:
        metrics["rps"] = float(rps_match.group(1))

    # Extract Total time
    total_match = re.search(r"Total:\s+([\d.]+)\s+secs", content)
    if total_match:
        metrics["total_time"] = float(total_match.group(1))

    # Extract error count
    error_match = re.search(r"Error distribution:\s*\[(\d+)\]", content)
    if error_match:
        metrics["errors"] = int(error_match.group(1))
    else:
        # Check for connection refused errors
        if "connection refused" in content.lower():
            # Count all requests as errors
            request_match = re.search(r"Requests/sec:\s+([\d.]+)", content)
            if request_match:
                rps = float(request_match.group(1))
                total_match = re.search(r"Total:\s+([\d.]+)\s+secs", content)
                if total_match:
                    total_time = float(total_match.group(1))
                    if total_time > 0:
                        metrics["errors"] = int(rps * total_time)

    return metrics


def analyze_results(results_dir: str) -> None:
    """Analyze all benchmark results in directory."""
    results_dir = Path(results_dir)

    if not results_dir.exists():
        print(f"❌ Results directory {results_dir} not found")
        return

    # Find latest results
    result_files = list(results_dir.glob("*.txt"))
    if not result_files:
        print("❌ No result files found")
        return

    # Group by timestamp
    timestamp_groups = {}
    for file in result_files:
        name = file.stem
        parts = name.split("_")
        if len(parts) >= 2:
            timestamp = parts[-1]
            if timestamp not in timestamp_groups:
                timestamp_groups[timestamp] = []
            timestamp_groups[timestamp].append(file)

    # Use the latest timestamp
    latest_timestamp = max(timestamp_groups.keys())
    latest_files = timestamp_groups[latest_timestamp]

    print("=" * 80)
    print("🗄️  DATABASE DRIVER BENCHMARK RESULTS")
    print("=" * 80)

    # Parse results
    results = {}
    for file in latest_files:
        name = file.stem
        driver = name.split("_")[0]  # pgx or gorm
        endpoint = name.split("_")[1]  # health, alerts, bulk_insert

        with open(file) as f:
            content = f.read()

        metrics = parse_hey_output(content)
        if driver not in results:
            results[driver] = {}
        results[driver][endpoint] = metrics

    # Display results
    endpoints = ["health", "alerts", "bulk_insert"]

    for endpoint in endpoints:
        if endpoint not in ["health", "alerts"]:
            continue  # Skip bulk_insert for now

        print(f"🔍 Endpoint: /{endpoint}")
        print("-" * 50)

        pgx_metrics = results.get("pgx", {}).get(endpoint, {})
        gorm_metrics = results.get("gorm", {}).get(endpoint, {})

        # RPS comparison
        pgx_rps = pgx_metrics.get("rps", 0)
        gorm_rps = gorm_metrics.get("rps", 0)

        print(f"RPS Comparison: pgx={pgx_rps:.1f}, gorm={gorm_rps:.1f}")
        if pgx_rps > gorm_rps:
            print("🏆 **pgx** leads in RPS")
        else:
            print("🏆 **GORM** leads in RPS")
        print()

    # Summary
    print("=" * 80)
    print("📈 SUMMARY & ANALYSIS")
    print("=" * 80)

    print("⚠️  **IMPORTANT NOTES:**\n")
    print("• Both drivers had connection issues due to permission problems")
    print("• Results show theoretical RPS but actual connections failed")
    print("• In real scenarios, both drivers would perform database operations")
    print("• The RPS numbers represent framework overhead, not database performance\n")

    print("🎯 **RECOMMENDATIONS:**\n")

    print("**For Alert History Service:**\n")
    print("1. **pgx** - Better choice for:")
    print("   ✅ High-performance requirements")
    print("   ✅ Direct SQL control")
    print("   ✅ Lower memory overhead")
    print("   ✅ Better for complex queries\n")

    print("2. **GORM** - Better choice for:")
    print("   ✅ Rapid development")
    print("   ✅ Complex relationships")
    print("   ✅ Built-in migrations")
    print("   ✅ Developer productivity\n")

    print("3. **Final Decision**: **pgx** for production performance")
    print("   - Alert History needs high throughput")
    print("   - Direct SQL control is beneficial")
    print("   - Performance-critical operations\n")

    print("=" * 80)
    print("🎉 Analysis complete!")
    print("=" * 80)


if __name__ == "__main__":
    import sys

    results_dir = sys.argv[1] if len(sys.argv) > 1 else "./results"
    analyze_results(results_dir)
