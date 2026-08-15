import { test, expect } from '@playwright/test';
import { openMenu, confirm, resetDB, resetDBBlank } from '../helpers.js';

const USERS = '/admin/users';
const USER_ROLES = '/admin/users/roles';
const LIST_ROLES = '/admin/users/roles/lists';

function roleRow(page, name) {
  const rows = page.getByTestId('role-row');
  return name ? rows.filter({ hasText: name }) : rows;
}

function userRow(page, name) {
  const rows = page.getByTestId('user-row');
  return name ? rows.filter({ hasText: name }) : rows;
}

// Open a role's edit view by name (via the listing), reloading it from the server.
async function gotoRoleEdit(page, base, name) {
  await page.goto(base);
  const id = await roleRow(page, name).getAttribute('data-id');
  await page.goto(`${base}/${id}`);
}

// Open a user's edit view by name, reloading it from the server.
async function gotoUserEdit(page, name) {
  await page.goto(USERS);
  const id = await userRow(page, name).getAttribute('data-id');
  await page.goto(`${USERS}/${id}`);
}

async function login(page, username, password) {
  await page.goto('/admin/login?next=/admin/lists');
  await page.locator('input[name=username]').fill(username);
  await page.locator('input[name=password]').fill(password);
  await Promise.all([
    page.waitForURL('**/admin/lists'),
    page.locator('form[action="/admin/login"] button[type=submit]').click(),
  ]);
}

test.describe.configure({ mode: 'serial' });

test.describe('First-time user setup', () => {
  // Start unauthenticated: the blank DB has no users yet.
  test.use({ storageState: { cookies: [], origins: [] } });

  test.beforeAll(async () => {
    await resetDBBlank();
  });

  test('creates the initial superadmin user', async ({ page }) => {
    await page.goto('/admin/login');

    await page.locator('input[name=email]').fill('super@domain.com');
    await page.locator('input[name=username]').fill('super');
    await page.locator('input[name=password]').fill('super123');
    await page.locator('input[name=password2]').fill('super123');
    await Promise.all([
      page.waitForURL((url) => !url.pathname.includes('/login')),
      page.locator('button[type=submit]').click(),
    ]);

    // The new user is the "Super Admin" (role id 1).
    await page.goto(USERS);
    const row = userRow(page, 'super');
    await openMenu(row);
    await Promise.all([page.waitForURL('**/admin/users/*'), row.getByTestId('btn-edit').click()]);
    await expect(page.locator('select[name=user_role]')).toHaveValue('1');
  });
});

