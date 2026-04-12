import { cn } from "@/lib/utils";

type DiffViewerProps = {
  leftValue: string;
  rightValue: string;
  leftTitle?: string;
  rightTitle?: string;
  className?: string;
};

type DiffRow = {
  lineNumber: number;
  left: string;
  right: string;
  changed: boolean;
};

function buildDiffRows(leftValue: string, rightValue: string): DiffRow[] {
  const leftLines = leftValue.split("\n");
  const rightLines = rightValue.split("\n");
  const length = Math.max(leftLines.length, rightLines.length);

  const rows: DiffRow[] = [];
  for (let index = 0; index < length; index += 1) {
    const left = leftLines[index] ?? "";
    const right = rightLines[index] ?? "";
    rows.push({
      lineNumber: index + 1,
      left,
      right,
      changed: left !== right,
    });
  }
  return rows;
}

export function DiffViewer({
  leftValue,
  rightValue,
  leftTitle = "Current",
  rightTitle = "Incoming",
  className,
}: DiffViewerProps) {
  const rows = buildDiffRows(leftValue, rightValue);

  return (
    <div className={cn("overflow-hidden rounded-lg border", className)}>
      <div className="grid grid-cols-2 border-b bg-muted/40 text-sm font-medium">
        <div className="px-3 py-2">{leftTitle}</div>
        <div className="border-l px-3 py-2">{rightTitle}</div>
      </div>

      <div className="max-h-[520px] overflow-auto">
        {rows.map((row) => (
          <div key={`diff-row-${row.lineNumber}`} className="grid grid-cols-2 text-xs md:text-sm">
            <div
              className={cn(
                "grid grid-cols-[56px_1fr] border-b border-r font-mono",
                row.changed ? "bg-warning/10" : "bg-background",
              )}
            >
              <span className="border-r px-2 py-1 text-right text-muted-foreground">{row.lineNumber}</span>
              <pre className="overflow-x-auto px-2 py-1 whitespace-pre-wrap break-words">{row.left || " "}</pre>
            </div>
            <div
              className={cn(
                "grid grid-cols-[56px_1fr] border-b font-mono",
                row.changed ? "bg-success/10" : "bg-background",
              )}
            >
              <span className="border-r px-2 py-1 text-right text-muted-foreground">{row.lineNumber}</span>
              <pre className="overflow-x-auto px-2 py-1 whitespace-pre-wrap break-words">{row.right || " "}</pre>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

