# Alertmanager Config Validator - Examples

This directory contains example Alertmanager configurations demonstrating various use cases and patterns.

## Directory Structure

```
examples/
├── configs/                    # Example configuration files
│   ├── valid-minimal.yml      # Minimal valid configuration
│   ├── valid-production.yml   # Production-grade configuration
│   ├── invalid-missing-receiver.yml  # Error example: Missing receiver
│   ├── invalid-matcher-syntax.yml    # Error example: Wrong syntax
│   └── best-practices.yml     # Recommended patterns
└── README.md                  # This file
```

## Usage

### Validate Examples

```bash
# Validate all examples
find examples/configs -name "*.yml" -exec configvalidator validate {} \;

# Validate specific example
configvalidator validate examples/configs/valid-minimal.yml

# Validate with verbose output
configvalidator validate --verbose examples/configs/valid-production.yml
```

### Learn from Examples

1. **Start with `valid-minimal.yml`**: Understand the minimum requirements
2. **Study `valid-production.yml`**: See production patterns
3. **Review `best-practices.yml`**: Learn recommended approaches
4. **Examine invalid examples**: Understand common errors

## Examples Overview

### Valid Configurations

#### `valid-minimal.yml`
- **Purpose**: Simplest possible valid configuration
- **Use Case**: Quick start, testing
- **Components**: 1 receiver, basic routing
- **Validation**: ✅ Passes all checks

#### `valid-production.yml`
- **Purpose**: Production-ready configuration
- **Use Case**: Real-world deployments
- **Components**: Multiple receivers, nested routes, inhibition rules
- **Features**:
  - Multiple notification channels (PagerDuty, Slack, Email)
  - Team-based routing
  - Critical alert prioritization
  - Inhibition rules for noise reduction
- **Validation**: ✅ Passes all checks

#### `best-practices.yml`
- **Purpose**: Demonstrate recommended patterns
- **Use Case**: Learning, reference
- **Components**: Comprehensive setup with comments
- **Highlights**:
  - Secret management (*_file variants)
  - TLS configuration
  - Intelligent grouping
  - Noise reduction strategies
  - Multiple redundant channels
- **Validation**: ✅ Passes all checks with best practices enabled

### Invalid Configurations

#### `invalid-missing-receiver.yml`
- **Error**: E102 - Receiver not found
- **Cause**: Route references non-existent receiver
- **Fix**: Add the receiver to `receivers` section
- **Validation**: ❌ Fails with specific error

#### `invalid-matcher-syntax.yml`
- **Error**: E104 - Invalid matcher syntax
- **Cause**: Wrong matcher format (`:` instead of `=`)
- **Fix**: Use correct matcher syntax
- **Valid Syntax**:
  - `label=value` (equality)
  - `label!=value` (inequality)
  - `label=~regex` (regex match)
  - `label!~regex` (regex non-match)
- **Validation**: ❌ Fails with specific error

## Testing Your Own Configurations

### Step 1: Validate Syntax
```bash
configvalidator validate your-config.yml
```

### Step 2: Enable Strict Mode
```bash
configvalidator validate --mode strict your-config.yml
```

### Step 3: Check Security
```bash
configvalidator validate --security your-config.yml
```

### Step 4: Apply Best Practices
```bash
configvalidator validate --best-practices your-config.yml
```

## Common Patterns

### Pattern 1: Team-Based Routing
```yaml
route:
  receiver: default
  routes:
    - receiver: team-backend
      matchers:
        - team=backend
    - receiver: team-frontend
      matchers:
        - team=frontend
```

### Pattern 2: Severity-Based Prioritization
```yaml
route:
  receiver: default
  routes:
    - receiver: critical-pagerduty
      matchers:
        - severity=critical
      group_wait: 10s
      continue: true
    - receiver: warning-slack
      matchers:
        - severity=warning
```

### Pattern 3: Inhibition Rules
```yaml
inhibit_rules:
  - source_matchers:
      - severity=critical
    target_matchers:
      - severity=warning
    equal:
      - alertname
      - instance
```

### Pattern 4: Multiple Channels for Critical Alerts
```yaml
routes:
  - receiver: critical-pagerduty
    matchers:
      - severity=critical
    continue: true  # Continue to next route
  - receiver: critical-slack
    matchers:
      - severity=critical
```

## Troubleshooting

### Error: "Receiver not found"
**Example**: `invalid-missing-receiver.yml`

**Problem**: Route references a receiver that doesn't exist
**Solution**: Ensure receiver name matches exactly

### Error: "Invalid matcher syntax"
**Example**: `invalid-matcher-syntax.yml`

**Problem**: Wrong matcher format
**Solution**: Use `label=value` format, not `label:value`

### Warning: "No inhibition rules"
**Solution**: Add inhibition rules to reduce alert noise (see `best-practices.yml`)

### Warning: "Hardcoded secrets"
**Solution**: Use `*_file` variants for sensitive data (see `best-practices.yml`)

## Additional Resources

- [User Guide](../docs/USER_GUIDE.md) - Comprehensive documentation
- [Alertmanager Docs](https://prometheus.io/docs/alerting/latest/configuration/) - Official documentation
- [GitHub Issues](https://github.com/yourusername/alertmanager-validator/issues) - Report problems

## Contributing Examples

Have a useful example? Please contribute!

1. Create your example in `examples/configs/`
2. Add documentation to this README
3. Validate: `configvalidator validate your-example.yml`
4. Submit a pull request

## License

Examples are provided under the same license as the main project.

---

**Generated by**: Alertmanager Config Validator
**Version**: 1.0.0
**Date**: 2025-11-24
