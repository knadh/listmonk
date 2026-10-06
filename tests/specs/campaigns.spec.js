import { test, expect } from '@playwright/test';
import { confirm, openMenu, resetDB, selectList, getMail } from '../helpers.js';

const CAMPAIGNS = '/admin/campaigns';
const expression = 'Hello {{ .Subscriber.Name }} from {{ .Subscriber.Attribs.city }}';
const rendered = 'Hello Demo Subscriber from Bengaluru';
const rows = (page) => page.locator('tbody tr[data-campaign-id]');
const row = (page, name) => rows(page).filter({ has: page.getByRole('link', { name, exact: true }) });
const tab = (page, name) => page.getByRole('tab', { name, exact: true });
const button = (scope, name) => scope.getByRole('button', { name, exact: true });

async function data(response) {
  expect(response.ok(), await response.text()).toBeTruthy();
  return (await response.json()).data;
}

async function create(page, overrides = {}) {
  return data(await page.request.post('/api/campaigns', { data: {
    name: 'Campaign', subject: 'Campaign subject', type: 'regular',
    content_type: 'html', body: `<p>${expression}</p>`, lists: [1], ...overrides,
  } }));
}

async function campaign(page, id) {
  return data(await page.request.get(`/api/campaigns/${id}`));
}

async function save(page, id, action = () => button(page, 'Save changes').click()) {
  const [response] = await Promise.all([
    page.waitForResponse((r) => r.url().endsWith(`/api/campaigns/${id}`) && r.request().method() === 'PUT'),
    action(),
  ]);
  return data(response);
}

async function menu(page, name, action) {
  const item = row(page, name).getByRole('menuitem', { name: action, exact: true });
  if (!(await item.isVisible())) await openMenu(row(page, name));
  await item.click();
}

async function switchFormat(page, format, accept = true) {
  const settings = page.locator('button[popovertarget="content-settings"]');
  await settings.click();
  await page.locator('#content-settings select[name="content_type"]').selectOption(format);
  if (accept) await confirm(page);
  else await button(page.locator('dialog[open]'), 'Cancel').click();
  await settings.click();
}

async function editorReady(page) {
  await expect(page.locator('.editor-body')).not.toHaveClass(/is-loading/);
}

async function visualDocument(page, document) {
  await editorReady(page);
  // Seed the builder through its public document API; serialization runs in the real iframe.
  await page.locator('visual-editor').evaluate((el, doc) => el.render(doc), document);
}

async function fillBody(page, format, text) {
  await editorReady(page);
  if (format === 'visual') {
    await visualDocument(page, {
      root: { type: 'EmailLayout', data: { childrenIds: ['text'] } },
      text: { type: 'Text', data: { props: { text: 'Write your message' }, style: {} } },
    });
    const frame = page.frameLocator('#visual-editor');
    await frame.getByText('Write your message', { exact: true }).click();
    await frame.getByRole('textbox', { name: 'Content', exact: true }).fill(text);
    await expect(frame.locator('td').getByText(text, { exact: true })).toBeVisible();
  } else if (format === 'richtext') {
    await page.locator('richtext-editor [contenteditable="true"]').fill(text);
  } else if (format === 'plain') {
    await page.locator('textarea[name="content"]').fill(text);
  } else {
    await page.locator('.editor-body .cm-content').fill(format === 'html' ? `<strong>${text}</strong>` : `**${text}**`);
  }
}

async function delivered(subject) {
  let mail;
  await expect.poll(async () => {
    mail = (await getMail()).find((m) => m.Content.Headers.Subject?.[0] === subject);
    return !!mail;
  }, { timeout: 20000 }).toBeTruthy();
  return mail;
}

async function preview(page, text) {
  await button(page, 'Preview').click();
  await expect(page.frameLocator('.preview-modal iframe').locator('body')).toContainText(text);
  await button(page.locator('.preview-modal'), 'Close').click();
}

test.beforeEach(async ({ browser }) => {
  await resetDB(browser);
});

