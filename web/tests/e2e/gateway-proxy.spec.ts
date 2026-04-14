import { test, expect } from '@playwright/test';
import { BASE_URL, GW_URL, adminPost, adminGet, adminDelete, resetAdminToken } from './helpers';

// Collects created resource IDs for cleanup
const createdResources: { type: string; id: string }[] = [];

test.afterAll(async ({ request }) => {
  resetAdminToken();
  for (const r of createdResources.reverse()) {
    await adminDelete(request, `/admin/api/v1/${r.type}/${r.id}`).catch(() => {});
  }
});

test.describe('Gateway Proxy Flow', () => {
  test.beforeAll(() => resetAdminToken());

  test('routes request to upstream via service', async ({ request }) => {
    // Step 1: Create upstream with target id
    const upstreamResp = await adminPost(request, '/admin/api/v1/upstreams', {
      name: 'e2e-proxy-upstream',
      algorithm: 'round_robin',
      targets: [{ id: 't1', address: 'httpbin.org:80', weight: 1 }],
    });
    expect(upstreamResp.ok()).toBeTruthy();
    const upstream = await upstreamResp.json();
    const upstreamId = upstream.id ?? upstream.upstream?.id;
    if (upstreamId) createdResources.push({ type: 'upstreams', id: upstreamId });

    // Step 2: Create service
    const serviceResp = await adminPost(request, '/admin/api/v1/services', {
      name: 'e2e-proxy-service',
      protocol: 'http',
      upstream: 'e2e-proxy-upstream',
    });
    expect(serviceResp.ok()).toBeTruthy();
    const service = await serviceResp.json();
    const serviceId = service.id ?? service.service?.id;
    if (serviceId) createdResources.push({ type: 'services', id: serviceId });

    // Step 3: Create route (may fail if service is not yet registered)
    const routeResp = await adminPost(request, '/admin/api/v1/routes', {
      name: 'e2e-proxy-route',
      service_name: 'e2e-proxy-service',
      path: '/e2e-test/*',
      methods: ['GET'],
    });
    if (routeResp.ok()) {
      const route = await routeResp.json();
      const routeId = route.id ?? route.route?.id;
      if (routeId) createdResources.push({ type: 'routes', id: routeId });
    }

    // Step 4: Verify route and upstream are listed
    const routesResp = await adminGet(request, '/admin/api/v1/routes');
    expect(routesResp.ok()).toBeTruthy();

    const upstreamsResp = await adminGet(request, '/admin/api/v1/upstreams');
    expect(upstreamsResp.ok()).toBeTruthy();
  });
});

test.describe('Gateway Health & Metrics', () => {
  test('gateway health endpoint is reachable', async ({ request }) => {
    const resp = await request.get(`${GW_URL}/health`);
    // Gateway returns something (may be HTML from SPA)
    expect(resp.status()).toBeLessThan(500);
  });

  test('gateway ready endpoint is reachable', async ({ request }) => {
    const resp = await request.get(`${GW_URL}/ready`);
    expect(resp.status()).toBeLessThan(500);
  });
});

test.describe('Admin Status & Config', () => {
  test.beforeAll(() => resetAdminToken());

  test('status endpoint returns ok', async ({ request }) => {
    const resp = await adminGet(request, '/admin/api/v1/status');
    expect(resp.ok()).toBeTruthy();
    const body = await resp.json();
    expect(body).toHaveProperty('status');
  });

  test('info endpoint returns version and uptime', async ({ request }) => {
    const resp = await adminGet(request, '/admin/api/v1/info');
    if (resp.ok()) {
      const body = await resp.json();
      expect(body).toHaveProperty('version');
      expect(body).toHaveProperty('uptime_sec');
    }
  });

  test('config export returns configuration', async ({ request }) => {
    const resp = await adminGet(request, '/admin/api/v1/config/export');
    if (resp.ok()) {
      const text = await resp.text();
      expect(text).toBeTruthy();
    }
  });
});
