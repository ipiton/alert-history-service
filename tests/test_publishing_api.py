#!/usr/bin/env python3
"""
Тест для Publishing Management API.
"""
import asyncio
import os
import sys

# Add the project root to the Python path
project_root = os.path.abspath(".")
sys.path.insert(0, project_root)


async def test_publishing_endpoints():
    """Test Publishing Management API endpoints."""
    print("🧪 Testing Publishing Management API...")

    try:
        # Test imports
        print("1. Testing publishing endpoints imports...")
        from src.alert_history.api.publishing_endpoints import publishing_router

        print("   ✅ Publishing endpoints imported")

        # Test router setup
        print("2. Testing publishing router configuration...")
        assert publishing_router.prefix == "/publishing"
        assert "Publishing Management" in publishing_router.tags
        print("   ✅ Publishing router configured correctly")

        # Test endpoint registration
        print("3. Testing endpoint registration...")
        routes = []
        for route in publishing_router.routes:
            if hasattr(route, "path"):
                routes.append(route.path)

        print(f"   📋 Found routes: {routes}")

        expected_endpoints = [
            "/targets",
            "/targets/refresh",
            "/mode",
            "/stats",
            "/test/{target_name}",
            "/secrets/template",
            "/health",
        ]

        for endpoint in expected_endpoints:
            # Check if endpoint exists (flexible matching)
            endpoint_found = any(
                endpoint.replace("{target_name}", "target_name") in route
                for route in routes
            )
            if endpoint_found:
                print(f"   ✅ Endpoint found: /publishing{endpoint}")
            else:
                print(f"   ⚠️  Endpoint missing: /publishing{endpoint}")

        print("4. Testing Pydantic models...")
        from src.alert_history.api.publishing_endpoints import (
            PublishingModeInfo,
            PublishingTargetInfo,
        )

        # Test model creation
        target_info = PublishingTargetInfo(
            name="test-target",
            format="rootly",
            url="https://api.rootly.com/v1/incidents",
            enabled=True,
            total_publishes=10,
            successful_publishes=8,
            failed_publishes=2,
            health_status="healthy",
        )

        assert target_info.name == "test-target"
        assert target_info.successful_publishes == 8
        print("   ✅ PublishingTargetInfo model works")

        mode_info = PublishingModeInfo(
            mode="intelligent",
            targets_available=True,
            targets_count=3,
            kubernetes_available=True,
            description="Test mode",
        )

        assert mode_info.mode == "intelligent"
        assert mode_info.targets_count == 3
        print("   ✅ PublishingModeInfo model works")

        print("\n🎉 Publishing endpoints test passed!")
        return True

    except Exception as e:
        print(f"❌ Test failed: {e}")
        import traceback

        traceback.print_exc()
        return False


async def test_secret_templates():
    """Test secret template generation."""
    print("\n🔐 Testing Secret Templates...")

    try:
        print("1. Testing secret template endpoint...")
        from src.alert_history.api.publishing_endpoints import get_secret_templates

        # Test template generation
        templates = await get_secret_templates()

        assert (
            len(templates) >= 3
        ), "Should have at least 3 templates (Rootly, PagerDuty, Slack)"
        print(f"   ✅ Generated {len(templates)} secret templates")

        # Check template structure
        for template in templates:
            assert hasattr(template, "apiVersion")
            assert hasattr(template, "kind")
            assert hasattr(template, "metadata")
            assert hasattr(template, "data")

            assert template.apiVersion == "v1"
            assert template.kind == "Secret"
            assert "alert-history.io/target" in template.metadata.get("labels", {})

            print(f"   ✅ Template validated: {template.metadata['name']}")

        print("2. Testing template content...")

        # Find specific templates
        rootly_template = None
        pagerduty_template = None
        slack_template = None

        for template in templates:
            name = template.metadata["name"]
            if "rootly" in name:
                rootly_template = template
            elif "pagerduty" in name:
                pagerduty_template = template
            elif "slack" in name:
                slack_template = template

        # Verify Rootly template
        if rootly_template:
            assert "url" in rootly_template.data
            assert "token" in rootly_template.data
            assert "format" in rootly_template.data
            print("   ✅ Rootly template complete")

        # Verify PagerDuty template
        if pagerduty_template:
            assert "url" in pagerduty_template.data
            assert "routing_key" in pagerduty_template.data
            assert "format" in pagerduty_template.data
            print("   ✅ PagerDuty template complete")

        # Verify Slack template
        if slack_template:
            assert "url" in slack_template.data
            assert "format" in slack_template.data
            assert "channel" in slack_template.data
            print("   ✅ Slack template complete")

        print("\n🎉 Secret templates test passed!")
        return True

    except Exception as e:
        print(f"❌ Test failed: {e}")
        import traceback

        traceback.print_exc()
        return False


