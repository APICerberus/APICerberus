import { Skeleton } from "@/components/ui/skeleton";

type LoadingStateProps = {
  rows?: number;
  columns?: number;
  className?: string;
};

export function LoadingState({ rows = 4, columns = 3, className }: LoadingStateProps) {
  return (
    <div className={className}>
      <div className="grid gap-3" style={{ gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))` }}>
        {Array.from({ length: rows * columns }).map((_, index) => (
          <Skeleton key={`loading-cell-${index}`} className="h-24 rounded-lg" />
        ))}
      </div>
    </div>
  );
}

