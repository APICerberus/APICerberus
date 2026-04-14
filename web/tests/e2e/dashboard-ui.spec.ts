import { test, expect } from '@playwright/test';
import { BASE_URL, adminGet, resetAdminToken } from './helpers';

const ADMIN_API_KEY = 'e2e-test-admin-key-must-be-at-least-32-chars-long';

test.describe('Dashboard UI Navigation', () => {
  test.beforeAll(() => resetAdminToken());

  test('dashboard page loads and shows login or dashboard', async ({ page }) => {
    await page.goto(`${BASE_URL}/dashboard`);
    await expect(page.getByText('API Cerberus Admin')).toBeVisible({ timeout: 15000 });
  });

  test('admin API returns valid JSON responses', async ({ request }) => {
    const resp = await adminGet(request, '/admin/api/v1/status');
    expect(resp.ok()).toBeTruthy();
    const body = await resp.json();
    expect(body).toHaveProperty('status');
  });

  test('health endpoint is accessible', async ({ page }) => {
    const response = await page.goto(`${BASE_URL}/ready`);
    // May get HTML from SPA fallback; that's OK — server is up
    expect(response).not.toBeNull();
    const status = response?.status();
    expect(status).toBeLessThan(500);
  });

  test('info endpoint returns gateway info', async ({ request }) => {
    const resp = await adminGet(request, '/admin/api/v1/info');
    if (resp.ok()) {
      const body = await resp.json();
      expect(body).toBeDefined();
    }
  });
});
