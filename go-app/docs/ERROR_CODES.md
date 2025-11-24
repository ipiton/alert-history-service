# Config Validator - Error Codes Reference

This document provides a comprehensive reference for all error, warning, info, and suggestion codes emitted by the Alertmanager++ Config Validator.

## Table of Contents

- [Understanding Issue Codes](#understanding-issue-codes)
- [Error Codes (E-series)](#error-codes-e-series)
  - [E000-E099: General & Parsing Errors](#e000-e099-general--parsing-errors)
  - [E100-E109: Route Validation Errors](#e100-e109-route-validation-errors)
  - [E110-E149: Receiver Validation Errors](#e110-e149-receiver-validation-errors)
  - [E150-E199: Inhibition Rule Errors](#e150-e199-inhibition-rule-errors)
  - [E200-E249: Global Configuration Errors](#e200-e249-global-configuration-errors)
- [Warning Codes (W-series)](#warning-codes-w-series)
  - [W000-W099: General Warnings](#w000-w099-general-warnings)
  - [W100-W149: Configuration Warnings](#w100-w149-configuration-warnings)
  - [W150-W199: Inhibition Warnings](#w150-w199-inhibition-warnings)
  - [W200-W249: Deprecation Warnings](#w200-w249-deprecation-warnings)
  - [W300-W399: Security Warnings](#w300-w399-security-warnings)
- [Info Codes (I-series)](#info-codes-i-series)
- [Suggestion Codes (S-series)](#suggestion-codes-s-series)

---

## Understanding Issue Codes

### Code Format

All issue codes follow the pattern: `{TYPE}{NUMBER}`

- **TYPE**: Single letter indicating issue severity
  - `E` = Error (blocking, must fix)
  - `W` = Warning (should review, may cause issues)
  - `I` = Info (informational, non-blocking)
  - `S` = Suggestion (improvement opportunity)

- **NUMBER**: Three-digit number indicating the category and specific issue
  - `000-099`: General and parsing issues
  - `100-199`: Route validation issues
  - `200-299`: Receiver validation issues
  - `300-399`: Security and TLS issues
  - etc.

### Severity Levels

| Code | Severity | Exit Code | Description |
|------|----------|-----------|-------------|
| E-series | ERROR | 2 | Blocking issues that must be fixed |
| W-series | WARNING | 1 | Non-blocking but should be reviewed |
| I-series | INFO | 0 | Informational messages |
| S-series | SUGGESTION | 0 | Improvement recommendations |

### Validation Modes

Different validation modes affect how issues are treated:

- **Strict Mode**: All warnings become errors
- **Lenient Mode**: Some errors become warnings
- **Permissive Mode**: Minimal validation, most issues become warnings

---

## Error Codes (E-series)

### E000-E099: General & Parsing Errors

#### E001: File Not Found
**Message**: Configuration file not found: {filename}

**Description**: The specified configuration file does not exist or is not accessible.

**Resolution**:
- Verify the file path is correct
- Check file permissions
- Ensure the file exists in the specified location

**Example**:
```bash
configvalidator validate non-existent.yml
# ERROR [E001]: Configuration file not found: non-existent.yml
```

---

#### E002: Invalid JSON Syntax
**Message**: Invalid JSON syntax: {error}

**Description**: The JSON configuration file contains syntax errors.

**Resolution**:
- Use a JSON validator to check syntax
- Check for missing commas, brackets, or quotes
- Ensure proper escaping of special characters

**Example**:
```json
{
  "global": {
    "resolve_timeout": "5m"  # Missing comma
    "smtp_from": "alerts@example.com"
  }
}
```

---

#### E003: Invalid YAML Syntax
**Message**: Invalid YAML syntax: {error}

**Description**: The YAML configuration file contains syntax errors.

**Resolution**:
- Use a YAML validator to check syntax
- Verify correct indentation (spaces, not tabs)
- Check for unquoted special characters

**Example**:
```yaml
global:
  resolve_timeout: 5m
    smtp_from: alerts@example.com  # Incorrect indentation
```

---

#### E004: Empty Configuration
**Message**: Configuration file is empty

**Description**: The configuration file contains no data.

**Resolution**:
- Add valid Alertmanager configuration
- Check if the file was created correctly

---

#### E005: Invalid Configuration Format
**Message**: Unable to unmarshal configuration: {error}

**Description**: The configuration structure is invalid or doesn't match the expected schema.

**Resolution**:
- Verify field names match Alertmanager schema
- Check data types are correct
- Review the Alertmanager documentation for correct structure

---

### E100-E109: Route Validation Errors

#### E100: Missing Receiver in Route
**Message**: Route must specify a receiver

**Description**: A route configuration is missing the required `receiver` field.

**Resolution**:
- Add a `receiver` field to the route
- Ensure the receiver name matches a defined receiver

**Example**:
```yaml
# ❌ Incorrect
route:
  group_by: ['alertname']

# ✅ Correct
route:
  receiver: 'default'
  group_by: ['alertname']
```

**Related**: E102, Receivers section

---

#### E101: Empty Receiver Name
**Message**: Receiver name cannot be empty

**Description**: The route specifies a receiver, but the name is an empty string.

**Resolution**:
- Provide a valid receiver name
- Ensure the receiver exists in the receivers list

---

#### E102: Receiver Not Found
**Message**: Receiver '{name}' not found in receivers list

**Description**: The route references a receiver that doesn't exist in the configuration.

**Resolution**:
- Add the missing receiver to the `receivers` section
- Fix the receiver name typo if applicable
- Verify receiver names match exactly (case-sensitive)

**Example**:
```yaml
route:
  receiver: 'pagerduty'  # ❌ Referenced but not defined

receivers:
  - name: 'default'  # ❌ Missing 'pagerduty' receiver
```

**Fix**:
```yaml
route:
  receiver: 'pagerduty'

receivers:
  - name: 'default'
  - name: 'pagerduty'  # ✅ Added missing receiver
    pagerduty_configs:
      - routing_key: '${PAGERDUTY_KEY}'
```

---

#### E103: Empty Matcher
**Message**: Matcher cannot be empty

**Description**: A route contains an empty matcher string.

**Resolution**:
- Remove the empty matcher
- Add a valid matcher expression

**Example**:
```yaml
# ❌ Incorrect
route:
  receiver: 'default'
  matchers:
    - 'severity="critical"'
    - ''  # Empty matcher

# ✅ Correct
route:
  receiver: 'default'
  matchers:
    - 'severity="critical"'
```

---

#### E104: Invalid Matcher Syntax
**Message**: Invalid matcher syntax: {error}

**Description**: A matcher expression has invalid syntax.

**Resolution**:
- Use valid matcher syntax: `label="value"`, `label=~"regex"`, `label!="value"`, `label!~"regex"`
- Ensure label names are valid
- Check for proper quoting

**Common Issues**:
- Missing operator: `severity` (should be `severity="critical"`)
- Invalid operator: `severity=="critical"` (should be single `=`)
- Empty value: `severity=""` (value required)
- Invalid label name: `@invalid="value"` (labels must be valid identifiers)

**Example**:
```yaml
# ❌ Incorrect
matchers:
  - 'severity'  # Missing operator
  - 'team=='  # Double equals, empty value
  - '@invalid="value"'  # Invalid label name

# ✅ Correct
matchers:
  - 'severity="critical"'
  - 'team="frontend"'
  - 'alertname=~".*Down"'
```

---

#### E105: Invalid Regex Pattern
**Message**: Invalid regular expression: {error}

**Description**: A regex matcher contains an invalid regular expression.

**Resolution**:
- Fix the regex pattern syntax
- Test the regex separately
- Escape special regex characters properly

**Example**:
```yaml
# ❌ Incorrect
matchers:
  - 'alertname=~"[invalid(regex"'  # Unclosed bracket

# ✅ Correct
matchers:
  - 'alertname=~"^Alert.*Down$"'
```

---

#### E106: Circular Route Reference
**Message**: Circular reference detected in route hierarchy

**Description**: Routes form a circular reference, causing infinite recursion.

**Resolution**:
- Restructure route hierarchy to avoid circular dependencies
- Remove or modify routes causing the cycle

---

#### E107: Invalid Group By Field
**Message**: Invalid group_by field: {field}

**Description**: The `group_by` field contains an invalid label name.

**Resolution**:
- Use valid label names (alphanumeric, underscore, starts with letter)
- Check for typos in label names

---

### E110-E149: Receiver Validation Errors

#### E110: No Receivers Defined
**Message**: No receivers defined. At least one receiver is required.

**Description**: The configuration has no receivers defined, but receivers are required.

**Resolution**:
- Add at least one receiver to the `receivers` section

**Example**:
```yaml
# ❌ Incorrect
receivers: []

# ✅ Correct
receivers:
  - name: 'default'
    email_configs:
      - to: 'team@example.com'
```

---

#### E111: Missing Receiver Name
**Message**: Receiver name is required.

**Description**: A receiver is defined without a name.

**Resolution**:
- Add a `name` field to the receiver
- Ensure the name is unique

**Example**:
```yaml
# ❌ Incorrect
receivers:
  - email_configs:
      - to: 'team@example.com'

# ✅ Correct
receivers:
  - name: 'default'
    email_configs:
      - to: 'team@example.com'
```

---

#### E112: Duplicate Receiver Name
**Message**: Duplicate receiver name '{name}'. Receiver names must be unique.

**Description**: Two or more receivers have the same name.

**Resolution**:
- Rename receivers to ensure uniqueness
- Consolidate duplicate receivers if appropriate

**Example**:
```yaml
# ❌ Incorrect
receivers:
  - name: 'default'
    email_configs: [...]
  - name: 'default'  # Duplicate!
    slack_configs: [...]

# ✅ Correct
receivers:
  - name: 'default'
    email_configs: [...]
  - name: 'slack-alerts'
    slack_configs: [...]
```

---

#### E113: Missing Webhook URL
**Message**: Webhook URL is required.

**Description**: A webhook configuration is missing the required `url` field.

**Resolution**:
- Add a `url` field with a valid HTTP/HTTPS URL

**Example**:
```yaml
# ❌ Incorrect
receivers:
  - name: 'webhook'
    webhook_configs:
      - send_resolved: true  # Missing url

# ✅ Correct
receivers:
  - name: 'webhook'
    webhook_configs:
      - url: 'https://api.example.com/alerts'
        send_resolved: true
```

---

#### E114: Invalid Webhook URL
**Message**: Invalid Webhook URL '{url}': {error}

**Description**: The webhook URL is not a valid URL format.

**Resolution**:
- Ensure the URL is absolute (includes scheme and host)
- Use `http://` or `https://` scheme
- Verify the URL is syntactically correct

**Example**:
```yaml
# ❌ Incorrect
webhook_configs:
  - url: 'not-a-valid-url'
  - url: '/relative/path'  # Must be absolute

# ✅ Correct
webhook_configs:
  - url: 'https://api.example.com/alerts'
```

---

#### E115: Missing Email Recipient
**Message**: Email recipient 'to' is required.

**Description**: An email configuration is missing the required `to` field.

**Resolution**:
- Add a `to` field with a valid email address

**Example**:
```yaml
# ❌ Incorrect
email_configs:
  - from: 'alerts@example.com'  # Missing 'to'

# ✅ Correct
email_configs:
  - to: 'team@example.com'
    from: 'alerts@example.com'
```

---

#### E116: Invalid Email Format (To)
**Message**: Invalid email address format for 'to': '{email}'.

**Description**: The recipient email address has an invalid format.

**Resolution**:
- Provide a valid email address (user@domain.tld)
- Check for typos

**Example**:
```yaml
# ❌ Incorrect
email_configs:
  - to: 'invalid-email'
  - to: 'user@'
  - to: '@example.com'

# ✅ Correct
email_configs:
  - to: 'team@example.com'
```

---

#### E117: Invalid Email Format (From)
**Message**: Invalid email address format for 'from': '{email}'.

**Description**: The sender email address has an invalid format.

**Resolution**:
- Provide a valid email address (user@domain.tld)

---

#### E118: Missing PagerDuty Key
**Message**: PagerDuty service_key or routing_key is required.

**Description**: A PagerDuty configuration must specify either `service_key` (v1) or `routing_key` (v2).

**Resolution**:
- Add either `service_key` or `routing_key`
- Use `routing_key` for PagerDuty Events API v2 (recommended)

**Example**:
```yaml
# ❌ Incorrect
pagerduty_configs:
  - description: 'Alert'  # Missing key

# ✅ Correct (v2)
pagerduty_configs:
  - routing_key: '${PAGERDUTY_ROUTING_KEY}'
    description: 'Alert'

# ✅ Correct (v1)
pagerduty_configs:
  - service_key: '${PAGERDUTY_SERVICE_KEY}'
    description: 'Alert'
```

---

#### E119: Invalid PagerDuty URL
**Message**: Invalid PagerDuty URL '{url}': {error}

**Description**: The PagerDuty API URL is not valid.

---

#### E120: Missing Slack API URL
**Message**: Slack API URL is required.

**Description**: A Slack configuration is missing the required `api_url` field.

**Resolution**:
- Add an `api_url` field with your Slack webhook URL

**Example**:
```yaml
# ❌ Incorrect
slack_configs:
  - channel: '#alerts'  # Missing api_url

# ✅ Correct
slack_configs:
  - api_url: '${SLACK_WEBHOOK_URL}'
    channel: '#alerts'
```

---

#### E121: Invalid Slack API URL
**Message**: Invalid Slack API URL '{url}': {error}

**Description**: The Slack API URL is not valid.

---

#### E122: Missing OpsGenie API Key
**Message**: OpsGenie API key is required.

**Description**: An OpsGenie configuration is missing the required `api_key`.

---

#### E123: Invalid OpsGenie API URL
**Message**: Invalid OpsGenie API URL '{url}': {error}

---

#### E124-E145: Additional Receiver Integration Errors

Similar patterns for other integrations (VictorOps, Pushover, SNS, Telegram, WeChat, MS Teams, Webex, Discord, Google Chat, Custom).

---

### E150-E199: Inhibition Rule Errors

#### E150: Missing Source Matchers
**Message**: Inhibition rule must define source_matchers

**Description**: An inhibition rule is missing the required `source_matchers` field.

**Resolution**:
- Add `source_matchers` to define which alerts will inhibit others

**Example**:
```yaml
# ❌ Incorrect
inhibit_rules:
  - target_matchers:
      - severity="warning"

# ✅ Correct
inhibit_rules:
  - source_matchers:
      - severity="critical"
    target_matchers:
      - severity="warning"
    equal:
      - alertname
```

---

#### E151: Missing Target Matchers
**Message**: Inhibition rule must define target_matchers

**Description**: An inhibition rule is missing the required `target_matchers` field.

**Resolution**:
- Add `target_matchers` to define which alerts will be inhibited

---

#### E152: Invalid Source Matcher
**Message**: Invalid source matcher: {error}

**Description**: A source matcher has invalid syntax.

**Resolution**:
- Fix the matcher syntax (see E104)

---

#### E153: Invalid Target Matcher
**Message**: Invalid target matcher: {error}

**Description**: A target matcher has invalid syntax.

---

### E200-E249: Global Configuration Errors

#### E200: Invalid Global Timeout
**Message**: Invalid global timeout: {error}

**Description**: A global timeout value is invalid or negative.

**Resolution**:
- Use valid duration format (e.g., "5m", "30s", "1h")
- Ensure durations are positive

**Example**:
```yaml
# ❌ Incorrect
global:
  resolve_timeout: "-5m"  # Negative

# ✅ Correct
global:
  resolve_timeout: "5m"
```

---

## Warning Codes (W-series)

### W000-W099: General Warnings

#### W001: Large Configuration File
**Message**: Configuration file is large ({size}KB). Consider splitting.

**Description**: The configuration file is very large, which may impact readability and maintainability.

**Suggestion**:
- Split configuration into multiple files
- Use configuration management tools
- Review for unnecessary complexity

---

### W100-W149: Configuration Warnings

#### W100: Receiver Without Integrations
**Message**: Receiver '{name}' has no integrations defined. It will not send any notifications.

**Description**: A receiver is defined but has no integration configurations (email, slack, etc.).

**Resolution**:
- Add at least one integration (email_configs, slack_configs, etc.)
- Remove the unused receiver

**Example**:
```yaml
# ⚠️ Warning
receivers:
  - name: 'empty-receiver'  # No integrations

# ✅ Fixed
receivers:
  - name: 'email-alerts'
    email_configs:
      - to: 'team@example.com'
```

---

#### W101: Missing Email From Field
**Message**: Email sender 'from' is not specified. Using global default.

**Description**: An email configuration doesn't specify a `from` address, so the global default will be used.

**Suggestion**:
- Explicitly set `from` field for clarity
- Or ensure global `smtp_from` is configured

---

#### W102: Missing Email Smarthost
**Message**: Email smarthost is not specified. Using global default.

---

#### W103: Missing Slack Channel
**Message**: Slack channel or channel_regex is not specified. Alerts might not be delivered.

**Description**: A Slack configuration doesn't specify where to send alerts.

**Resolution**:
- Add `channel` field (e.g., '#alerts')
- Or add `channel_regex` for dynamic routing

---

### W150-W199: Inhibition Warnings

#### W150: No Equal Labels
**Message**: Inhibition rule has no equal labels. This may inhibit unrelated alerts.

**Description**: An inhibition rule doesn't specify `equal` labels, which means it may inhibit alerts that shouldn't be related.

**Suggestion**:
- Add `equal` labels to match related alerts (e.g., instance, cluster)

**Example**:
```yaml
# ⚠️ Warning
inhibit_rules:
  - source_matchers:
      - severity="critical"
    target_matchers:
      - severity="warning"
    # Missing 'equal' field

# ✅ Improved
inhibit_rules:
  - source_matchers:
      - severity="critical"
    target_matchers:
      - severity="warning"
    equal:
      - alertname
      - instance
```

---

#### W151: Overlapping Source and Target
**Message**: Source and target matchers overlap. Rule may not work as expected.

**Description**: The source and target matchers match the same alerts, which may cause unexpected behavior.

---

#### W152: Too Many Equal Labels
**Message**: Inhibition rule has {count} equal labels. Consider reducing for broader inhibition.

**Description**: Many equal labels make inhibition very specific, potentially missing related alerts.

---

#### W153: Empty Equal Labels
**Message**: Equal labels list is empty. Consider adding labels or removing the field.

---

#### W154: Regex in Source Matcher
**Message**: Source matcher uses regex. This may impact performance with many alerts.

**Suggestion**:
- Use exact matchers when possible for better performance

---

#### W155: Complex Target Regex
**Message**: Target matcher uses complex regex. Consider simplifying.

---

#### W156: Broad Inhibition Rule
**Message**: Inhibition rule is very broad and may inhibit many alerts.

---

### W200-W249: Deprecation Warnings

#### W200: Deprecated Continue Field
**Message**: The 'continue' field is deprecated. Use routes with explicit continue behavior.

**Description**: The `continue` field in routes is deprecated in newer Alertmanager versions.

**Resolution**:
- Remove the `continue` field
- Use route hierarchy for correct alert routing

---

#### W201: Deprecated Service Key
**Message**: PagerDuty service_key is deprecated. Use routing_key (Events API v2).

**Description**: The PagerDuty Events API v1 (`service_key`) is deprecated.

**Resolution**:
- Migrate to Events API v2 using `routing_key`

---

### W300-W399: Security Warnings

#### W300: Hardcoded Secret
**Message**: Potential hardcoded secret detected in {field}.

**Description**: A field that typically contains sensitive data appears to have a hardcoded value.

**Resolution**:
- Use environment variables for secrets: `${VARIABLE_NAME}`
- Use secret management tools (Vault, Sealed Secrets, etc.)

**Example**:
```yaml
# ⚠️ Warning - Hardcoded
slack_configs:
  - api_url: 'https://hooks.slack.com/services/T00/B00/XXX'

# ✅ Secure - Environment variable
slack_configs:
  - api_url: '${SLACK_WEBHOOK_URL}'
```

---

#### W301: Insecure Protocol (HTTP)
**Message**: Insecure protocol (HTTP) detected. Consider using HTTPS.

**Description**: A URL uses HTTP instead of HTTPS, which transmits data in plaintext.

**Resolution**:
- Use HTTPS URLs for all integrations
- Configure TLS/SSL on target services

---

#### W310: Weak TLS Version
**Message**: TLS version {version} is weak. Consider using TLS 1.2+.

---

#### W311: Insecure Skip Verify
**Message**: TLS certificate verification is disabled (insecure_skip_verify: true).

**Description**: TLS certificate verification is disabled, which allows MITM attacks.

**Resolution**:
- Set `insecure_skip_verify: false`
- Use proper CA certificates
- Only disable for local development/testing

**Example**:
```yaml
# ⚠️ Insecure
webhook_configs:
  - url: 'https://internal.example.com/alerts'
    http_config:
      tls_config:
        insecure_skip_verify: true  # Dangerous!

# ✅ Secure
webhook_configs:
  - url: 'https://internal.example.com/alerts'
    http_config:
      tls_config:
        ca_file: '/etc/ssl/certs/ca.pem'
        insecure_skip_verify: false
```

---

## Info Codes (I-series)

#### I001: Configuration Parsed Successfully
**Message**: Configuration parsed successfully.

#### I002: Using Default Value
**Message**: Using default value for {field}: {value}

#### I003: Configuration Statistics
**Message**: Configuration has {receivers} receivers, {routes} routes, {inhibit_rules} inhibition rules.

---

## Suggestion Codes (S-series)

#### S001: Add Group By
**Message**: Consider adding 'group_by' labels for better alert grouping.

**Before**:
```yaml
route:
  receiver: 'default'
```

**After**:
```yaml
route:
  receiver: 'default'
  group_by: ['alertname', 'cluster', 'service']
```

---

#### S002: Add Continue to Route
**Message**: Consider adding 'continue: true' to allow alert processing by multiple routes.

---

#### S003: Consolidate Receivers
**Message**: Receivers '{receiver1}' and '{receiver2}' have similar configurations. Consider consolidating.

---

#### S004: Use Regex Matcher
**Message**: Multiple exact matchers can be simplified with regex: {suggestion}

**Before**:
```yaml
routes:
  - receiver: 'team-a'
    matchers:
      - team="frontend"
  - receiver: 'team-a'
    matchers:
      - team="backend"
```

**After**:
```yaml
routes:
  - receiver: 'team-a'
    matchers:
      - team=~"frontend|backend"
```

---

#### S005: Add Documentation
**Message**: Consider adding comments to document complex routing logic.

---

## Error Code Index

Quick reference table of all error codes:

| Code | Category | Severity | Description |
|------|----------|----------|-------------|
| E001 | General | Error | File not found |
| E002 | Parsing | Error | Invalid JSON syntax |
| E003 | Parsing | Error | Invalid YAML syntax |
| E004 | Parsing | Error | Empty configuration |
| E005 | Parsing | Error | Invalid configuration format |
| E100 | Route | Error | Missing receiver in route |
| E101 | Route | Error | Empty receiver name |
| E102 | Route | Error | Receiver not found |
| E103 | Route | Error | Empty matcher |
| E104 | Route | Error | Invalid matcher syntax |
| E105 | Route | Error | Invalid regex pattern |
| E106 | Route | Error | Circular route reference |
| E107 | Route | Error | Invalid group_by field |
| E110 | Receiver | Error | No receivers defined |
| E111 | Receiver | Error | Missing receiver name |
| E112 | Receiver | Error | Duplicate receiver name |
| E113 | Receiver | Error | Missing webhook URL |
| E114 | Receiver | Error | Invalid webhook URL |
| E115 | Receiver | Error | Missing email recipient |
| E116 | Receiver | Error | Invalid email format (to) |
| E117 | Receiver | Error | Invalid email format (from) |
| E118 | Receiver | Error | Missing PagerDuty key |
| E119 | Receiver | Error | Invalid PagerDuty URL |
| E120 | Receiver | Error | Missing Slack API URL |
| E121 | Receiver | Error | Invalid Slack API URL |
| E122 | Receiver | Error | Missing OpsGenie API key |
| E123 | Receiver | Error | Invalid OpsGenie API URL |
| E124-E145 | Receiver | Error | Other integration errors |
| E150 | Inhibition | Error | Missing source matchers |
| E151 | Inhibition | Error | Missing target matchers |
| E152 | Inhibition | Error | Invalid source matcher |
| E153 | Inhibition | Error | Invalid target matcher |
| E200 | Global | Error | Invalid global timeout |
| W100 | Receiver | Warning | Receiver without integrations |
| W101 | Receiver | Warning | Missing email from field |
| W102 | Receiver | Warning | Missing email smarthost |
| W103 | Receiver | Warning | Missing Slack channel |
| W150 | Inhibition | Warning | No equal labels |
| W151 | Inhibition | Warning | Overlapping source and target |
| W152 | Inhibition | Warning | Too many equal labels |
| W153 | Inhibition | Warning | Empty equal labels |
| W154 | Inhibition | Warning | Regex in source matcher |
| W155 | Inhibition | Warning | Complex target regex |
| W156 | Inhibition | Warning | Broad inhibition rule |
| W200 | Deprecation | Warning | Deprecated continue field |
| W201 | Deprecation | Warning | Deprecated service key |
| W300 | Security | Warning | Hardcoded secret |
| W301 | Security | Warning | Insecure protocol (HTTP) |
| W310 | Security | Warning | Weak TLS version |
| W311 | Security | Warning | Insecure skip verify |

---

## Getting Help

If you encounter an error code not documented here or need additional assistance:

1. Check the [User Guide](USER_GUIDE.md) for general usage
2. Review [Examples](EXAMPLES.md) for common patterns
3. Consult the Alertmanager documentation: https://prometheus.io/docs/alerting/latest/configuration/
4. Open an issue on the project repository

---

## Contributing

If you find an error code that needs clarification or want to add examples:

1. Submit a PR with documentation improvements
2. Include real-world examples where helpful
3. Ensure error codes are accurate and up-to-date

---

**Last Updated**: 2025-11-24
**Validator Version**: 1.0.0 (TN-151 Implementation)
