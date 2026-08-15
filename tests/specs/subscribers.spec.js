import { test, expect } from '@playwright/test';
import { setTimeout as sleep } from 'node:timers/promises';
import {
  openMenu, confirm, resetDB, selectList, getMail, clearMail,
} from '../helpers.js';

const SUBSCRIBERS = '/admin/subscribers';
const SUB_FORM = '/subscription/form';

// Seeded lists: 1 = Default (private, single opt-in), 2 = Opt-in (public, double opt-in).
const LIST_DEFAULT = 1;
const LIST_OPTIN = 2;

function subRow(page, text) {
  const rows = page.getByTestId('sub-row');
  return text ? rows.filter({ hasText: text }) : rows;
}

function subRowIDs(page) {
  return page.getByTestId('sub-row').evaluateAll((els) => els.map((el) => Number(el.dataset.id)));
}

// The (filtered) total the listing header shows, e.g. "Subscribers (47)".
async function readTotal(page) {
  const txt = await page.locator('.page-title').innerText();
  return Number(txt.match(/\((\d+)\)/)[1]);
}

// Create `n` subscribers via the API. Returns nothing; assert on the DB after.
async function seedSubscribers(page, n, { prefix = 'seed', lists = [], status = 'enabled' } = {}) {
  for (let i = 0; i < n; i += 1) {
    const res = await page.request.post('/api/subscribers', {
      data: {
        email: `${prefix}${i}@example.com`,
        name: `${prefix} ${i}`,
        status,
        lists,
        attribs: { city: 'Bengaluru', age: 20 + (i % 40) },
      },
    });
    expect(res.ok()).toBeTruthy();
  }
}

// Select rows and open the bulk-actions menu. With selectAll, promote the visible
// selection to the whole query (needs total > per-page for the button to render).
async function openBulkMenu(page, { selectAll = false } = {}) {
  await page.locator('thead input[type=checkbox]').check();
  await page.getByTestId('btn-bulk-actions').click();
  if (selectAll) {
    await page.getByTestId('select-all-subscribers').click();
  }
}

// Open the bulk manage-lists dialog and apply an action to the given lists.
async function bulkManageLists(page, {
  action, lists, preconfirm = false, selectAll = false,
}) {
  await openBulkMenu(page, { selectAll });
  await page.getByTestId('btn-manage-lists').click();

  const dialog = page.locator('dialog[open]');
  await dialog.getByTestId(`check-list-${action}`).check();
  for (const l of lists) {
    await selectList(dialog, l);
  }
  if (preconfirm) {
    await dialog.getByTestId('preconfirm').check();
  }

  await Promise.all([
    page.waitForResponse((r) => /\/api\/subscribers(\/query)?\/lists/.test(r.url()) && r.request().method() === 'PUT'),
    dialog.locator('button[type=submit]').click(),
  ]);
}

// Trigger a confirm-guarded bulk action and wait for its API call.
async function bulkConfirm(page, {
  testid, selectAll = false, match,
}) {
  await openBulkMenu(page, { selectAll });
  await page.getByTestId(testid).click();
  await Promise.all([page.waitForResponse(match), confirm(page)]);
}

test.describe.configure({ mode: 'serial' });

