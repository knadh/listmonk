import { chromium } from '@playwright/test';
import { mkdirSync } from 'node:fs';
import { getMail, login } from './helpers.js';
import { launchOptions } from './browser.js';

// Login and create a persistent session for tests.
export default async function globalSetup() {
  // Check the shared mail service once before every test tries to reset it.
  try {
    await getMail();
  } catch (err) {
    throw new Error(`Cannot reach MailHog at ${process.env.MAILHOG_URL || 'http://localhost:8025'}. Ensure the service is running and reachable from the test container.`, { cause: err });
  }

  mkdirSync(new URL('.auth', import.meta.url).pathname, { recursive: true });

  const browser = await chromium.launch(launchOptions);
  await login(await browser.newContext());
  await browser.close();
}
