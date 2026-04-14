import { test, expect } from '@playwright/test';
import { BASE_URL, adminGet, resetAdminToken } from './helpers';

const ADMIN_API_KEY = 'e2e-test-admin-key-must-be-at-least-32-chars-long';

test.describe('Admin API Auth', () => {
  test.beforeAll(() => resetAdminToken());

  test('authenticates and returns status', async ({ request }) => {
    const response = await adminGet(request, '/admin/api/v1/status');
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body).toHaveProperty('status');
  });

  test('rejects requests without auth', async ({ request }) => {
    const response = await request.get(`${BASE_URL}/admin/api/v1/status`);
    expect(response.status()).toBe(401);
  });

  test('rejects requests with invalid bearer token', async ({ request }) => {
    const response = await request.get(`${BASE_URL}/admin/api/v1/status`, {
      headers: { Authorization: 'Bearer invalid-token' },
    });
    expect(response.status()).toBe(401);
  });
});

test.describe('Admin Login Page', () => {
  test('renders the login page', async ({ page }) => {
    await page.goto(`${BASE_URL}/dashboard`);
    await expect(page.getByText('API Cerberus Admin')).toBeVisible({ timeout: 15000 });
    await expect(page.getByLabel('Admin API Key')).toBeVisible();
  });

  test('has password input and submit button', async ({ page }) => {
    await page.goto(`${BASE_URL}/dashboard`);
    await expect(page.getByText('API Cerberus Admin')).toBeVisible({ timeout: 15000 });
    await expect(page.getByLabel('Admin API Key')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Continue' })).toBeVisible();
  });

  test('login form submits to /admin/login', async ({ page }) => {
    await page.goto(`${BASE_URL}/dashboard`);
    await expect(page.getByText('API Cerberus Admin')).toBeVisible({ timeout: 15000 });
    const form = page.locator('form');
    await expect(form).toHaveAttribute('action', '/admin/login');
    await expect(form).toHaveAttribute('method', 'POST');
  });
});
