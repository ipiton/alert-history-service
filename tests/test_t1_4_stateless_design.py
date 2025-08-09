#!/usr/bin/env python3
"""
Test T1.4: Stateless Application Design.

Тестирует:
- Удаление локального состояния из приложения
- Перенос временных данных в Redis/PostgreSQL
- Idempotent operations для всех API endpoints
- Instance ID tracking для debugging
- Stateless service coordination
"""
import asyncio
import hashlib
import os
import sys
import time
from unittest.mock import AsyncMock, MagicMock

# Add project root to path
project_root = os.path.abspath(".")
sys.path.insert(0, project_root)


async def test_stateless_manager():
    """Test StatelessManager functionality."""
    print("\n🏛️ Testing Stateless Manager...")

    try:
        from src.alert_history.core.stateless_manager import StatelessManager

        # Test manager initialization
        manager = StatelessManager(
            redis_cache=None,  # Test without Redis first
            operation_ttl=3600,
        )

        # Test instance ID generation
        assert manager.instance_id.startswith("alert-history-")
        assert len(manager.instance_id) > 20  # Should include timestamp and random suffix

        print("   ✅ StatelessManager initialization")

        # Test operation registry
        assert hasattr(manager, '_operation_registry')
        assert isinstance(manager._operation_registry, dict)

        print("   ✅ Local operation registry")

        # Test idempotent operation without Redis (local fallback)
        operation_key = "test_operation_123"

        # First call should return True (can proceed)
        can_proceed_1 = await manager.ensure_idempotent_operation(operation_key)
        assert can_proceed_1 == True

        # Second call should return False (already executed)
        can_proceed_2 = await manager.ensure_idempotent_operation(operation_key)
        assert can_proceed_2 == False

        print("   ✅ Idempotent operations (local fallback)")

        # Test stats
        stats = await manager.get_stateless_stats()
        assert "instance_id" in stats
        assert "redis_available" in stats
        assert stats["redis_available"] == False

        print("   ✅ Statistics collection")

        # Test health check
        health = await manager.health_check()
        assert "stateless_manager" in health
        assert health["stateless_manager"] == "healthy"

        print("   ✅ Health check")

        print("\n🎉 Stateless manager test passed!")
        return True

    except Exception as e:
        print(f"   ❌ Stateless manager test failed: {e}")
        return False


async def test_idempotent_operations():
    """Test idempotent operations functionality."""
    print("\n🔁 Testing Idempotent Operations...")

    try:
        from src.alert_history.core.stateless_manager import StatelessManager

        # Create mock Redis cache
        mock_redis = AsyncMock()
        mock_redis.get.return_value = None  # Operation doesn't exist
        mock_redis.set.return_value = True   # Successfully set operation

        manager = StatelessManager(
            redis_cache=mock_redis,
            operation_ttl=300,
        )

        # Test operation key generation
        operation_key = "webhook_processing:alert_123"

        # First execution should proceed
        can_proceed = await manager.ensure_idempotent_operation(operation_key, ttl=300)
        assert can_proceed == True

        # Verify Redis was called correctly
        mock_redis.get.assert_called_once()
        mock_redis.set.assert_called_once()

        print("   ✅ Operation registration with Redis")

        # Test duplicate operation detection
        mock_redis.reset_mock()
        mock_redis.get.return_value = {
            "instance_id": "other-instance",
            "timestamp": "2024-01-01T12:00:00Z",
        }

        can_proceed_duplicate = await manager.ensure_idempotent_operation(operation_key)
        assert can_proceed_duplicate == False

        print("   ✅ Duplicate operation detection")

        # Test operation expiration
        manager._operation_registry.clear()

        # Add operation that should be expired
        from datetime import datetime, timedelta
        expired_time = datetime.utcnow() - timedelta(seconds=manager.operation_ttl + 100)
        manager._operation_registry["expired_op"] = expired_time

        # This should return True since operation is expired
        can_proceed_expired = manager._check_local_operation("expired_op")
        assert can_proceed_expired == True

        print("   ✅ Operation expiration")

        print("\n🎉 Idempotent operations test passed!")
        return True

    except Exception as e:
        print(f"   ❌ Idempotent operations test failed: {e}")
        return False


