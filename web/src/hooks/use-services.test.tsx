import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement, type ReactNode } from "react";
import { useServices, useService, useCreateService, useDeleteService } from "./use-services";
import { adminApiRequest } from "@/lib/api";

// Mock the API module
vi.mock("@/lib/api", () => ({
  adminApiRequest: vi.fn(),
}));

const mockApi = vi.mocked(adminApiRequest);

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(
      QueryClientProvider,
      { client: queryClient },
      children,
    );
  };
}

const mockServices = [
  { id: "svc-1", name: "API Service", protocol: "http", upstream: "up-1" },
  { id: "svc-2", name: "Auth Service", protocol: "grpc", upstream: "up-2" },
];

describe("useServices", () => {
  beforeEach(() => {
    mockApi.mockReset();
  });

  it("fetches and returns services list", async () => {
    mockApi.mockResolvedValueOnce(mockServices);

    const { result } = renderHook(() => useServices(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data).toEqual(mockServices);
    expect(mockApi).toHaveBeenCalledWith("/admin/api/v1/services");
  });

  it("handles fetch error", async () => {
    mockApi.mockRejectedValueOnce(new Error("Network error"));

    const { result } = renderHook(() => useServices(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeDefined();
  });
});

describe("useService", () => {
  beforeEach(() => {
    mockApi.mockReset();
  });

  it("fetches a single service by id", async () => {
    mockApi.mockResolvedValueOnce(mockServices[0]);

    const { result } = renderHook(() => useService("svc-1"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(mockServices[0]);
    expect(mockApi).toHaveBeenCalledWith("/admin/api/v1/services/svc-1");
  });

  it("does not fetch when id is empty", () => {
    const { result } = renderHook(() => useService(""), {
      wrapper: createWrapper(),
    });

    expect(result.current.fetchStatus).toBe("idle");
    expect(mockApi).not.toHaveBeenCalled();
  });
});

describe("useCreateService", () => {
  beforeEach(() => {
    mockApi.mockReset();
  });

  it("posts a new service and returns it", async () => {
    const newService = { name: "New Service", protocol: "http", upstream: "up-1" };
    const created = { id: "svc-3", ...newService };
    mockApi.mockResolvedValueOnce(created);

    const { result } = renderHook(() => useCreateService(), {
      wrapper: createWrapper(),
    });

    result.current.mutate(newService);

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(created);
    expect(mockApi).toHaveBeenCalledWith("/admin/api/v1/services", {
      method: "POST",
      body: newService,
    });
  });
});

describe("useDeleteService", () => {
  beforeEach(() => {
    mockApi.mockReset();
  });

  it("deletes a service by id", async () => {
    mockApi.mockResolvedValueOnce(null);

    const { result } = renderHook(() => useDeleteService(), {
      wrapper: createWrapper(),
    });

    result.current.mutate("svc-1");

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockApi).toHaveBeenCalledWith("/admin/api/v1/services/svc-1", {
      method: "DELETE",
    });
  });
});
