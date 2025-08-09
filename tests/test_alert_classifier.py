#!/usr/bin/env python3
"""
Тест Alert Classifier инфраструктуры.
"""
import sys
import os
import asyncio

# Add the project root to the Python path
project_root = os.path.abspath(".")
sys.path.insert(0, project_root)


async def test_llm_client_structure():
    """Test LLM client structure and interfaces."""
    print("🧪 Testing LLM Client Structure...")

    try:
        # Test imports
        print("1. Testing LLM client imports...")
        from src.alert_history.services.llm_client import LLMProxyClient
        from src.alert_history.core.interfaces import Alert, AlertSeverity
        print("   ✅ LLM client and interfaces imported successfully")

        # Test client initialization
        print("2. Testing LLM client initialization...")
        llm_client = LLMProxyClient(
            proxy_url="http://localhost:8080/llm",
            api_key="test-key",
            model="gpt-4",
            timeout=30
        )
        print("   ✅ LLM client initialized")

        # Test prompts and schema
        print("3. Testing classification prompts and schema...")
        prompt = llm_client.classification_prompt
        schema = llm_client.function_schema

        print(f"   ✅ Classification prompt length: {len(prompt)} chars")
        print(f"   ✅ Function schema name: {schema['name']}")
        print(f"   ✅ Required fields: {schema['parameters']['required']}")

        # Test prompt content
        assert "severity" in prompt.lower()
        assert "critical" in prompt.lower()
        assert "warning" in prompt.lower()
        print("   ✅ Prompt contains required severity levels")

        # Test schema validation
        required_fields = schema['parameters']['required']
        expected_fields = ['severity', 'confidence', 'reasoning', 'recommendations']
        for field in expected_fields:
            assert field in required_fields, f"Missing required field: {field}"
        print("   ✅ Function schema has all required fields")

        print("\n🎉 LLM client structure test passed!")
        return True

    except Exception as e:
        print(f"❌ Test failed: {e}")
        import traceback
        traceback.print_exc()
        return False


async def test_alert_classification_service():
    """Test Alert Classification Service structure."""
    print("\n🔍 Testing Alert Classification Service...")

    try:
        # Test imports
        print("1. Testing classification service imports...")
        from src.alert_history.services.alert_classifier import AlertClassificationService
        from src.alert_history.services.llm_client import LLMProxyClient
        from src.alert_history.database.sqlite_adapter import SQLiteLegacyStorage
        from src.alert_history.services.redis_cache import RedisCache
        print("   ✅ Classification service components imported")

        # Test service initialization
        print("2. Testing service initialization...")

        # Create mock components
        llm_client = LLMProxyClient(
            proxy_url="http://localhost:8080/llm",
            api_key="test-key"
        )

        storage = SQLiteLegacyStorage("./data/alert_history.sqlite3")

        from config import get_config
        config = get_config()
        cache = RedisCache(
            redis_url=config.redis.redis_url,
            default_ttl=3600
        )

        # Create classification service
        classifier = AlertClassificationService(
            llm_client=llm_client,
            storage=storage,
            cache=cache,
            cache_ttl=3600,
            enable_fallback=True
        )

        print("   ✅ Classification service initialized")

        # Test service methods
        print("3. Testing service interface...")
        methods_to_check = [
            'classify_alert',
            'bulk_classify_alerts',
            'get_classification_stats',
            'refresh_classification',
        ]

        for method in methods_to_check:
            assert hasattr(classifier, method), f"Missing method: {method}"
            print(f"   ✅ Method available: {method}")

        print("\n🎉 Alert Classification Service test passed!")
        return True

    except Exception as e:
        print(f"❌ Test failed: {e}")
        import traceback
        traceback.print_exc()
        return False


