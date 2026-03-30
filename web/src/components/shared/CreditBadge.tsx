import { Coins } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

type CreditBadgeProps = {
  value: number;
  kind?: "cost" | "balance";
  className?: string;
};

function formatCredits(value: number) {
  return new Intl.NumberFormat("en-US", { maximumFractionDigits: 2 }).format(value);
}

export function CreditBadge({ value, kind = "cost", className }: CreditBadgeProps) {
  return (
    <Badge
      variant="outline"
      className={cn(
        "gap-1 border px-2 py-0.5 font-medium tabular-nums",
        kind === "balance"
          ? "border-success/30 bg-success/15 text-success"
          : "border-warning/30 bg-warning/15 text-warning",
        className,
      )}
    >
      <Coins className="size-3.5" />
      {kind === "balance" ? "Balance" : "Cost"}: {formatCredits(value)}
    </Badge>
  );
}

