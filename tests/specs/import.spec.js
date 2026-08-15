import { test, expect } from '@playwright/test';
import { confirm, resetDB } from '../helpers.js';

const IMPORT = '/admin/subscribers/import';
const SUBSCRIBERS = '/admin/subscribers';
const CSV = new URL('../fixtures/subs.csv', import.meta.url).pathname;

// Pick a list in the taginput selector. Filling the input matches a datalist
// option, which the component turns into a tag carrying the list's ID.
async function selectList(page, name) {
  const input = page.locator('ot-taginput input');
  await input.click();
  await input.fill(name);
  await expect(page.locator('ot-taginput .badge')).toContainText(name);
}

test.describe.configure({ mode: 'serial' });

test.describe('Import', () => {
  test.beforeAll(async ({ browser }) => {
    await resetDB(browser);
  });

  test('imports subscribers across modes and statuses', async ({ page }) => {
    const cases = [
      { mode: 'subscribe', subStatus: 'unconfirmed', overwrite: true },
      { mode: 'subscribe', subStatus: 'confirmed', overwrite: true },
      { mode: 'subscribe', subStatus: 'unconfirmed', overwrite: false },
      { mode: 'blocklist', subStatus: 'unsubscribed', overwrite: false },
    ];

    for (const c of cases) {
      await page.goto(IMPORT);
      await page.getByTestId(`check-${c.mode}`).check();
      await page.getByTestId(`check-${c.subStatus}`).check();

      if (c.overwrite) {
        await page.getByTestId('overwrite-user-info').check();
        await page.getByTestId('overwrite-sub-status').check();
      }

      if (c.mode === 'subscribe') {
        await selectList(page, 'Default list');
      }

      await page.locator('input[type=file]').setInputFiles(CSV);
      await page.getByTestId('btn-upload').click();

      // Overwriting the subscription status prompts for confirmation.
      if (c.overwrite) {
        await confirm(page);
      }

      // Wait for the import to finish, then clear it. Clearing reloads the
      // page back to the upload form.
      await expect(page.getByTestId('btn-done')).toBeVisible();
      await page.getByTestId('btn-done').click();
      await expect(page.getByTestId('btn-upload')).toBeVisible();

      // 100 imported + 2 seeded subscribers.
      await page.goto(SUBSCRIBERS);
      await expect(page.locator('.page-title')).toContainText('(102)');
    }
  });

  test.describe('rejects a malformed file', () => {
    test.beforeAll(async ({ browser }) => {
      await resetDB(browser);
    });

    test('fails on a wrong delimiter', async ({ page }) => {
      await page.goto(IMPORT);
      await selectList(page, 'Default list');
      await page.locator('input[name=delim]').fill('|');
      await page.locator('input[type=file]').setInputFiles(CSV);
      await page.getByTestId('btn-upload').click();

      await expect(page.locator('.stat-value')).toContainText('failed');
    });
  });
});
