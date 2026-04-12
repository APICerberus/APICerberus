import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApiRequest } from "@/lib/api";
import type { User, UserListResponse } from "@/lib/types";
import { normalizeUserListResponse, toQueryRecord } from "./helpers";
import { queryKeys } from "./query-keys";

export type UserListParams = {
  search?: string;
  status?: string;
  role?: string;
  sort_by?: string;
  sort_desc?: boolean;
  limit?: number;
  offset?: number;
};

export function useUsers(params: UserListParams = {}) {
  const normalizedParams = toQueryRecord(params);

  return useQuery({
    queryKey: queryKeys.users(normalizedParams),
    queryFn: async () => {
      const payload = await adminApiRequest<unknown>("/admin/api/v1/users", {
        query: normalizedParams,
      });
      return normalizeUserListResponse(payload) as UserListResponse;
    },
  });
}

export function useUser(id: string) {
  return useQuery({
    queryKey: queryKeys.user(id),
    queryFn: () => adminApiRequest<User>(`/admin/api/v1/users/${id}`),
    enabled: Boolean(id),
  });
}

export function useCreateUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: Record<string, unknown>) =>
      adminApiRequest<User>("/admin/api/v1/users", {
        method: "POST",
        body: payload,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.users() });
    },
  });
}

export function useUpdateUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: Record<string, unknown> }) =>
      adminApiRequest<User>(`/admin/api/v1/users/${id}`, {
        method: "PUT",
        body: payload,
      }),
    onSuccess: async (_data, variables) => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.users() });
      await queryClient.invalidateQueries({ queryKey: queryKeys.user(variables.id) });
    },
  });
}

export function useDeleteUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      adminApiRequest<null>(`/admin/api/v1/users/${id}`, {
        method: "DELETE",
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.users() });
    },
  });
}

export function useSuspendUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      adminApiRequest<User>(`/admin/api/v1/users/${id}/suspend`, {
        method: "POST",
        body: {},
      }),
    onSuccess: async (_data, id) => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.users() });
      await queryClient.invalidateQueries({ queryKey: queryKeys.user(id) });
    },
  });
}

export function useActivateUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      adminApiRequest<User>(`/admin/api/v1/users/${id}/activate`, {
        method: "POST",
        body: {},
      }),
    onSuccess: async (_data, id) => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.users() });
      await queryClient.invalidateQueries({ queryKey: queryKeys.user(id) });
    },
  });
}

