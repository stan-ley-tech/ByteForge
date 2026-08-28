import { defineConfig, devices } from "@playwright/test";

// E2E tests drive the real, built frontend against the real Go backend —
// no mocked network layer. The server under test points its own requests
// at itself (see e2e/run.spec.ts), which keeps the suite deterministic
// without depending on a public API being reachable in CI.
export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: "list",
  use: {
    baseURL: "http://127.0.0.1:8089",
    trace: "on-first-retry",
  },
  webServer: {
    command: "npm run e2e:server",
    url: "http://127.0.0.1:8089/healthz",
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