test.describe('Subscribers: listing, search & sort', () => {
  test.beforeAll(async ({ browser }) => {
    await resetDB(browser);
  });

  test('shows the two seeded subscribers', async ({ page }) => {
    await page.goto(SUBSCRIBERS);

    await expect(subRow(page)).toHaveCount(2);
    await expect(subRow(page, 'john@example.com')).toBeVisible();
    await expect(subRow(page, 'anon@example.com')).toBeVisible();
  });

  test('searches subscribers', async ({ page }) => {
    await page.goto(SUBSCRIBERS);
    const search = page.getByTestId('search');

    for (const c of [
      { q: 'john', email: 'john@example.com' },
      { q: 'anon', email: 'anon@example.com' },
    ]) {
      await search.fill(c.q);
      await search.press('Enter');
      await expect(subRow(page)).toHaveCount(1);
      await expect(subRow(page)).toContainText(c.email);
    }

    await search.fill('');
    await search.press('Enter');
    await expect(subRow(page)).toHaveCount(2);
  });

  test('runs advanced SQL queries', async ({ page }) => {
    const cases = [
      { sql: "subscribers.attribs->>'city'='Bengaluru'", count: 2 },
      { sql: "subscribers.attribs->>'city'='Bengaluru' AND id=1", count: 1 },
      { sql: "(subscribers.attribs->>'good')::BOOLEAN = true AND name like 'Anon%'", count: 1 },
    ];

    for (const c of cases) {
      await page.goto(SUBSCRIBERS);
      await page.getByTestId('btn-advanced-search').click();
      await page.getByTestId('query').fill(c.sql);
      await Promise.all([page.waitForLoadState(), page.getByTestId('btn-query').click()]);
      await expect(subRow(page)).toHaveCount(c.count);
    }
  });

  test('resets the advanced query editor', async ({ page }) => {
    await page.goto(SUBSCRIBERS);
    await page.getByTestId('btn-advanced-search').click();

    const query = page.getByTestId('query');
    await query.fill("name LIKE '%x%'");
    await expect(page.getByTestId('btn-query-reset')).toBeEnabled();

    await page.getByTestId('btn-query-reset').click();
    await expect(query).toHaveValue('');
    await expect(page.getByTestId('btn-query-reset')).toBeDisabled();
  });

  test('sorts subscribers by every column', async ({ page }) => {
    await page.goto(SUBSCRIBERS);

    const clickSort = async (field, expected) => {
      await page.locator(`a[data-sort-field="${field}"]`).click();
      await expect.poll(() => subRowIDs(page)).toEqual(expected);
    };

    // Seeded: john (id 1, "John Doe"), anon (id 2, "Anon Doe").
    await clickSort('email', [2, 1]);
    await clickSort('email', [1, 2]);
    await clickSort('name', [2, 1]);
    await clickSort('name', [1, 2]);
    await clickSort('created_at', [1, 2]);
    await clickSort('created_at', [2, 1]);
    await clickSort('updated_at', [1, 2]);
    await clickSort('updated_at', [2, 1]);
  });

  test('exports subscribers via the API', async ({ page }) => {
    const cases = [
      { query: '', length: 3 },
      { query: "name ILIKE '%anon%'", length: 2 },
      { query: "name like 'nope'", length: 1 },
    ];

    for (const c of cases) {
      const res = await page.request.get('/api/subscribers/export', { params: { query: c.query } });
      expect(res.ok()).toBeTruthy();
      const lines = (await res.text()).trim().split('\n');
      expect(lines).toHaveLength(c.length);
    }
  });
});

