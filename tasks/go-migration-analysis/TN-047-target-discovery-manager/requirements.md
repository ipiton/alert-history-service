# TN-047: Target Discovery Manager - Requirements

## 1. Overview

Target Discovery Manager автоматически обнаруживает publishing targets из Kubernetes Secrets, парсит конфигурацию и предоставляет unified interface для access к targets.

## 2. Functional Requirements

### FR-1: Target Discovery
- List secrets через K8s client с label selector "publishing-target=true"
- Parse secret data в PublishingTarget structures
- Validate target configuration
- Store discovered targets в in-memory cache

### FR-2: Target Management
- GetTarget(name) - retrieve specific target
- ListTargets() - get all active targets
- GetTargetsByType(type) - filter by type (rootly/pagerduty/slack/webhook)

### FR-3: Secret Parsing
- Parse base64 encoded secret data
- Support fields: name, type, url, format, enabled, auth headers
- Validate required fields (name, type, url)
- Handle malformed secrets gracefully

## 3. Non-Functional Requirements

- Performance: Discovery <2s for 20 secrets
- Thread-safe operations
- 80%+ test coverage
- Zero breaking changes

## 4. Dependencies

- ✅ TN-046: K8s Client
- core.PublishingTarget model

## 5. Acceptance Criteria

- [ ] TargetDiscoveryManager interface defined
- [ ] DefaultTargetDiscoveryManager implemented
- [ ] Secret parsing logic complete
- [ ] Target validation implemented
- [ ] 10+ unit tests passing
- [ ] Documentation complete

---
**Status**: Ready for implementation
**Estimated**: 8-10 hours
