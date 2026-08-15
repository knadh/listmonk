import { test, expect } from '@playwright/test';
import { resetDB, clearMail, getMail } from '../helpers.js';

const FORMS = '/admin/lists/forms';
const SUB_FORM = '/subscription/form';

const REDIRECT_URLS = [
  'http://localhost:9000/thank-you',
  'https://example.com/welcome',
];

function publicLists(page) {
  return page.locator('ul[data-testid=lists] li');
}

function formHTML(page) {
  return page.locator('[data-testid=form] [role=textbox]');
}

function redirectURLs(page) {
  return page.locator('[data-testid=redirect-urls] input[type=radio]');
}

// Saving trusted URLs SIGHUPs the process, which re-execs. Wait for the reload
// to expose the new redirect URLs in the server config.
async function setTrustedURLs(page, urls) {
  const res = await page.request.put('/api/settings/security.trusted_urls', { data: urls });
  expect(res.ok()).toBeTruthy();

  await expect.poll(async () => {
    try {
      const { data } = await (await page.request.get('/api/config')).json();
      return data.public_subscription?.redirect_urls ?? [];
    } catch {
      // The server is mid-reload and not accepting connections yet.
      return [];
    }
  }, { timeout: 20000, intervals: [500] }).toEqual(urls);
}

async function publicListUUID(page) {
  const { data } = await (await page.request.get('/api/lists')).json();
  return data.results.find((l) => l.type === 'public').uuid;
}

async function subscribe(page, { email, name }) {
  await page.goto(SUB_FORM);
  await page.locator('input[name=email]').fill(email);
  await page.locator('input[name=name]').fill(name);
  await Promise.all([
    page.waitForLoadState(),
    page.locator('button[type=submit]').click(),
  ]);
  // A missing SMTP config surfaces a retry message instead of a success one.
  await expect(page.locator('body')).toContainText(/has been sent|successfully|retry/i);
}

test.describe.configure({ mode: 'serial' });

test.describe('Forms', () => {
  test.beforeAll(async ({ browser }) => {
    await resetDB(browser);
  });

  test('lists only public lists and hides the form until one is picked', async ({ page }) => {
    await page.goto(FORMS);

    await expect(publicLists(page)).toHaveCount(1);
    await expect(publicLists(page)).toContainText('Opt-in list');
    await expect(formHTML(page)).toBeHidden();
  });

  test('hides the redirect URL options when none are whitelisted', async ({ page }) => {
    await page.goto(FORMS);

    // No trusted URLs on a fresh install, so the picker isn't rendered at all.
    await expect(page.locator('[data-testid=redirect-urls]')).toHaveCount(0);
  });

  test('generates form HTML for the selected list', async ({ page }) => {
    await page.goto(FORMS);

    const checkbox = publicLists(page).locator('input');
    const uuid = await checkbox.inputValue();

    await checkbox.check();
    await expect(formHTML(page)).toBeVisible();
    await expect(formHTML(page)).toContainText(uuid);

    await checkbox.uncheck();
    await expect(formHTML(page)).toBeHidden();
  });

  test('subscribes new subscribers via the public form', async ({ page }) => {
    // Add a second public list so the form offers a choice.
    const res = await page.request.post('/api/lists', {
      data: { name: 'test-list', type: 'public', optin: 'single' },
    });
    expect(res.ok()).toBeTruthy();

    await clearMail();
    await page.goto(SUB_FORM);
    await expect(page.locator('ul.lists input[type=checkbox]')).toHaveCount(2);

    for (let i = 0; i < 2; i += 1) {
      await subscribe(page, { email: `test${i}@test.com`, name: `test${i}` });
    }

    const { data } = await (await page.request.get('/api/subscribers')).json();

    // Two new + two seeded subscribers.
    expect(data.total).toBe(4);
    for (let i = 0; i < 2; i += 1) {
      expect(data.results.find((s) => s.email === `test${i}@test.com`).lists.length).toBe(2);
    }

    // The double opt-in list sent each of them a confirmation email.
    await expect.poll(async () => {
      const tos = (await getMail()).map((m) => m.Content.Headers.To[0]);
      return ['test0', 'test1'].every((n) => tos.some((t) => t.includes(`${n}@test.com`)));
    }).toBe(true);
  });

  test('unsubscribes from a list and then blocklists', async ({ page }) => {
    // Point the seeded campaign at the opt-in list the subscribers are on.
    await page.request.put('/api/campaigns/1', { data: { lists: [2] } });

    const subs = (await (await page.request.get('/api/subscribers')).json()).data.results;
    const sub = subs.find((s) => s.email === 'test0@test.com');
    const camp = (await (await page.request.get('/api/campaigns')).json()).data.results[0];
    const url = `/subscription/${camp.uuid}/${sub.uuid}`;

    // Unsubscribe from the campaign's list only.
    await page.goto(url);
    await page.locator('#btn-unsub').click();

    let lists = (await (await page.request.get('/api/subscribers')).json())
      .data.results.find((s) => s.email === 'test0@test.com').lists;
    expect(lists.find((l) => l.id === 2).subscription_status).toBe('unsubscribed');
    expect(lists.find((l) => l.id === 3).subscription_status).toBe('unconfirmed');

    // Unsubscribe from everything and blocklist.
    await page.goto(url);
    await page.locator('#privacy-blocklist').check();
    await page.locator('#btn-unsub').click();

    const sub0 = (await (await page.request.get('/api/subscribers')).json())
      .data.results.find((s) => s.email === 'test0@test.com');
    expect(sub0.status).toBe('blocklisted');
    expect(sub0.lists.find((l) => l.id === 2).subscription_status).toBe('unsubscribed');
    expect(sub0.lists.find((l) => l.id === 3).subscription_status).toBe('unsubscribed');
  });

  test('manages subscription preferences', async ({ page }) => {
    const subs = (await (await page.request.get('/api/subscribers')).json()).data.results;
    const sub = subs.find((s) => s.email === 'test1@test.com');
    const camp = (await (await page.request.get('/api/campaigns')).json()).data.results[0];

    await page.goto(`/subscription/${camp.uuid}/${sub.uuid}?manage=true`);

    await page.locator('input[name=name]').fill('new-name');
    await page.locator('ul.lists input').first().uncheck();
    await page.locator('#btn-unsub').click();

    const sub1 = (await (await page.request.get('/api/subscribers')).json())
      .data.results.find((s) => s.email === 'test1@test.com');
    expect(sub1.name).toBe('new-name');
    expect(sub1.lists.find((l) => l.id === 2).subscription_status).toBe('unsubscribed');
    expect(sub1.lists.find((l) => l.id === 3).subscription_status).toBe('unconfirmed');
  });
});

