# TN-130: Inhibition API Endpoints - Requirements

**Task**: HTTP API for inhibition rules and status
**Priority**: HIGH
**Dependencies**: TN-127, TN-129
**Estimated**: 2 hours

## Key Requirements

1. **GET /api/v2/inhibition/rules** - list rules
2. **GET /api/v2/inhibition/status** - check if alert inhibited
3. **POST /api/v2/inhibition/check** - check specific alert
4. **Integration**: main.go + AlertProcessor
5. **OpenAPI 3.0 spec**

## Acceptance Criteria

- [ ] 3 HTTP endpoints (Alertmanager compatible)
- [ ] main.go integration
- [ ] AlertProcessor integration
- [ ] OpenAPI spec
- [ ] Module documentation

**Status**: READY