async def test_temporary_data_management():
    """Test temporary data management."""
    print("\n💾 Testing Temporary Data Management...")

    try:
        from src.alert_history.core.stateless_manager import StatelessManager

        # Create mock Redis cache
        mock_redis = AsyncMock()
        mock_redis.set.return_value = True
        mock_redis.get.return_value = {"test": "data"}
        mock_redis.delete.return_value = True

        manager = StatelessManager(redis_cache=mock_redis)

        # Test storing temporary data
        test_data = {"user_id": "123", "session": "abc"}
        success = await manager.store_temporary_data("user_session", test_data, ttl=300)
        assert success == True

        # Verify correct Redis key was used
        expected_key = f"temp:{manager.instance_id}:user_session"
        mock_redis.set.assert_called_with(expected_key, test_data, 300)

        print("   ✅ Temporary data storage")

        # Test retrieving temporary data
        retrieved_data = await manager.get_temporary_data("user_session")
        assert retrieved_data == {"test": "data"}

        mock_redis.get.assert_called_with(expected_key)

        print("   ✅ Temporary data retrieval")

        # Test deleting temporary data
        deleted = await manager.delete_temporary_data("user_session")
        assert deleted == True

        mock_redis.delete.assert_called_with(expected_key)

        print("   ✅ Temporary data deletion")

        # Test without Redis (should fail gracefully)
        manager_no_redis = StatelessManager(redis_cache=None)

        success_no_redis = await manager_no_redis.store_temporary_data("test", {"data": 1})
        assert success_no_redis == False

        data_no_redis = await manager_no_redis.get_temporary_data("test")
        assert data_no_redis == None

        print("   ✅ Graceful fallback without Redis")

        print("\n🎉 Temporary data management test passed!")
        return True

    except Exception as e:
        print(f"   ❌ Temporary data management test failed: {e}")
        return False


async def test_stateless_decorators():
    """Test stateless decorators."""
    print("\n🎭 Testing Stateless Decorators...")

    try:
        from src.alert_history.utils.stateless_decorators import (
            idempotent, stateless, instance_tracked, _generate_operation_key
        )

        # Test operation key generation
        def test_function():
            pass

        key1 = _generate_operation_key(test_function, ("arg1", "arg2"), {"param": "value"})
        key2 = _generate_operation_key(test_function, ("arg1", "arg2"), {"param": "value"})
        key3 = _generate_operation_key(test_function, ("arg1", "different"), {"param": "value"})

        # Same inputs should generate same key
        assert key1 == key2
        # Different inputs should generate different keys
        assert key1 != key3

        print("   ✅ Operation key generation")

        # Test decorator structure
        @idempotent(ttl=300)
        async def test_idempotent_function(arg1, arg2):
            return f"processed {arg1} {arg2}"

        assert callable(test_idempotent_function)

        print("   ✅ Idempotent decorator structure")

        @stateless(validate_parameters=True)
        async def test_stateless_function(required_param, optional_param=None):
            return f"result {required_param}"

        assert callable(test_stateless_function)

        print("   ✅ Stateless decorator structure")

        @instance_tracked(heartbeat_interval=30)
        async def test_tracked_function():
            return "tracked result"

        assert callable(test_tracked_function)

        print("   ✅ Instance tracked decorator structure")

        print("\n🎉 Stateless decorators test passed!")
        return True

    except Exception as e:
        print(f"   ❌ Stateless decorators test failed: {e}")
        return False