test.describe('Subscribers: create, edit & delete', () => {
  test.beforeAll(async ({ browser }) => {
    await resetDB(browser);
  });

  test('creates a subscriber on a list', async ({ page }) => {
    await page.goto(SUBSCRIBERS);
    await page.getByTestId('btn-new').click();

    const dialog = page.locator('dialog[open]');
    await dialog.locator('input[name=email]').fill('created@example.com');
    await dialog.locator('input[name=name]').fill('Created');
    await selectList(dialog, 'Default list');
    await Promise.all([
      page.waitForResponse((r) => /\/api\/subscribers$/.test(r.url()) && r.request().method() === 'POST'),
      dialog.getByTestId('btn-save').click(),
    ]);

    // The listing shows the new subscriber on the Default list.
    await page.goto(SUBSCRIBERS);
    const row = subRow(page, 'created@example.com');
    await expect(row).toBeVisible();
    await expect(row).toContainText('Default list');
    await expect(row.locator('td.align-center')).toContainText('1');
  });

  test('creates a pre-confirmed subscriber on a double opt-in list', async ({ page }) => {
    await page.goto(SUBSCRIBERS);
    await page.getByTestId('btn-new').click();

    const dialog = page.locator('dialog[open]');
    await dialog.locator('input[name=email]').fill('confirmed@example.com');
    await dialog.locator('input[name=name]').fill('Confirmed');
    await selectList(dialog, 'Opt-in list');
    // Pre-confirm only enables once a double opt-in list is picked.
    await expect(dialog.getByTestId('preconfirm')).toBeEnabled();
    await dialog.getByTestId('preconfirm').check();
    await Promise.all([
      page.waitForResponse((r) => /\/api\/subscribers$/.test(r.url()) && r.request().method() === 'POST'),
      dialog.getByTestId('btn-save').click(),
    ]);

    // The listing badge shows the Opt-in list subscription as confirmed.
    await page.goto(SUBSCRIBERS);
    const row = subRow(page, 'confirmed@example.com');
    await expect(row).toContainText('Opt-in list');
    await expect(row).toContainText('Confirmed');
  });

  test('edits a subscriber and blocklists via the status field', async ({ page }) => {
    await page.goto(`${SUBSCRIBERS}/1`);
    await page.locator('input[name=email]').fill('edited@example.com');
    await page.locator('input[name=name]').fill('Edited Name');
    await page.locator('select[name=status]').selectOption('blocklisted');
    await page.locator('textarea[name=attribs]').fill('{"string": "hello", "ints": [1, 2, 3], "sub": {"bool": true}}');
    // Saving reloads the edit view; it re-renders with the persisted values.
    await Promise.all([page.waitForEvent('load'), page.getByTestId('btn-save').click()]);
    await expect(page.locator('input[name=email]')).toHaveValue('edited@example.com');
    await expect(page.locator('input[name=name]')).toHaveValue('Edited Name');
    await expect(page.locator('select[name=status]')).toHaveValue('blocklisted');
    const attribs = await page.locator('textarea[name=attribs]').inputValue();
    expect(attribs).toContain('"string": "hello"');
    expect(attribs).toContain('"bool": true');

    // The listing marks blocklisted subscribers.
    await page.goto(SUBSCRIBERS);
    await expect(subRow(page, 'edited@example.com')).toHaveClass(/blocklisted/);
  });

  test('rejects invalid attribs JSON', async ({ page }) => {
    await page.goto(`${SUBSCRIBERS}/1`);
    await page.locator('textarea[name=attribs]').fill('{ not valid json');
    await page.getByTestId('btn-save').click();

    await expect(page.locator('.toast')).toBeVisible();
    // No navigation happened: still on the edit page.
    expect(page.url()).toContain(`${SUBSCRIBERS}/1`);
  });

  test('exposes a per-row data download link', async ({ page }) => {
    await page.goto(SUBSCRIBERS);
    const row = subRow(page, 'anon@example.com');
    await openMenu(row);
    await expect(row.getByTestId('btn-download')).toHaveAttribute('href', /\/api\/subscribers\/2\/export$/);
  });

  test('deletes a subscriber from the row menu', async ({ page }) => {
    await page.goto(SUBSCRIBERS);
    const before = await subRow(page).count();

    const row = subRow(page, 'created@example.com');
    await openMenu(row);
    await row.getByTestId('btn-delete').click();
    await confirm(page);

    await expect(subRow(page)).toHaveCount(before - 1);
    await expect(subRow(page, 'created@example.com')).toHaveCount(0);
  });
});

