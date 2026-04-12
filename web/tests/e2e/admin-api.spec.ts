import { test, expect } from '@playwright/test';

const ADMIN_API_KEY = 'e2e-test-admin-key-must-be-at-least-32-chars-long';
const BASE_URL = 'http://127.0.0.1:9876';

test.describe('Admin API', () => {
  test('authenticates with admin key', async ({ request }) => {
    const response = await request.get(`${BASE_URL}/admin/api/v1/status`, {
      headers: { 'X-Admin-Key': ADMIN_API_KEY },
    });
    expect(response.ok()).toBeTruthy();
    const body = await response.json();
    expect(body).toHaveProperty('version');
    expect(body).toHaveProperty('uptime');
  });

  test('rejects requests without admin key', async ({ request }) => {
    const response = await request.get(`${BASE_URL}/admin/api/v1/status`);
    expect(response.status()).toBe(401);
  });

  test('rejects requests with invalid admin key', async ({ request }) => {
    const response = await request.get(`${BASE_URL}/admin/api/v1/status`, {
      headers: { 'X-Admin-Key': 'invalid-key' },
    });
    expect(response.status()).toBe(401);
  });
});

test.describe('Admin Login Page', () => {
  test('renders the login page', async ({ page }) => {
    await page.goto(`${BASE_URL}/dashboard`);
    await expect(page.getByText('Admin Login')).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Admin Login' })).toBeVisible();
  });

  test('has password input and submit button', async ({ page }) => {
    await page.goto(`${BASE_URL}/dashboard`);
    await expect(page.getByLabel('Admin API Key')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Continue' })).toBeVisible();
  });

  test('login form submits POST to correct action', async ({ page }) => {
    await page.goto(`${BASE_URL}/dashboard`);
    const form = page.locator('form');
    await expect(form).toHaveAttribute('action', '/admin/login');
    await expect(form).toHaveAttribute('method', 'POST');
  });
});
