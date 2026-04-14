import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  adminApiRequest,
  ApiError,
  isAdminAuthenticated,
  setAdminAuthenticated,
} from "./api";

// Mock global fetch
const mockFetch = vi.fn();
vi.stubGlobal("fetch", mockFetch);

// Mock sessionStorage
const sessionStore: Record<string, string> = {};
vi.stubGlobal(
  "sessionStorage",
  {
    getItem: vi.fn((key: string) => sessionStore[key] ?? null),
    setItem: vi.fn((key: string, value: string) => {
      sessionStore[key] = value;
    }),
    removeItem: vi.fn((key: string) => {
      delete sessionStore[key];
    }),
    clear: vi.fn(() => Object.keys(sessionStore).forEach((k) => delete sessionStore[k])),
  },
);

describe("ApiError", () => {
  it("constructs with status, code, and payload", () => {
    const err = new ApiError("test error", 400, "bad_request", { detail: "x" });
    expect(err.message).toBe("test error");
    expect(err.status).toBe(400);
    expect(err.code).toBe("bad_request");
    expect(err.payload).toEqual({ detail: "x" });
    expect(err.name).toBe("ApiError");
    expect(err).toBeInstanceOf(Error);
  });
});

describe("adminApiRequest", () => {
  beforeEach(() => {
    mockFetch.mockReset();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("makes a GET request and returns parsed JSON", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      text: async () => JSON.stringify({ id: "1", name: "test" }),
    });

    const result = await adminApiRequest<{ id: string }>("/admin/api/v1/services");
    expect(result).toEqual({ id: "1", name: "test" });
    expect(mockFetch).toHaveBeenCalledTimes(1);

    const [url, init] = mockFetch.mock.calls[0];
    expect(url).toContain("/admin/api/v1/services");
    expect(init.method).toBe("GET");
  });

  it("makes a POST request with JSON body", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      text: async () => JSON.stringify({ id: "2" }),
    });

    await adminApiRequest("/admin/api/v1/services", {
      method: "POST",
      body: { name: "new-service" },
    });

    const [, init] = mockFetch.mock.calls[0];
    expect(init.method).toBe("POST");
    expect(init.body).toBe(JSON.stringify({ name: "new-service" }));
    expect(init.headers).toBeDefined();
  });

  it("throws ApiError on non-ok response with error object", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 400,
      statusText: "Bad Request",
      text: async () =>
        JSON.stringify({ error: { message: "name is required", code: "invalid_input" } }),
    });

    try {
      await adminApiRequest("/admin/api/v1/services", { method: "POST", body: {} });
      expect.unreachable("Should have thrown");
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError);
      expect((err as ApiError).status).toBe(400);
      expect((err as ApiError).code).toBe("invalid_input");
      expect((err as ApiError).message).toBe("name is required");
    }
  });

  it("uses statusText when error has no message", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
      statusText: "Internal Server Error",
      text: async () => JSON.stringify({}),
    });

    try {
      await adminApiRequest("/admin/api/v1/services");
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError);
      expect((err as ApiError).status).toBe(500);
    }
  });

  it("handles empty response body", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      text: async () => "",
    });

    const result = await adminApiRequest("/admin/api/v1/services/1", {
      method: "DELETE",
    });
    expect(result).toBeNull();
  });

  it("handles non-JSON response body", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      text: async () => "plain text",
    });

    const result = await adminApiRequest<string>("/admin/api/v1/test");
    expect(result).toBe("plain text");
  });

  it("appends query parameters", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      text: async () => "[]",
    });

    await adminApiRequest("/admin/api/v1/services", {
      query: { limit: 10, offset: 0, active: true },
    });

    const [url] = mockFetch.mock.calls[0];
    expect(url).toContain("limit=10");
    expect(url).toContain("offset=0");
    expect(url).toContain("active=true");
  });

  it("skips null/undefined/empty query params", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      text: async () => "[]",
    });

    await adminApiRequest("/admin/api/v1/services", {
      query: { limit: 10, empty: "", nil: null, undef: undefined },
    });

    const [url] = mockFetch.mock.calls[0];
    expect(url).toContain("limit=10");
    expect(url).not.toContain("empty=");
    expect(url).not.toContain("nil=");
    expect(url).not.toContain("undef=");
  });

  it("wraps AbortError as timeout ApiError", async () => {
    mockFetch.mockImplementationOnce(() =>
      Promise.reject(new DOMException("The operation was aborted", "AbortError")),
    );

    try {
      await adminApiRequest("/admin/api/v1/services", { timeoutMs: 50 });
      expect.unreachable("Should have thrown");
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError);
      expect((err as ApiError).message).toBe("Request timed out");
      expect((err as ApiError).code).toBe("request_timeout");
    }
  });

  it("throws network error on fetch failure", async () => {
    mockFetch.mockRejectedValueOnce(new TypeError("Failed to fetch"));

    await expect(adminApiRequest("/admin/api/v1/services")).rejects.toThrow(
      "Network request failed",
    );
  });
});

describe("isAdminAuthenticated / setAdminAuthenticated", () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it("returns false when not authenticated", () => {
    expect(isAdminAuthenticated()).toBe(false);
  });

  it("returns true after setting authenticated", () => {
    setAdminAuthenticated(true);
    expect(isAdminAuthenticated()).toBe(true);
  });

  it("returns false after clearing authentication", () => {
    setAdminAuthenticated(true);
    expect(isAdminAuthenticated()).toBe(true);
    setAdminAuthenticated(false);
    expect(isAdminAuthenticated()).toBe(false);
  });
});