test.describe('Subscribers: filters & pagination', () => {
  test.beforeAll(async ({ browser }) => {
    await resetDB(browser);
    const context = await browser.newContext({
      storageState: new URL('../.auth/admin.json', import.meta.url).pathname,
    });
    const page = await context.newPage();
    // 25 on the Default list → 27 total (2 seeded), spanning two pages.
    await seedSubscribers(page, 25, { prefix: 'pag', lists: [LIST_DEFAULT] });
    await context.close();
  });

  test('paginates the listing', async ({ page }) => {
    await page.goto(SUBSCRIBERS);
    await expect(subRow(page)).toHaveCount(20);

    const nav = page.locator('nav.pagination');
    await expect(nav).toBeVisible();
    await Promise.all([page.waitForLoadState(), nav.locator('a', { hasText: '2' }).click()]);

    await expect(page).toHaveURL(/[?&]page=2/);
    await expect(subRow(page)).toHaveCount(7);
    await expect(nav.locator('a.pg-selected')).toHaveText('2');
  });

  test('filters by list', async ({ page }) => {
    // Only anon (id 2) is on the Opt-in list.
    await page.goto(`${SUBSCRIBERS}/lists/${LIST_OPTIN}`);
    await expect(subRow(page)).toHaveCount(1);
    await expect(subRow(page, 'anon@example.com')).toBeVisible();
    await expect(page.locator('.page-title')).toContainText('Opt-in list');

    // Clearing the filter badge returns to the full listing.
    await Promise.all([page.waitForLoadState(), page.getByTestId('btn-remove-filter').click()]);
    await expect(page).toHaveURL(/\/admin\/subscribers$/);
  });

  test('filters by subscriber status', async ({ page }) => {
    await page.request.put('/api/subscribers/blocklist', { data: { ids: [1] } });

    await page.goto(`${SUBSCRIBERS}?status=blocklisted`);
    await expect(subRow(page)).toHaveCount(1);
    await expect(subRow(page, 'john@example.com')).toBeVisible();
    await expect(page.locator('.page-title')).toContainText('Blocklisted');
  });

  test('filters a list by subscription status', async ({ page }) => {
    // anon (id 2) is the sole Opt-in list member; confirm its subscription.
    await page.request.put('/api/subscribers/lists', {
      data: {
        ids: [2], action: 'add', target_list_ids: [LIST_OPTIN], status: 'confirmed',
      },
    });

    // The confirmed filter shows only anon, with a Confirmed badge.
    await page.goto(`${SUBSCRIBERS}/lists/${LIST_OPTIN}?subscription_status=confirmed`);
    await expect(page.locator('.page-title')).toContainText('Confirmed');
    await expect(subRow(page)).toHaveCount(1);
    await expect(subRow(page, 'anon@example.com')).toContainText('Confirmed');
  });
});

test.describe('Subscribers: bulk actions', () => {
  test.beforeAll(async ({ browser }) => {
    await resetDB(browser);
    const context = await browser.newContext({
      storageState: new URL('../.auth/admin.json', import.meta.url).pathname,
    });
    const page = await context.newPage();
    // 45 fresh subscribers → 47 total, spanning three pages.
    await seedSubscribers(page, 45, { prefix: 'bulk' });
    await context.close();
  });

  test('adds a list to the selected (visible) page', async ({ page }) => {
    await page.goto(SUBSCRIBERS);
    const count = await subRow(page).count();

    await bulkManageLists(page, { action: 'add', lists: ['Default list'] });

    // Every subscriber on the reloaded page now carries the Default list badge.
    await expect(subRow(page)).toHaveCount(count);
    await expect(subRow(page).filter({ hasText: 'Default list' })).toHaveCount(count);
  });

  test('adds a pre-confirmed list to all via select-all query', async ({ page }) => {
    await page.goto(SUBSCRIBERS);
    await bulkManageLists(page, {
      action: 'add', lists: ['Opt-in list'], preconfirm: true, selectAll: true,
    });

    // Everyone (47) ends up on the Opt-in list.
    await page.goto(`${SUBSCRIBERS}/lists/${LIST_OPTIN}`);
    expect(await readTotal(page)).toBe(47);
    // New subscriptions are confirmed; only the pre-existing anon stays unconfirmed.
    await page.goto(`${SUBSCRIBERS}/lists/${LIST_OPTIN}?subscription_status=confirmed`);
    expect(await readTotal(page)).toBe(46);
  });

  test('unsubscribes all from a list via query', async ({ page }) => {
    await page.goto(SUBSCRIBERS);
    await bulkManageLists(page, { action: 'unsubscribe', lists: ['Opt-in list'], selectAll: true });

    // All 47 Opt-in subscriptions are now unsubscribed.
    await page.goto(`${SUBSCRIBERS}/lists/${LIST_OPTIN}?subscription_status=unsubscribed`);
    expect(await readTotal(page)).toBe(47);
  });

  test('removes a list from all via query', async ({ page }) => {
    await page.goto(SUBSCRIBERS);
    await bulkManageLists(page, { action: 'remove', lists: ['Opt-in list'], selectAll: true });

    // The Opt-in list has no members left.
    await page.goto(`${SUBSCRIBERS}/lists/${LIST_OPTIN}`);
    await expect(page.locator('.empty-state')).toBeVisible();
  });

  test('blocklists the selected (visible) page', async ({ page }) => {
    await page.goto(SUBSCRIBERS);
    const count = await subRow(page).count();

    await bulkConfirm(page, {
      testid: 'btn-manage-blocklist',
      match: (r) => /\/api\/subscribers\/blocklist$/.test(r.url()) && r.request().method() === 'PUT',
    });

    // Exactly the selected page's subscribers are blocklisted; nobody else.
    await page.goto(`${SUBSCRIBERS}?status=blocklisted`);
    expect(await readTotal(page)).toBe(count);
  });

  test('blocklists all via query', async ({ page }) => {
    await page.goto(SUBSCRIBERS);
    await bulkConfirm(page, {
      testid: 'btn-manage-blocklist',
      selectAll: true,
      match: (r) => /\/api\/subscribers\/query\/blocklist$/.test(r.url()) && r.request().method() === 'PUT',
    });

    await page.goto(`${SUBSCRIBERS}?status=blocklisted`);
    expect(await readTotal(page)).toBe(47);
  });

  test('exports the selection via the UI', async ({ page }) => {
    await page.goto(SUBSCRIBERS);
    await openBulkMenu(page, { selectAll: true });

    const [download] = await Promise.all([
      page.waitForEvent('download'),
      (async () => {
        await page.getByTestId('btn-export-subscribers').click();
        await confirm(page);
      })(),
    ]);
    expect(download.suggestedFilename()).toBe('subscribers.csv');
  });

  test('deletes the selected (visible) page', async ({ page }) => {
    await page.goto(SUBSCRIBERS);
    const before = await readTotal(page);
    const count = await subRow(page).count();

    await bulkConfirm(page, {
      testid: 'btn-delete-subscribers',
      match: (r) => /\/api\/subscribers\?/.test(r.url()) && r.request().method() === 'DELETE',
    });

    // The total dropped by exactly the page's worth of subscribers.
    expect(await readTotal(page)).toBe(before - count);
  });

  test('deletes everything via query', async ({ page }) => {
    await page.goto(SUBSCRIBERS);
    await bulkConfirm(page, {
      testid: 'btn-delete-subscribers',
      selectAll: true,
      match: (r) => /\/api\/subscribers\/query\/delete$/.test(r.url()) && r.request().method() === 'POST',
    });

    await expect(page.locator('.empty-state')).toBeVisible();
    expect(await readTotal(page)).toBe(0);
  });
});

