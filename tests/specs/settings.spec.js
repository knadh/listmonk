import { test, expect } from '@playwright/test';
import { resetDB, getMail } from '../helpers.js';

const SETTINGS = '/admin/settings';
const smtpServers = (page) => page.locator('.mail-servers details');
const tab = (page, name) => page.getByRole('tab', { name, exact: true });

async function settings(page) {
  const res = await page.request.get('/api/settings');
  expect(res.ok()).toBeTruthy();
  return (await res.json()).data;
}

// Saving reloads the backend. Wait for the UI's health check before navigating.
async function save(page, status = 200) {
  const response = page.waitForResponse((r) => new URL(r.url()).pathname === '/api/settings'
    && r.request().method() === 'PUT');
  await page.getByRole('button', { name: 'Save', exact: true }).click();
  expect((await response).status()).toBe(status);
  if (status === 200) {
    await expect(page.getByRole('button', { name: 'Save', exact: true })).toBeDisabled();
    await expect(page.getByText('"Settings" updated', { exact: true })).toBeVisible();
  }
  await expect(page.getByRole('button', { name: 'Save', exact: true })).toBeEnabled();
}

async function remove(page, item, accept = true) {
  await item.getByRole('button', { name: 'Delete', exact: true }).click();
  await page.getByRole('dialog').getByRole('button', { name: accept ? 'Ok' : 'Cancel', exact: true }).click();
}

