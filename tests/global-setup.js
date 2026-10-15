import { chromium } from '@playwright/test';
import { mkdirSync } from 'node:fs';
import { login } from './helpers.js';
import { launchOptions } from './browser.js';

// Login and create a persistent session for tests.
export default async function globalSetup() {
  mkdirSync(new URL('.auth', import.meta.url).pathname, { recursive: true });

  const browser = await chromium.launch(launchOptions);
  await login(await browser.newContext());
  await browser.close();
}
