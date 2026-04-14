import { describe, it, expect } from "vitest";
import { queryKeys } from "./query-keys";

describe("queryKeys", () => {
  it("has stable base keys", () => {
    expect(queryKeys.services).toEqual(["services"]);
    expect(queryKeys.routes).toEqual(["routes"]);
    expect(queryKeys.upstreams).toEqual(["upstreams"]);
  });

  it("generates entity-specific keys", () => {
    expect(queryKeys.service("svc-1")).toEqual(["services", "svc-1"]);
    expect(queryKeys.route("r-1")).toEqual(["routes", "r-1"]);
    expect(queryKeys.upstream("up-1")).toEqual(["upstreams", "up-1"]);
    expect(queryKeys.upstreamHealth("up-1")).toEqual(["upstreams", "up-1", "health"]);
  });

  it("generates user keys with optional params", () => {
    expect(queryKeys.users()).toEqual(["users", {}]);
    expect(queryKeys.users({ limit: 10 })).toEqual(["users", { limit: 10 }]);
    expect(queryKeys.user("u-1")).toEqual(["users", "u-1"]);
  });

  it("generates credit keys", () => {
    expect(queryKeys.creditsOverview).toEqual(["credits", "overview"]);
    expect(queryKeys.creditsBalance("u-1")).toEqual(["credits", "balance", "u-1"]);
    expect(queryKeys.creditsTransactions("u-1")).toEqual(["credits", "transactions", "u-1", {}]);
    expect(queryKeys.creditsTransactions("u-1", { limit: 5 })).toEqual([
      "credits",
      "transactions",
      "u-1",
      { limit: 5 },
    ]);
  });

  it("generates audit keys", () => {
    expect(queryKeys.auditLogs()).toEqual(["audit", "logs", {}]);
    expect(queryKeys.auditLog("a-1")).toEqual(["audit", "log", "a-1"]);
    expect(queryKeys.auditStats()).toEqual(["audit", "stats", {}]);
    expect(queryKeys.auditExport()).toEqual(["audit", "export", {}]);
  });

  it("generates analytics keys", () => {
    expect(queryKeys.analyticsOverview()).toEqual(["analytics", "overview", {}]);
    expect(queryKeys.analyticsTimeseries()).toEqual(["analytics", "timeseries", {}]);
    expect(queryKeys.analyticsTopRoutes()).toEqual(["analytics", "top-routes", {}]);
    expect(queryKeys.analyticsOverview({ period: "7d" })).toEqual([
      "analytics",
      "overview",
      { period: "7d" },
    ]);
  });

  it("produces unique keys for different params", () => {
    const keys = [
      queryKeys.auditLogs({ status: 200 }),
      queryKeys.auditLogs({ status: 500 }),
      queryKeys.auditLogs({ method: "GET" }),
    ];
    const serialized = keys.map((k) => JSON.stringify(k));
    expect(new Set(serialized).size).toBe(3);
  });
});
