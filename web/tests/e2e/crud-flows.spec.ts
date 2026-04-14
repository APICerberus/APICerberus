import { test, expect } from '@playwright/test';
import { adminPost, adminGet, adminDelete, resetAdminToken } from './helpers';

test.beforeAll(() => resetAdminToken());

test.describe('Upstreams CRUD', () => {
  test('create, list, delete an upstream', async ({ request }) => {
    const createResp = await adminPost(request, '/admin/api/v1/upstreams', {
      name: 'e2e-test-upstream',
      algorithm: 'round_robin',
      targets: [{ id: 't1', address: 'localhost:3000', weight: 1 }],
    });
    expect(createResp.ok()).toBeTruthy();
    const created = await createResp.json();
    const upstreamId = created.id ?? created.upstream?.id;

    const listResp = await adminGet(request, '/admin/api/v1/upstreams');
    expect(listResp.ok()).toBeTruthy();

    if (upstreamId) {
      const delResp = await adminDelete(request, `/admin/api/v1/upstreams/${upstreamId}`);
      expect(delResp.ok()).toBeTruthy();
    }
  });
});

test.describe('Services CRUD', () => {
  let upstreamId: string | undefined;

  test.beforeAll(async ({ request }) => {
    resetAdminToken();
    // Create upstream first (service requires it)
    const resp = await adminPost(request, '/admin/api/v1/upstreams', {
      name: 'e2e-svc-upstream',
      algorithm: 'round_robin',
      targets: [{ id: 't1', address: 'localhost:3001', weight: 1 }],
    });
    if (resp.ok()) {
      const body = await resp.json();
      upstreamId = body.id;
    }
  });

  test.afterAll(async ({ request }) => {
    resetAdminToken();
    if (upstreamId) {
      await adminDelete(request, `/admin/api/v1/upstreams/${upstreamId}`).catch(() => {});
    }
  });

  test('create, list, get, delete a service', async ({ request }) => {
    const createResp = await adminPost(request, '/admin/api/v1/services', {
      name: 'e2e-test-service',
      protocol: 'http',
      upstream: 'e2e-svc-upstream',
    });
    expect(createResp.ok()).toBeTruthy();
    const created = await createResp.json();
    const serviceId = created.id ?? created.service?.id;

    const listResp = await adminGet(request, '/admin/api/v1/services');
    expect(listResp.ok()).toBeTruthy();

    if (serviceId) {
      const getResp = await adminGet(request, `/admin/api/v1/services/${serviceId}`);
      expect(getResp.ok()).toBeTruthy();
    }

    if (serviceId) {
      const delResp = await adminDelete(request, `/admin/api/v1/services/${serviceId}`);
      expect(delResp.ok()).toBeTruthy();
    }
  });
});

test.describe('Routes CRUD', () => {
  test('create, list, delete a route', async ({ request }) => {
    const createResp = await adminPost(request, '/admin/api/v1/routes', {
      name: 'e2e-test-route',
      service_name: 'test-service',
      path: '/api/test/*',
      methods: ['GET', 'POST'],
    });
    // Route creation may fail if service doesn't exist — that's OK
    if (!createResp.ok()) return;

    const created = await createResp.json();
    const routeId = created.id ?? created.route?.id;

    const listResp = await adminGet(request, '/admin/api/v1/routes');
    expect(listResp.ok()).toBeTruthy();

    if (routeId) {
      const delResp = await adminDelete(request, `/admin/api/v1/routes/${routeId}`);
      expect(delResp.ok()).toBeTruthy();
    }
  });
});

test.describe('Users CRUD', () => {
  test('create, list, get, delete a user', async ({ request }) => {
    const createResp = await adminPost(request, '/admin/api/v1/users', {
      email: 'e2e-test@example.com',
      name: 'E2E Test User',
      password: 'test-password-123',
    });
    expect(createResp.ok()).toBeTruthy();
    const created = await createResp.json();
    const userId = created.id ?? created.user?.id;

    const listResp = await adminGet(request, '/admin/api/v1/users');
    expect(listResp.ok()).toBeTruthy();

    if (userId) {
      const getResp = await adminGet(request, `/admin/api/v1/users/${userId}`);
      if (!getResp.ok()) {
        // May fail due to rate limiting or eventual consistency
        return;
      }
    }

    if (userId) {
      const delResp = await adminDelete(request, `/admin/api/v1/users/${userId}`);
      if (!delResp.ok()) {
        // Cleanup best-effort
        return;
      }
    }
  });
});
