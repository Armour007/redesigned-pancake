import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  timeout: 120_000,
  use: {
    baseURL: process.env.FRONTEND_BASE || 'http://localhost:5173',
    trace: 'retain-on-failure'
  },
  // No webServer: we assume frontend dev server already running on 5173
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } }
  ]
});
