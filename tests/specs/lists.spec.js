import { test, expect } from '@playwright/test';
import { openMenu, confirm } from '../helpers.js';

const LISTS = '/admin/lists';

// Find a table row by name.
function listRow(page, name) {
  const rows = page.getByTestId('list-row');
  return name ? rows.filter({ hasText: name }) : rows;
}

// IDs currently rendered in the table.
function rowIDs(page) {
  return page.getByTestId('list-row').evaluateAll((els) => els.map((el) => Number(el.dataset.id)));
}

// Fill the list form.
async function fillListForm(scope, { name, type, optin, tag, description } = {}) {
  await scope.locator('input[name="name"]').fill(name);
  await scope.locator('select[name="type"]').selectOption(type);
  await scope.locator('select[name="optin"]').selectOption(optin);
  if (tag) {
    const input = scope.locator('input[name="tags"]');
    await input.fill(tag);
    await input.press('Enter');
  }
  await scope.locator('textarea[name="description"]').fill(description);
}

// Run `action` and wait for the API response.
async function withListAPI(page, method, action) {
  const [res] = await Promise.all([
    page.waitForResponse((r) => /\/api\/lists(\/|\?|$)/.test(r.url()) && r.request().method() === method),
    action(),
  ]);
  expect(res.ok()).toBeTruthy();
}

test.describe.configure({ mode: 'serial' });

