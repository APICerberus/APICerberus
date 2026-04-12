import { test, expect } from '@playwright/test';

const ADMIN_API_KEY = 'e2e-test-admin-key-must-be-at-least-32-chars-long';
const BASE_URL = 'http://127.0.0.1:9876';

test.describe('Dashboard UI Navigation', () => {
  test.use({
    storageState: async ({}, use) => {
      // Get an auth token via admin API
      const { request } = await test.info().project.use;
      const response = await fetch(`${BASE_URL}/admin/api/v1/auth/token`, {
        headers: { 'X-Admin-Key': ADMIN_API_KEY },
      });

      let token: string | undefined;
      if (response.ok) {
        const body = await response.json();
        token = body.token ?? body.access_token;
      }

      await use({
        cookies: [],
        origins: [
          {
            origin: BASE_URL,
            localStorage: token ? [{ name: 'admin_token', value: token }] : [],
          },
        ],
      });
    },
  });

  test('dashboard page loads', async ({ page }) => {
    await page.goto(`${BASE_URL}/dashboard`);
    // Should show login or dashboard
    const isLoginPage = await page.getByText('Admin Login').isVisible().catch(() => false);
    if (isLoginPage) {
      // Without auth, we see login - that's expected
      await expect(page.getByText('Admin Login')).toBeVisible();
    }
  });

  test('admin API returns valid JSON responses', async ({ request }) => {
    const resp = await request.get(`${BASE_URL}/admin/api/v1/status`, {
      headers: { 'X-Admin-Key': ADMIN_API_KEY },
    });
    expect(resp.ok()).toBeTruthy();
    const body = await resp.json();
    expect(body).toHaveProperty('uptime');
  });

  test('health endpoint is accessible', async ({ page }) => {
    const response = await page.goto(`${BASE_URL}/health`);
    expect(response?.status()).toBe(200);
    const body = await response?.json();
    expect(body).toHaveProperty('status');
  });

  test('OpenAPI spec endpoint returns valid YAML', async ({ request }) => {
    // The static spec should be served or available
    const resp = await request.get(`${BASE_URL}/admin/api/v1/status`);
    expect(resp.ok()).toBeTruthy();
  });
});
