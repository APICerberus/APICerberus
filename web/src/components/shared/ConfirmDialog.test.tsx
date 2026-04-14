import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ConfirmDialog } from "./ConfirmDialog";

describe("ConfirmDialog", () => {
  it("renders when open", () => {
    render(
      <ConfirmDialog
        open={true}
        onOpenChange={vi.fn()}
        title="Delete Service?"
        description="This action cannot be undone."
        onConfirm={vi.fn()}
      />,
    );

    expect(screen.getByText("Delete Service?")).toBeInTheDocument();
    expect(screen.getByText("This action cannot be undone.")).toBeInTheDocument();
  });

  it("does not render content when closed", () => {
    render(
      <ConfirmDialog
        open={false}
        onOpenChange={vi.fn()}
        title="Delete Service?"
        description="This action cannot be undone."
        onConfirm={vi.fn()}
      />,
    );

    expect(screen.queryByText("Delete Service?")).not.toBeInTheDocument();
  });

  it("renders confirm and cancel buttons", () => {
    render(
      <ConfirmDialog
        open={true}
        onOpenChange={vi.fn()}
        title="Delete Item?"
        description="Are you sure?"
        onConfirm={vi.fn()}
      />,
    );

    // Both buttons should be present
    expect(screen.getByRole("button", { name: "Confirm" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
  });

  it("renders custom labels", () => {
    render(
      <ConfirmDialog
        open={true}
        onOpenChange={vi.fn()}
        title="Delete?"
        description="Sure?"
        confirmLabel="Yes, Delete"
        cancelLabel="No, Keep"
        onConfirm={vi.fn()}
      />,
    );

    expect(screen.getByText("Yes, Delete")).toBeInTheDocument();
    expect(screen.getByText("No, Keep")).toBeInTheDocument();
  });
});
