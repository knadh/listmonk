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

test.describe('Bounces: bulk actions', () => {
  test.beforeEach(async ({ browser }) => {
    await resetDB(browser, { bounces: true });
  });

  test('rejects invalid scopes, preserves explicit IDs, and uses bounce permissions for blocklisting', async ({ page, browser }) => {
    const data = async (res) => {
      expect(res.ok(), await res.text()).toBeTruthy();
      return (await res.json()).data;
    };
    const subs = (await data(await page.request.get('/api/subscribers'))).results;
    for (const sub of subs) await postBounce(page, 'soft', sub.email);
    const bounces = (await data(await page.request.get('/api/bounces'))).results;
    expect(bounces).toHaveLength(2);
    for (const url of ['?all=true&campaign_id=invalid', '?all=true&id=invalid', '']) {
      expect((await page.request.delete(`/api/bounces${url}`)).status()).toBe(400);
    }
    expect((await page.request.put('/api/bounces/blocklist?id=invalid')).status()).toBe(400);
    await data(await page.request.delete('/api/bounces?all=true&source=missing'));
    await data(await page.request.put('/api/bounces/blocklist?source=missing'));
    for (const sub of subs) expect(await subStatus(page, sub.id)).toBe('enabled');

    const role = await data(await page.request.post('/api/roles/users', { data: {
      name: 'Bounce manager', permissions: ['bounces:get', 'bounces:manage'],
    } }));
    const user = await data(await page.request.post('/api/users', { data: {
      username: 'bounce-manager', type: 'api', status: 'enabled', user_role_id: role.id,
    } }));
    const context = await browser.newContext({
      storageState: { cookies: [], origins: [] },
      extraHTTPHeaders: { Authorization: `Basic ${Buffer.from(`${user.username}:${user.password}`).toString('base64')}` },
    });
    try {
      await data(await context.request.put(`/api/bounces/blocklist?id=${bounces[0].id}`));
    } finally {
      await context.close();
    }
    expect(await subStatus(page, bounces[0].subscriber_id)).toBe('blocklisted');
    expect(await subStatus(page, bounces[1].subscriber_id)).toBe('enabled');
    await data(await page.request.delete(`/api/bounces?all=true&id=${bounces[0].id}`));
    expect((await data(await page.request.get('/api/bounces'))).results.map((b) => b.id)).toEqual([bounces[1].id]);
  });

  for (const filter of ['source', 'type', 'campaign_id', 'combined', 'selected IDs', 'deselected row']) {
    for (const action of ['delete', 'blocklist']) {
      test(`${action} preserves bounces and subscribers outside ${filter}`, async ({ page }) => {
        const data = async (res) => {
          expect(res.ok(), await res.text()).toBeTruthy();
          return (await res.json()).data;
        };
        const campaign = await data(await page.request.post('/api/campaigns', { data: {
          name: 'Bounce target', subject: 'Bounce target', type: 'regular', content_type: 'html', body: 'Test', lists: [1],
        } }));
        const targets = [];
        const outsiders = [];
        for (let i = 0; i < 25; i += 1) {
          const sub = await data(await page.request.post('/api/subscribers', { data: {
            email: `bounce${i}@example.com`, name: `Bounce ${i}`, status: 'enabled', lists: [1],
          } }));
          const outside = i >= 22;
          const source = outside && ['source', 'combined'].includes(filter) ? 'other' : 'target';
          const type = outside && ['type', 'combined'].includes(filter) ? 'hard' : 'soft';
          const campaignUUID = outside && ['campaign_id', 'combined'].includes(filter) ? '' : campaign.uuid;
          await data(await page.request.post('/webhooks/bounce', { data: { email: sub.email, source, type, campaign_uuid: campaignUUID } }));
          // Hard-bounce processing blocklists automatically; restore the fixture so
          // accidental bulk blocklisting of an unrelated subscriber is observable.
          if (type === 'hard') await data(await page.request.put(`/api/subscribers/${sub.id}`, { data: {
            email: sub.email, name: sub.name, status: 'enabled', lists: [1],
          } }));
          (outside ? outsiders : targets).push(sub.id);
        }
        const query = {
          source: 'source=target', type: 'type=soft', campaign_id: `campaign_id=${campaign.id}`,
          combined: `source=target&type=soft&campaign_id=${campaign.id}`,
          'selected IDs': '', 'deselected row': '',
        }[filter];
        await page.goto(`${BOUNCES}?${query}`);
        await page.locator('thead input[type=checkbox]').check();
        await page.getByTestId('btn-bulk-actions').click();
        if (filter !== 'selected IDs') await page.getByTestId('select-all-bounces').click();
        if (filter === 'deselected row') {
          await page.getByTestId('btn-bulk-actions').click();
          await bounceRow(page).first().locator('input[name=id]').uncheck();
          await page.getByTestId('btn-bulk-actions').click();
        }
        const selected = filter.includes('selected')
          ? await page.locator('input[name=id]:checked').evaluateAll((els) => els.map((el) => Number(el.dataset.subscriberId)))
          : targets;
        await page.getByTestId(action === 'delete' ? 'btn-delete-bounces' : 'btn-manage-blocklist').click();
        const [res] = await Promise.all([
          page.waitForResponse((r) => r.request().method() === (action === 'delete' ? 'DELETE' : 'PUT')),
          confirm(page),
        ]);
        expect(res.ok()).toBeTruthy();
        const bounces = (await data(await page.request.get('/api/bounces?per_page=100'))).results;
        const subscribers = (await data(await page.request.get('/api/subscribers?per_page=100'))).results;
        for (const id of [...targets, ...outsiders]) {
          const affected = selected.includes(id);
          expect(bounces.some((b) => b.subscriber_id === id)).toBe(action !== 'delete' || !affected);
          expect(subscribers.find((s) => s.id === id).status).toBe(action === 'blocklist' && affected ? 'blocklisted' : 'enabled');
        }
      });
    }
  }
});
