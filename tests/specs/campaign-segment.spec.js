import { test, expect } from '@playwright/test';
import { clearMail, getMail, openMenu, resetDB } from '../helpers.js';

const CAMPAIGNS = '/admin/campaigns';
const SEGMENT = "subscribers.attribs->>'tenant_id' = 'tenant_x' AND subscribers.attribs->>'lang' = 'tr'";
const button = (scope, name) => scope.getByRole('button', { name, exact: true });
const segmentInput = (page) => page.locator('[data-cy=segment-query]');
const row = (page, name) => page.locator('tbody tr[data-campaign-id]')
  .filter({ has: page.getByRole('link', { name, exact: true }) });

async function data(response) {
  expect(response.ok(), await response.text()).toBeTruthy();
  return (await response.json()).data;
}

async function createCampaign(page, overrides = {}) {
  return data(await page.request.post('/api/campaigns', { data: {
    name: 'Segmented', subject: 'Segmented subject', type: 'regular',
    content_type: 'html', body: '<p>Hello</p>', lists: [1], ...overrides,
  } }));
}

// The default list has the seeded john@example.com (no lang/tenant). Add subscribers with
// attributes to segment on: two matching, one not matching, and a matching but blocklisted one.
async function addSubscribers(page) {
  const subs = [
    { email: 'tr1@example.com', attribs: { lang: 'tr', tenant_id: 'tenant_x' } },
    { email: 'tr2@example.com', attribs: { lang: 'tr', tenant_id: 'tenant_x' } },
    { email: 'en1@example.com', attribs: { lang: 'en', tenant_id: 'tenant_x' } },
    { email: 'blocked@example.com', attribs: { lang: 'tr', tenant_id: 'tenant_x' }, status: 'blocklisted' },
  ];
  for (const s of subs) {
    await data(await page.request.post('/api/subscribers', { data: {
      name: s.email, status: 'enabled', lists: [1], preconfirm_subscriptions: true, ...s,
    } }));
  }
}

async function dryRun(page) {
  const [response] = await Promise.all([
    page.waitForResponse((r) => r.url().endsWith('/api/campaigns/segment/preview')),
    button(page, 'Dry run').click(),
  ]);
  return response;
}

async function openSegmentTab(page, id) {
  await page.goto(`${CAMPAIGNS}/${id}#tab=segment`);
  await page.getByRole('tab', { name: 'Segmentation' }).click();
}

test.beforeEach(async ({ browser }) => {
  await resetDB(browser);
});

test('dry-runs a segment, shows a sample and persists it on save', async ({ page }) => {
  await addSubscribers(page);
  const c = await createCampaign(page);
  await openSegmentTab(page, c.id);

  // Without a segment, all eligible subscribers in the lists match.
  await dryRun(page);
  await expect(page.locator('[data-cy=segment-result]')).toContainText('Matches 4 of 4 subscribers');

  await segmentInput(page).fill(SEGMENT);
  await dryRun(page);
  await expect(page.locator('[data-cy=segment-result]')).toContainText('Matches 2 of 4 subscribers');
  const sample = page.locator('[data-cy=segment-sample] tbody tr');
  await expect(sample).toHaveCount(2);
  await expect(sample.nth(0)).toContainText('tr1@example.com');
  await expect(sample.nth(1)).toContainText('tr2@example.com');

  // Dry runs don't save anything.
  expect((await data(await page.request.get(`/api/campaigns/${c.id}`))).segment_query).toBe('');

  await Promise.all([
    page.waitForResponse((r) => r.url().endsWith(`/api/campaigns/${c.id}`) && r.request().method() === 'PUT'),
    button(page, 'Save changes').click(),
  ]);
  expect((await data(await page.request.get(`/api/campaigns/${c.id}`))).segment_query).toBe(SEGMENT);

  await page.reload();
  await page.getByRole('tab', { name: 'Segmentation' }).click();
  await expect(segmentInput(page)).toHaveValue(SEGMENT);

  // The campaign listing shows the segment.
  await page.goto(CAMPAIGNS);
  await expect(row(page, 'Segmented').locator('[data-cy=campaign-segment]')).toContainText(SEGMENT.slice(0, 20));

  // Reset clears the segment.
  await openSegmentTab(page, c.id);
  await button(page, 'Reset').click();
  await expect(segmentInput(page)).toHaveValue('');
});

