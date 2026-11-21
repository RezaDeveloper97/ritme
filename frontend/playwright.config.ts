import { tmpdir } from 'node:os';
import { join } from 'node:path';

import { defineConfig } from '@playwright/test';

import { E2E } from './e2e/support/env';

/**
 * Browser smoke harness (T-M12-25). Runs the main routine against a LOCAL stack: backend-go with
 * `SMS_PROVIDER=log` + its own database, the Next.js dev server on :3100. Boot recipe and env vars:
 * `frontend/CLAUDE.md` § Testing (Playwright smoke). Specs are `e2e/*.e2e.ts` (vitest never sees them).
 */
export default defineConfig({
  testDir: './e2e',
  testMatch: /.*\.e2e\.ts$/,
  globalSetup: './e2e/global-setup.ts',
  timeout: 150_000,
  expect: { timeout: 15_000 },
  // One worker: every journey asks for an OTP and send-otp allows 5 per minute per IP.
  workers: 1,
  fullyParallel: false,
  retries: 0,
  reporter: [['list']],
  // Outside the project: `next dev` watches frontend/, and traces written inside it trigger reload loops mid-test.
  outputDir: join(tmpdir(), 'ritme-e2e-results'),
  use: {
    baseURL: E2E.webUrl,
    locale: 'fa-IR',
    timezoneId: 'Asia/Tehran',
    viewport: { width: 390, height: 844 },
    actionTimeout: 15_000,
    navigationTimeout: 60_000,
    trace: 'off',
    screenshot: 'only-on-failure',
    // Locally the system Chrome (no browser download needed); CI installs Playwright's chromium.
    channel: E2E.browserChannel,
  },
  webServer: E2E.startWeb
    ? {
        command: `npx next dev -p ${E2E.webPort}`,
        url: E2E.webUrl,
        reuseExistingServer: true,
        timeout: 180_000,
        env: { NEXT_PUBLIC_API_BASE_URL: `${E2E.apiUrl}/api/v1` },
      }
    : undefined,
});