test.describe('Settings', () => {
  // Reset before creating the page/context so it uses the refreshed login.
  test.beforeEach(async ({ browser }) => { await resetDB(browser); });
  test.beforeEach(async ({ page }) => { await page.goto(SETTINGS); });

  test('saves edits across tabs, enables SMTP and deletes a server', async ({ page }) => {
    const root = 'http://127.0.0.1:9000';
    const favicon = `${root}/public/static/logo.png`;
    await page.locator('[name="app.root_url"]').fill(root);
    await page.locator('[name="app.favicon_url"]').fill(favicon);
    await tab(page, 'Performance').click();
    await page.locator('[name="app.concurrency"]').fill('9');
    await tab(page, 'SMTP').click();
    await page.getByRole('button', { name: 'Add new' }).click();
    const servers = smtpServers(page);
    const added = servers.nth(1);
    await added.getByPlaceholder('smtp.yourmailserver.net').fill('localhost');
    await added.getByRole('spinbutton', { name: /^Port/ }).fill('1025');
    await added.getByRole('group', { name: 'TLS', exact: true }).getByRole('combobox').selectOption('none');
    await added.getByRole('switch', { name: 'Enabled', exact: true }).uncheck();
    await expect(added.getByPlaceholder('smtp.yourmailserver.net')).toBeDisabled();
    await expect(added.getByRole('button', { name: 'Test connection' })).toBeDisabled();
    await added.getByRole('switch', { name: 'Enabled', exact: true }).check();
    await remove(page, servers.first(), false);
    await expect(servers).toHaveCount(2);
    await remove(page, servers.first());
    await expect(servers).toHaveCount(1);
    await expect(servers.getByRole('button', { name: 'Delete' })).toBeHidden();
    await save(page);
    expect(await settings(page)).toMatchObject({
      'app.root_url': root, 'app.favicon_url': favicon, 'app.concurrency': 9,
      smtp: [{ enabled: true, host: 'localhost', port: 1025, tls_type: 'none' }],
    });
    await page.reload();
    await expect(servers.getByRole('switch', { name: 'Enabled', exact: true })).toBeChecked();
    await tab(page, 'General').click();
    await expect(page.locator('[name="app.root_url"]')).toHaveValue(root);
    await expect(page.locator('[name="app.favicon_url"]')).toHaveValue(favicon);
    await tab(page, 'Performance').click();
    await expect(page.locator('[name="app.concurrency"]')).toHaveValue('9');
  });

  test('searches across tabs using all terms', async ({ page }) => {
    const search = page.getByRole('searchbox', { name: 'Search' });
    await search.fill('  RATE   workers  ');
    await expect(page.locator('[name="app.concurrency"]')).toBeVisible();
    await expect(tab(page, 'General')).toBeHidden();
    await expect(tab(page, 'Performance')).toBeVisible();
    await search.fill('email');
    await expect(page.locator('[name="app.from_email"]')).toBeVisible();
    await expect(tab(page, 'SMTP')).toBeVisible();
    await tab(page, 'SMTP').click();
    await expect(search).toBeEmpty();
    await expect(smtpServers(page)).toBeVisible();
    await search.fill('no-such-setting');
    await expect(tab(page, 'General')).toBeVisible();
    await expect(smtpServers(page)).toBeVisible();
    await search.fill('concurrency');
    await search.fill('');
    await expect(smtpServers(page)).toBeVisible();
    await expect(page.locator('[name="app.concurrency"]')).toBeHidden();
  });

  test('SMTP presets reset credentials and preserve unrelated fields and accordion state', async ({ page }) => {
    await tab(page, 'SMTP').click();
    const server = smtpServers(page);
    await server.locator('summary').click();
    await server.getByPlaceholder('email-primary').fill('email-primary');
    await server.getByRole('group', { name: 'Auth protocol', exact: true }).getByRole('combobox').selectOption('login');
    await server.getByRole('textbox', { name: /^Username/ }).fill('old-user');
    await server.getByLabel(/Password/).fill('old-secret');
    for (const [provider, host, port, auth, tls] of [
      ['Gmail', 'smtp.gmail.com', '465', 'login', 'TLS'],
      ['Postmark', 'smtp.postmarkapp.com', '587', 'cram', 'STARTTLS'],
    ]) {
      await server.getByRole('link', { name: provider, exact: true }).click();
      await expect(server.getByPlaceholder('smtp.yourmailserver.net')).toHaveValue(host);
      await expect(server.getByRole('spinbutton', { name: /^Port/ })).toHaveValue(port);
      await expect(server.getByRole('group', { name: 'Auth protocol', exact: true }).getByRole('combobox')).toHaveValue(auth);
      await expect(server.getByRole('group', { name: 'TLS', exact: true }).getByRole('combobox')).toHaveValue(tls);
      await expect(server.getByRole('textbox', { name: /^Username/ })).toBeFocused();
      await expect(server.getByRole('textbox', { name: /^Username/ })).toBeEmpty();
      await expect(server.getByLabel(/Password/)).toBeEmpty();
      await expect(server.getByPlaceholder('email-primary')).toHaveValue('email-primary');
    }
    await server.getByRole('group', { name: 'Auth protocol', exact: true }).getByRole('combobox').selectOption('none');
    await expect(server.getByRole('textbox', { name: /^Username/ })).toBeDisabled();
    await expect(server.getByLabel(/Password/)).toBeDisabled();
    await page.reload();
    await expect(server.getByPlaceholder('smtp.yourmailserver.net')).toBeVisible();
    await server.locator('summary').click();
    await page.reload();
    await expect(server.getByPlaceholder('smtp.yourmailserver.net')).toBeHidden();
  });

  test('tests custom SMTP headers and masks saved secrets', async ({ page }) => {
    await tab(page, 'SMTP').click();
    const server = smtpServers(page);
    await server.locator('summary').click();
    await server.getByRole('group', { name: 'Auth protocol', exact: true }).getByRole('combobox').selectOption('login');
    await server.getByRole('textbox', { name: /^Username/ }).fill('test-user');
    await server.getByLabel(/Password/).fill('settings-test-secret');
    await server.getByRole('link', { name: 'Set custom headers' }).click();
    const headers = server.getByPlaceholder('[{"X-Custom": "value"}]');
    await headers.fill('[{"X-Settings-Test":"saved"}]');
    await save(page);
    await page.reload();
    await expect(server.getByLabel(/Password/)).toHaveValue(/^•+$/);
    expect(JSON.parse(await headers.inputValue())).toEqual([{ 'X-Settings-Test': 'saved' }]);
    await server.getByLabel(/Password/).fill('••partial');
    await page.getByRole('button', { name: 'Save', exact: true }).click();
    await expect(page.getByText(/Clear and re-enter the full password/)).toBeVisible();
    await server.getByLabel(/Password/).fill('replacement-secret');
    await save(page);
    await page.reload();
    await expect(server.getByLabel(/Password/)).toHaveValue(/^•+$/);
    // An unrelated save sends no replacement for the masked password.
    const request = page.waitForRequest((r) => r.url().endsWith('/api/settings') && r.method() === 'PUT');
    await save(page);
    expect((await request).postDataJSON().smtp[0].password).toBe('');
    await page.reload();
    await expect(server.getByLabel(/Password/)).toHaveValue(/^•+$/);
  });

  test('normalizes domain lists and retains tracking preferences while disabled', async ({ page }) => {
    await tab(page, 'Privacy').click();
    const individual = page.getByRole('switch', { name: /^Individual subscriber tracking/ });
    await individual.check();
    await page.getByRole('switch', { name: /^Disable tracking/ }).check();
    await expect(individual).toBeDisabled();
    await expect(individual).toBeChecked();
    const domainText = page.locator('textarea:visible');
    await domainText.fill('  EXAMPLE.COM  \n\n spam.test\n  ');
    await expect(page.getByRole('tab', { name: /Domain blocklist \(\s*2\s*\)/ })).toBeVisible();
    await page.getByRole('tab', { name: /^Domain allowlist/ }).click();
    await domainText.fill(' TRUSTED.TEST \n\n');
    await save(page);
    expect(await settings(page)).toMatchObject({
      'privacy.domain_blocklist': ['example.com', 'spam.test'],
      'privacy.domain_allowlist': ['trusted.test'],
      'privacy.disable_tracking': true, 'privacy.individual_tracking': true,
    });
    await page.reload();
    await expect(domainText).toHaveValue('example.com\nspam.test');
    await page.getByRole('switch', { name: /^Disable tracking/ }).uncheck();
    await expect(individual).toBeEnabled();
    await expect(individual).toBeChecked();
  });

  test('enables sliding window controls and persists their values', async ({ page }) => {
    await tab(page, 'Performance').click();
    const group = page.getByRole('group', { name: 'Enable sliding window limit', exact: true });
    await expect(group.getByRole('textbox', { name: /^Duration/ })).toBeDisabled();
    await group.getByRole('switch').check();
    await group.getByRole('spinbutton', { name: /^Max. messages/ }).fill('42');
    await group.getByRole('textbox', { name: /^Duration/ }).fill('2h');
    await save(page);
    await page.reload();
    await expect(group.getByRole('switch')).toBeChecked();
    await expect(group.getByRole('spinbutton', { name: /^Max. messages/ })).toHaveValue('42');
    await expect(group.getByRole('textbox', { name: /^Duration/ })).toHaveValue('2h');
    await group.getByRole('switch').uncheck();
    await expect(group.getByRole('textbox', { name: /^Duration/ })).toBeDisabled();
    await expect(group.getByRole('textbox', { name: /^Duration/ })).toHaveValue('2h');
  });

  test('adds, disables and deletes messengers with confirmation', async ({ page }) => {
    await tab(page, 'Messengers').click();
    await page.getByRole('button', { name: 'Add new' }).click();
    const messenger = page.locator('.messengers details');
    await expect(messenger.getByPlaceholder('mymessenger')).toBeFocused();
    await messenger.getByPlaceholder('mymessenger').fill('sms-test');
    await messenger.getByPlaceholder('https://postback.messenger.net/path').fill('http://localhost:9999/send');
    await messenger.getByRole('switch', { name: 'Enabled' }).uncheck();
    await save(page);
    await page.reload();
    await expect(messenger).toHaveCount(1);
    await expect(messenger.getByRole('switch', { name: 'Enabled' })).not.toBeChecked();
    expect((await settings(page)).messengers).toMatchObject([
      { name: 'sms-test', root_url: 'http://localhost:9999/send', enabled: false },
    ]);
    await remove(page, messenger, false);
    await expect(messenger).toHaveCount(1);
    await remove(page, messenger);
    await expect(messenger).toHaveCount(0);
    await save(page);
    await page.reload();
    await expect(messenger).toHaveCount(0);
    expect((await settings(page)).messengers).toEqual([]);
  });

  test('persists trusted URLs as separate entries', async ({ page }) => {
    await tab(page, 'Security').click();
    const urls = page.getByPlaceholder('https://example.com', { exact: true });
    await urls.fill('https://one.example.com\nhttps://two.example.com/thanks');
    await save(page);
    expect((await settings(page))['security.trusted_urls']).toEqual([
      'https://one.example.com', 'https://two.example.com/thanks',
    ]);
    await page.reload();
    await expect(urls).toHaveValue('https://one.example.com\nhttps://two.example.com/thanks');
    await urls.fill('');
    await save(page);
    await page.reload();
    await expect(urls).toBeEmpty();
  });

  test('persists bounce actions and gates mailbox and webhook controls', async ({ page }) => {
    await tab(page, 'Bounces').click();
    await page.getByRole('switch', { name: 'Enable bounce processing', exact: true }).check();
    const hard = page.getByRole('group', { name: 'Hard', exact: true });
    await hard.getByRole('spinbutton').fill('3');
    await hard.getByRole('combobox').selectOption('unsubscribe');
    const mailbox = page.getByRole('switch', { name: 'Enable bounce mailbox', exact: true });
    await mailbox.check();
    await expect(page.getByPlaceholder('bounce.yourmailserver.net')).toBeVisible();
    await mailbox.uncheck();
    await expect(page.getByPlaceholder('bounce.yourmailserver.net')).toHaveCount(0);
    const webhooks = page.getByRole('switch', { name: /^Enable bounce webhooks/ });
    await webhooks.check();
    await page.getByRole('group', { name: 'SES', exact: true }).getByRole('switch').check();
    await save(page);
    expect(await settings(page)).toMatchObject({
      'bounce.enabled': true, 'bounce.webhooks_enabled': true, 'bounce.ses_enabled': true,
      'bounce.actions': { hard: { count: 3, action: 'unsubscribe' } },
      'bounce.mailboxes': [{ enabled: false }],
    });
    await page.reload();
    await expect(hard.getByRole('spinbutton')).toHaveValue('3');
    await expect(hard.getByRole('combobox')).toHaveValue('unsubscribe');
    await webhooks.uncheck();
    await expect(page.getByRole('group', { name: 'SES', exact: true })).toHaveCount(0);
  });

  test('saves independent admin and public appearance editors', async ({ page }) => {
    await tab(page, 'Appearance').click();
    const css = page.getByRole('group', { name: 'Custom CSS', exact: true }).getByRole('textbox');
    const js = page.getByRole('group', { name: 'Custom JavaScript', exact: true }).getByRole('textbox');
    await css.fill('/* admin settings test */');
    await js.fill('// admin settings test');
    await tab(page, 'Public').click();
    await css.fill('/* public settings test */');
    await js.fill('// public settings test');
    await save(page);
    expect(await settings(page)).toMatchObject({
      'appearance.admin.custom_css': '/* admin settings test */',
      'appearance.admin.custom_js': '// admin settings test',
      'appearance.public.custom_css': '/* public settings test */',
      'appearance.public.custom_js': '// public settings test',
    });
    await page.reload();
    await expect(css).toHaveText('/* admin settings test */');
    await expect(js).toHaveText('// admin settings test');
    await tab(page, 'Public').click();
    await expect(css).toHaveText('/* public settings test */');
    await expect(js).toHaveText('// public settings test');
  });

});
