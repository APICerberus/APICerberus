import { useQuery } from "@tanstack/react-query";
import { adminApiRequest } from "@/lib/api";
import type { AuditEntry, AuditListResponse } from "@/lib/types";
import { toQueryRecord } from "./helpers";
import { queryKeys } from "./query-keys";

export type AuditLogSearchParams = {
  user_id?: string;
  api_key_prefix?: string;
  route?: string;
  method?: string;
  status_min?: number;
  status_max?: number;
  client_ip?: string;
  blocked?: boolean;
  block_reason?: string;
  date_from?: string;
  date_to?: string;
  min_latency_ms?: number;
  q?: string;
  limit?: number;
  offset?: number;
};

export function useAuditLogs(params: AuditLogSearchParams = {}) {
  const normalizedParams = toQueryRecord(params);
  return useQuery({
    queryKey: queryKeys.auditLogs(normalizedParams),
    queryFn: () =>
      adminApiRequest<AuditListResponse>("/admin/api/v1/audit-logs", {
        query: normalizedParams,
      }),
  });
}

export function useAuditLog(id: string) {
  return useQuery({
    queryKey: queryKeys.auditLog(id),
    queryFn: () => adminApiRequest<AuditEntry>(`/admin/api/v1/audit-logs/${id}`),
    enabled: Boolean(id),
  });
}

export function useAuditLogStats(params: Omit<AuditLogSearchParams, "limit" | "offset"> = {}) {
  const normalizedParams = toQueryRecord(params);
  return useQuery({
    queryKey: queryKeys.auditStats(normalizedParams),
    queryFn: () =>
      adminApiRequest<Record<string, unknown>>("/admin/api/v1/audit-logs/stats", {
        query: normalizedParams,
      }),
  });
}

export function useAuditLogExport(
  params: AuditLogSearchParams & { format?: "jsonl" | "csv" } = { format: "jsonl" },
) {
  const normalizedParams = toQueryRecord(params);
  return useQuery({
    queryKey: queryKeys.auditExport(normalizedParams),
    queryFn: () =>
      adminApiRequest<string>("/admin/api/v1/audit-logs/export", {
        query: normalizedParams,
      }),
  });
}