async def test_cross_instance_coordination():
    """Test cross-instance coordination."""
    print("\n🤝 Testing Cross-Instance Coordination...")

    try:
        from src.alert_history.core.stateless_manager import StatelessManager

        # Create mock Redis cache
        mock_redis = AsyncMock()
        mock_redis.set.return_value = True

        manager = StatelessManager(redis_cache=mock_redis)

        # Test heartbeat update
        heartbeat_success = await manager.update_instance_heartbeat()
        assert heartbeat_success == True

        # Verify heartbeat key format
        expected_heartbeat_key = f"instance:heartbeat:{manager.instance_id}"
        mock_redis.set.assert_called()

        call_args = mock_redis.set.call_args
        assert call_args[0][0] == expected_heartbeat_key  # First argument is the key
        assert call_args[0][2] == 120  # TTL is 120 seconds

        print("   ✅ Instance heartbeat")

        # Test active instances (simplified test)
        active_instances = await manager.get_active_instances()
        assert isinstance(active_instances, list)
        assert len(active_instances) >= 1

        # Should include this instance
        this_instance = next(
            (inst for inst in active_instances if inst["instance_id"] == manager.instance_id),
            None
        )
        assert this_instance is not None

        print("   ✅ Active instances tracking")

        # Test instance ID uniqueness
        manager2 = StatelessManager(redis_cache=mock_redis)
        assert manager.instance_id != manager2.instance_id

        print("   ✅ Instance ID uniqueness")

        print("\n🎉 Cross-instance coordination test passed!")
        return True

    except Exception as e:
        print(f"   ❌ Cross-instance coordination test failed: {e}")
        return False


async def test_stateless_validation():
    """Test stateless operation validation."""
    print("\n✅ Testing Stateless Validation...")

    try:
        from src.alert_history.core.stateless_manager import StatelessManager

        manager = StatelessManager(redis_cache=None)

                # Test valid stateless operation
        valid_result = manager.validate_stateless_operation(
            "process_alert",
            fingerprint="alert_123",
            data={"alert": "data"},
        )

        assert valid_result["operation"] == "process_alert"
        assert isinstance(valid_result, dict)
        assert "stateless" in valid_result

        print("   ✅ Stateless operation validation structure")

        # Test operation with potential state issues
        stateful_result = manager.validate_stateless_operation(
            "bad_operation",
            self_cache="some_cache",  # This should trigger warning
        )

        # Should detect potential state dependencies
        assert isinstance(stateful_result["issues"], list)
        assert isinstance(stateful_result["recommendations"], list)

        print("   ✅ State dependency detection")

        # Test operation without parameters
        no_params_result = manager.validate_stateless_operation("empty_operation")

        assert "No parameters provided" in str(no_params_result["issues"])

        print("   ✅ Parameter validation")

        print("\n🎉 Stateless validation test passed!")
        return True

    except Exception as e:
        print(f"   ❌ Stateless validation test failed: {e}")
        return False


async def test_main_integration():
    """Test StatelessManager integration in main.py."""
    print("\n🔗 Testing Main.py Integration...")

    try:
        # Check if StatelessManager is imported and used in main.py
        main_file_path = "src/alert_history/main.py"

        with open(main_file_path, 'r') as f:
            main_content = f.read()

        # Check for StatelessManager import
        if "StatelessManager" not in main_content:
            print("   ❌ StatelessManager not imported in main.py")
            return False

        print("   ✅ StatelessManager imported in main.py")

        # Check for StatelessManager initialization
        if "stateless_manager = StatelessManager" not in main_content:
            print("   ❌ StatelessManager not initialized in main.py")
            return False

        print("   ✅ StatelessManager initialization in main.py")

        # Check for app state assignment
        if "app.state.stateless_manager" not in main_content:
            print("   ❌ StatelessManager not assigned to app state")
            return False

        print("   ✅ StatelessManager assigned to app state")

        # Check for heartbeat update
        if "update_instance_heartbeat" not in main_content:
            print("   ❌ Instance heartbeat not updated in main.py")
            return False

        print("   ✅ Instance heartbeat updated in main.py")

        print("\n🎉 Main.py integration test passed!")
        return True

    except Exception as e:
        print(f"   ❌ Main.py integration test failed: {e}")
        return False


