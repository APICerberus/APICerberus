import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ErrorState } from "./ErrorState";

describe("ErrorState", () => {
  it("renders error message with default title", () => {
    render(<ErrorState message="Failed to load services" />);
    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
    expect(screen.getByText("Failed to load services")).toBeInTheDocument();
  });

  it("renders custom title", () => {
    render(<ErrorState title="Not Found" message="Service does not exist" />);
    expect(screen.getByText("Not Found")).toBeInTheDocument();
    expect(screen.getByText("Service does not exist")).toBeInTheDocument();
  });

  it("renders retry button when onRetry is provided", () => {
    const onRetry = vi.fn();
    render(<ErrorState message="Error" onRetry={onRetry} />);
    expect(screen.getByText("Retry")).toBeInTheDocument();
  });

  it("does not render retry button when onRetry is omitted", () => {
    render(<ErrorState message="Error" />);
    expect(screen.queryByText("Retry")).not.toBeInTheDocument();
  });

  it("calls onRetry when retry button is clicked", async () => {
    const onRetry = vi.fn();
    render(<ErrorState message="Error" onRetry={onRetry} />);

    await userEvent.click(screen.getByText("Retry"));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it("renders custom retry label", () => {
    const onRetry = vi.fn();
    render(<ErrorState message="Error" onRetry={onRetry} retryLabel="Try Again" />);
    expect(screen.getByText("Try Again")).toBeInTheDocument();
  });
});
