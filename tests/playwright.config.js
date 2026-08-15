import { defineConfig, devices } from '@playwright/test';

// listmonk root address (see ../config.toml).
const BASE_URL = process.env.LISTMONK_URL || 'http://localhost:9000';

export default defineConfig({
  testDir: './specs',
  workers: 1,
  reporter: process.env.CI ? 'list' : [['list'], ['html', { open: 'never' }]],

  // Log in once and save the session before any tests run.
  globalSetup: './global-setup.js',

  // Create a fresh listmonk DB instance for every test run.
  webServer: {
    command: './listmonk --install --yes && ./listmonk --static-dir static',
    cwd: '..',
    url: `${BASE_URL}/health`,
    reuseExistingServer: false,
    env: { LISTMONK_ADMIN_USER: 'admin', LISTMONK_ADMIN_PASSWORD: 'listmonk' },
  },

  use: {
    baseURL: BASE_URL,
    storageState: './.auth/admin.json',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },

  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'], viewport: { width: 1400, height: 950 } } },
  ],
});
