import { test, expect } from '@playwright/test';
import { openMenu, confirm, resetDB } from '../helpers.js';

const BOUNCES = '/admin/subscribers/bounces';

function bounceRow(page, email) {
  const rows = page.getByTestId('bounce-row');
  return email ? rows.filter({ hasText: email }) : rows;
}

// Register a bounce for an email via the webhook endpoint.
async function postBounce(page, type, email) {
  const res = await page.request.post('/webhooks/bounce', { data: { source: 'api', type, email } });
  expect(res.ok()).toBeTruthy();
}

async function subStatus(page, id) {
  const res = await page.request.get(`/api/subscribers/${id}`);
  return res.ok() ? (await res.json()).data.status : null;
}

test.describe.configure({ mode: 'serial' });

test.describe('Bounces', () => {
  let subs;

  // Bounce processing + webhooks are enabled at install (they're read at boot).
  test.beforeAll(async ({ browser }) => {
    await resetDB(browser, { bounces: true });
  });

  test('reads the seeded subscribers', async ({ page }) => {
    subs = (await (await page.request.get('/api/subscribers')).json()).data.results;
    expect(subs.length).toBe(2);
  });

  test('applies bounce actions by type', async ({ page }) => {
    const [a, b] = subs;

    // Soft bounces don't cross the threshold: the subscriber stays enabled.
    await postBounce(page, 'soft', a.email);
    await postBounce(page, 'soft', a.email);
    expect(await subStatus(page, a.id)).toBe('enabled');

    // Hard bounces blocklist.
    await postBounce(page, 'hard', a.email);
    await postBounce(page, 'hard', a.email);
    expect(await subStatus(page, a.id)).toBe('blocklisted');

    // Complaints delete the subscriber outright.
    await postBounce(page, 'complaint', b.email);
    const res = await page.request.get(`/api/subscribers/${b.id}`);
    expect(res.status()).toBe(400);
  });

  test('lists the recorded bounces', async ({ page }) => {
    await page.goto(BOUNCES);

    // The complaint subscriber (and its bounce) were deleted. The blocklisted
    // subscriber kept its bounces up to the point it was blocklisted.
    const count = await bounceRow(page).count();
    expect(count).toBeGreaterThan(0);
    await expect(bounceRow(page, subs[0].email)).toHaveCount(count);
    await expect(bounceRow(page, subs[1].email)).toHaveCount(0);
  });

  test('views a bounce record and deletes it', async ({ page }) => {
    await page.goto(BOUNCES);
    const before = await bounceRow(page).count();

    const row = bounceRow(page).first();
    await openMenu(row);
    await row.getByTestId('btn-meta').click();
    await expect(page.locator('dialog[open] pre')).toBeVisible();
    await page.locator('dialog[open] button').click();

    await openMenu(row);
    await row.getByTestId('btn-delete').click();
    await confirm(page);
    await expect(bounceRow(page)).toHaveCount(before - 1);
  });
});
