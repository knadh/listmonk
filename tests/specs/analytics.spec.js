import { test, expect } from '@playwright/test';
import { resetDB, psql } from '../helpers.js';

// Chart axis labels are formatted in the browser's local time.
test.use({ timezoneId: 'UTC' });

async function create(page, name) {
  const res = await page.request.post('/api/campaigns', { data: {
    name, subject: name, type: 'regular', content_type: 'html', body: '<p>Hi</p>', lists: [1],
  } });
  expect(res.ok(), await res.text()).toBeTruthy();
  return (await res.json()).data;
}

// Chart.js isn't a global, so import the bundle chunk the page already loaded
// (the same module instance) and read the chart's x-axis from its registry.
function axisLabels(page, id) {
  return page.evaluate(async (id) => {
    const urls = performance.getEntriesByType('resource').map((e) => e.name).filter((u) => u.includes('/chunks/'));
    for (const url of urls) {
      const Chart = Object.values(await import(url)).find((v) => typeof v?.getChart === 'function');
      if (Chart) {
        return Chart.getChart(document.getElementById(id))?.scales.x.getLabels();
      }
    }
    return null;
  }, id);
}

test.describe('Analytics', () => {
  test.beforeAll(async ({ browser }) => {
    await resetDB(browser);
  });

  test('orders the x-axis by date across campaigns', async ({ page }) => {
    const a = await create(page, 'Analytics A');
    const b = await create(page, 'Analytics B');

    // B bounced on a day that A, which is plotted first, didn't.
    psql(`INSERT INTO bounces (subscriber_id, campaign_id, type, source, created_at) VALUES
      (1, ${a.id}, 'soft', 'api', '2026-09-27 10:00:00+00'),
      (1, ${a.id}, 'soft', 'api', '2026-10-07 10:00:00+00'),
      (1, ${b.id}, 'soft', 'api', '2026-09-24 10:00:00+00'),
      (1, ${b.id}, 'soft', 'api', '2026-09-27 10:00:00+00');`);

    await page.goto(`/admin/campaigns/analytics?id=${a.id}&id=${b.id}&from=2026-09-20&to=2026-10-10&interval=day`);
    await expect.poll(() => axisLabels(page, 'chart-bounces'))
      .toEqual(['2026-09-24 00:00', '2026-09-27 00:00', '2026-10-07 00:00']);
  });
});
