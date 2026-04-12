import type { LucideIcon } from "lucide-react";
import { TrendingDown, TrendingUp } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { cn } from "@/lib/utils";

type KPICardProps = {
  label: string;
  value: string | number;
  icon: LucideIcon;
  trend?: number;
  description?: string;
  className?: string;
};

export function KPICard({ label, value, icon: Icon, trend, description, className }: KPICardProps) {
  const hasTrend = typeof trend === "number";
  const positive = hasTrend && trend >= 0;

  return (
    <Card className={className}>
      <CardHeader className="pb-2">
        <CardDescription>{label}</CardDescription>
        <div className="flex items-center justify-between">
          <CardTitle className="text-2xl font-semibold tabular-nums">{value}</CardTitle>
          <span className="rounded-md bg-primary/10 p-1.5 text-primary">
            <Icon className="size-4" />
          </span>
        </div>
      </CardHeader>
      <CardContent className="pt-0">
        {hasTrend ? (
          <div
            className={cn(
              "inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-xs font-medium",
              positive ? "bg-success/15 text-success" : "bg-destructive/15 text-destructive",
            )}
          >
            {positive ? <TrendingUp className="size-3.5" /> : <TrendingDown className="size-3.5" />}
            {Math.abs(trend).toFixed(1)}%
          </div>
        ) : null}
        {description ? <p className="mt-2 text-xs text-muted-foreground">{description}</p> : null}
      </CardContent>
    </Card>
  );
}

