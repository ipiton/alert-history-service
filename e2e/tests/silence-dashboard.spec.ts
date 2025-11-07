import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

/**
 * E2E Tests for Silence Dashboard (TN-136)
 *
 * Tests cover:
 * - Dashboard rendering
 * - Filters functionality
 * - Bulk operations
 * - Pagination
 * - WebSocket real-time updates
 * - Accessibility (WCAG 2.1 AA)
 */

test.describe('Silence Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to silence dashboard
    await page.goto('/ui/silences');

    // Wait for page to load
    await page.waitForLoadState('networkidle');
  });

  test('should load dashboard successfully', async ({ page }) => {
    // Check page title
    await expect(page).toHaveTitle(/Silence Management/i);

    // Check main elements are visible
    await expect(page.locator('h1')).toContainText('Silence Dashboard');
    await expect(page.locator('.dashboard-filters')).toBeVisible();
    await expect(page.locator('.silences-table')).toBeVisible();
  });

  test('should filter silences by status', async ({ page }) => {
    // Click status filter dropdown
    await page.locator('#filter-status').click();

    // Select "Active" status
    await page.locator('[data-value="active"]').click();

    // Wait for filter to apply
    await page.waitForResponse(resp =>
      resp.url().includes('/api/v2/silences') && resp.status() === 200
    );

    // Verify filtered results
    const statusBadges = page.locator('.status-badge');
    const count = await statusBadges.count();

    for (let i = 0; i < count; i++) {
      await expect(statusBadges.nth(i)).toContainText(/active/i);
    }
  });

  test('should perform bulk delete operation', async ({ page }) => {
    // Select multiple silences
    await page.locator('input[type="checkbox"][name="select-silence"]').nth(0).check();
    await page.locator('input[type="checkbox"][name="select-silence"]').nth(1).check();

    // Click bulk delete button
    await page.locator('button:has-text("Delete Selected")').click();

    // Confirm deletion in modal
    await page.locator('button:has-text("Confirm")').click();

    // Wait for success message
    await expect(page.locator('.toast-success')).toBeVisible();
    await expect(page.locator('.toast-success')).toContainText(/deleted successfully/i);
  });

  test('should navigate through pagination', async ({ page }) => {
    // Check initial page
    await expect(page.locator('.pagination .current-page')).toContainText('1');

    // Click next page
    await page.locator('button:has-text("Next")').click();

    // Wait for new data to load
    await page.waitForResponse(resp =>
      resp.url().includes('/api/v2/silences') && resp.status() === 200
    );

    // Verify page changed
    await expect(page.locator('.pagination .current-page')).toContainText('2');
  });

  test('should receive real-time WebSocket updates', async ({ page }) => {
    // Listen for WebSocket connection
    const wsPromise = page.waitForEvent('websocket', ws =>
      ws.url().includes('/ws/silences')
    );

    const ws = await wsPromise;
    expect(ws.url()).toContain('/ws/silences');

    // Wait for initial connection
    await page.waitForTimeout(1000);

    // Simulate silence creation in another tab/user
    // (In real test, would trigger via API or second browser context)

    // Verify toast notification appears
    await expect(page.locator('.toast-info')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('.toast-info')).toContainText(/silence created/i);
  });

  test('should be accessible (WCAG 2.1 AA)', async ({ page }) => {
    // Run axe accessibility scan
    const accessibilityScanResults = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
      .analyze();

    // Assert no accessibility violations
    expect(accessibilityScanResults.violations).toEqual([]);
  });

  test('should be keyboard navigable', async ({ page }) => {
    // Focus on first focusable element
    await page.keyboard.press('Tab');

    // Check filter dropdown is focused
    await expect(page.locator('#filter-status')).toBeFocused();

    // Navigate to create button
    for (let i = 0; i < 5; i++) {
      await page.keyboard.press('Tab');
    }

    // Press Enter to activate create button
    await page.keyboard.press('Enter');

    // Verify navigation to create form
    await expect(page).toHaveURL(/\/ui\/silences\/create/);
  });

  test('should display responsive layout on mobile', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });

    // Check mobile-specific elements
    await expect(page.locator('.mobile-menu-toggle')).toBeVisible();

    // Verify table is scrollable horizontally
    const table = page.locator('.silences-table');
    const scrollWidth = await table.evaluate(el => el.scrollWidth);
    const clientWidth = await table.evaluate(el => el.clientWidth);

    expect(scrollWidth).toBeGreaterThan(clientWidth);
  });

  test('should persist filters in URL query params', async ({ page }) => {
    // Apply filters
    await page.locator('#filter-status').selectOption('active');
    await page.locator('#filter-creator').fill('ops@example.com');

    // Wait for URL to update
    await page.waitForURL(/status=active/);
    await page.waitForURL(/creator=ops@example.com/);

    // Reload page
    await page.reload();

    // Verify filters persisted
    await expect(page.locator('#filter-status')).toHaveValue('active');
    await expect(page.locator('#filter-creator')).toHaveValue('ops@example.com');
  });
});

test.describe('Create Silence Form', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/ui/silences/create');
    await page.waitForLoadState('networkidle');
  });

  test('should create silence successfully', async ({ page }) => {
    // Fill in required fields
    await page.locator('#creator').fill('test@example.com');
    await page.locator('#comment').fill('Test maintenance window');

    // Set time range (2 hours from now)
    await page.locator('button:has-text("2 Hours")').click();

    // Add matcher
    await page.locator('input[name="matchers[0][name]"]').fill('alertname');
    await page.locator('select[name="matchers[0][operator]"]').selectOption('=');
    await page.locator('input[name="matchers[0][value]"]').fill('HighCPU');

    // Submit form
    await page.locator('button[type="submit"]:has-text("Create Silence")').click();

    // Wait for success toast
    await expect(page.locator('.toast-success')).toBeVisible();

    // Verify redirect to dashboard
    await expect(page).toHaveURL(/\/ui\/silences$/);
  });

  test('should validate required fields', async ({ page }) => {
    // Try to submit without filling required fields
    await page.locator('button[type="submit"]').click();

    // Check validation errors
    await expect(page.locator('.field-error')).toHaveCount(3); // creator, comment, time
  });

  test('should add/remove matchers dynamically', async ({ page }) => {
    // Initial matcher count
    const initialCount = await page.locator('.matcher-row').count();

    // Add new matcher
    await page.locator('button:has-text("Add Matcher")').click();

    // Verify matcher added
    await expect(page.locator('.matcher-row')).toHaveCount(initialCount + 1);

    // Remove matcher
    await page.locator('.btn-remove-matcher').last().click();

    // Verify matcher removed
    await expect(page.locator('.matcher-row')).toHaveCount(initialCount);
  });
});

// TODO: Add more test suites for:
// - Detail view page
// - Edit form page
// - Templates page
// - Analytics dashboard
// - PWA offline functionality
// - WebSocket reconnection
// - Error handling scenarios