test.describe('Subscriber detail: lists & subscriptions', () => {
  const listRow = (page, name) => page.locator('.subscriptions tbody tr').filter({ hasText: name });

  // Per-subscription actions confirm, then reload the page on success.
  async function rowAction(page, name, testid) {
    const row = listRow(page, name);
    await openMenu(row);
    await Promise.all([
      page.waitForEvent('load'),
      (async () => { await row.getByTestId(testid).click(); await confirm(page); })(),
    ]);
  }

  test.beforeAll(async ({ browser }) => {
    await resetDB(browser);
  });

  test('adds a list via the manage dialog', async ({ page }) => {
    await page.goto(`${SUBSCRIBERS}/1/lists`);
    await page.getByTestId('btn-manage').click();

    const dialog = page.locator('dialog[open]');
    await dialog.getByTestId('check-list-add').check();
    await selectList(dialog, 'Opt-in list');
    await Promise.all([
      page.waitForEvent('load'),
      dialog.locator('button[type=submit]').click(),
    ]);

    await expect(listRow(page, 'Opt-in list')).toBeVisible();
    await expect(listRow(page, 'Opt-in list').locator('.status-unconfirmed')).toBeVisible();
  });

  test('sends an opt-in confirmation email', async ({ page }) => {
    await clearMail();
    await page.goto(`${SUBSCRIBERS}/1/lists`);

    const btn = page.getByTestId('btn-send-optin');
    await expect(btn).toBeEnabled();
    // Sending shows a toast (no reload), so wait on the API call.
    await Promise.all([
      page.waitForResponse((r) => /\/api\/subscribers\/1\/optin$/.test(r.url()) && r.request().method() === 'POST'),
      (async () => { await btn.click(); await confirm(page); })(),
    ]);

    await expect.poll(async () => (await getMail()).length, { timeout: 10000 }).toBeGreaterThan(0);
    expect((await getMail())[0].Content.Headers.To[0]).toContain('john@example.com');
  });

  test('confirms a double opt-in subscription', async ({ page }) => {
    await page.goto(`${SUBSCRIBERS}/1/lists`);
    await rowAction(page, 'Opt-in list', 'btn-confirm-sub');
    await expect(listRow(page, 'Opt-in list').locator('.status-confirmed')).toBeVisible();
  });

  test('unsubscribes and re-subscribes a subscription', async ({ page }) => {
    await page.goto(`${SUBSCRIBERS}/1/lists`);
    await rowAction(page, 'Opt-in list', 'btn-unsubscribe');
    await expect(listRow(page, 'Opt-in list').locator('.status-unsubscribed')).toBeVisible();

    await page.goto(`${SUBSCRIBERS}/1/lists`);
    await rowAction(page, 'Opt-in list', 'btn-resubscribe');
    await expect(listRow(page, 'Opt-in list').locator('.status-unconfirmed')).toBeVisible();
  });

  test('deletes a subscription', async ({ page }) => {
    await page.goto(`${SUBSCRIBERS}/1/lists`);
    await rowAction(page, 'Default list', 'btn-delete-sub');
    await expect(listRow(page, 'Default list')).toHaveCount(0);
  });
});

