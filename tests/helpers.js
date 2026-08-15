import { execSync, execFileSync, spawn } from 'node:child_process';
import { setTimeout as sleep } from 'node:timers/promises';
import { expect } from '@playwright/test';

const BASE_URL = process.env.LISTMONK_URL || 'http://localhost:9000';
const MAILHOG_URL = process.env.MAILHOG_URL || 'http://localhost:8025';
const ROOT = '..';
const AUTH_FILE = new URL('.auth/admin.json', import.meta.url).pathname;
const ENV = { ...process.env, LISTMONK_ADMIN_USER: 'admin', LISTMONK_ADMIN_PASSWORD: 'listmonk' };

// A single SMTP server pointing at MailHog so tests capture (and can assert on) mail.
const MAILHOG_SMTP = JSON.stringify([{
  enabled: true, host: 'localhost', port: 1025, auth_protocol: 'none',
  username: '', password: '', hello_hostname: '', max_conns: 10,
  idle_timeout: '15s', wait_timeout: '5s', max_msg_retries: 2, msg_retry_delay: '10ms',
  tls_type: 'none', tls_skip_verify: false, email_headers: [], from_addresses: [],
}]);

// Open a table row's "More" actions dropdown.
export async function openMenu(row) {
  await row.getByRole('button', { name: 'More' }).click();
}

// Click the "Ok" button on the confirmation dialog.
export async function confirm(page) {
  await page.locator('dialog[open] [data-confirm-ok]').click();
}

// Pick a list by name in an <ot-taginput> list selector. Filling the input
// matches a datalist option, which the component turns into a tag/badge.
export async function selectList(scope, name) {
  const input = scope.locator('ot-taginput input');
  await input.click();
  await input.fill(name);
  await expect(scope.locator('ot-taginput .badge').filter({ hasText: name })).toBeVisible();
}

// Log in and persist the session to the shared storage state file.
export async function login(context) {
  const page = await context.newPage();
  await page.goto(`${BASE_URL}/admin/login?next=/admin/lists`);
  await page.getByLabel('Username').fill('admin');
  await page.getByLabel('Password').fill('listmonk');
  await page.getByRole('button', { name: 'Login' }).click();
  await page.waitForURL('**/admin/lists');
  await context.storageState({ path: AUTH_FILE });
  await page.close();
}

// Delete all mail captured by MailHog.
export async function clearMail() {
  await fetch(`${MAILHOG_URL}/api/v2/messages`, { method: 'DELETE' });
}

// All mail captured by MailHog, newest first.
export async function getMail() {
  return (await (await fetch(`${MAILHOG_URL}/api/v2/messages`)).json()).items;
}

// Run a SQL statement against the listmonk DB. Credentials come from the
// LISTMONK_db__* env, falling back to config.toml defaults.
function psql(sql) {
  execFileSync('psql', [
    '-h', process.env.LISTMONK_db__host || 'localhost',
    '-p', process.env.LISTMONK_db__port || '5432',
    '-U', process.env.LISTMONK_db__user || 'listmonk',
    '-d', process.env.LISTMONK_db__database || 'listmonk',
    '-c', sql,
  ], { env: { ...process.env, PGPASSWORD: process.env.LISTMONK_db__password || 'listmonk' }, stdio: 'ignore' });
}

// Point listmonk's SMTP config at MailHog on the freshly installed DB.
function configureSMTP() {
  psql(`UPDATE settings SET value = '${MAILHOG_SMTP}' WHERE key = 'smtp';`);
}

// Enable bounce processing + the webhook endpoint, and delete on complaints.
// Bounce settings are read into config at boot, so this must run before start.
function configureBounces() {
  const actions = JSON.stringify({
    soft: { count: 2, action: 'none' },
    hard: { count: 1, action: 'blocklist' },
    complaint: { count: 1, action: 'delete' },
  });
  psql(`UPDATE settings SET value = 'true' WHERE key = 'bounce.enabled';
        UPDATE settings SET value = 'true' WHERE key = 'bounce.webhooks_enabled';
        UPDATE settings SET value = '${actions}' WHERE key = 'bounce.actions';`);
}

// Wipe and reinstall the DB by restarting listmonk, then refresh the login
// session (a reinstall clears the DB-backed session). The schema can't be
// reinstalled under a running server, so the process is killed and relaunched.
export async function resetDB(browser, { bounces = false } = {}) {
  try { execSync('pkill -9 listmonk', { stdio: 'ignore' }); } catch { /* none running */ }
  execSync('./listmonk --install --yes', { cwd: ROOT, env: ENV, stdio: 'ignore' });
  configureSMTP();
  if (bounces) {
    configureBounces();
  }
  spawn('./listmonk', ['--static-dir', 'static'], {
    cwd: ROOT, env: ENV, detached: true, stdio: 'ignore',
  }).unref();

  for (let i = 0; i < 100; i += 1) {
    try {
      if ((await fetch(`${BASE_URL}/health`)).ok) break;
    } catch { /* not up yet */ }
    await sleep(200);
  }

  await clearMail();

  const context = await browser.newContext();
  await login(context);
  await context.close();
}

// Reinstall the DB with no admin user so the app boots into first-time user
// setup mode. Unlike resetDB, it doesn't create a session (there's no user yet).
export async function resetDBBlank() {
  try { execSync('pkill -9 listmonk', { stdio: 'ignore' }); } catch { /* none running */ }

  const env = { ...process.env };
  delete env.LISTMONK_ADMIN_USER;
  delete env.LISTMONK_ADMIN_PASSWORD;

  execSync('./listmonk --install --yes', { cwd: ROOT, env, stdio: 'ignore' });
  configureSMTP();
  spawn('./listmonk', ['--static-dir', 'static'], {
    cwd: ROOT, env, detached: true, stdio: 'ignore',
  }).unref();

  for (let i = 0; i < 100; i += 1) {
    try {
      if ((await fetch(`${BASE_URL}/health`)).ok) break;
    } catch { /* not up yet */ }
    await sleep(200);
  }
}
