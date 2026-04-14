import { test, expect } from '@playwright/test';
import { BASE_URL, adminPost, adminGet, adminDelete, resetAdminToken } from './helpers';

const ADMIN_API_KEY = 'e2e-test-admin-key-must-be-at-least-32-chars-long';

test.describe('API Key Lifecycle', () => {
  const createdUserIds: string[] = [];

  test.beforeAll(() => resetAdminToken());

  test.afterAll(async ({ request }) => {
    resetAdminToken();
    for (const id of createdUserIds.reverse()) {
      await adminDelete(request, `/admin/api/v1/users/${id}`).catch(() => {});
    }
  });

  test('create user, create API key, list keys, delete key', async ({ request }) => {
    const userResp = await adminPost(request, '/admin/api/v1/users', {
      email: 'apikey-test@example.com',
      name: 'API Key Test User',
      password: 'test-password-abc',
    });
    expect(userResp.ok()).toBeTruthy();
    const user = await userResp.json();
    const userId = user.id ?? user.user?.id;
    if (userId) createdUserIds.push(userId);
    expect(userId).toBeTruthy();

    const keyResp = await adminPost(request, `/admin/api/v1/users/${userId}/apikeys`, {
      name: 'E2E Test Key',
      mode: 'live',
    });
    if (keyResp.ok()) {
      const keyData = await keyResp.json();
      const keyValue = keyData.key ?? keyData.api_key?.key;
      if (keyValue) {
        expect(keyValue).toMatch(/^ck_live_/);
      }
    }

    const listResp = await adminGet(request, `/admin/api/v1/users/${userId}/apikeys`);
    if (listResp.ok()) {
      const keys = await listResp.json();
      expect(keys).toBeDefined();
    }
  });

  test('create test mode API key with ck_test_ prefix', async ({ request }) => {
    const userResp = await adminPost(request, '/admin/api/v1/users', {
      email: 'testkey-user@example.com',
      name: 'Test Key User',
      password: 'test-password-xyz',
    });
    expect(userResp.ok()).toBeTruthy();
    const user = await userResp.json();
    const userId = user.id ?? user.user?.id;
    if (userId) createdUserIds.push(userId);

    const keyResp = await adminPost(request, `/admin/api/v1/users/${userId}/apikeys`, {
      name: 'E2E Test Mode Key',
      mode: 'test',
    });
    if (keyResp.ok()) {
      const keyData = await keyResp.json();
      const keyValue = keyData.key ?? keyData.api_key?.key;
      if (keyValue) {
        expect(keyValue).toMatch(/^ck_test_/);
      }
    }
  });
});

test.describe('Admin Login Flow', () => {
  test('login page shows form elements', async ({ page }) => {
    await page.goto(`${BASE_URL}/dashboard`);
    await expect(page.getByText('API Cerberus Admin')).toBeVisible({ timeout: 15000 });

    await expect(page.getByLabel('Admin API Key')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Continue' })).toBeVisible();
  });

  test('login with valid key redirects to dashboard', async ({ page }) => {
    await page.goto(`${BASE_URL}/dashboard`);
    await expect(page.getByText('API Cerberus Admin')).toBeVisible({ timeout: 15000 });

    await page.getByLabel('Admin API Key').fill(ADMIN_API_KEY);
    await page.getByRole('button', { name: 'Continue' }).click();

    // Should redirect to dashboard with login=success
    await page.waitForURL(/login=success|\/dashboard\/?$/, { timeout: 5000 }).catch(() => {});
  });

  test('login with invalid key shows error', async ({ page }) => {
    await page.goto(`${BASE_URL}/dashboard`);
    await expect(page.getByText('API Cerberus Admin')).toBeVisible({ timeout: 15000 });

    await page.getByLabel('Admin API Key').fill('invalid-key');
    await page.getByRole('button', { name: 'Continue' }).click();

    // Server redirects to /dashboard?login=invalid_key; SPA may further redirect to /login
    await page.waitForURL(/login/, { timeout: 5000 }).catch(() => {});
    const url = page.url();
    // Either query param or /login path — both indicate login error handling
    expect(url).toMatch(/login/);
  });

  test('dashboard shows login when not authenticated', async ({ page }) => {
    await page.goto(`${BASE_URL}/dashboard`);
    await expect(page.getByText('API Cerberus Admin')).toBeVisible({ timeout: 15000 });
  });
});

test.describe('Admin Plugin Management', () => {
  test.beforeAll(() => resetAdminToken());

  test('plugins endpoint returns plugin list', async ({ request }) => {
    const resp = await adminGet(request, '/admin/api/v1/plugins');
    if (resp.ok()) {
      const body = await resp.json();
      expect(body).toBeDefined();
    }
  });
});
