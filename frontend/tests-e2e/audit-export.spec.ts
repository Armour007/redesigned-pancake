import { test, expect, type Page } from '@playwright/test';

const loginWithToken = async (page: Page, token: string, orgId: string) => {
  await page.goto(`/login/sso-callback?token=${encodeURIComponent(token)}&orgId=${encodeURIComponent(orgId)}`);
  await page.waitForURL('**/dashboard');
};

test('audit export page shows controls and can save', async ({ page }) => {
  // Mock API endpoints used by layout and this page
  await page.route('**/me', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ full_name: 'Test User', email: 'test@example.com' }) });
  });
  await page.route('**/admin/queue/dlq?**', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ total: 0, items: [] }) });
  });
  await page.route('**/admin/webhooks/dlq?**', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ total: 0, items: [] }) });
  });

  // State for schedule
  let schedule = { cron: '0 1 * * *', destType: 'file', dest: '/tmp/audit.zip', format: 'json', lookback: '720h' };

  await page.route('**/organizations/**/regulator/audit-export/schedule', async (route, request) => {
    if (request.method() === 'GET') {
      return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ schedule }) });
    }
    // POST save
    const body = request.postDataJSON() || {};
    schedule = { cron: body.cron || schedule.cron, destType: body.dest_type || schedule.destType, dest: body.dest || schedule.dest, format: body.format || schedule.format, lookback: body.lookback || schedule.lookback } as any;
    return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ ok: true }) });
  });

  await loginWithToken(page, 'e2e_token', 'org1');

  await page.goto('/organizations/org1/regulator/audit-export');

  // Labels and inputs present
  await expect(page.getByLabel('Cron')).toBeVisible();
  await expect(page.getByLabel('Destination Type')).toBeVisible();
  await expect(page.getByLabel('Destination', { exact: true })).toBeVisible();
  await expect(page.getByLabel('Format')).toBeVisible();
  await expect(page.getByLabel('Lookback (duration)')).toBeVisible();

  // Save flow
  await page.getByLabel('Cron').fill('0 2 * * *');
  await page.getByRole('button', { name: 'Save' }).click();

  await expect(page.getByText('Saved')).toBeVisible();
});

test('audit export page shows toast on failing save', async ({ page }) => {
  // Mock layout calls
  await page.route('**/me', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ full_name: 'Test User', email: 'test@example.com' }) });
  });
  await page.route('**/admin/queue/dlq?**', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ total: 0, items: [] }) });
  });
  await page.route('**/admin/webhooks/dlq?**', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ total: 0, items: [] }) });
  });

  // GET returns existing schedule
  await page.route('**/organizations/**/regulator/audit-export/schedule', async (route, request) => {
    if (request.method() === 'GET') {
      return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ schedule: { cron: '0 1 * * *', destType: 'file', dest: '/tmp/audit.zip', format: 'json', lookback: '720h' } }) });
    }
    // POST save fails
    return route.fulfill({ status: 500, contentType: 'text/plain', body: 'internal error' });
  });

  await loginWithToken(page, 'e2e_token', 'org1');
  await page.goto('/organizations/org1/regulator/audit-export');

  // Attempt to save and expect failure status text visible
  await page.getByRole('button', { name: 'Save' }).click();
  await expect(page.getByText('Failed to save')).toBeVisible();
});
