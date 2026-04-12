export type TimeWindow = "15m" | "1h" | "6h" | "24h" | "7d" | "all";

export const TIME_WINDOW_OPTIONS: Array<{ label: string; value: TimeWindow }> = [
  { label: "15m", value: "15m" },
  { label: "1h", value: "1h" },
  { label: "6h", value: "6h" },
  { label: "24h", value: "24h" },
  { label: "7d", value: "7d" },
  { label: "All", value: "all" },
];

const WINDOW_MS: Record<Exclude<TimeWindow, "all">, number> = {
  "15m": 15 * 60 * 1000,
  "1h": 60 * 60 * 1000,
  "6h": 6 * 60 * 60 * 1000,
  "24h": 24 * 60 * 60 * 1000,
  "7d": 7 * 24 * 60 * 60 * 1000,
};

export function filterDataByWindow<T>(
  data: T[],
  window: TimeWindow,
  getTimestamp: (item: T) => string | number | Date,
): T[] {
  if (window === "all") {
    return data;
  }
  const now = Date.now();
  const cutoff = now - WINDOW_MS[window];
  return data.filter((item) => {
    const timestamp = new Date(getTimestamp(item)).getTime();
    return Number.isFinite(timestamp) && timestamp >= cutoff;
  });
}

export function formatTimeTick(value: string | number | Date) {
  return new Date(value).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

export function formatDateTick(value: string | number | Date) {
  return new Date(value).toLocaleDateString([], { month: "short", day: "numeric" });
}

