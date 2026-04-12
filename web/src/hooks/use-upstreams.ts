import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApiRequest } from "@/lib/api";
import type { Upstream, UpstreamTarget } from "@/lib/types";
import { queryKeys } from "./query-keys";

export function useUpstreams() {
  return useQuery({
    queryKey: queryKeys.upstreams,
    queryFn: () => adminApiRequest<Upstream[]>("/admin/api/v1/upstreams"),
  });
}

export function useUpstream(id: string) {
  return useQuery({
    queryKey: queryKeys.upstream(id),
    queryFn: () => adminApiRequest<Upstream>(`/admin/api/v1/upstreams/${id}`),
    enabled: Boolean(id),
  });
}

export function useUpstreamHealth(id: string) {
  return useQuery({
    queryKey: queryKeys.upstreamHealth(id),
    queryFn: () => adminApiRequest<Record<string, unknown>>(`/admin/api/v1/upstreams/${id}/health`),
    enabled: Boolean(id),
  });
}

export function useCreateUpstream() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: Partial<Upstream>) =>
      adminApiRequest<Upstream>("/admin/api/v1/upstreams", {
        method: "POST",
        body: payload,
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.upstreams });
    },
  });
}

export function useUpdateUpstream() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: Partial<Upstream> }) =>
      adminApiRequest<Upstream>(`/admin/api/v1/upstreams/${id}`, {
        method: "PUT",
        body: payload,
      }),
    onSuccess: async (_data, variables) => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.upstreams });
      await queryClient.invalidateQueries({ queryKey: queryKeys.upstream(variables.id) });
      await queryClient.invalidateQueries({ queryKey: queryKeys.upstreamHealth(variables.id) });
    },
  });
}

export function useDeleteUpstream() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) =>
      adminApiRequest<null>(`/admin/api/v1/upstreams/${id}`, {
        method: "DELETE",
      }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.upstreams });
    },
  });
}

export function useAddUpstreamTarget() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: Partial<UpstreamTarget> }) =>
      adminApiRequest<UpstreamTarget>(`/admin/api/v1/upstreams/${id}/targets`, {
        method: "POST",
        body: payload,
      }),
    onSuccess: async (_data, variables) => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.upstream(variables.id) });
      await queryClient.invalidateQueries({ queryKey: queryKeys.upstreamHealth(variables.id) });
    },
  });
}

export function useDeleteUpstreamTarget() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, targetId }: { id: string; targetId: string }) =>
      adminApiRequest<null>(`/admin/api/v1/upstreams/${id}/targets/${targetId}`, {
        method: "DELETE",
      }),
    onSuccess: async (_data, variables) => {
      await queryClient.invalidateQueries({ queryKey: queryKeys.upstream(variables.id) });
      await queryClient.invalidateQueries({ queryKey: queryKeys.upstreamHealth(variables.id) });
    },
  });
}