test.describe('Forms redirect URL whitelist', () => {
  test.beforeAll(async ({ browser }) => {
    await resetDB(browser);
    // Reuse the logged-in session persisted by resetDB so the settings API is authorized.
    const context = await browser.newContext({
      storageState: new URL('../.auth/admin.json', import.meta.url).pathname,
    });
    const page = await context.newPage();
    await setTrustedURLs(page, REDIRECT_URLS);
    await context.close();
  });

  test('lists whitelisted URLs as redirect options', async ({ page }) => {
    await page.goto(FORMS);

    // A "None" radio plus one per whitelisted URL, with "None" selected by default.
    await expect(redirectURLs(page)).toHaveCount(REDIRECT_URLS.length + 1);
    await expect(page.locator('[data-testid=redirect-urls] input[value=""]')).toBeChecked();
    for (const url of REDIRECT_URLS) {
      await expect(page.locator(`[data-testid=redirect-urls] input[value="${url}"]`)).toHaveCount(1);
    }
  });

  test('injects the selected redirect URL as a hidden "next" field', async ({ page }) => {
    await page.goto(FORMS);

    // A list has to be picked before any form HTML is generated.
    await publicLists(page).locator('input').first().check();
    await expect(formHTML(page)).toBeVisible();
    await expect(formHTML(page)).not.toContainText('name="next"');

    await page.locator(`[data-testid=redirect-urls] input[value="${REDIRECT_URLS[1]}"]`).check();
    await expect(formHTML(page)).toContainText(`<input type="hidden" name="next" value="${REDIRECT_URLS[1]}" />`);

    // Switching back to "None" removes it again.
    await page.locator('[data-testid=redirect-urls] input[value=""]').check();
    await expect(formHTML(page)).not.toContainText('name="next"');
  });

  test('redirects to a whitelisted "next" URL after submission', async ({ page }) => {
    const uuid = await publicListUUID(page);

    const res = await page.request.post(SUB_FORM, {
      form: { email: 'redirect@test.com', name: 'redirect', l: uuid, next: REDIRECT_URLS[0] },
      maxRedirects: 0,
    });

    expect(res.status()).toBe(303);
    expect(res.headers().location).toBe(REDIRECT_URLS[0]);
  });

  test('ignores a non-whitelisted "next" URL to prevent open redirects', async ({ page }) => {
    const uuid = await publicListUUID(page);

    const res = await page.request.post(SUB_FORM, {
      form: { email: 'noredirect@test.com', name: 'noredirect', l: uuid, next: 'https://evil.example.com/phish' },
      maxRedirects: 0,
    });

    // No redirect: the normal confirmation page is rendered instead.
    expect(res.status()).toBe(200);
    expect(res.headers().location).toBeUndefined();
    expect(await res.text()).toMatch(/subscri|confirm/i);
  });
});
