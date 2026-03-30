export const queryKeys = {
  services: ["services"] as const,
  service: (id: string) => ["services", id] as const,

  routes: ["routes"] as const,
  route: (id: string) => ["routes", id] as const,

  upstreams: ["upstreams"] as const,
  upstream: (id: string) => ["upstreams", id] as const,
  upstreamHealth: (id: string) => ["upstreams", id, "health"] as const,

  users: (params?: Record<string, unknown>) => ["users", params ?? {}] as const,
  user: (id: string) => ["users", id] as const,

  creditsOverview: ["credits", "overview"] as const,
  creditsBalance: (userID: string) => ["credits", "balance", userID] as const,
  creditsTransactions: (userID: string, params?: Record<string, unknown>) =>
    ["credits", "transactions", userID, params ?? {}] as const,

  auditLogs: (params?: Record<string, unknown>) => ["audit", "logs", params ?? {}] as const,
  auditLog: (id: string) => ["audit", "log", id] as const,
  auditStats: (params?: Record<string, unknown>) => ["audit", "stats", params ?? {}] as const,
  auditExport: (params?: Record<string, unknown>) => ["audit", "export", params ?? {}] as const,

  analyticsOverview: (params?: Record<string, unknown>) => ["analytics", "overview", params ?? {}] as const,
  analyticsTimeseries: (params?: Record<string, unknown>) => ["analytics", "timeseries", params ?? {}] as const,
  analyticsTopRoutes: (params?: Record<string, unknown>) => ["analytics", "top-routes", params ?? {}] as const,
} as const;

