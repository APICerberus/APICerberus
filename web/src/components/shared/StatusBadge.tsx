import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

type StatusBadgeProps = {
  status: string;
  className?: string;
};

const STATUS_STYLE_MAP: Record<string, string> = {
  active: "bg-success/15 text-success border-success/30",
  suspended: "bg-destructive/15 text-destructive border-destructive/30",
  pending: "bg-warning/15 text-warning border-warning/30",
};

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const normalized = status.trim().toLowerCase();
  const styleClass = STATUS_STYLE_MAP[normalized] ?? "bg-muted text-muted-foreground border-border";

  return (
    <Badge variant="outline" className={cn("capitalize", styleClass, className)}>
      {status}
    </Badge>
  );
}