test.describe('Lists', () => {
  test('shows the default lists and their subscriber counts', async ({ page }) => {
    await page.goto(LISTS);

    await expect(listRow(page)).toHaveCount(2);
    await expect(listRow(page, 'Default list')).toBeVisible();
    await expect(listRow(page, 'Opt-in list')).toBeVisible();

    // Both seeded lists have exactly one subscriber.
    await expect(listRow(page, 'Default list').locator('.subscriber-total')).toContainText('1');
    await expect(listRow(page, 'Opt-in list').locator('.subscriber-total')).toContainText('1');

    // Type / opt-in badges.
    await expect(listRow(page, 'Default list')).toContainText('Private');
    await expect(listRow(page, 'Default list')).toContainText('Single opt-in');
    await expect(listRow(page, 'Opt-in list')).toContainText('Public');
    await expect(listRow(page, 'Opt-in list')).toContainText('Double opt-in');
  });

  test('creates a campaign for a list', async ({ page }) => {
    await page.goto(LISTS);

    const row = listRow(page, 'Default list');
    await openMenu(row);
    await Promise.all([
      page.waitForURL('**/admin/campaigns/new**'),
      row.getByTestId('btn-campaign').click(),
    ]);

    // The list is pre-selected on the new campaign form.
    await expect(page.getByText('Default list')).toBeVisible();
  });

  test('creates an opt-in campaign for a double opt-in list', async ({ page }) => {
    await page.goto(LISTS);

    const row = listRow(page, 'Opt-in list');
    await openMenu(row);
    await row.getByTestId('btn-send-optin-campaign').click();
    await confirm(page);

    // Redirects to the newly created campaign's content tab.
    await page.waitForURL(/\/admin\/campaigns\/\d+/);
    expect(page.url()).toContain('#tab=content');
  });

  test('shows the subscribers of each list', async ({ page }) => {
    const cases = [
      { list: 'Default list', email: 'john@example.com' },
      { list: 'Opt-in list', email: 'anon@example.com' },
    ];

    for (const c of cases) {
      await page.goto(LISTS);
      await listRow(page, c.list).locator('.subscriber-total').click();

      await expect(page).toHaveURL(/\/admin\/subscribers\/lists\/\d+/);
      await expect(page.locator('tbody tr')).toHaveCount(1);
      await expect(page.locator('td.email')).toContainText(c.email);
    }
  });

  test('edits the default lists', async ({ page }) => {
    await page.goto(LISTS);
    const ids = await rowIDs(page);

    for (const [n, id] of ids.entries()) {
      await page.goto(`${LISTS}/${id}`);
      await fillListForm(page, {
        name: `list-${n}`,
        type: 'public',
        optin: 'double',
        tag: `tag${n}`,
        description: `desc${n}`,
      });
      await withListAPI(page, 'PUT', () => page.getByTestId('btn-save').click());
    }

    // Confirm the edits on the listing.
    await page.goto(LISTS);
    for (let n = 0; n < ids.length; n += 1) {
      const row = listRow(page, `list-${n}`);
      await expect(row).toContainText('Public');
      await expect(row).toContainText('Double opt-in');
      await expect(row).toContainText('#test');
      await expect(row).toContainText(`#tag${n}`);
    }
  });

  test('deletes all lists', async ({ page }) => {
    await page.goto(LISTS);

    let count = await listRow(page).count();
    while (count > 0) {
      const row = listRow(page).first();
      const id = await row.getAttribute('data-id');
      await openMenu(row);
      await row.getByTestId('btn-delete').click();
      await withListAPI(page, 'DELETE', () => confirm(page));

      count -= 1;
      await expect(listRow(page)).toHaveCount(count);
    }

    await expect(page.locator('.empty-state')).toBeVisible();
  });

  test('adds new lists of every type/opt-in combination', async ({ page }) => {
    const combos = [
      { type: 'private', optin: 'single' },
      { type: 'private', optin: 'double' },
      { type: 'public', optin: 'single' },
      { type: 'public', optin: 'double' },
    ];

    for (const [n, c] of combos.entries()) {
      const name = `list-${c.type}-${c.optin}-${n}`;
      await page.goto(LISTS);
      await page.getByTestId('btn-new').click();

      const dialog = page.locator('dialog[open]');
      await fillListForm(dialog, {
        name, type: c.type, optin: c.optin, tag: `tag${n}`, description: `desc-${c.type}-${n}`,
      });
      await withListAPI(page, 'POST', () => dialog.getByTestId('btn-save').click());

      const row = listRow(page, name);
      await expect(row).toBeVisible();
      await expect(row).toContainText(c.type === 'private' ? 'Private' : 'Public');
      await expect(row).toContainText(c.optin === 'single' ? 'Single opt-in' : 'Double opt-in');
    }
  });

  test('searches lists by name', async ({ page }) => {
    await page.goto(LISTS);

    await page.getByTestId('query').fill('list-public-single-2');
    await page.getByTestId('query').press('Enter');

    await expect(listRow(page)).toHaveCount(1);
    await expect(listRow(page)).toContainText('list-public-single-2');
  });

  test('filters lists by type', async ({ page }) => {
    await page.goto(LISTS);

    // Clicking a row's type badge filters the listing by that type.
    await listRow(page, 'list-private-single-0').locator('a[href*="type="]').click();
    await expect(page).toHaveURL(/[?&]type=private/);

    // Only the two private lists remain.
    await expect(listRow(page)).toHaveCount(2);
    await expect(listRow(page, 'list-private-single-0')).toBeVisible();
    await expect(listRow(page, 'list-private-double-1')).toBeVisible();

    // The active filter shows as a removable badge.
    await expect(page.locator('.page-title')).toContainText('Private');
    await page.getByTestId('btn-remove-filter').click();
    await expect(listRow(page)).toHaveCount(4);
  });

  test('filters lists by opt-in', async ({ page }) => {
    await page.goto(LISTS);

    await listRow(page, 'list-private-single-0').locator('a[href*="optin="]').click();
    await expect(page).toHaveURL(/[?&]optin=single/);

    // Only the two single opt-in lists remain.
    await expect(listRow(page)).toHaveCount(2);
    await expect(listRow(page, 'list-private-single-0')).toBeVisible();
    await expect(listRow(page, 'list-public-single-2')).toBeVisible();

    await expect(page.locator('.page-title')).toContainText('Single opt-in');
    await page.getByTestId('btn-remove-filter').click();
    await expect(listRow(page)).toHaveCount(4);
  });

  test('filters lists by tag', async ({ page }) => {
    await page.goto(LISTS);

    await listRow(page, 'list-private-single-0').locator('a[href*="tag="]').click();
    await expect(page).toHaveURL(/[?&]tag=tag0/);

    // Each list carries a unique tag, so only one row matches.
    await expect(listRow(page)).toHaveCount(1);
    await expect(listRow(page, 'list-private-single-0')).toBeVisible();

    await expect(page.locator('.page-title')).toContainText('#tag0');
    await page.getByTestId('btn-remove-filter').click();
    await expect(listRow(page)).toHaveCount(4);
  });

  test('filters lists by multiple criteria', async ({ page }) => {
    await page.goto(`${LISTS}?type=public&optin=double`);

    await expect(listRow(page)).toHaveCount(1);
    await expect(listRow(page, 'list-public-double-3')).toBeVisible();

    // Both active filters appear as badges in the header.
    await expect(page.locator('.page-title')).toContainText('Public');
    await expect(page.locator('.page-title')).toContainText('Double opt-in');
  });

  test('sorts lists by column', async ({ page }) => {
    await page.goto(LISTS);

    // Four lists exist with IDs [3,4,5,6] (see the "adds new lists" test).
    const clickSort = async (field, expected) => {
      await page.locator(`a[data-sort-field="${field}"]`).click();
      await expect.poll(() => rowIDs(page)).toEqual(expected);
    };

    await clickSort('name', [4, 3, 6, 5]);
    await clickSort('name', [5, 6, 3, 4]);

    await clickSort('type', [3, 4, 5, 6]);
    await clickSort('type', [6, 5, 4, 3]);

    await clickSort('created_at', [3, 4, 5, 6]);
    await clickSort('created_at', [6, 5, 4, 3]);
  });

  test('lists public lists on the public subscription form', async ({ page }) => {
    await page.goto('/subscription/form');

    const items = page.locator('ul.lists li');
    await expect(items).toHaveCount(2);
    await expect(items.filter({ hasText: 'list-public-single-2' })).toContainText('desc-public-2');
    await expect(items.filter({ hasText: 'list-public-double-3' })).toContainText('desc-public-3');
  });

  test('bulk deletes lists', async ({ page }) => {
    // Create enough lists to span more than one page.
    for (let i = 0; i < 30; i += 1) {
      const res = await page.request.post('/api/lists', {
        data: { name: `bulk-${i}`, type: 'public', optin: 'single' },
      });
      expect(res.ok()).toBeTruthy();
    }

    // Delete every list across all pages via the "select all" query option.
    await page.goto(LISTS);
    await page.locator('thead input[type="checkbox"]').check();
    await page.getByTestId('btn-bulk-actions').click();

    const selectAll = page.getByTestId('select-all-lists');
    if (await selectAll.isVisible()) {
      await selectAll.click();
    }
    await withListAPI(page, 'DELETE', async () => {
      await page.getByTestId('btn-delete-lists').click();
      await confirm(page);
    });
    await expect(page.locator('.empty-state')).toBeVisible();

    // Now delete a single page's worth using the selected-IDs.
    for (let i = 0; i < 5; i += 1) {
      await page.request.post('/api/lists', { data: { name: `bulk-again-${i}`, type: 'public', optin: 'single' } });
    }

    await page.goto(LISTS);
    await page.locator('thead input[type="checkbox"]').check();
    await page.getByTestId('btn-bulk-actions').click();
    await withListAPI(page, 'DELETE', async () => {
      await page.getByTestId('btn-delete-lists').click();
      await confirm(page);
    });
    await expect(page.locator('.empty-state')).toBeVisible();
  });
});
