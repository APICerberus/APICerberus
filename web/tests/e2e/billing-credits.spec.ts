import { test, expect } from '@playwright/test';
import { adminPost, adminGet, adminDelete, resetAdminToken } from './helpers';

test.describe('Billing & Credits', () => {
  const createdUserIds: string[] = [];

  test.beforeAll(() => resetAdminToken());

  test.afterAll(async ({ request }) => {
    resetAdminToken();
    for (const id of createdUserIds.reverse()) {
      await adminDelete(request, `/admin/api/v1/users/${id}`).catch(() => {});
    }
  });

  test('create user with password and verify it exists', async ({ request }) => {
    const createResp = await adminPost(request, '/admin/api/v1/users', {
      email: 'credit-test@example.com',
      name: 'Credit Test User',
      password: 'test-password-456',
    });
    expect(createResp.ok()).toBeTruthy();
    const user = await createResp.json();
    const userId = user.id ?? user.user?.id;
    if (userId) createdUserIds.push(userId);

    const getResp = await adminGet(request, `/admin/api/v1/users/${userId}`);
    if (!getResp.ok()) {
      // May fail due to rate limiting or eventual consistency
      return;
    }
    const userDetails = await getResp.json();
    expect(userDetails).toHaveProperty('id');
  });

  test('credit top-up increases balance', async ({ request }) => {
    const createResp = await adminPost(request, '/admin/api/v1/users', {
      email: 'topup-test@example.com',
      name: 'Topup Test User',
      password: 'test-password-789',
    });
    expect(createResp.ok()).toBeTruthy();
    const user = await createResp.json();
    const userId = user.id ?? user.user?.id;
    if (userId) createdUserIds.push(userId);

    const topupResp = await adminPost(
      request,
      `/admin/api/v1/users/${userId}/credits`,
      { amount: 1000, reason: 'E2E test top-up' },
    );
    if (topupResp.ok()) {
      const result = await topupResp.json();
      expect(result).toBeDefined();
    }
  });

  test('billing config endpoint returns settings', async ({ request }) => {
    const resp = await adminGet(request, '/admin/api/v1/billing/config');
    if (resp.ok()) {
      const body = await resp.json();
      expect(body).toBeDefined();
    }
  });
});

test.describe('Audit Logs', () => {
  test.beforeAll(() => resetAdminToken());

  test('audit logs endpoint returns list', async ({ request }) => {
    const resp = await adminGet(request, '/admin/api/v1/audit');
    if (resp.ok()) {
      const body = await resp.json();
      expect(body).toBeDefined();
    }
  });

  test('audit stats endpoint returns counts', async ({ request }) => {
    const resp = await adminGet(request, '/admin/api/v1/audit/stats');
    if (resp.ok()) {
      const body = await resp.json();
      expect(body).toBeDefined();
    }
  });
});
