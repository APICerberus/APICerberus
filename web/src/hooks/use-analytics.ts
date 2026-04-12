import { useQuery } from "@tanstack/react-query";
import { adminApiRequest } from "@/lib/api";
import type { AnalyticsOverview, AnalyticsTimeseries, AnalyticsTopRoutes } from "@/lib/types";
import { toQueryRecord } from "./helpers";
import { queryKeys } from "./query-keys";

export type AnalyticsRangeParams = {
  window?: string;
  from?: string;
  to?: string;
};

export type AnalyticsTimeseriesParams = AnalyticsRangeParams & {
  granularity?: string;
};

export type AnalyticsTopRoutesParams = AnalyticsRangeParams & {
  limit?: number;
};

export function useAnalyticsOverview(params: AnalyticsRangeParams = {}) {
  const normalizedParams = toQueryRecord(params);
  return useQuery({
    queryKey: queryKeys.analyticsOverview(normalizedParams),
    queryFn: () =>
      adminApiRequest<AnalyticsOverview>("/admin/api/v1/analytics/overview", {
        query: normalizedParams,
      }),
  });
}

export function useAnalyticsTimeseries(params: AnalyticsTimeseriesParams = {}) {
  const normalizedParams = toQueryRecord(params);
  return useQuery({
    queryKey: queryKeys.analyticsTimeseries(normalizedParams),
    queryFn: () =>
      adminApiRequest<AnalyticsTimeseries>("/admin/api/v1/analytics/timeseries", {
        query: normalizedParams,
      }),
  });
}

export function useAnalyticsTopRoutes(params: AnalyticsTopRoutesParams = {}) {
  const normalizedParams = toQueryRecord(params);
  return useQuery({
    queryKey: queryKeys.analyticsTopRoutes(normalizedParams),
    queryFn: () =>
      adminApiRequest<AnalyticsTopRoutes>("/admin/api/v1/analytics/top-routes", {
        query: normalizedParams,
      }),
  });
}