test.describe('Users, roles & login', () => {
  test.beforeAll(async ({ browser }) => {
    await resetDB(browser);
  });

  test('adds user roles', async ({ page }) => {
    // "first": accept the default permissions.
    await page.goto(USER_ROLES);
    await page.getByTestId('btn-new').click();
    await page.locator('input[name=name]').fill('first');
    await Promise.all([
      page.waitForURL((url) => url.pathname.endsWith('/users/roles')),
      page.getByTestId('btn-save').click(),
    ]);

    // "second": every permission checked.
    await page.getByTestId('btn-new').click();
    await page.locator('input[name=name]').fill('second');
    const boxes = page.locator('form input[type=checkbox]');
    for (let i = 0, n = await boxes.count(); i < n; i += 1) {
      await boxes.nth(i).check();
    }
    await Promise.all([
      page.waitForURL((url) => url.pathname.endsWith('/users/roles')),
      page.getByTestId('btn-save').click(),
    ]);

    await expect(roleRow(page)).toHaveCount(3);

    // Reopen "second": every permission checkbox came back checked.
    await gotoRoleEdit(page, USER_ROLES, 'second');
    await expect(page.locator('form input[type=checkbox]:checked')).not.toHaveCount(0);
    await expect(page.locator('form input[type=checkbox]:not(:checked)')).toHaveCount(0);
  });

  test('edits a user role', async ({ page }) => {
    await page.goto(USER_ROLES);
    const row = roleRow(page, 'first');
    await openMenu(row);
    await row.getByTestId('btn-edit').click();

    await page.locator('input[value="users:get"]').check();
    await Promise.all([
      page.waitForResponse((r) => /\/api\/roles\/users\/\d+/.test(r.url()) && r.request().method() === 'PUT'),
      page.getByTestId('btn-save').click(),
    ]);

    // Reopen the role: "users:get" (not a default) is checked, proving the edit stuck.
    await gotoRoleEdit(page, USER_ROLES, 'first');
    await expect(page.locator('input[value="users:get"]')).toBeChecked();
  });

  test('clones and deletes a user role', async ({ page }) => {
    await page.goto(USER_ROLES);

    const clone = roleRow(page).last();
    page.once('dialog', (d) => d.accept('second copy'));
    await openMenu(clone);
    await clone.getByTestId('btn-clone').click();
    await expect(roleRow(page)).toHaveCount(4);
    await expect(roleRow(page, 'second copy')).toBeVisible();

    const last = roleRow(page).last();
    await openMenu(last);
    await last.getByTestId('btn-delete').click();
    await confirm(page);
    await expect(roleRow(page)).toHaveCount(3);
    await expect(roleRow(page, 'second copy')).toHaveCount(0);
  });

  test('adds list roles', async ({ page }) => {
    // "first": a single list.
    await page.goto(LIST_ROLES);
    await page.getByTestId('btn-new').click();
    await page.locator('input[name=name]').fill('first');
    await page.getByTestId('btn-add-list').click();
    await Promise.all([
      page.waitForURL((url) => url.pathname.endsWith('/roles/lists')),
      page.getByTestId('btn-save').click(),
    ]);

    // "second": both lists.
    await page.getByTestId('btn-new').click();
    await page.locator('input[name=name]').fill('second');
    await page.getByTestId('btn-add-list').click();
    await page.getByTestId('btn-add-list').click();
    await Promise.all([
      page.waitForURL((url) => url.pathname.endsWith('/roles/lists')),
      page.getByTestId('btn-save').click(),
    ]);

    await expect(roleRow(page)).toHaveCount(2);

    // Reopen each: "first" lists one row, "second" lists both.
    await gotoRoleEdit(page, LIST_ROLES, 'first');
    await expect(page.locator('form tbody tr')).toHaveCount(1);
    await gotoRoleEdit(page, LIST_ROLES, 'second');
    await expect(page.locator('form tbody tr')).toHaveCount(2);
  });

  test('edits a list role', async ({ page }) => {
    await page.goto(LIST_ROLES);
    const row = roleRow(page, 'second');
    await openMenu(row);
    await row.getByTestId('btn-edit').click();

    // Drop the "manage" permission on the second list.
    await page.locator('tbody tr').nth(1).locator('input[value="list:manage"]').uncheck();
    await Promise.all([
      page.waitForResponse((r) => /\/api\/roles\/lists\/\d+/.test(r.url()) && r.request().method() === 'PUT'),
      page.getByTestId('btn-save').click(),
    ]);

    // Reopen the role: both lists keep "view", exactly one lost "manage".
    await gotoRoleEdit(page, LIST_ROLES, 'second');
    await expect(page.locator('input[value="list:get"]:checked')).toHaveCount(2);
    await expect(page.locator('input[value="list:manage"]:checked')).toHaveCount(1);
  });

  test('clones and deletes a list role', async ({ page }) => {
    await page.goto(LIST_ROLES);

    const clone = roleRow(page).last();
    page.once('dialog', (d) => d.accept('second copy'));
    await openMenu(clone);
    await clone.getByTestId('btn-clone').click();
    await expect(roleRow(page)).toHaveCount(3);
    await expect(roleRow(page, 'second copy')).toBeVisible();

    const last = roleRow(page).last();
    await openMenu(last);
    await last.getByTestId('btn-delete').click();
    await confirm(page);
    await expect(roleRow(page)).toHaveCount(2);
    await expect(roleRow(page, 'second copy')).toHaveCount(0);
  });

  test('adds users', async ({ page }) => {
    const cases = [
      { name: 'first', userRole: 'first', listRole: 'first' },
      { name: 'second', userRole: 'second', listRole: 'second' },
      { name: 'third', userRole: 'first', listRole: 'first' },
    ];

    for (const c of cases) {
      await page.goto(USERS);
      await page.getByTestId('btn-new').click();
      await page.locator('input[name=username]').fill(c.name);
      await page.locator('input[name=name]').fill(c.name);
      await page.locator('input[name=email]').fill(`${c.name}@domain.com`);
      await page.getByTestId('password-login').check();
      await page.locator('input[name=password]').fill(`${c.name}000000`);
      await page.locator('input[name=password2]').fill(`${c.name}000000`);
      await page.locator('select[name=user_role]').selectOption({ label: c.userRole });
      await page.locator('select[name=list_role]').selectOption({ label: c.listRole });
      await Promise.all([
        page.waitForURL((url) => url.pathname.endsWith('/admin/users')),
        page.getByTestId('btn-save').click(),
      ]);
    }

    // admin + 3 created.
    await expect(userRow(page)).toHaveCount(4);

    // The listing renders each user's role badges. Read the rows and confirm the
    // roles match (usernames and role names overlap, so match on the username link).
    const rows = await userRow(page).evaluateAll((els) => els.map((el) => ({
      username: el.querySelector('a[href*="/admin/users/"]').textContent.trim(),
      userRole: el.querySelector('a[href*="user_role_id"]')?.textContent.trim(),
      listRole: el.querySelector('a[href*="list_role_id"]')?.textContent.trim(),
    })));
    for (const c of cases) {
      const row = rows.find((r) => r.username === c.name);
      expect(row.userRole).toBe(c.userRole);
      expect(row.listRole).toBe(c.listRole);
    }
  });

  test('edits a user', async ({ page }) => {
    await gotoUserEdit(page, 'third');
    await page.getByTestId('password-login').uncheck();
    await page.locator('select[name=user_role]').selectOption({ label: 'second' });
    await page.locator('select[name=list_role]').selectOption({ label: 'second' });
    await Promise.all([
      page.waitForResponse((r) => /\/api\/users\/\d+/.test(r.url()) && r.request().method() === 'PUT'),
      page.getByTestId('btn-save').click(),
    ]);

    // Reopen the edit view: the new roles and disabled password login are reflected.
    await gotoUserEdit(page, 'third');
    await expect(page.getByTestId('password-login')).not.toBeChecked();
    await expect(page.locator('select[name=user_role] option:checked')).toContainText('second');
    await expect(page.locator('select[name=list_role] option:checked')).toContainText('second');
  });

  test('deletes a user', async ({ page }) => {
    await page.goto(USERS);
    const row = userRow(page, 'third');
    await openMenu(row);
    await row.getByTestId('btn-delete').click();
    await confirm(page);

    await expect(userRow(page)).toHaveCount(3);
    await expect(userRow(page, 'third')).toHaveCount(0);
  });

  test('logs in as a single-list user', async ({ browser }) => {
    const context = await browser.newContext({ storageState: { cookies: [], origins: [] } });
    const page = await context.newPage();
    await login(page, 'first', 'first000000');

    // Only the one list granted by the list role, with no create button.
    await expect(page.getByTestId('list-row')).toHaveCount(1);
    await expect(page.getByTestId('list-row')).toContainText('Default list');
    await expect(page.getByTestId('btn-new')).toHaveCount(0);
    await expect(page.getByTestId('list-row').getByTestId('btn-edit')).toHaveCount(1);

    await context.close();
  });

  test('logs in as an all-permissions user', async ({ browser }) => {
    const context = await browser.newContext({ storageState: { cookies: [], origins: [] } });
    const page = await context.newPage();
    await login(page, 'second', 'second000000');

    // The superadmin-equivalent role sees every list with full actions.
    await expect(page.getByTestId('list-row')).toHaveCount(2);
    await expect(page.getByTestId('list-row').getByTestId('btn-edit')).toHaveCount(2);
    await expect(page.getByTestId('list-row').getByTestId('btn-delete')).toHaveCount(2);

    await context.close();
  });
});
