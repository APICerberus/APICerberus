import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApiRequest } from "@/lib/api";
import type { Route } from "@/lib/types";
import { queryKeys } from "./query-keys";

export function useRoutes() {
  return useQuery({
    queryKey: queryKeys.routes,
    queryFn: () => adminApiRequest<Route[]>("/admin/api/v1/routes"),
  });
}

export function useRoute(id: string) {
  return useQuery({
    queryKey: queryKeys.route(id),
    queryFn: () => adminApiRequest<Route>(`/admin/api/v1/routes/${id}`),
    enabled: Boolean(id),
  });
}

export function useCreateRoute() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: Partial<Route>) =>
      adminApiRequest<Route>("/admin/api/v1/routes", {
        method: "POST",
        body: payload,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.routes });
    },
  });
}

export function useUpdateRoute() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: Partial<Route> }) =>
      adminApiRequest<Route>(`/admin/api/v1/routes/${id}`, {
        method: "PUT",
        body: payload,
      }),
    onSuccess: async (_data, variables) => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.routes });
      await queryClient.invalidateQueries({ queryKey: queryKeys.route(variables.id) });
    },
  });
}

export function useDeleteRoute() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      adminApiRequest<null>(`/admin/api/v1/routes/${id}`, {
        method: "DELETE",
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.routes });
    },
  });
}

