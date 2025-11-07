import { defineConfig, devices } from '@playwright/test';

/**
 * E2E Testing Configuration for TN-136: Silence UI Components
 *
 * This configuration sets up Playwright for testing the Silence Management UI
 * with real-time WebSocket updates, PWA features, and accessibility validation.
 *
 * Run tests:
 *   npm run test:e2e           # Run all E2E tests
 *   npm run test:e2e:ui        # Run with UI mode
 *   npm run test:e2e:debug     # Debug mode
 *   npm run test:e2e:report    # Show HTML report
 */
export default defineConfig({
  testDir: './tests',

  // Maximum time one test can run for
  timeout: 30 * 1000,

  // Run tests in files in parallel
  fullyParallel: true,

  // Fail the build on CI if you accidentally left test.only in the source code
  forbidOnly: !!process.env.CI,

  // Retry on CI only
  retries: process.env.CI ? 2 : 0,

  // Opt out of parallel tests on CI
  workers: process.env.CI ? 1 : undefined,

  // Reporter to use
  reporter: [
    ['html', { outputFolder: 'playwright-report' }],
    ['json', { outputFile: 'test-results/results.json' }],
    ['junit', { outputFile: 'test-results/junit.xml' }],
    ['list'],
  ],

  use: {
    // Base URL for tests
    baseURL: process.env.BASE_URL || 'http://localhost:8080',

    // Collect trace when retrying the failed test
    trace: 'on-first-retry',

    // Screenshot on failure
    screenshot: 'only-on-failure',

    // Video on failure
    video: 'retain-on-failure',

    // Emulate locale and timezone
    locale: 'en-US',
    timezoneId: 'America/New_York',
  },

  // Configure projects for major browsers
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },
    // Mobile viewports
    {
      name: 'Mobile Chrome',
      use: { ...devices['Pixel 5'] },
    },
    {
      name: 'Mobile Safari',
      use: { ...devices['iPhone 12'] },
    },
    // Accessibility tests
    {
      name: 'accessibility',
      use: {
        ...devices['Desktop Chrome'],
        // Run accessibility tests with axe-core
      },
    },
  ],

  // Run your local dev server before starting the tests
  webServer: {
    command: 'cd ../go-app && go run cmd/server/main.go',
    url: 'http://localhost:8080/healthz',
    reuseExistingServer: !process.env.CI,
    timeout: 120 * 1000,
  },
});
