import { type APIRequestContext } from '@playwright/test';

export const ADMIN_API_KEY = 'e2e-test-admin-key-must-be-at-least-32-chars-long';
export const BASE_URL = 'http://127.0.0.1:9876';
export const GW_URL = 'http://127.0.0.1:8080';

// Cached JWT token — reset via resetAdminToken()
let _adminToken: string | null = null;

export async function getAdminToken(request: APIRequestContext): Promise<string> {
  if (_adminToken) return _adminToken;
  return login(request);
}

async function login(request: APIRequestContext): Promise<string> {
  // POST form login to get JWT in Set-Cookie header
  const loginResp = await request.post(`${BASE_URL}/admin/login`, {
    form: { admin_key: ADMIN_API_KEY },
    maxRedirects: 0,
  });

  // Extract token from Set-Cookie
  const setCookie = loginResp.headers()['set-cookie'] ?? '';
  const match = setCookie.match(/apicerberus_admin_session=([^;]+)/);
  if (!match) {
    _adminToken = null;
    throw new Error(
      `Login failed. Status: ${loginResp.status()}, ` +
      `Location: ${loginResp.headers()['location'] ?? 'none'}`,
    );
  }

  _adminToken = match[1];
  return _adminToken;
}

export function resetAdminToken(): void {
  _adminToken = null;
}

export async function adminGet(request: APIRequestContext, path: string) {
  const token = await getAdminToken(request);
  const resp = await request.get(`${BASE_URL}${path}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  // If unauthorized, reset token and retry once
  if (resp.status() === 401) {
    resetAdminToken();
    const newToken = await getAdminToken(request);
    return request.get(`${BASE_URL}${path}`, {
      headers: { Authorization: `Bearer ${newToken}` },
    });
  }
  return resp;
}

export async function adminPost(request: APIRequestContext, path: string, body?: object) {
  const token = await getAdminToken(request);
  const opts: Record<string, unknown> = {
    headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
  };
  if (body) opts.data = body;
  const resp = await request.post(`${BASE_URL}${path}`, opts);
  if (resp.status() === 401) {
    resetAdminToken();
    const newToken = await getAdminToken(request);
    const retryOpts: Record<string, unknown> = {
      headers: { Authorization: `Bearer ${newToken}`, 'Content-Type': 'application/json' },
    };
    if (body) retryOpts.data = body;
    return request.post(`${BASE_URL}${path}`, retryOpts);
  }
  return resp;
}

export async function adminPut(request: APIRequestContext, path: string, body?: object) {
  const token = await getAdminToken(request);
  const opts: Record<string, unknown> = {
    headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
  };
  if (body) opts.data = body;
  return request.put(`${BASE_URL}${path}`, opts);
}

export async function adminDelete(request: APIRequestContext, path: string) {
  const token = await getAdminToken(request);
  return request.delete(`${BASE_URL}${path}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
}
