import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApiRequest } from "@/lib/api";
import type { Service } from "@/lib/types";
import { queryKeys } from "./query-keys";

export function useServices() {
  return useQuery({
    queryKey: queryKeys.services,
    queryFn: () => adminApiRequest<Service[]>("/admin/api/v1/services"),
  });
}

export function useService(id: string) {
  return useQuery({
    queryKey: queryKeys.service(id),
    queryFn: () => adminApiRequest<Service>(`/admin/api/v1/services/${id}`),
    enabled: Boolean(id),
  });
}

export function useCreateService() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: Partial<Service>) =>
      adminApiRequest<Service>("/admin/api/v1/services", {
        method: "POST",
        body: payload,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.services });
    },
  });
}

export function useUpdateService() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: Partial<Service> }) =>
      adminApiRequest<Service>(`/admin/api/v1/services/${id}`, {
        method: "PUT",
        body: payload,
      }),
    onSuccess: async (_data, variables) => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.services });
      await queryClient.invalidateQueries({ queryKey: queryKeys.service(variables.id) });
    },
  });
}

export function useDeleteService() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      adminApiRequest<null>(`/admin/api/v1/services/${id}`, {
        method: "DELETE",
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.services });
    },
  });
}