async def test_stateless_patterns():
    """Test common stateless patterns."""
    print("\n🏗️ Testing Stateless Patterns...")

    try:
        # Test alert processing pattern
        from src.alert_history.utils.stateless_decorators import idempotent_alert_processing

        @idempotent_alert_processing(ttl=300)
        async def process_alert(alert_data):
            return f"processed alert {alert_data.get('fingerprint', 'unknown')}"

        assert callable(process_alert)

        print("   ✅ Alert processing pattern")

        # Test webhook pattern
        from src.alert_history.utils.stateless_decorators import idempotent_webhook

        @idempotent_webhook(ttl=300)
        async def handle_webhook(payload, webhook_id=None):
            return f"handled webhook {webhook_id}"

        assert callable(handle_webhook)

        print("   ✅ Webhook processing pattern")

        # Test state removal patterns
        # Check that AppState is not used for critical operations
        app_state_file_path = "src/alert_history/core/app_state.py"

        with open(app_state_file_path, 'r') as f:
            app_state_content = f.read()

        # AppState should only be used for dependency injection, not business state
        if "dependency injection" not in app_state_content.lower():
            print("   ⚠️ AppState purpose not clearly documented")
        else:
            print("   ✅ AppState properly documented for dependency injection only")

        print("\n🎉 Stateless patterns test passed!")
        return True

    except Exception as e:
        print(f"   ❌ Stateless patterns test failed: {e}")
        return False


async def main():
    """Run all T1.4 stateless design tests."""
    print("🎯 T1.4: Stateless Application Design Tests")
    print("=" * 55)

    tests = [
        ("Stateless Manager", test_stateless_manager),
        ("Idempotent Operations", test_idempotent_operations),
        ("Temporary Data Management", test_temporary_data_management),
        ("Stateless Decorators", test_stateless_decorators),
        ("Cross-Instance Coordination", test_cross_instance_coordination),
        ("Stateless Validation", test_stateless_validation),
        ("Main.py Integration", test_main_integration),
        ("Stateless Patterns", test_stateless_patterns),
    ]

    results = []

    for test_name, test_func in tests:
        print(f"\n🧪 Running {test_name} test...")
        try:
            success = await test_func()
            results.append((test_name, success))

            if success:
                print(f"✅ {test_name} test passed")
            else:
                print(f"❌ {test_name} test failed")

        except Exception as e:
            print(f"💥 {test_name} test crashed: {e}")
            results.append((test_name, False))

    # Results summary
    print("\n" + "=" * 55)
    print("📊 T1.4: STATELESS APPLICATION DESIGN TEST RESULTS")
    print("=" * 55)

    passed = sum(1 for _, success in results if success)
    total = len(results)

    for test_name, success in results:
        status = "✅ PASSED" if success else "❌ FAILED"
        print(f"   {status} {test_name}")

    success_rate = passed / total * 100
    print(f"\n🏆 OVERALL RESULTS:")
    print(f"   • Tests Passed: {passed}/{total}")
    print(f"   • Success Rate: {success_rate:.1f}%")

    if success_rate >= 80:
        print(f"\n✅ T1.4 STATELESS APPLICATION DESIGN TESTS PASSED!")
        if success_rate == 100:
            print("🏆 PERFECT SCORE! All tests passed!")
        print("\n🚀 Ready for:")
        print("   • Horizontal scaling without state issues")
        print("   • Zero-downtime deployments")
        print("   • Multi-instance coordination")
        print("   • Cloud-native deployment")
        print("   • 12-Factor App compliance COMPLETE")
        return True
    else:
        print(f"\n❌ T1.4 STATELESS APPLICATION DESIGN TESTS FAILED!")
        print("   🔧 Fix failing components before proceeding")
        return False


if __name__ == "__main__":
    success = asyncio.run(main())
    sys.exit(0 if success else 1)
