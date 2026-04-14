import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, act } from "@testing-library/react";
import { TimeAgo } from "./TimeAgo";

describe("TimeAgo", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders relative time for a recent date", () => {
    const now = new Date("2026-04-14T12:00:00Z");
    vi.setSystemTime(now);

    const fiveMinutesAgo = new Date("2026-04-14T11:55:00Z");
    render(<TimeAgo value={fiveMinutesAgo} />);

    expect(screen.getByText(/5 minutes? ago/)).toBeInTheDocument();
  });

  it("accepts ISO string input", () => {
    const now = new Date("2026-04-14T12:00:00Z");
    vi.setSystemTime(now);

    render(<TimeAgo value="2026-04-14T11:30:00Z" />);
    expect(screen.getByText(/30 minutes? ago/)).toBeInTheDocument();
  });

  it("accepts timestamp number input", () => {
    const now = new Date("2026-04-14T12:00:00Z");
    vi.setSystemTime(now);

    const oneHourAgo = new Date("2026-04-14T11:00:00Z").getTime();
    render(<TimeAgo value={oneHourAgo} />);
    expect(screen.getByText(/1 hour ago/)).toBeInTheDocument();
  });

  it("sets the datetime attribute on the time element", () => {
    const now = new Date("2026-04-14T12:00:00Z");
    vi.setSystemTime(now);

    const date = new Date("2026-04-14T10:00:00Z");
    render(<TimeAgo value={date} />);

    const timeEl = document.querySelector("time");
    expect(timeEl).not.toBeNull();
    expect(timeEl!.getAttribute("datetime")).toBe(date.toISOString());
  });

  it("sets the title attribute with formatted date", () => {
    const now = new Date("2026-04-14T12:00:00Z");
    vi.setSystemTime(now);

    render(<TimeAgo value={new Date("2026-04-14T10:00:00Z")} />);
    const timeEl = document.querySelector("time");
    expect(timeEl).not.toBeNull();
    expect(timeEl!.getAttribute("title")).toBeTruthy();
  });

  it("applies custom className", () => {
    render(<TimeAgo value={new Date()} className="text-muted" />);
    const timeEl = screen.getByText(/ago/);
    expect(timeEl.classList.contains("text-muted")).toBe(true);
  });

  it("updates the display on interval tick", () => {
    const now = new Date("2026-04-14T12:00:00Z");
    vi.setSystemTime(now);

    const startTime = new Date("2026-04-14T11:59:00Z");
    render(<TimeAgo value={startTime} />);

    // Initially shows "1 minute ago"
    expect(screen.getByText(/1 minute ago/)).toBeInTheDocument();

    // Advance 60 seconds
    act(() => {
      vi.advanceTimersByTime(60_000);
    });

    // Should now show "2 minutes ago"
    expect(screen.getByText(/2 minutes ago/)).toBeInTheDocument();
  });
});
