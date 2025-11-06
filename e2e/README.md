# E2E Testing Guide for TN-136: Silence UI Components

## 📋 Overview

This directory contains End-to-End (E2E) tests for the Silence Management UI using [Playwright](https://playwright.dev/).

**Status**: 🚧 **INFRASTRUCTURE READY** (Tests pending implementation)
**Target Coverage**: 80%+ of critical user flows
**Browsers**: Chromium, Firefox, WebKit (Desktop + Mobile)
**Accessibility**: WCAG 2.1 AA compliance validation

---

## 🚀 Quick Start

### Prerequisites

```bash
# Node.js >= 18.x required
node --version

# Install dependencies
cd e2e
npm install

# Install Playwright browsers
npx playwright install
```

### Run Tests

```bash
# Run all E2E tests
npm run test:e2e

# Run with UI mode (interactive)
npm run test:e2e:ui

# Run specific browser
npm run test:e2e:chrome
npm run test:e2e:firefox
npm run test:e2e:mobile

# Debug mode
npm run test:e2e:debug

# View HTML report
npm run test:e2e:report
```

---

## 📁 Directory Structure

```
e2e/
├── playwright.config.ts       # Playwright configuration
├── package.json               # Dependencies
├── tests/
│   ├── silence-dashboard.spec.ts   # Dashboard tests ✅
│   ├── silence-create.spec.ts      # Create form tests (TODO)
│   ├── silence-detail.spec.ts      # Detail view tests (TODO)
│   ├── silence-edit.spec.ts        # Edit form tests (TODO)
│   ├── silence-templates.spec.ts   # Templates tests (TODO)
│   ├── silence-analytics.spec.ts   # Analytics tests (TODO)
│   ├── websocket.spec.ts           # WebSocket tests (TODO)
│   ├── pwa.spec.ts                 # PWA offline tests (TODO)
│   └── accessibility.spec.ts       # Accessibility tests (TODO)
├── fixtures/                  # Test data fixtures
├── helpers/                   # Test utilities
└── playwright-report/         # Generated test reports
```

---

## 🧪 Test Suites

### 1. Dashboard Tests (`silence-dashboard.spec.ts`)

**Status**: ✅ Skeleton implemented

Tests:
- [x] Dashboard loads successfully
- [x] Filter by status (active/pending/expired)
- [x] Bulk delete operations
- [x] Pagination navigation
- [x] Real-time WebSocket updates
- [x] WCAG 2.1 AA accessibility
- [x] Keyboard navigation
- [x] Mobile responsive layout
- [x] URL query param persistence

**Coverage**: ~70% of dashboard flows

---

### 2. Create Form Tests (`silence-create.spec.ts`)

**Status**: 🔄 Partially implemented

Planned tests:
- [ ] Create silence with valid data
- [ ] Form validation (required fields)
- [ ] Add/remove matchers dynamically
- [ ] Time presets (1h, 4h, 8h, 24h)
- [ ] Regex matcher validation
- [ ] Maximum matcher limit (100)
- [ ] Character counter (comment field)
- [ ] Cancel confirmation dialog

---

### 3. Detail View Tests (`silence-detail.spec.ts`)

**Status**: ⏳ TODO

Planned tests:
- [ ] View silence details
- [ ] Matched alerts count updates
- [ ] Quick actions (extend, clone)
- [ ] Delete confirmation
- [ ] Auto-refresh every 10 seconds
- [ ] Status badge updates

---

### 4. Edit Form Tests (`silence-edit.spec.ts`)

**Status**: ⏳ TODO

Planned tests:
- [ ] Edit silence comment
- [ ] Extend end time
- [ ] Update matchers
- [ ] Read-only fields (creator, start time)
- [ ] Validation on update
- [ ] Discard changes confirmation

---

### 5. Templates Tests (`silence-templates.spec.ts`)

**Status**: ⏳ TODO

Planned tests:
- [ ] View template library
- [ ] Preview template (modal)
- [ ] Use template (pre-fill form)
- [ ] 3 built-in templates (Maintenance, OnCall, Incident)

---

### 6. Analytics Tests (`silence-analytics.spec.ts`)

**Status**: ⏳ TODO

Planned tests:
- [ ] View analytics dashboard
- [ ] Statistics cards (Total, Active, Pending, Expired)
- [ ] Top creators table
- [ ] Most silenced alerts
- [ ] Time range selector (24h, 7d, 30d, 90d)
- [ ] Auto-refresh every 5 minutes

---

### 7. WebSocket Tests (`websocket.spec.ts`)

**Status**: ⏳ TODO

Planned tests:
- [ ] WebSocket connection established
- [ ] Receive `silence_created` event
- [ ] Receive `silence_updated` event
- [ ] Receive `silence_deleted` event
- [ ] Receive `silence_expired` event
- [ ] Reconnect on connection loss
- [ ] Ping/pong keep-alive
- [ ] Toast notifications on events

---

### 8. PWA Tests (`pwa.spec.ts`)

**Status**: ⏳ TODO

Planned tests:
- [ ] Service Worker registration
- [ ] Offline page fallback
- [ ] Cache-first for static assets
- [ ] Network-first for UI pages
- [ ] Install prompt (add to home screen)
- [ ] Manifest validation

---

### 9. Accessibility Tests (`accessibility.spec.ts`)

**Status**: ⏳ TODO

Planned tests:
- [ ] WCAG 2.1 AA compliance (all pages)
- [ ] Keyboard navigation (Tab, Enter, Escape)
- [ ] Screen reader compatibility (ARIA labels)
- [ ] Focus indicators visible
- [ ] Color contrast ratios
- [ ] Touch target sizes (44px minimum)

---

## 🎯 Testing Strategy

### Critical User Flows (Priority 1)

1. **Create Silence** (end-to-end)
   - Navigate to dashboard → Click "Create" → Fill form → Submit → Verify creation

2. **Edit Silence** (end-to-end)
   - View silence → Click "Edit" → Modify fields → Save → Verify update

3. **Delete Silence** (end-to-end)
   - View silence → Click "Delete" → Confirm → Verify deletion

4. **Bulk Delete** (dashboard)
   - Select multiple → Click "Delete Selected" → Confirm → Verify

5. **Real-time Updates** (WebSocket)
   - Open dashboard → Create silence in another tab → Verify toast notification

---

### Performance Benchmarks

Target metrics (Lighthouse):
- **Performance**: >90
- **Accessibility**: 100
- **Best Practices**: >90
- **SEO**: >80

---

## 🐛 Debugging Tests

### Visual Debugging

```bash
# Run with browser visible
npm run test:e2e:headed

# Run in UI mode (interactive)
npm run test:e2e:ui

# Debug specific test
npx playwright test tests/silence-dashboard.spec.ts --debug
```

### Trace Viewer

```bash
# Generate trace on failure
npx playwright test --trace on

# View trace
npx playwright show-trace trace.zip
```

### Screenshots & Videos

Playwright automatically captures:
- **Screenshots**: On test failure
- **Videos**: On test failure (retained)
- **Traces**: On first retry

---

## 🔧 Configuration

### Environment Variables

```bash
# Base URL for tests
export BASE_URL=http://localhost:8080

# Test timeout (milliseconds)
export TEST_TIMEOUT=30000

# Number of workers (parallel tests)
export WORKERS=4
```

### Custom Config

Edit `playwright.config.ts` to customize:
- Browsers to test
- Viewport sizes
- Timeouts
- Reporters
- WebServer command

---

## 📊 Test Reports

After running tests, view reports:

```bash
# HTML report (interactive)
npm run test:e2e:report

# JSON report
cat test-results/results.json | jq

# JUnit XML (for CI integration)
cat test-results/junit.xml
```

---

## 🚀 CI/CD Integration

### GitHub Actions

```yaml
name: E2E Tests

on:
  pull_request:
    branches: [main]
  push:
    branches: [main]

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '18'

      - name: Install dependencies
        run: |
          cd e2e
          npm ci
          npx playwright install --with-deps

      - name: Start Go server
        run: |
          cd go-app
          go run cmd/server/main.go &
          sleep 5

      - name: Run E2E tests
        run: |
          cd e2e
          npm run test:e2e

      - name: Upload test reports
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: playwright-report
          path: e2e/playwright-report/
```

---

## 📚 Resources

- [Playwright Documentation](https://playwright.dev/docs/intro)
- [Axe Accessibility Testing](https://github.com/dequelabs/axe-core)
- [WCAG 2.1 AA Guidelines](https://www.w3.org/WAI/WCAG21/quickref/)
- [TN-136 Requirements](../tasks/go-migration-analysis/TN-136-silence-ui-components/requirements.md)
- [TN-136 Design](../tasks/go-migration-analysis/TN-136-silence-ui-components/design.md)

---

## 🔜 Next Steps

1. **Implement remaining test suites** (8 TODOs)
2. **Add test fixtures** (sample silence data)
3. **Create test helpers** (common actions, assertions)
4. **Integrate with CI/CD** (GitHub Actions)
5. **Run performance tests** (Lighthouse)
6. **Add visual regression tests** (Percy/Chromatic)

---

**Document Version**: 1.0
**Created**: 2025-11-06
**Author**: Alert History Team
**Status**: Infrastructure Ready, Tests Pending
