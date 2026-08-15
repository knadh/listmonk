import { chromium } from '@playwright/test';
import { mkdirSync } from 'node:fs';

const BASE_URL = process.env.LISTMONK_URL || 'http://localhost:9000';

// Login and create a persistent session for tests.
export default async function globalSetup() {
  const browser = await chromium.launch();
  const page = await browser.newPage();

  await page.goto(`${BASE_URL}/admin/login?next=/admin/lists`);
  await page.getByLabel('Username').fill('admin');
  await page.getByLabel('Password').fill('listmonk');
  await page.getByRole('button', { name: 'Login' }).click();
  await page.waitForURL('**/admin/lists');

  mkdirSync(new URL('.auth', import.meta.url).pathname, { recursive: true });
  await page.context().storageState({ path: new URL('.auth/admin.json', import.meta.url).pathname });
  await browser.close();
}
