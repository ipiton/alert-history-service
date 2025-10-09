#!/usr/bin/env python3

import os
import re
import sys
from pathlib import Path
from typing import Dict, Tuple


class BenchmarkAnalyzer:
    def __init__(self, results_dir: str):
        self.results_dir = Path(results_dir)
        self.results: Dict[str, Dict[str, float]] = {}

    def parse_hey_results(self, file_path: Path) -> Dict[str, float]:
        """Parse hey benchmark results"""
        results = {}

        try:
            with open(file_path) as f:
                content = f.read()

            # Extract RPS
            rps_match = re.search(r"Requests/sec:\s*([\d.]+)", content)
            if rps_match:
                results["rps"] = float(rps_match.group(1))

            # Extract latency percentiles
            latency_matches = re.findall(r"(\d+)\%\s*in\s*([\d.]+)ms", content)
            for percentile, value in latency_matches:
                results[f"latency_p{percentile}"] = float(value)

            # Extract total requests
            total_match = re.search(r"(\d+)\s*requests\s*in", content)
            if total_match:
                results["total_requests"] = int(total_match.group(1))

        except Exception as e:
            print(f"Error parsing {file_path}: {e}")

        return results

    def parse_wrk_results(self, file_path: Path) -> Dict[str, float]:
        """Parse wrk benchmark results"""
        results = {}

        try:
            with open(file_path) as f:
                content = f.read()

            # Extract RPS
            rps_match = re.search(r"Requests/sec:\s*([\d.]+)", content)
            if rps_match:
                results["rps"] = float(rps_match.group(1))

            # Extract latency percentiles
            latency_matches = re.findall(r"(\d+)\%\s*([\d.]+)ms", content)
            for percentile, value in latency_matches:
                results[f"latency_p{percentile}"] = float(value)

        except Exception as e:
            print(f"Error parsing {file_path}: {e}")

        return results

    def collect_results(
        self,
    ) -> Tuple[Dict[str, Dict[str, float]], Dict[str, Dict[str, float]]]:
        """Collect all benchmark results"""
        fiber_results = {}
        gin_results = {}

        for file_path in self.results_dir.glob("*"):
            if not file_path.is_file():
                continue

            filename = file_path.name

            if filename.startswith("fiber_"):
                if "sustained" in filename:
                    fiber_results["sustained"] = self.parse_wrk_results(file_path)
                else:
                    endpoint = (
                        filename.replace("fiber_", "").replace(".txt", "").split("_")[0]
                    )
                    fiber_results[endpoint] = self.parse_hey_results(file_path)

            elif filename.startswith("gin_"):
                if "sustained" in filename:
                    gin_results["sustained"] = self.parse_wrk_results(file_path)
                else:
                    endpoint = (
                        filename.replace("gin_", "").replace(".txt", "").split("_")[0]
                    )
                    gin_results[endpoint] = self.parse_hey_results(file_path)

        return fiber_results, gin_results

    def print_comparison(
        self,
        fiber_results: Dict[str, Dict[str, float]],
        gin_results: Dict[str, Dict[str, float]],
    ) -> None:
        """Print detailed comparison"""
        print("=" * 80)
        print("HTTP FRAMEWORK BENCHMARK RESULTS")
        print("=" * 80)

        endpoints = set(fiber_results.keys()) & set(gin_results.keys())

        for endpoint in sorted(endpoints):
            print(f"\n🔍 Endpoint: /{endpoint}")
            print("-" * 40)

            fiber_data = fiber_results[endpoint]
            gin_data = gin_results[endpoint]

            if "rps" in fiber_data and "rps" in gin_data:
                fiber_rps = fiber_data["rps"]
                gin_rps = gin_data["rps"]

                print(".1f")
                print(".1f")

                if fiber_rps > gin_rps:
                    ratio = fiber_rps / gin_rps
                    print(".2f")
                else:
                    ratio = gin_rps / fiber_rps
                    print(".2f")
                _ = ratio  # Use the variable to avoid linting warning
            # Latency comparison
            if "latency_p50" in fiber_data and "latency_p50" in gin_data:
                print("\nLatency (ms):")
                print("2d" ".2f" ".2f" ".2f")

    def generate_report(self) -> None:
        """Generate comprehensive benchmark report"""
        fiber_results, gin_results = self.collect_results()

        if not fiber_results and not gin_results:
            print("❌ No benchmark results found!")
            return

        print("\n" + "=" * 80)
        print("📊 BENCHMARK ANALYSIS REPORT")
        print("=" * 80)

        self.print_comparison(fiber_results, gin_results)

        print("\n" + "=" * 80)
        print("📈 SUMMARY & RECOMMENDATIONS")
        print("=" * 80)

        # Calculate overall performance
        fiber_rps_total = sum(
            data.get("rps", 0) for data in fiber_results.values() if "rps" in data
        )
        gin_rps_total = sum(
            data.get("rps", 0) for data in gin_results.values() if "rps" in data
        )

        if fiber_rps_total > 0 and gin_rps_total > 0:
            if fiber_rps_total > gin_rps_total:
                ratio = fiber_rps_total / gin_rps_total
                print(".2f")
            else:
                ratio = gin_rps_total / fiber_rps_total
                print(".2f")
            _ = ratio  # Use the variable to avoid linting warning
        print("\n🎯 RECOMMENDATIONS:")
        print("• Fiber v2: Best for high-performance applications")
        print("• Gin: Best for rapid development and ecosystem maturity")
        print("• Consider your specific requirements and team experience")


def main() -> None:
    if len(sys.argv) != 2:
        print("Usage: python3 analyze_results.py <results_directory>")
        sys.exit(1)

    results_dir = sys.argv[1]

    if not os.path.exists(results_dir):
        print(f"❌ Results directory '{results_dir}' does not exist!")
        sys.exit(1)

    analyzer = BenchmarkAnalyzer(results_dir)
    analyzer.generate_report()


if __name__ == "__main__":
    main()
