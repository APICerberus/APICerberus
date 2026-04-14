import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { CreditBadge } from "./CreditBadge";

describe("CreditBadge", () => {
  it("renders cost badge by default", () => {
    render(<CreditBadge value={42} />);
    expect(screen.getByText(/Cost.*42/)).toBeInTheDocument();
  });

  it("renders balance badge when kind=balance", () => {
    render(<CreditBadge value={100} kind="balance" />);
    expect(screen.getByText(/Balance.*100/)).toBeInTheDocument();
  });

  it("formats large numbers with commas", () => {
    render(<CreditBadge value={1234567} />);
    expect(screen.getByText(/Cost.*1,234,567/)).toBeInTheDocument();
  });

  it("formats fractional values", () => {
    render(<CreditBadge value={3.14159} />);
    expect(screen.getByText(/Cost.*3.14/)).toBeInTheDocument();
  });

  it("renders zero value", () => {
    render(<CreditBadge value={0} />);
    expect(screen.getByText(/Cost.*0/)).toBeInTheDocument();
  });

  it("applies custom className", () => {
    const { container } = render(<CreditBadge value={10} className="test-class" />);
    expect(container.firstChild).toHaveClass("test-class");
  });
});
