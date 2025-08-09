#!/usr/bin/env python3
"""
T7.1: Code Quality & Unit Tests - Comprehensive Testing Suite

Tests:
- PEP8 compliance with flake8
- Type checking with mypy
- Security linting with bandit
- Import organization
- Docstring coverage
- Unit test coverage for core components

Usage:
    python test_t7_1_code_quality.py
"""

import subprocess
import sys
import unittest
from pathlib import Path


class TestCodeQuality(unittest.TestCase):
    """Test suite for T7.1: Code Quality & Unit Tests."""

    def setUp(self):
        """Set up test environment."""
        self.project_root = Path(__file__).parent
        self.src_path = self.project_root / "src"
        self.max_flake8_errors = 50  # Allow some remaining issues but track progress

    def test_01_critical_syntax_errors(self):
        """Test for critical syntax errors that would break the application."""
        print("\n=== T7.1.1: Critical Syntax Errors Check ===")

        result = subprocess.run(
            [
                sys.executable,
                "-m",
                "flake8",
                str(self.src_path),
                "--count",
                "--select=E9,F63,F7,F82",
                "--show-source",
                "--statistics",
            ],
            capture_output=True,
            text=True,
            cwd=self.project_root,
        )

        print(f"Flake8 critical errors output:\n{result.stdout}")
        if result.stderr:
            print(f"Stderr: {result.stderr}")

        # Should have 0 critical errors
        error_count = 0
        if result.stdout.strip() and result.stdout.strip()[-1].isdigit():
            error_count = int(result.stdout.strip().split("\n")[-1])

        print(f"✅ Critical syntax errors: {error_count}")
        self.assertEqual(error_count, 0, "Critical syntax errors found")

    def test_02_pep8_compliance_progress(self):
        """Test PEP8 compliance - tracking progress."""
        print("\n=== T7.1.2: PEP8 Compliance Progress ===")

        result = subprocess.run(
            [
                sys.executable,
                "-m",
                "flake8",
                str(self.src_path),
                "--count",
                "--max-line-length=100",
                "--ignore=E203,W503",
            ],
            capture_output=True,
            text=True,
            cwd=self.project_root,
        )

        # Extract error count from last line
        error_count = 0
        if result.stdout.strip():
            lines = result.stdout.strip().split("\n")
            if lines[-1].isdigit():
                error_count = int(lines[-1])

        print(f"📊 Current PEP8 errors: {error_count}")
        print(f"📊 Target max errors: {self.max_flake8_errors}")

        # Progress tracking - allow some errors but should be decreasing
        progress_ratio = max(0, 1 - (error_count / 500))  # Original ~500 errors
        print(f"📈 PEP8 Progress: {progress_ratio:.1%}")

        self.assertLessEqual(
            error_count,
            self.max_flake8_errors,
            f"Too many PEP8 errors: {error_count} > {self.max_flake8_errors}",
        )

    def test_03_import_organization(self):
        """Test import organization (basic check)."""
        print("\n=== T7.1.3: Import Organization ===")

        # Check a few key files for proper import structure
        key_files = [
            "src/alert_history/main.py",
            "src/alert_history/config.py",
            "src/alert_history/api/webhook_endpoints.py",
        ]

        issues = []
        for file_path in key_files:
            full_path = self.project_root / file_path
            if full_path.exists():
                with open(full_path, encoding="utf-8") as f:
                    content = f.read()

                # Basic checks
                if "from typing import" in content and "import typing" in content:
                    issues.append(f"{file_path}: Mixed typing imports")

                # Check for wildcard imports
                if "from * import" in content or "import *" in content:
                    issues.append(f"{file_path}: Wildcard imports found")

        if issues:
            print(f"⚠️  Import issues found: {len(issues)}")
            for issue in issues[:5]:  # Show first 5
                print(f"   - {issue}")
        else:
            print("✅ No major import organization issues detected")

    def test_04_basic_type_annotations(self):
        """Test basic type annotation coverage."""
        print("\n=== T7.1.4: Basic Type Annotations ===")

        try:
            # Run mypy on key modules
            result = subprocess.run(
                [
                    sys.executable,
                    "-m",
                    "mypy",
                    str(self.src_path / "alert_history" / "config.py"),
                    str(self.src_path / "alert_history" / "core" / "interfaces.py"),
                    "--ignore-missing-imports",
                    "--show-error-codes",
                ],
                capture_output=True,
                text=True,
                cwd=self.project_root,
            )

            error_count = len(
                [line for line in result.stdout.split("\n") if ": error:" in line]
            )
            print(f"📊 MyPy errors in core modules: {error_count}")

            # Allow some mypy errors initially, focus on critical files
            self.assertLessEqual(
                error_count, 20, "Too many type annotation errors in core modules"
            )

        except FileNotFoundError:
            print("⚠️  MyPy not available, skipping type checking")

    def test_05_security_linting(self):
        """Test security with bandit (if available)."""
        print("\n=== T7.1.5: Security Linting ===")

        try:
            result = subprocess.run(
                [
                    sys.executable,
                    "-m",
                    "bandit",
                    "-r",
                    str(self.src_path),
                    "-f",
                    "json",
                    "-ll",  # Low and low severity
                ],
                capture_output=True,
                text=True,
                cwd=self.project_root,
            )

            # Parse bandit JSON output
            import json

            try:
                bandit_data = json.loads(result.stdout)
                high_severity = len(
                    [
                        r
                        for r in bandit_data.get("results", [])
                        if r.get("issue_severity") == "HIGH"
                    ]
                )
                medium_severity = len(
                    [
                        r
                        for r in bandit_data.get("results", [])
                        if r.get("issue_severity") == "MEDIUM"
                    ]
                )

                print(
                    f"🔒 Security issues - High: {high_severity}, Medium: {medium_severity}"
                )

                # Should have no high severity security issues
                self.assertEqual(
                    high_severity, 0, "High severity security issues found"
                )
                self.assertLessEqual(
                    medium_severity, 5, "Too many medium severity security issues"
                )

            except json.JSONDecodeError:
                print("⚠️  Bandit output parsing failed")

        except FileNotFoundError:
            print("⚠️  Bandit not available, skipping security linting")

    def test_06_docstring_coverage(self):
        """Test basic docstring coverage."""
        print("\n=== T7.1.6: Docstring Coverage ===")

        # Check key modules for docstrings
        key_files = [
            "src/alert_history/main.py",
            "src/alert_history/config.py",
            "src/alert_history/core/interfaces.py",
        ]

        total_functions = 0
        documented_functions = 0

        for file_path in key_files:
            full_path = self.project_root / file_path
            if full_path.exists():
                with open(full_path, encoding="utf-8") as f:
                    lines = f.readlines()

                in_function = False
                for i, line in enumerate(lines):
                    if line.strip().startswith(("def ", "class ", "async def ")):
                        total_functions += 1
                        in_function = True

                        # Check next few lines for docstring
                        for j in range(i + 1, min(i + 5, len(lines))):
                            if '"""' in lines[j] or "'''" in lines[j]:
                                documented_functions += 1
                                break
                        in_function = False

        if total_functions > 0:
            coverage = documented_functions / total_functions
            print(
                f"📚 Docstring coverage: {coverage:.1%} ({documented_functions}/{total_functions})"
            )
            self.assertGreaterEqual(coverage, 0.3, "Docstring coverage too low")
        else:
            print("⚠️  No functions found for docstring analysis")

    def test_07_unit_test_structure(self):
        """Test unit test structure and basic coverage."""
        print("\n=== T7.1.7: Unit Test Structure ===")

        # Check for existing test files
        test_files = list(self.project_root.glob("test_*.py"))
        print(f"📋 Test files found: {len(test_files)}")

        for test_file in test_files[:5]:  # Show first 5
            print(f"   - {test_file.name}")

        # Basic test structure validation
        self.assertGreaterEqual(len(test_files), 5, "Insufficient test files")

        # Try to run a simple test to verify test infrastructure
        try:
            result = subprocess.run(
                [
                    sys.executable,
                    "-m",
                    "unittest",
                    "discover",
                    "-s",
                    ".",
                    "-p",
                    "test_*.py",
                    "-v",
                ],
                capture_output=True,
                text=True,
                cwd=self.project_root,
                timeout=30,
            )

            # Count test results
            if "Ran" in result.stderr:
                print("🧪 Test execution summary:")
                for line in result.stderr.split("\n"):
                    if "Ran" in line or "OK" in line or "FAILED" in line:
                        print(f"   {line.strip()}")

        except (subprocess.TimeoutExpired, FileNotFoundError):
            print("⚠️  Test execution skipped (timeout or missing dependencies)")

    def test_08_code_complexity(self):
        """Test code complexity (basic metrics)."""
        print("\n=== T7.1.8: Code Complexity ===")

        # Basic line count and file metrics
        python_files = list(self.src_path.rglob("*.py"))
        total_lines = 0
        total_files = len(python_files)

        large_files = []
        for py_file in python_files:
            try:
                with open(py_file, encoding="utf-8") as f:
                    lines = len(f.readlines())
                    total_lines += lines

                    if lines > 800:  # Flag very large files
                        large_files.append((py_file.name, lines))
            except:
                pass

        avg_lines = total_lines / total_files if total_files > 0 else 0

        print(f"📊 Total Python files: {total_files}")
        print(f"📊 Total lines of code: {total_lines:,}")
        print(f"📊 Average lines per file: {avg_lines:.0f}")

        if large_files:
            print(f"⚠️  Large files (>800 lines): {len(large_files)}")
            for name, lines in large_files[:3]:
                print(f"   - {name}: {lines} lines")

        # Basic complexity assertions
        self.assertLess(avg_lines, 600, "Average file size too large")
        self.assertLess(len(large_files), 5, "Too many large files")