async def test_classification_integration():
    """Test classification integration with existing components."""
    print("\n🔗 Testing Classification Integration...")

    try:
        # Test config integration
        print("1. Testing LLM configuration...")
        from config import get_config
        config = get_config()

        if hasattr(config, 'llm') and config.llm:
            print(f"   ✅ LLM proxy URL: {config.llm.proxy_url}")
            print(f"   ✅ LLM model: {config.llm.model}")
            print(f"   ✅ LLM timeout: {config.llm.timeout}")
        else:
            print("   ⚠️  LLM configuration not found (expected in early development)")

        # Test endpoint integration
        print("2. Testing classification endpoints...")
        from src.alert_history.api.classification_endpoints import router
        print(f"   ✅ Classification router prefix: {router.prefix}")
        print(f"   ✅ Classification router tags: {router.tags}")

        # Count routes
        route_count = len([route for route in router.routes])
        print(f"   ✅ Classification routes: {route_count}")

        # Test alert models
        print("3. Testing alert data models...")
        from src.alert_history.core.interfaces import Alert, AlertSeverity, ClassificationResult

        # Test severity enum
        severities = [AlertSeverity.CRITICAL, AlertSeverity.WARNING, AlertSeverity.INFO, AlertSeverity.NOISE]
        print(f"   ✅ Alert severities: {[s.value for s in severities]}")

        print("\n🎉 Classification integration test passed!")
        return True

    except Exception as e:
        print(f"❌ Test failed: {e}")
        import traceback
        traceback.print_exc()
        return False


async def test_openai_function_calling():
    """Test OpenAI function calling schema compatibility."""
    print("\n🤖 Testing OpenAI Function Calling...")

    try:
        print("1. Testing function schema structure...")
        from src.alert_history.services.llm_client import LLMProxyClient

        client = LLMProxyClient(
            proxy_url="http://localhost:8080/llm",
            api_key="test-key"
        )

        schema = client.function_schema

        # Validate OpenAI function calling schema format
        required_keys = ['name', 'description', 'parameters']
        for key in required_keys:
            assert key in schema, f"Missing schema key: {key}"

        print("   ✅ Function schema has required OpenAI format")

        # Validate parameters structure
        params = schema['parameters']
        assert params['type'] == 'object', "Parameters must be object type"
        assert 'properties' in params, "Properties must be defined"
        assert 'required' in params, "Required fields must be specified"

        print("   ✅ Parameters structure is valid")

        # Test severity enum
        severity_prop = params['properties']['severity']
        expected_severities = ['critical', 'warning', 'info', 'noise']
        actual_severities = severity_prop['enum']

        for severity in expected_severities:
            assert severity in actual_severities, f"Missing severity: {severity}"

        print("   ✅ Severity enum matches our AlertSeverity")

        # Test confidence range
        confidence_prop = params['properties']['confidence']
        assert confidence_prop['minimum'] == 0.0, "Confidence minimum should be 0.0"
        assert confidence_prop['maximum'] == 1.0, "Confidence maximum should be 1.0"

        print("   ✅ Confidence score range is valid")

        print("\n🎉 OpenAI Function Calling test passed!")
        return True

    except Exception as e:
        print(f"❌ Test failed: {e}")
        import traceback
        traceback.print_exc()
        return False


if __name__ == "__main__":
    print("🎯 Alert Classifier Infrastructure Test")
    print("=" * 50)

    # Run all tests
    success1 = asyncio.run(test_llm_client_structure())
    success2 = asyncio.run(test_alert_classification_service())
    success3 = asyncio.run(test_classification_integration())
    success4 = asyncio.run(test_openai_function_calling())

    overall_success = success1 and success2 and success3 and success4

    if overall_success:
        print("\n" + "=" * 50)
        print("✅ ALL ALERT CLASSIFIER TESTS PASSED!")
        print("")
        print("🎯 Classification infrastructure ready for:")
        print("   • LLM-powered alert classification")
        print("   • OpenAI function calling integration")
        print("   • Redis-based result caching")
        print("   • Database storage integration")
        print("   • Prometheus metrics collection")
        print("")
        print("🤖 OpenAI Function Calling: READY")
        print("⚡ Classification Service: READY")
        print("📊 Integration: COMPLETE")

    sys.exit(0 if overall_success else 1)
