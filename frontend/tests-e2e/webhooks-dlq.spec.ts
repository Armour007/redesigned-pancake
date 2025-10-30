import { test, expect, type Page } from '@playwright/test';

const loginWithToken = async (page: Page, token: string, orgId: string) => {
  await page.goto(`/login/sso-callback?token=${encodeURIComponent(token)}&orgId=${encodeURIComponent(orgId)}`);
  await page.waitForURL('**/dashboard');
};

test('webhooks DLQ listing and actions', async ({ page }) => {
  // Mock layout calls
  await page.route('**/me', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ full_name: 'Test User', email: 'test@example.com' }) });
  });
  await page.route('**/admin/queue/dlq?**', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ total: 0, items: [] }) });
  });

  // Stateful DLQ list and action endpoints
  let page1 = [
    { id: '1', endpoint: 'wh_ep_1', event: 'user.created', at: 1700000000, url: 'https://example.com/webhook', attempts: 3, last_code: 500, org_id: 'org1' },
    { id: '2', endpoint: 'wh_ep_2', event: 'user.updated', at: 1700000100, url: 'https://example.com/webhook2', attempts: 2, last_code: 429, org_id: 'org1' }
  ];
  let page2 = [
    { id: '0', endpoint: 'wh_ep_0', event: 'org.created', at: 1699999999, url: 'https://example.com/old', attempts: 5, last_code: 500, org_id: 'org1' }
  ];
  let items = [...page1];
  let next_before_id: string | null = '2';

  await page.route('**/admin/webhooks/dlq?**', async (route, request) => {
    const url = new URL(request.url());
    const beforeId = url.searchParams.get('before_id');
    if (!beforeId) {
      items = [...page1];
      next_before_id = '2';
    } else {
      items = [...page1, ...page2];
      next_before_id = null;
    }
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ total: items.length, items, next_before_id }) });
  });
  await page.route('**/admin/webhooks/dlq/requeue', async (route, request) => {
    const body = (request.postDataJSON() || {}) as { ids?: string[] };
    if (body.ids?.length) {
      items = items.filter(i => !body.ids!.includes(i.id));
    }
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ ok: true }) });
  });
  await page.route('**/admin/webhooks/dlq/delete', async (route, request) => {
    const body = (request.postDataJSON() || {}) as { ids?: string[] };
    if (body.ids?.length) {
      items = items.filter(i => !body.ids!.includes(i.id));
    }
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ ok: true }) });
  });

  await loginWithToken(page, 'e2e_token', 'org1');
  await page.goto('/admin/webhooks/dlq');

  // Wait for table to render (items > 0)
  await expect(page.getByRole('table')).toBeVisible();
  // Wait for a row checkbox to confirm body rendered
  await expect(page.getByRole('checkbox', { name: 'Select 1' })).toBeVisible();

  // Select first row
  await page.getByRole('checkbox', { name: 'Select 1' }).check();

  // Requeue selected
  await page.getByRole('button', { name: 'Requeue selected' }).click();

  // After requeue, the list should update
  await expect(page.getByText('Requeued 1 item')).toBeVisible();

  // Delete remaining
  await page.getByRole('checkbox', { name: 'Select 2' }).check();
  await page.getByRole('button', { name: 'Delete selected' }).click();
  await expect(page.getByText('Deleted 1 item')).toBeVisible();

  // Load more (older)
  await page.getByRole('button', { name: 'Load more (older)' }).click();
  await expect(page.getByText('wh_ep_0')).toBeVisible();

  // Open link navigates to endpoint page for org/endpoint
  await page.getByRole('link', { name: 'Open' }).first().click();
  await expect(page).toHaveURL(/\/organizations\/org1\/webhooks\?endpointId=/);
});