test.describe('Subscriber detail: activity tab', () => {
  test.beforeAll(async ({ browser }) => {
    await resetDB(browser);
  });

  test('shows the activity stat cards', async ({ page }) => {
    await page.goto(`${SUBSCRIBERS}/1/activity`);
    const stats = page.locator('.activity .stat-value');
    await expect(stats).toHaveCount(3);
    // No campaigns yet: campaigns, views and clicks are all zero.
    for (let i = 0; i < 3; i += 1) {
      await expect(stats.nth(i)).toHaveText('0');
    }
  });
});

test.describe('Domain blocklist', () => {
  let publicUUID;

  test.beforeAll(async ({ browser }) => {
    await resetDB(browser);

    const context = await browser.newContext({
      storageState: new URL('../.auth/admin.json', import.meta.url).pathname,
    });
    const page = await context.newPage();

    const { data } = await (await page.request.get('/api/lists')).json();
    publicUUID = data.results.find((l) => l.type === 'public').uuid;

    // Blocklisting domains SIGHUPs the server, which re-execs. Wait for it to boot.
    await page.request.put('/api/settings/privacy.domain_blocklist', { data: ['ban.net', 'ban.org', 'ban.com'] });
    await sleep(1500);
    await expect.poll(async () => {
      try {
        return (await page.request.get('/health')).ok();
      } catch {
        return false;
      }
    }, { timeout: 20000, intervals: [300] }).toBe(true);

    await context.close();
  });

  test('rejects blocklisted domains on the public form', async ({ page }) => {
    const submit = async (email) => {
      const res = await page.request.post(SUB_FORM, {
        form: { email, name: 'test', l: publicUUID }, maxRedirects: 0,
      });
      return res.text();
    };

    expect(await submit('test@noban.net')).not.toMatch(/error/i);
    expect(await submit('test@ban.net')).toMatch(/error/i);
  });

  test('rejects blocklisted domains via the admin API', async ({ page }) => {
    // Allowed domain is accepted.
    let res = await page.request.post('/api/subscribers', {
      data: { email: 'ok@noban.net', name: 'test', lists: [1], status: 'enabled' },
    });
    expect(res.status()).toBe(200);

    // New subscriber on a banned domain is rejected.
    res = await page.request.post('/api/subscribers', {
      data: { email: 'nope@ban.com', name: 'test', lists: [1], status: 'enabled' },
    });
    expect(res.status()).toBe(400);

    // Editing an existing subscriber onto a banned domain is rejected.
    res = await page.request.put('/api/subscribers/1', {
      data: { email: 'nope@ban.org', name: 'test', lists: [1], status: 'enabled' },
    });
    expect(res.status()).toBe(400);
  });
});