test('shows the seeded campaign and previews it from the listing', async ({ page }) => {
  await page.goto(CAMPAIGNS);
  await expect(rows(page)).toHaveCount(1);
  await expect(rows(page)).toContainText('Draft');
  await expect(rows(page).locator('.lists')).toContainText('Default list');
  await openMenu(rows(page));
  await rows(page).getByRole('menuitem', { name: 'Preview', exact: true }).click();
  await expect(page.frameLocator('.preview-modal iframe').locator('body')).toContainText('Hi');
  await button(page.locator('.preview-modal'), 'Close').click();
  await expect(page.locator('.preview-modal iframe')).not.toHaveAttribute('src');
});

for (const format of ['richtext', 'html', 'markdown', 'plain', 'visual']) {
  for (const lists of [['Default list'], ['Default list', 'Opt-in list']]) {
    test(`creates, edits and previews ${format} with ${lists.length} list(s)`, async ({ page }) => {
      await page.goto(CAMPAIGNS);
      await page.getByRole('link', { name: 'New', exact: true }).click();
      await expect(tab(page, 'Content')).toHaveAttribute('aria-disabled', 'true');
      await page.getByLabel('Name', { exact: true }).fill(`New ${format}`);
      await page.getByLabel('Subject', { exact: true }).fill(`Subject ${format}`);
      await page.locator('input[name="from_email"]').fill('Sender <sender@example.com>');
      await page.getByRole('combobox', { name: 'Format', exact: true }).selectOption(format);
      for (const name of lists) await selectList(page.locator('fieldset').filter({ has: page.locator('#campaign-lists') }), name);
      for (const tag of ['news', 'monthly']) {
        await page.getByPlaceholder('Tags', { exact: true }).fill(tag);
        await page.getByPlaceholder('Tags', { exact: true }).press('Enter');
      }
      await page.getByRole('switch', { name: 'Custom headers', exact: true }).check();
      await page.locator('textarea[name="headers"]').fill('[{"X-Custom":"value"}]');
      await button(page, 'Continue').click();
      await expect(page).toHaveURL(/\/campaigns\/\d+#tab=content$/);
      const id = Number(new URL(page.url()).pathname.split('/').pop());
      await expect(tab(page, 'Content')).toHaveAttribute('aria-selected', 'true');
      await fillBody(page, format, expression);
      await save(page, id);
      await page.reload();
      await editorReady(page);
      const saved = await campaign(page, id);
      expect(saved).toMatchObject({ name: `New ${format}`, subject: `Subject ${format}`, status: 'draft',
        content_type: format, from_email: 'Sender <sender@example.com>',
        headers: [{ 'X-Custom': 'value' }], send_at: null, altbody: null });
      expect(saved.tags.sort()).toEqual(['monthly', 'news']);
      expect(saved.lists.map((l) => l.id).sort()).toEqual(lists.length === 1 ? [1] : [1, 2]);
      expect(saved.body).toContain(expression);
      if (format === 'visual') expect(JSON.parse(saved.body_source).text.data.props.text).toBe(expression);
      await preview(page, rendered);
      await page.goto(CAMPAIGNS);
      await expect(row(page, `New ${format}`)).toContainText(`Subject ${format}`);
      await expect(row(page, `New ${format}`).locator('.lists li')).toHaveCount(lists.length);
    });
  }
}

test('edits metadata, schedules in local time, unschedules and clears the schedule', async ({ page }) => {
  const c = await create(page, { lists: [2], tags: ['old'] });
  await page.goto(`${CAMPAIGNS}/${c.id}`);
  await page.getByLabel('Name', { exact: true }).fill('Scheduled newsletter');
  await page.getByLabel('Subject', { exact: true }).fill('Updated subject');
  await page.locator('input[name="from_email"]').fill('New <new@example.com>');
  const listField = page.locator('fieldset').filter({ has: page.locator('#campaign-lists') });
  await listField.locator('.badge').hover();
  await button(listField, 'Remove Opt-in list').click();
  await selectList(listField, 'Default list');
  const tags = page.getByPlaceholder('Tags', { exact: true });
  await tags.press('Backspace');
  await tags.fill('new-tag');
  await tags.press('Enter');
  await page.getByRole('switch', { name: 'Custom headers', exact: true }).check();
  await page.locator('textarea[name="headers"]').fill('[{"X-Custom":"Custom-Value"}]');
  await page.getByRole('switch', { name: 'Send later', exact: true }).check();
  const when = `${new Date().getFullYear() + 1}-06-15T09:30`;
  await page.locator('input[name="send_at"]').fill(when);
  const iso = await page.evaluate((value) => new Date(value).toISOString(), when);
  await expect(button(page, 'Start campaign')).toHaveCount(0);
  await tab(page, 'Content').click();
  await switchFormat(page, 'plain');
  await fillBody(page, 'plain', 'Scheduled body');
  await tab(page, 'Campaign').click();
  await button(page, 'Schedule campaign').click();
  await confirm(page);
  await expect(page).toHaveURL(/\/admin\/campaigns$/);
  await expect(row(page, 'Scheduled newsletter')).toContainText('Scheduled');
  const saved = await campaign(page, c.id);
  expect(saved).toMatchObject({ status: 'scheduled', name: 'Scheduled newsletter', subject: 'Updated subject',
    from_email: 'New <new@example.com>', content_type: 'plain', body: 'Scheduled body', altbody: null,
    tags: ['new-tag'], headers: [{ 'X-Custom': 'Custom-Value' }] });
  expect(new Date(saved.send_at).toISOString()).toBe(iso);
  expect(saved.lists.map((l) => l.id)).toEqual([1]);
  await page.goto(`${CAMPAIGNS}/${c.id}`);
  await expect(page.locator('input[name="send_at"]')).toHaveValue(when);
  await button(page, 'Unschedule').click();
  await confirm(page);
  await expect(page.locator('header .status-draft')).toBeVisible();
  await page.getByRole('switch', { name: 'Send later', exact: true }).uncheck();
  await save(page, c.id);
  expect((await campaign(page, c.id)).send_at).toBeNull();
  await expect(button(page, 'Start campaign')).toBeVisible();
});

test('converts formats and cancels conversion without losing content', async ({ page }) => {
  const c = await create(page, { content_type: 'richtext' });
  await page.goto(`${CAMPAIGNS}/${c.id}#tab=content`);
  await editorReady(page);
  await switchFormat(page, 'plain', false);
  await expect(page.locator('richtext-editor')).toBeVisible();
  for (const format of ['html', 'markdown', 'plain']) {
    await switchFormat(page, format);
    await editorReady(page);
    await save(page, c.id);
    expect(await campaign(page, c.id)).toMatchObject({ content_type: format });
    await preview(page, rendered);
    await page.reload();
  }
});

test('previews and test-sends unsaved content without modifying the draft', async ({ page }) => {
  const c = await create(page, { content_type: 'plain', body: 'Saved body', subject: 'Preview test' });
  await page.goto(`${CAMPAIGNS}/${c.id}#tab=content`);
  await fillBody(page, 'plain', 'Unsaved hello {{ .Subscriber.Name }}');
  await button(page, 'Preview').click();
  await expect(page.frameLocator('.preview-modal iframe').locator('body')).toContainText('Unsaved hello Demo Subscriber');
  await button(page, 'Send test message').click();
  await page.locator('#campaign-test input[type="email"]').fill('john@example.com');
  await page.locator('#campaign-test input[type="email"]').press('Enter');
  await button(page.locator('#campaign-test'), 'Send').click();
  expect(JSON.stringify(await delivered('Preview test'))).toContain('Unsaved hello');
  expect(await campaign(page, c.id)).toMatchObject({ status: 'draft', body: 'Saved body' });
});

test('generates, persists and removes alternate text; plain format clears it', async ({ page }) => {
  const c = await create(page, { body: '<p>Alternate message</p>' });
  await page.goto(`${CAMPAIGNS}/${c.id}#tab=content`);
  await editorReady(page);
  const toggle = page.getByRole('tabpanel').filter({ has: page.locator('.editor-body') }).getByRole('switch');
  await toggle.check();
  const alt = page.getByRole('textbox', { name: 'Add alternate plain text message', exact: true });
  await expect(alt).toHaveValue('Alternate message');
  await alt.fill('Custom alternate');
  await save(page, c.id);
  await page.reload();
  await expect(alt).toHaveValue('Custom alternate');
  await toggle.uncheck();
  await confirm(page);
  await save(page, c.id);
  expect((await campaign(page, c.id)).altbody).toBeNull();
  await toggle.check();
  await switchFormat(page, 'plain');
  await save(page, c.id);
  expect((await campaign(page, c.id)).altbody).toBeNull();
});

test('persists attributes and archive settings with keyboard save', async ({ page }) => {
  const c = await create(page);
  await page.goto(`${CAMPAIGNS}/${c.id}`);
  await tab(page, 'Attributes').click();
  await page.getByRole('textbox', { name: 'Attributes', exact: true }).fill('{"edition":42}');
  await tab(page, 'Archive').click();
  await page.getByRole('switch').filter({ visible: true }).check();
  await page.locator('input[name="archive_slug"]').fill('newsletter-edition-42');
  await page.locator('textarea[name="archive_meta"]').fill('{"name":"Archive reader","attribs":{"city":"Mumbai"}}');
  await save(page, c.id, () => page.locator('textarea[name="archive_meta"]').press('ControlOrMeta+s'));
  await page.reload();
  await expect(page.locator('input[name="archive_slug"]')).toHaveValue('newsletter-edition-42');
  await button(page.getByRole('tabpanel').filter({ has: page.locator('input[name="archive_slug"]') }), 'Preview').click();
  await expect(page.frameLocator('.preview-modal iframe').locator('body')).toContainText('Hello Archive reader from Mumbai');
  await button(page.locator('.preview-modal'), 'Close').click();
  expect(await campaign(page, c.id)).toMatchObject({ attribs: { edition: 42 }, archive: true,
    archive_slug: 'newsletter-edition-42', archive_meta: { name: 'Archive reader', attribs: { city: 'Mumbai' } } });
});

test('rejects malformed JSON without overwriting saved metadata', async ({ page }) => {
  const c = await create(page, { headers: [{ 'X-Custom': 'saved' }], attribs: { edition: 1 } });
  await page.goto(`${CAMPAIGNS}/${c.id}`);
  for (const [name, field, valid] of [
    ['Campaign', 'textarea[name="headers"]', '[{"X-Custom":"saved"}]'],
    ['Attributes', 'textarea[aria-labelledby="campaign-attribs-label"]', '{"edition":1}'],
  ]) {
    await tab(page, name).click();
    await page.locator(field).fill('{broken');
    await button(page, 'Save changes').click();
    await expect(page.locator('output[data-variant="danger"]').last()).toContainText(/JSON|SyntaxError/);
    expect(await campaign(page, c.id)).toMatchObject({ headers: [{ 'X-Custom': 'saved' }], attribs: { edition: 1 } });
    await page.locator(field).fill(valid);
    await save(page, c.id);
  }
});

test('clones full content into a draft and cancels cloning and deletion', async ({ page }) => {
  const c = await create(page, { tags: ['clone'], headers: [{ 'X-Custom': 'copy' }],
    attribs: { edition: 2 }, altbody: 'Alternative', lists: [1, 2] });
  await page.goto(CAMPAIGNS);
  page.once('dialog', (dialog) => dialog.dismiss());
  await menu(page, c.name, 'Clone');
  await expect(rows(page)).toHaveCount(2);
  page.once('dialog', (dialog) => dialog.accept('Cloned newsletter'));
  await menu(page, c.name, 'Clone');
  await expect(page).toHaveURL(/\/campaigns\/\d+$/);
  const id = Number(new URL(page.url()).pathname.split('/').pop());
  const clone = await campaign(page, id);
  for (const key of ['subject', 'body', 'content_type', 'altbody', 'headers', 'attribs', 'tags']) expect(clone[key]).toEqual(c[key]);
  expect(clone).toMatchObject({ name: 'Cloned newsletter', status: 'draft' });
  expect(clone.uuid).not.toBe(c.uuid);
  expect(clone.lists.map((l) => l.id)).toEqual([1, 2]);
  await page.goto(CAMPAIGNS);
  await menu(page, clone.name, 'Delete');
  await button(page.locator('dialog[open]'), 'Cancel').click();
  await expect(row(page, clone.name)).toBeVisible();
  await menu(page, clone.name, 'Delete');
  await confirm(page);
  await expect(row(page, clone.name)).toHaveCount(0);
  await expect(row(page, c.name)).toBeVisible();
});

test('searches, filters and sorts campaigns through SSR navigation', async ({ page }) => {
  await data(await page.request.delete('/api/campaigns?all=true'));
  for (const name of ['Bravo', 'Alpha', 'Charlie']) await create(page, { name, tags: [name === 'Charlie' ? 'other' : 'news'] });
  const scheduled = await create(page, { name: 'Scheduled', send_at: new Date(Date.now() + 86400000).toISOString() });
  await data(await page.request.put(`/api/campaigns/${scheduled.id}/status`, { data: { status: 'scheduled' } }));
  await page.goto(CAMPAIGNS);
  await page.getByRole('textbox', { name: 'Search', exact: true }).fill('Alpha');
  await page.getByRole('textbox', { name: 'Search', exact: true }).press('Enter');
  await expect(rows(page)).toHaveCount(1);
  await expect(row(page, 'Alpha')).toBeVisible();
  await page.goto(CAMPAIGNS);
  await row(page, 'Alpha').getByRole('link', { name: '#news', exact: true }).click();
  await expect(rows(page)).toHaveCount(2);
  await expect(page).toHaveURL(/[?&]tag=news/);
  await page.getByRole('textbox', { name: 'Search', exact: true }).fill('Alpha');
  await page.getByRole('textbox', { name: 'Search', exact: true }).press('Enter');
  await expect(rows(page)).toHaveCount(1);
  await expect(page).toHaveURL(/[?&]tag=news/);
  await page.locator('.page-title').getByRole('link', { name: 'Remove', exact: true }).click();
  await expect(page).not.toHaveURL(/[?&]tag=/);
  await expect(row(page, 'Alpha')).toBeVisible();
  await page.goto(CAMPAIGNS);
  await row(page, 'Alpha').getByRole('link', { name: 'View Draft', exact: true }).click();
  await expect(page).toHaveURL(/[?&]status=draft/);
  await expect(rows(page)).toHaveCount(3);
  for (const [field, orders] of [
    ['name', [['Alpha', 'Bravo', 'Charlie'], ['Charlie', 'Bravo', 'Alpha']]],
    ['created_at', [['Bravo', 'Alpha', 'Charlie'], ['Charlie', 'Alpha', 'Bravo']]],
  ]) {
    for (const names of orders) {
      await page.locator(`a[data-sort-field="${field}"]`).click();
      await expect(rows(page).locator('td.name .name-stack > div > div > a.unstyled')).toHaveText(names);
      await expect(page).toHaveURL(/[?&]status=draft/);
    }
  }
});

for (const all of [false, true]) {
  test(`bulk deletes ${all ? 'all matching pages' : 'selected IDs'} and preserves other campaigns`, async ({ page }) => {
    for (let i = 0; i < (all ? 25 : 3); i += 1) await create(page, { name: `Bulk ${i}` });
    await page.goto(`${CAMPAIGNS}?query=Bulk`);
    await page.locator('thead input[type="checkbox"]').check();
    await page.locator('button[popovertarget="camp-bulk-actions"]').click();
    if (all) await page.getByRole('menuitem', { name: /Select all/ }).click();
    await page.getByRole('menuitem', { name: /^Delete/ }).click();
    await confirm(page);
    await expect(rows(page)).toHaveCount(0);
    await expect(page.locator('.empty-state')).toBeVisible();
    await page.goto(CAMPAIGNS);
    await expect(rows(page)).toHaveCount(1);
  });
}

test('uploads an attachment, persists it and delivers it when starting from the editor', async ({ page }) => {
  const c = await create(page, { subject: 'Attachment delivery' });
  await page.goto(`${CAMPAIGNS}/${c.id}`);
  await tab(page, 'Attachments').click();
  await button(page, 'Add attachments').click();
  const picker = page.frameLocator('.media-picker-modal iframe');
  await picker.locator('button[popovertarget="media-upload"]').click();
  await picker.locator('input[type="file"]').setInputFiles({ name: 'campaign-example.json', mimeType: 'application/json', buffer: Buffer.from('{"hello":"attachment"}') });
  await picker.locator('button[type="submit"]').filter({ hasText: 'Upload' }).click();
  await picker.getByRole('link', { name: 'campaign-example.json', exact: true }).click();
  await expect(page.locator('.media-picker-modal')).not.toBeVisible();
  await save(page, c.id);
  await page.reload();
  await expect(page.locator('.attachments .filename')).toHaveText('campaign-example.json');
  expect((await campaign(page, c.id)).media).toHaveLength(1);
  await tab(page, 'Campaign').click();
  await button(page, 'Start campaign').click();
  await confirm(page);
  await expect(page).toHaveURL(/\/admin\/campaigns$/);
  expect(JSON.stringify(await delivered(c.subject))).toContain('campaign-example.json');
  await expect.poll(async () => (await campaign(page, c.id)).status, { timeout: 20000 }).toBe('finished');
  await page.reload();
  await expect(row(page, c.name)).toContainText('Finished');
  await page.goto(`${CAMPAIGNS}/${c.id}`);
  await expect(page.getByLabel('Name', { exact: true })).toBeDisabled();
  await expect(button(page, 'Start campaign')).toHaveCount(0);
  await tab(page, 'Archive').click();
  await page.getByRole('switch').filter({ visible: true }).check();
  await page.locator('input[name="archive_slug"]').fill('delivered-newsletter');
  const [response] = await Promise.all([
    page.waitForResponse((r) => r.url().endsWith(`/api/campaigns/${c.id}/archive`) && r.request().method() === 'PUT'),
    button(page, 'Save changes').click(),
  ]);
  await data(response);
  expect(await campaign(page, c.id)).toMatchObject({ status: 'finished', archive: true, archive_slug: 'delivered-newsletter' });
});

test('starts, pauses, resumes and cancels from the listing', async ({ page }) => {
  // Keep sending long enough to change status at the default rate of 10 messages/s.
  const list = await data(await page.request.post('/api/lists', { data: { name: 'Slow audience', type: 'private', optin: 'single' } }));
  for (let i = 0; i < 150; i += 1) {
    await data(await page.request.post('/api/subscribers', { data: {
      email: `lifecycle-${i}@example.com`, name: `Reader ${i}`, status: 'enabled', lists: [list.id],
    } }));
  }
  const c = await create(page, { lists: [list.id] });
  await page.goto(CAMPAIGNS);
  for (const [action, status] of [['Start campaign', 'running'], ['Pause', 'paused'], ['Send', 'running'], ['Cancel', 'cancelled']]) {
    await menu(page, c.name, action);
    await confirm(page);
    await expect(row(page, c.name)).toHaveClass(`status-${status}`);
    expect((await campaign(page, c.id)).status).toBe(status);
  }
  await openMenu(row(page, c.name));
  await expect(row(page, c.name).getByRole('menuitem', { name: 'Start campaign', exact: true })).toHaveCount(0);
});

for (const format of ['richtext', 'visual']) {
  for (const embed of [true, false]) {
    test(`${format} image ${embed ? 'embeds as MIME/CID' : 'stays a URL'} after reload`, async ({ page }) => {
      const c = await create(page, { content_type: format, body: '', subject: `${format} image ${embed}` });
      const media = await data(await page.request.post('/api/media', { multipart: { file: {
        name: 'campaign-pixel.png', mimeType: 'image/png',
        buffer: Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAIAAACQd1PeAAAADElEQVR4nGP4z8AAAAMBAQDJ/pLvAAAAAElFTkSuQmCC', 'base64'),
      } } }));
      await page.goto(`${CAMPAIGNS}/${c.id}#tab=content`);
      await editorReady(page);
      if (format === 'richtext') {
        await button(page, 'Image').click();
        const dialog = page.locator('richtext-editor dialog[open]');
        await button(dialog, 'Media').click();
        await page.frameLocator('.media-picker-modal iframe').getByRole('link', { name: media.filename, exact: true }).click();
        await expect(dialog.locator('input[name="src"]')).toHaveValue(media.url);
        await dialog.getByRole('checkbox', { name: 'Embed', exact: true }).setChecked(embed);
        await button(dialog, 'Insert').click();
        await expect(page.locator('richtext-editor [contenteditable] img')).toHaveAttribute('src', media.url);
      } else {
        await visualDocument(page, {
          root: { type: 'EmailLayout', data: { childrenIds: ['image'] } },
          image: { type: 'Image', data: { props: { url: media.url, embed: false }, style: {} } },
        });
        const frame = page.frameLocator('#visual-editor');
        await frame.locator('#visual-editor-container img').click();
        await frame.getByRole('checkbox').setChecked(embed);
      }
      await save(page, c.id);
      await page.reload();
      await editorReady(page);
      if (format === 'richtext') {
        await page.locator('richtext-editor [contenteditable] img').click();
        await button(page, 'Image').click();
        await expect(page.locator('richtext-editor dialog[open]').getByRole('checkbox', { name: 'Embed', exact: true })).toBeChecked({ checked: embed });
        await button(page.locator('richtext-editor dialog[open]'), 'Close').click();
      } else {
        const frame = page.frameLocator('#visual-editor');
        await frame.locator('#visual-editor-container img').click();
        await expect(frame.getByRole('checkbox')).toBeChecked({ checked: embed });
      }
      await tab(page, 'Campaign').click();
      await button(page, 'Start campaign').click();
      await confirm(page);
      const raw = JSON.stringify(await delivered(c.subject));
      expect(raw.includes('image/png')).toBe(embed);
      expect(raw.includes('cid:')).toBe(embed);
      expect(raw.includes('Content-Id')).toBe(embed);
    });
  }
}

for (const tracked of [true, false]) {
  test(`richtext link tracking follows the URL (${tracked ? 'on' : 'off'}) and persists toggles`, async ({ page }) => {
    const url = 'https://example.com/docs?edition=42#intro';
    const href = (track) => `${url}${track ? '@TrackLink' : ''}`;
    const c = await create(page, { content_type: 'richtext', body:
      `<p><a href="${href(tracked)}">Documentation</a> <a href="${href(!tracked)}">Other link</a></p>` });
    await page.goto(`${CAMPAIGNS}/${c.id}#tab=content`);
    await editorReady(page);

    // Existing links must override the remembered preference for newly inserted links.
    await page.evaluate((value) => localStorage.setItem('trackLink', String(value)), !tracked);
    const link = page.locator('richtext-editor [contenteditable]').getByRole('link', { name: 'Documentation', exact: true });
    const dialog = page.locator('richtext-editor dialog[open]');
    const checkbox = dialog.getByRole('checkbox', { name: 'Track link?', exact: true });
    await link.click();
    await button(page, 'Link').click();
    await expect(dialog.locator('input[name="url"]')).toHaveValue(url);
    await expect(checkbox).toBeChecked({ checked: tracked });
    await button(dialog, 'Close').click();

    // Reusing the URL dialog for another link must recompute the checkbox state.
    await page.locator('richtext-editor [contenteditable]').getByRole('link', { name: 'Other link', exact: true }).click();
    await button(page, 'Link').click();
    await expect(checkbox).toBeChecked({ checked: !tracked });
    await button(dialog, 'Close').click();
    await link.click();
    await button(page, 'Link').click();
    await expect(checkbox).toBeChecked({ checked: tracked });
    await checkbox.setChecked(!tracked);
    await button(dialog, 'Insert').click();
    await expect(link).toHaveAttribute('href', href(!tracked));
    await save(page, c.id);
    expect((await campaign(page, c.id)).body).toContain(`href="${href(!tracked)}"`);
    await page.reload();
    await editorReady(page);
    await link.click();
    await button(page, 'Link').click();
    await expect(dialog.locator('input[name="url"]')).toHaveValue(url);
    await expect(checkbox).toBeChecked({ checked: !tracked });
    await button(dialog, 'Close').click();
  });
}