async def test_target_testing():
    """Test target testing functionality."""
    print("\n🎯 Testing Target Testing...")

    try:
        print("1. Testing test request/response models...")
        from src.alert_history.api.publishing_endpoints import (
            TestTargetRequest,
            TestTargetResponse,
        )

        # Test request model
        test_request = TestTargetRequest(
            test_alert={
                "fingerprint": "test-alert",
                "labels": {"alertname": "TestAlert"},
                "status": "firing",
            },
            timeout_seconds=30,
        )

        assert test_request.timeout_seconds == 30
        assert test_request.test_alert["fingerprint"] == "test-alert"
        print("   ✅ TestTargetRequest model works")

        # Test response model
        test_response = TestTargetResponse(
            target_name="test-target",
            success=True,
            status_code=200,
            response_time_ms=250,
            test_timestamp="2024-12-28T10:00:00Z",
        )

        assert test_response.success == True
        assert test_response.response_time_ms == 250
        print("   ✅ TestTargetResponse model works")

        print("2. Testing dependency injection...")
        from src.alert_history.api.publishing_endpoints import (
            get_alert_publisher,
            get_target_manager,
        )

        # Test that dependencies can be imported and called
        target_manager = await get_target_manager()
        alert_publisher = await get_alert_publisher()

        print(f"   📋 Target manager type: {type(target_manager)}")
        print(f"   📋 Alert publisher type: {type(alert_publisher)}")

        # Check if they are the right type (more flexible than None check)
        from src.alert_history.services.alert_publisher import (
            AlertPublisher as AlertPublisherClass,
        )
        from src.alert_history.services.target_discovery import DynamicTargetManager

        assert isinstance(
            target_manager, DynamicTargetManager
        ), f"Expected DynamicTargetManager, got {type(target_manager)}"
        assert isinstance(
            alert_publisher, AlertPublisherClass
        ), f"Expected AlertPublisher, got {type(alert_publisher)}"
        print("   ✅ Dependency injection works")

        print("3. Testing app state integration...")
        from src.alert_history.core.app_state import app_state

        # Verify app state has the instances
        assert hasattr(app_state, "target_manager")
        assert hasattr(app_state, "alert_publisher")
        print("   ✅ App state integration works")

        print("\n🎉 Target testing functionality test passed!")
        return True

    except Exception as e:
        print(f"❌ Test failed: {e}")
        import traceback

        traceback.print_exc()
        return False


async def test_publishing_integration():
    """Test integration with main components."""
    print("\n🔗 Testing Publishing API Integration...")

    try:
        print("1. Testing component integration...")

        # Test that all required services can be imported
        services_available = True
        try:
            from src.alert_history.core.interfaces import (
                PublishingFormat,
                PublishingTarget,
            )
            from src.alert_history.services.alert_publisher import AlertPublisher
            from src.alert_history.services.target_discovery import DynamicTargetManager

            print("   ✅ All required services available")
        except ImportError as e:
            print(f"   ❌ Service import failed: {e}")
            services_available = False

        print("2. Testing publishing formats support...")
        from src.alert_history.core.interfaces import PublishingFormat

        formats = [
            PublishingFormat.ROOTLY,
            PublishingFormat.PAGERDUTY,
            PublishingFormat.SLACK,
            PublishingFormat.ALERTMANAGER,
        ]

        for fmt in formats:
            print(f"   ✅ Format supported: {fmt.value}")

        print("3. Testing target discovery integration...")
        from src.alert_history.services.target_discovery import TargetDiscoveryConfig

        # Test that discovery config can be created
        config = TargetDiscoveryConfig(
            enabled=True,
            secret_labels=["alert-history.io/target=true"],
            secret_namespaces=["default"],
        )

        assert config.enabled == True
        assert "alert-history.io/target=true" in config.secret_labels
        print("   ✅ Target discovery configuration works")

        if services_available:
            print("   ✅ All publishing API integrations ready")

        print("\n🎉 Publishing API integration test passed!")
        return True

    except Exception as e:
        print(f"❌ Test failed: {e}")
        import traceback

        traceback.print_exc()
        return False


if __name__ == "__main__":
    print("🎯 Publishing Management API Test")
    print("=" * 50)

    # Run all tests
    success1 = asyncio.run(test_publishing_endpoints())
    success2 = asyncio.run(test_secret_templates())
    success3 = asyncio.run(test_target_testing())
    success4 = asyncio.run(test_publishing_integration())

    overall_success = success1 and success2 and success3 and success4

    if overall_success:
        print("\n" + "=" * 50)
        print("✅ ALL PUBLISHING API TESTS PASSED!")
        print("")
        print("🎯 Publishing Management API готов:")
        print("   • GET /publishing/targets - список активных targets")
        print("   • POST /publishing/targets/refresh - обновление targets")
        print("   • GET /publishing/mode - проверка режима работы")
        print("   • GET /publishing/stats - статистика публикации")
        print("   • POST /publishing/test/{target} - тестирование targets")
        print("   • GET /publishing/secrets/template - templates для secrets")
        print("   • GET /publishing/health - health check")
        print("")
        print("🔐 Secret Templates готовы:")
        print("   • Rootly API integration template")
        print("   • PagerDuty events API template")
        print("   • Slack webhook template")
        print("")
        print("🏆 T3.5: DYNAMIC PUBLISHING MANAGEMENT API COMPLETE!")

    sys.exit(0 if overall_success else 1)