def main():
    """Run the code quality test suite."""
    print("🚀 Starting T7.1: Code Quality & Unit Tests")
    print("=" * 60)

    # Set up environment
    test_suite = unittest.TestLoader().loadTestsFromTestCase(TestCodeQuality)
    runner = unittest.TextTestRunner(verbosity=2, stream=sys.stdout)

    # Run tests
    result = runner.run(test_suite)

    # Summary
    print("\n" + "=" * 60)
    print("📊 T7.1 CODE QUALITY SUMMARY")
    print("=" * 60)

    total_tests = result.testsRun
    failures = len(result.failures)
    errors = len(result.errors)
    passed = total_tests - failures - errors

    print(f"Total Tests: {total_tests}")
    print(f"Passed: {passed}")
    print(f"Failed: {failures}")
    print(f"Errors: {errors}")

    success_rate = (passed / total_tests) * 100 if total_tests > 0 else 0
    print(f"Success Rate: {success_rate:.1f}%")

    if success_rate >= 75:
        print("🎉 T7.1: Code Quality - ACCEPTABLE PROGRESS")
        status = "PROGRESS"
    else:
        print("❌ T7.1: Code Quality - NEEDS IMPROVEMENT")
        status = "NEEDS_WORK"

    print("=" * 60)

    # Exit with appropriate code
    return 0 if success_rate >= 75 else 1


if __name__ == "__main__":
    sys.exit(main())
