import { test, expect } from '@playwright/test';
import { confirm, openMenu, resetDB } from '../helpers.js';

const MEDIA = '/admin/campaigns/media';
const cards = (page) => page.locator('.gallery figure');
const card = (page, name) => cards(page).filter({ has: page.getByTitle(name, { exact: true }) });
const preview = (page) => page.locator('dialog.media-preview');

async function data(response) {
  expect(response.ok(), await response.text()).toBeTruthy();
  return (await response.json()).data;
}

async function seed(page, name, tags = '') {
  return data(await page.request.post('/api/media', { multipart: {
    file: { name, mimeType: 'application/json', buffer: Buffer.from('{}') }, tags,
  } }));
}

async function addTag(scope, tag) {
  await scope.getByRole('combobox', { name: 'Tags', exact: true }).fill(tag);
  await scope.getByRole('combobox', { name: 'Tags', exact: true }).press('Enter');
}

test.beforeEach(async ({ browser }) => {
  await resetDB(browser);
});

test('uploads multiple files with normalized tags and previews images and documents', async ({ page }) => {
  await seed(page, 'existing.json', 'news');
  await page.goto(MEDIA);
  await page.locator('button[popovertarget="media-upload"]').click();
  
  const upload = page.locator('#media-upload');
  await expect(upload.locator('button[type="submit"]')).toBeDisabled();
  await upload.locator('input[type="file"]').setInputFiles([
    { name: 'pixel.png', mimeType: 'image/png', buffer: Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAIAAACQd1PeAAAADElEQVR4nGP4z8AAAAMBAQDJ/pLvAAAAAElFTkSuQmCC', 'base64') },
    { name: 'document.json', mimeType: 'application/json', buffer: Buffer.from('{}') },
  ]);
  await upload.getByRole('combobox', { name: 'Tags', exact: true }).fill('new');
  await page.getByRole('option', { name: '#news', exact: true }).click();
  await addTag(upload, ' Summer Sale ');
  await upload.locator('button[type="submit"]').click();
  await expect(cards(page)).toHaveCount(3);
  expect(await data(await page.request.get('/api/media/tags'))).toEqual([
    { tag: 'news', count: 3 }, { tag: 'summer-sale', count: 2 },
  ]);

  for (const name of ['pixel.png', 'document.json']) {
    await expect(card(page, name).locator('.media-tags a')).toHaveText(['#news', '#summer-sale']);
    await card(page, name).locator('a.thumb').click();
    await expect(preview(page)).toBeVisible();
    await expect(preview(page).getByRole('heading')).toHaveText(name);
    await expect(preview(page).getByRole('button', { name: 'Save', exact: true })).toBeDisabled();
    if (name.endsWith('.png')) await expect(preview(page).getByRole('img')).toHaveAttribute('src', /pixel\.png$/);
    else await expect(preview(page).locator('.ext')).toHaveText('JSON');
    await preview(page).getByRole('button', { name: 'Close', exact: true }).click();
    await expect(preview(page)).toBeHidden();
  }
});

test('edits, discards and clears preview tags, then deletes media', async ({ page }) => {
  const media = await seed(page, 'editable.json', 'old');
  await page.goto(MEDIA);
  await openMenu(card(page, media.filename));
  await card(page, media.filename).getByRole('menuitem', { name: 'Edit', exact: true }).click();
  await addTag(preview(page), 'discarded');
  await preview(page).getByRole('button', { name: 'Close', exact: true }).click();
  await card(page, media.filename).locator('a.thumb').click();
  await expect(preview(page).locator('ot-taginput .badge')).toHaveCount(1);
  await expect(preview(page).getByRole('button', { name: 'Save', exact: true })).toBeDisabled();
  await preview(page).locator('ot-taginput .badge').hover();
  await preview(page).getByRole('button', { name: 'Remove old', exact: true }).click();
  await addTag(preview(page), 'New Tag');
  await preview(page).getByRole('button', { name: 'Save', exact: true }).click();
  await expect(card(page, media.filename).locator('.media-tags')).toHaveText('#new-tag');
  expect((await data(await page.request.get(`/api/media/${media.id}`))).tags).toEqual(['new-tag']);
  await card(page, media.filename).locator('a.thumb').click();
  await preview(page).locator('ot-taginput .badge').hover();
  await preview(page).getByRole('button', { name: 'Remove new-tag', exact: true }).click();
  await preview(page).getByRole('button', { name: 'Save', exact: true }).click();
  await expect(card(page, media.filename).locator('.media-tags')).toHaveCount(0);
  expect((await data(await page.request.get(`/api/media/${media.id}`))).tags).toEqual([]);
  expect(await data(await page.request.get('/api/media/tags'))).toEqual([]);
  await openMenu(card(page, media.filename));
  await card(page, media.filename).getByRole('menuitem', { name: 'Delete', exact: true }).click();
  await confirm(page);
  await expect(cards(page)).toHaveCount(0);
  expect((await page.request.get(`/api/media/${media.id}`)).status()).toBe(404);
});

test('filters by sidebar, card and preview tags and searches filenames and tags', async ({ page }) => {
  await seed(page, 'first.json', 'news');
  await seed(page, 'second.json', 'news');
  await seed(page, 'other.json', 'other');
  await page.goto(MEDIA);

  const sidebar = page.getByRole('navigation', { name: 'Tags', exact: true });
  await sidebar.getByRole('link', { name: '# news (2)', exact: true }).click();
  await expect(cards(page).locator('.filename')).toHaveText(['second.json', 'first.json']);
  await expect(sidebar.getByRole('link', { name: '# news (2)', exact: true })).toHaveAttribute('aria-current', 'page');
  await sidebar.getByRole('link', { name: 'All', exact: true }).click();
  await card(page, 'other.json').getByRole('link', { name: '#other', exact: true }).click();
  await expect(cards(page).locator('.filename')).toHaveText(['other.json']);
  await sidebar.getByRole('link', { name: 'All', exact: true }).click();
  await card(page, 'first.json').locator('a.thumb').click();
  await preview(page).getByRole('link', { name: '#news', exact: true }).click();
  await expect(cards(page)).toHaveCount(2);
  await sidebar.getByRole('link', { name: 'All', exact: true }).click();

  const search = page.getByRole('combobox', { name: 'Search', exact: true });
  await search.fill('new');
  await page.getByRole('option', { name: '#news', exact: true }).click();
  await page.getByRole('button', { name: 'Search', exact: true }).click();
  await expect(cards(page)).toHaveCount(2);
  await search.fill('FIRST');
  await search.press('Enter');
  await expect(cards(page).locator('.filename')).toHaveText(['first.json']);
  await search.fill('missing');
  await search.press('Enter');
  await expect(cards(page)).toHaveCount(0);
});
