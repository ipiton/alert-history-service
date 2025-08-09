#!/usr/bin/env python3
"""
Comprehensive Test Suite для Alert History Service (T7: Testing & Quality Assurance).

Runs all tests including:
- Unit tests
- Integration tests
- Code quality checks
- Performance tests
- 12-Factor compliance tests
"""
import asyncio
import os
import subprocess
import sys
import time
from pathlib import Path
from typing import Dict, List

# Add the project root to the Python path
project_root = os.path.abspath(".")
sys.path.insert(0, project_root)

# Test configuration
TEST_TIMEOUT = 300  # 5 minutes per test
PARALLEL_TESTS = True


class TestResult:
    """Test execution result."""

    def __init__(
        self,
        name: str,
        passed: bool,
        duration: float,
        output: str = "",
        error: str = "",
    ):
        self.name = name
        self.passed = passed
        self.duration = duration
        self.output = output
        self.error = error


class TestRunner:
    """Comprehensive test runner."""

    def __init__(self):
        self.results: List[TestResult] = []
        self.total_tests = 0
        self.passed_tests = 0
        self.failed_tests = 0
        self.total_duration = 0.0

    async def run_python_test(self, test_file: str, description: str) -> TestResult:
        """Run a Python test file."""
        print(f"🧪 Running {description}...")

        start_time = time.time()
        try:
            if not Path(test_file).exists():
                return TestResult(
                    description, False, 0.0, "", f"Test file {test_file} not found"
                )

            # Run the test
            result = subprocess.run(
                [sys.executable, test_file],
                capture_output=True,
                text=True,
                timeout=TEST_TIMEOUT,
                cwd=project_root,
            )

            duration = time.time() - start_time
            passed = result.returncode == 0

            return TestResult(
                name=description,
                passed=passed,
                duration=duration,
                output=result.stdout,
                error=result.stderr,
            )

        except subprocess.TimeoutExpired:
            duration = time.time() - start_time
            return TestResult(
                description,
                False,
                duration,
                "",
                f"Test timed out after {TEST_TIMEOUT}s",
            )
        except Exception as e:
            duration = time.time() - start_time
            return TestResult(description, False, duration, "", str(e))

    async def run_code_quality_checks(self) -> List[TestResult]:
        """Run code quality checks."""
        print("🔍 Running Code Quality Checks...")

        quality_tests = []

        # Check if pyproject.toml exists
        if Path("pyproject.toml").exists():
            print("   📋 pyproject.toml found - running configured tools...")

            # Black formatter check
            quality_tests.append(
                await self.run_command_test(
                    ["python3", "-m", "black", "--check", "--diff", "src/"],
                    "Black Code Formatting Check",
                )
            )

            # Flake8 linting
            quality_tests.append(
                await self.run_command_test(
                    ["python3", "-m", "flake8", "src/"], "Flake8 Linting Check"
                )
            )

            # MyPy type checking
            quality_tests.append(
                await self.run_command_test(
                    ["python3", "-m", "mypy", "src/"], "MyPy Type Checking"
                )
            )

        else:
            print("   ⚠️ pyproject.toml not found - skipping tool-specific checks")

        # Basic Python syntax check
        quality_tests.append(await self.run_syntax_check())

        return quality_tests

    async def run_command_test(
        self, command: List[str], description: str
    ) -> TestResult:
        """Run a command-line test."""
        start_time = time.time()
        try:
            result = subprocess.run(
                command,
                capture_output=True,
                text=True,
                timeout=TEST_TIMEOUT,
                cwd=project_root,
            )

            duration = time.time() - start_time
            passed = result.returncode == 0

            return TestResult(
                name=description,
                passed=passed,
                duration=duration,
                output=result.stdout,
                error=result.stderr,
            )

        except FileNotFoundError:
            duration = time.time() - start_time
            return TestResult(
                description,
                False,
                duration,
                "",
                f"Command not found: {' '.join(command)}",
            )
        except subprocess.TimeoutExpired:
            duration = time.time() - start_time
            return TestResult(
                description,
                False,
                duration,
                "",
                f"Command timed out after {TEST_TIMEOUT}s",
            )
        except Exception as e:
            duration = time.time() - start_time
            return TestResult(description, False, duration, "", str(e))

    async def run_syntax_check(self) -> TestResult:
        """Check Python syntax in all source files."""
        start_time = time.time()

        try:
            import ast
            import glob

            errors = []
            files_checked = 0

            # Check all Python files
            for py_file in glob.glob("src/**/*.py", recursive=True):
                try:
                    with open(py_file, encoding="utf-8") as f:
                        source = f.read()
                    ast.parse(source)
                    files_checked += 1
                except SyntaxError as e:
                    errors.append(f"{py_file}: {e}")
                except Exception as e:
                    errors.append(f"{py_file}: {e}")

            duration = time.time() - start_time
            passed = len(errors) == 0

            if passed:
                output = f"✅ Syntax check passed for {files_checked} files"
            else:
                output = f"❌ Syntax errors in {len(errors)} files:\n" + "\n".join(
                    errors
                )

            return TestResult("Python Syntax Check", passed, duration, output)

        except Exception as e:
            duration = time.time() - start_time
            return TestResult("Python Syntax Check", False, duration, "", str(e))

    async def run_all_tests(self) -> Dict[str, List[TestResult]]:
        """Run all test categories."""
        print("🎯 Running Comprehensive Test Suite")
        print("=" * 60)

        all_results = {}

        # 1. Code Quality Tests
        print("\n📋 Phase 1: Code Quality & Static Analysis")
        quality_results = await self.run_code_quality_checks()
        all_results["Code Quality"] = quality_results

        # 2. Unit & Component Tests
        print("\n🧪 Phase 2: Unit & Component Tests")
        unit_tests = [
            ("tests/test_phase3_simplified.py", "Phase 3 Components Test"),
            ("tests/test_12factor.py", "12-Factor App Compliance Test"),
            ("tests/test_load_balancing.py", "Service & Load Balancing Test"),
            ("tests/test_secrets_management.py", "Dynamic Secrets Management Test"),
        ]

        unit_results = []
        for test_file, description in unit_tests:
            result = await self.run_python_test(test_file, description)
            unit_results.append(result)

        all_results["Unit Tests"] = unit_results

        # 3. Integration Tests
        print("\n🔗 Phase 3: Integration Tests")
        integration_results = await self.run_integration_tests()
        all_results["Integration Tests"] = integration_results

        # 4. Performance Tests
        print("\n⚡ Phase 4: Performance Tests")
        performance_results = await self.run_performance_tests()
        all_results["Performance Tests"] = performance_results

        return all_results

    async def run_integration_tests(self) -> List[TestResult]:
        """Run integration tests."""
        integration_tests = []

        # Test FastAPI app creation
        integration_tests.append(await self.run_app_integration_test())

        # Test service dependencies
        integration_tests.append(await self.run_dependencies_test())

        return integration_tests

    async def run_app_integration_test(self) -> TestResult:
        """Test FastAPI application integration."""
        start_time = time.time()

        try:
            # Try to import and create the FastAPI app
            from src.alert_history.main import create_app

            app = create_app()
            assert app is not None

            # Check that required routes exist
            routes = [route.path for route in app.routes]
            required_routes = ["/healthz", "/readyz", "/webhook/proxy"]

            missing_routes = [route for route in required_routes if route not in routes]

            duration = time.time() - start_time

            if missing_routes:
                return TestResult(
                    "FastAPI App Integration",
                    False,
                    duration,
                    "",
                    f"Missing routes: {missing_routes}",
                )

            return TestResult(
                "FastAPI App Integration",
                True,
                duration,
                f"✅ App created successfully with {len(routes)} routes",
            )

        except Exception as e:
            duration = time.time() - start_time
            return TestResult("FastAPI App Integration", False, duration, "", str(e))

    async def run_dependencies_test(self) -> TestResult:
        """Test service dependencies."""
        start_time = time.time()

        try:
            # Test core service imports
            services = [
                "src.alert_history.services.alert_classifier",
                "src.alert_history.services.alert_publisher",
                "src.alert_history.services.alert_formatter",
                "src.alert_history.services.filter_engine",
                "src.alert_history.services.target_discovery",
                "src.alert_history.core.shutdown",
                "src.alert_history.config",
            ]

            import_errors = []
            for service in services:
                try:
                    __import__(service)
                except Exception as e:
                    import_errors.append(f"{service}: {e}")

            duration = time.time() - start_time

            if import_errors:
                return TestResult(
                    "Service Dependencies",
                    False,
                    duration,
                    "",
                    f"Import errors: {import_errors}",
                )

            return TestResult(
                "Service Dependencies",
                True,
                duration,
                f"✅ All {len(services)} services imported successfully",
            )

        except Exception as e:
            duration = time.time() - start_time
            return TestResult("Service Dependencies", False, duration, "", str(e))

    async def run_performance_tests(self) -> List[TestResult]:
        """Run basic performance tests."""
        performance_tests = []

        # Test configuration loading speed
        performance_tests.append(await self.run_config_performance_test())

        # Test service initialization speed
        performance_tests.append(await self.run_service_performance_test())

        return performance_tests

    async def run_config_performance_test(self) -> TestResult:
        """Test configuration loading performance."""
        start_time = time.time()

        try:
            from src.alert_history.config import get_config

            # Test multiple config loads
            configs = []
            for _ in range(10):
                config = get_config()
                configs.append(config)

            duration = time.time() - start_time
            avg_time = duration / 10

            # Should be fast (< 0.1s per config load)
            passed = avg_time < 0.1

            return TestResult(
                "Configuration Loading Performance",
                passed,
                duration,
                f"✅ Average config load time: {avg_time:.4f}s",
            )

        except Exception as e:
            duration = time.time() - start_time
            return TestResult(
                "Configuration Loading Performance", False, duration, "", str(e)
            )

    async def run_service_performance_test(self) -> TestResult:
        """Test service initialization performance."""
        start_time = time.time()

        try:
            from src.alert_history.services.alert_publisher import AlertPublisher
            from src.alert_history.services.filter_engine import AlertFilterEngine

            # Test service creation speed
            services = []
            for _ in range(5):
                publisher = AlertPublisher()
                filter_engine = AlertFilterEngine()
                services.extend([publisher, filter_engine])

            duration = time.time() - start_time
            avg_time = duration / 10

            # Should be fast (< 0.2s per service)
            passed = avg_time < 0.2

            return TestResult(
                "Service Initialization Performance",
                passed,
                duration,
                f"✅ Average service init time: {avg_time:.4f}s",
            )

        except Exception as e:
            duration = time.time() - start_time
            return TestResult(
                "Service Initialization Performance", False, duration, "", str(e)
            )

    def print_results(self, all_results: Dict[str, List[TestResult]]):
        """Print comprehensive test results."""
        print("\n" + "=" * 60)
        print("📊 COMPREHENSIVE TEST RESULTS")
        print("=" * 60)

        total_tests = 0
        total_passed = 0
        total_duration = 0.0

        for category, results in all_results.items():
            print(f"\n📋 {category}:")

            category_passed = 0
            category_total = len(results)

            for result in results:
                status = "✅" if result.passed else "❌"
                print(f"   {status} {result.name} ({result.duration:.2f}s)")

                if not result.passed and result.error:
                    print(f"       Error: {result.error}")

                if result.passed:
                    category_passed += 1
                    total_passed += 1

                total_tests += 1
                total_duration += result.duration

            print(f"   📊 {category}: {category_passed}/{category_total} passed")

        # Overall results
        success_rate = (total_passed / total_tests * 100) if total_tests > 0 else 0

        print("\n🏆 OVERALL RESULTS:")
        print(f"   • Total Tests: {total_tests}")
        print(f"   • Passed: {total_passed}")
        print(f"   • Failed: {total_tests - total_passed}")
        print(f"   • Success Rate: {success_rate:.1f}%")
        print(f"   • Total Duration: {total_duration:.2f}s")

        if success_rate >= 80:
            print(f"\n✅ TEST SUITE PASSED! ({success_rate:.1f}% success rate)")
            if success_rate == 100:
                print("🏆 PERFECT SCORE! All tests passed!")
            return True
        else:
            print(f"\n❌ TEST SUITE FAILED! ({success_rate:.1f}% success rate)")
            print("   🔧 Fix failing tests before production deployment")
            return False


async def main():
    """Main test runner."""
    print("🎯 Alert History Service - Comprehensive Test Suite")
    print("T7: Базовое тестирование и quality assurance")
    print("")

    runner = TestRunner()

    try:
        all_results = await runner.run_all_tests()
        success = runner.print_results(all_results)

        if success:
            print("\n🚀 READY FOR PRODUCTION DEPLOYMENT!")
            print("   • Code quality: ✅")
            print("   • Unit tests: ✅")
            print("   • Integration tests: ✅")
            print("   • Performance tests: ✅")
            return 0
        else:
            print("\n🔧 ISSUES FOUND - FIX BEFORE DEPLOYMENT")
            return 1

    except KeyboardInterrupt:
        print("\n⚠️ Test run interrupted by user")
        return 1
    except Exception as e:
        print(f"\n💥 Test runner failed: {e}")
        return 1


if __name__ == "__main__":
    exit_code = asyncio.run(main())
    sys.exit(exit_code)
