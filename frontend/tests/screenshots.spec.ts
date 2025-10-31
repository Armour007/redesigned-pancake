import { test, expect } from '@playwright/test';

const FRONTEND = process.env.FRONTEND_BASE || 'http://localhost:5173';
const API = process.env.API_BASE || 'http://localhost:8081';

async function createUser() {
  const rand = Math.floor(Math.random() * 1e9);
  const email = `ui${rand}@example.com`;
  const password = 'P@ssw0rd12345!';
  // Register
  const reg = await fetch(`${API}/auth/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ full_name: 'UI Tester', email, password })
  });
  if (!reg.ok) throw new Error('register failed');
  // Login
  const login = await fetch(`${API}/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password })
  });
  if (!login.ok) throw new Error('login failed');
  const j = await login.json();
  const token: string = j.token;
  // Get org
  const orgsRes = await fetch(`${API}/organizations/mine`, { headers: { Authorization: `Bearer ${token}` } });
  if (!orgsRes.ok) throw new Error('orgs failed');
  const orgs = await orgsRes.json();
  const orgId: string = orgs[0]?.id;
  return { token, orgId };
}

test('brand screenshots', async ({ page }) => {
  test.setTimeout(120000);
  const { token, orgId } = await createUser();

  // Seed localStorage before first navigation
  await page.addInitScript(([t, o]) => {
    window.localStorage.setItem('aura_token', t as string);
    window.localStorage.setItem('aura_org_id', o as string);
  }, [token, orgId]);

  const shots = [
    '/',
    '/dashboard',
    '/agents',
    '/apikeys',
    '/logs',
    '/devices',
    '/settings',
    '/admin/queue',
    '/admin/webhooks/dlq'
  ];

  for (const path of shots) {
    const url = FRONTEND.replace(/\/$/, '') + path;
    await page.goto(url, { waitUntil: 'networkidle' });
    // Wait for header or primary content
    await page.waitForTimeout(500);
    await page.screenshot({ path: `screenshots${path.replace(/\//g, '_') || '_home'}.png`, fullPage: true });
  }
});
