import { Download } from "lucide-react";
import type { Table } from "@tanstack/react-table";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

type DataTableExportProps<TData> = {
  table: Table<TData>;
  fileName?: string;
};

function sanitizeFileName(value: string) {
  return value.replace(/[^\w\-]+/g, "_").toLowerCase();
}

function escapeCsv(value: unknown) {
  const text = String(value ?? "");
  if (!text.includes(",") && !text.includes("\"") && !text.includes("\n")) {
    return text;
  }
  return `"${text.replaceAll("\"", "\"\"")}"`;
}

function exportFile(content: string, fileName: string, mimeType: string) {
  const blob = new Blob([content], { type: mimeType });
  const link = document.createElement("a");
  link.href = URL.createObjectURL(blob);
  link.download = fileName;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(link.href);
}

export function DataTableExport<TData>({ table, fileName = "table-export" }: DataTableExportProps<TData>) {
  const rows = table.getFilteredRowModel().rows;
  const columns = table
    .getAllLeafColumns()
    .filter((column) => column.getCanHide() === false || column.getIsVisible());

  const baseName = sanitizeFileName(fileName);

  const handleExportCsv = () => {
    const header = columns.map((column) => escapeCsv(column.id)).join(",");
    const body = rows
      .map((row) =>
        columns
          .map((column) => {
            const value = row.getValue(column.id);
            return escapeCsv(value);
          })
          .join(","),
      )
      .join("\n");
    exportFile(`${header}\n${body}`, `${baseName}.csv`, "text/csv;charset=utf-8");
  };

  const handleExportJson = () => {
    const payload = rows.map((row) => row.original);
    exportFile(JSON.stringify(payload, null, 2), `${baseName}.json`, "application/json;charset=utf-8");
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm">
          <Download className="mr-2 size-4" />
          Export
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={handleExportCsv}>Export CSV</DropdownMenuItem>
        <DropdownMenuItem onClick={handleExportJson}>Export JSON</DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

