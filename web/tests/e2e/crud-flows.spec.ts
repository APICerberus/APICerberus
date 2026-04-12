import { test, expect } from '@playwright/test';

const ADMIN_API_KEY = 'e2e-test-admin-key-must-be-at-least-32-chars-long';
const BASE_URL = 'http://127.0.0.1:9876';

async function adminApi(request: ReturnType<typeof test>, method: string, path: string, body?: object) {
  const opts: Record<string, unknown> = {
    headers: { 'X-Admin-Key': ADMIN_API_KEY, 'Content-Type': 'application/json' },
  };
  if (body) opts.data = body;
  return request[method.toLowerCase()](path, opts);
}

test.describe('Services CRUD', () => {
  test('create, list, get, delete a service', async ({ request }) => {
    // Create a service
    const createResp = await adminApi(request, 'POST', `${BASE_URL}/admin/api/v1/services`, {
      name: 'e2e-test-service',
      protocol: 'http',
      upstream: 'test-upstream',
    });
    expect(createResp.ok()).toBeTruthy();
    const created = await createResp.json();
    const serviceId = created.id ?? created.service?.id;

    // List services
    const listResp = await adminApi(request, 'GET', `${BASE_URL}/admin/api/v1/services`);
    expect(listResp.ok()).toBeTruthy();
    const list = await listResp.json();
    expect(Array.isArray(list.items ?? list.services ?? list)).toBeTruthy();

    // Get service by ID
    if (serviceId) {
      const getResp = await adminApi(request, 'GET', `${BASE_URL}/admin/api/v1/services/${serviceId}`);
      expect(getResp.ok()).toBeTruthy();
    }

    // Delete service
    if (serviceId) {
      const delResp = await adminApi(request, 'DELETE', `${BASE_URL}/admin/api/v1/services/${serviceId}`);
      expect(delResp.ok()).toBeTruthy();
    }
  });
});

test.describe('Upstreams CRUD', () => {
  test('create, list, delete an upstream', async ({ request }) => {
    // Create an upstream
    const createResp = await adminApi(request, 'POST', `${BASE_URL}/admin/api/v1/upstreams`, {
      name: 'e2e-test-upstream',
      algorithm: 'round_robin',
      targets: [{ address: 'localhost:3000', weight: 1 }],
    });
    expect(createResp.ok()).toBeTruthy();
    const created = await createResp.json();
    const upstreamId = created.id ?? created.upstream?.id;

    // List upstreams
    const listResp = await adminApi(request, 'GET', `${BASE_URL}/admin/api/v1/upstreams`);
    expect(listResp.ok()).toBeTruthy();

    // Delete upstream
    if (upstreamId) {
      const delResp = await adminApi(request, 'DELETE', `${BASE_URL}/admin/api/v1/upstreams/${upstreamId}`);
      expect(delResp.ok()).toBeTruthy();
    }
  });
});

test.describe('Routes CRUD', () => {
  test('create, list, delete a route', async ({ request }) => {
    // Create a route
    const createResp = await adminApi(request, 'POST', `${BASE_URL}/admin/api/v1/routes`, {
      name: 'e2e-test-route',
      service_name: 'test-service',
      path: '/api/test/*',
      methods: ['GET', 'POST'],
    });
    expect(createResp.ok()).toBeTruthy();
    const created = await createResp.json();
    const routeId = created.id ?? created.route?.id;

    // List routes
    const listResp = await adminApi(request, 'GET', `${BASE_URL}/admin/api/v1/routes`);
    expect(listResp.ok()).toBeTruthy();

    // Delete route
    if (routeId) {
      const delResp = await adminApi(request, 'DELETE', `${BASE_URL}/admin/api/v1/routes/${routeId}`);
      expect(delResp.ok()).toBeTruthy();
    }
  });
});

test.describe('Users CRUD', () => {
  test('create, list, get, delete a user', async ({ request }) => {
    // Create a user
    const createResp = await adminApi(request, 'POST', `${BASE_URL}/admin/api/v1/users`, {
      email: 'e2e-test@example.com',
      name: 'E2E Test User',
    });
    expect(createResp.ok()).toBeTruthy();
    const created = await createResp.json();
    const userId = created.id ?? created.user?.id;

    // List users
    const listResp = await adminApi(request, 'GET', `${BASE_URL}/admin/api/v1/users`);
    expect(listResp.ok()).toBeTruthy();

    // Get user by ID
    if (userId) {
      const getResp = await adminApi(request, 'GET', `${BASE_URL}/admin/api/v1/users/${userId}`);
      expect(getResp.ok()).toBeTruthy();
    }

    // Delete user
    if (userId) {
      const delResp = await adminApi(request, 'DELETE', `${BASE_URL}/admin/api/v1/users/${userId}`);
      expect(delResp.ok()).toBeTruthy();
    }
  });
});
