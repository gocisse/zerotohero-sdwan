import { defineConfig, devices } from '@playwright/test'

// The suite runs against the already-running docker-compose stack (see `make e2e`).
// Tests share one database, so they run serially in file order.
export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  workers: 1,
  retries: 0,
  timeout: 90_000,
  expect: { timeout: 10_000 },
  reporter: [['list']],
  use: {
    baseURL: process.env.E2E_BASE_URL ?? 'http://localhost:5173',
    ...devices['Desktop Chrome'],
    viewport: { width: 1280, height: 1000 },
    trace: 'off',
  },
})