test('rejects invalid and unsafe segments', async ({ page }) => {
  const c = await createCampaign(page);
  await openSegmentTab(page, c.id);

  for (const [query, error] of [
    ['1=1) OR (1=1', 'unbalanced parentheses'],
    ['subscribers.id IN (SELECT id FROM users)', "table 'users' is not allowed"],
    ["subscribers.mail = 'x'", 'does not exist'],
  ]) {
    await segmentInput(page).fill(query);
    const response = await dryRun(page);
    expect(response.status()).toBe(400);
    await expect(page.locator('output[data-variant="danger"]').last()).toContainText(error);
    await expect(page.locator('[data-cy=segment-result]')).toHaveCount(0);
  }

  // Saving an invalid segment is rejected and the saved campaign is unchanged.
  await segmentInput(page).fill('1=1) OR (1=1');
  await button(page, 'Save changes').click();
  await expect(page.locator('output[data-variant="danger"]').last()).toContainText('unbalanced parentheses');
  expect((await data(await page.request.get(`/api/campaigns/${c.id}`))).segment_query).toBe('');
});

test('sends a segmented campaign only to matching subscribers and shows the segment once finished', async ({ page }) => {
  await addSubscribers(page);
  // A unique subject to tell this run's mail apart from earlier ones in MailHog.
  const subject = `Segment delivery ${Date.now()}`;
  const c = await createCampaign(page, { subject, segment_query: SEGMENT });
  await clearMail();

  await data(await page.request.put(`/api/campaigns/${c.id}/status`, { data: { status: 'running' } }));
  await expect.poll(async () => (await data(await page.request.get(`/api/campaigns/${c.id}`))).status,
    { timeout: 30000 }).toBe('finished');

  const camp = await data(await page.request.get(`/api/campaigns/${c.id}`));
  expect(camp).toMatchObject({ to_send: 2, sent: 2 });

  const recipients = (await getMail())
    .filter((m) => m.Content.Headers.Subject?.[0] === subject)
    .map((m) => m.Content.Headers.To[0]).sort();
  expect(recipients.map((r) => r.match(/<?([^<>\s]+@[^<>\s]+)>?$/)[1])).toEqual(['tr1@example.com', 'tr2@example.com']);

  // The finished campaign's segment is visible (read-only) and can still be dry-run.
  await openSegmentTab(page, c.id);
  await expect(segmentInput(page)).toHaveValue(SEGMENT);
  await expect(segmentInput(page)).toHaveJSProperty('readOnly', true);
  await dryRun(page);
  await expect(page.locator('[data-cy=segment-result]')).toContainText('Matches 2 of 4 subscribers');
});

test('clones the segment and previews a saved segment over the API', async ({ page }) => {
  await addSubscribers(page);
  const c = await createCampaign(page, { segment_query: SEGMENT });

  await page.goto(CAMPAIGNS);
  page.once('dialog', (dialog) => dialog.accept('Segmented clone'));
  await openMenu(row(page, 'Segmented'));
  await row(page, 'Segmented').getByRole('menuitem', { name: 'Clone', exact: true }).click();
  await expect(page).toHaveURL(/\/campaigns\/\d+$/);
  const id = Number(new URL(page.url()).pathname.split('/').pop());
  expect((await data(await page.request.get(`/api/campaigns/${id}`))).segment_query).toBe(SEGMENT);

  const preview = await data(await page.request.get(`/api/campaigns/${c.id}/segment/preview?limit=1`));
  expect(preview).toMatchObject({ matched: 2, total: 4 });
  expect(preview.results.map((s) => s.email)).toEqual(['tr1@example.com']);
});
