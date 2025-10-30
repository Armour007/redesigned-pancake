import { test, expect, type Page } from '@playwright/test';

const loginWithToken = async (page: Page, token: string, orgId: string) => {
  await page.goto(`/login/sso-callback?token=${encodeURIComponent(token)}&orgId=${encodeURIComponent(orgId)}`);
  await page.waitForURL('**/dashboard');
};

test('unauthenticated redirect to login', async ({ page }) => {
  await page.goto('/dashboard');
  await expect(page).toHaveURL(/\/login$/);
});

test('authenticated nav works and layout fetches do not break', async ({ page }) => {
  // Mock layout calls
  await page.route('**/me', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ full_name: 'Test User', email: 'test@example.com' }) });
  });
  await page.route('**/admin/queue/dlq?**', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ total: 1, items: [] }) });
  });
  await page.route('**/admin/webhooks/dlq?**', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ total: 2, items: [] }) });
  });

  await loginWithToken(page, 'e2e_token', 'org1');

  // On dashboard, click Webhooks DLQ in the sidebar
  await page.getByRole('link', { name: 'Webhooks DLQ' }).click();
  await expect(page).toHaveURL(/\/admin\/webhooks\/dlq$/);
});
