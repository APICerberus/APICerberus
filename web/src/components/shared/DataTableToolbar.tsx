import type { Table } from "@tanstack/react-table";
import { Columns3, RotateCcw } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { DataTableExport } from "./DataTableExport";

export type DataTableFilterOption = {
  label: string;
  value: string;
};

export type DataTableFilterConfig<TData> = {
  columnId: keyof TData | string;
  title: string;
  options: DataTableFilterOption[];
};

type DataTableToolbarProps<TData> = {
  table: Table<TData>;
  searchColumn?: keyof TData | string;
  searchPlaceholder?: string;
  filters?: DataTableFilterConfig<TData>[];
  fileName?: string;
};

export function DataTableToolbar<TData>({
  table,
  searchColumn,
  searchPlaceholder = "Search...",
  filters = [],
  fileName,
}: DataTableToolbarProps<TData>) {
  const hasFilters = table.getState().columnFilters.length > 0 || table.getState().globalFilter;

  return (
    <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
      <div className="flex flex-1 flex-col gap-2 sm:flex-row">
        {searchColumn ? (
          <Input
            placeholder={searchPlaceholder}
            className="h-9 w-full sm:max-w-xs"
            value={(table.getColumn(String(searchColumn))?.getFilterValue() as string) ?? ""}
            onChange={(event) => table.getColumn(String(searchColumn))?.setFilterValue(event.target.value)}
          />
        ) : null}

        {filters.map((filter) => {
          const column = table.getColumn(String(filter.columnId));
          if (!column) {
            return null;
          }

          return (
            <Select
              key={`${String(filter.columnId)}-${filter.title}`}
              value={(column.getFilterValue() as string) ?? "__all__"}
              onValueChange={(value) => column.setFilterValue(value === "__all__" ? undefined : value)}
            >
              <SelectTrigger className="h-9 w-full sm:w-44">
                <SelectValue placeholder={filter.title} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="__all__">All {filter.title}</SelectItem>
                {filter.options.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          );
        })}
      </div>

      <div className="flex items-center gap-2">
        {hasFilters ? (
          <Button variant="ghost" size="sm" onClick={() => table.resetColumnFilters()}>
            <RotateCcw className="mr-2 size-4" />
            Reset
          </Button>
        ) : null}

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="sm">
              <Columns3 className="mr-2 size-4" />
              Columns
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuLabel>Toggle Columns</DropdownMenuLabel>
            <DropdownMenuSeparator />
            {table
              .getAllLeafColumns()
              .filter((column) => column.getCanHide())
              .map((column) => (
                <DropdownMenuCheckboxItem
                  key={column.id}
                  checked={column.getIsVisible()}
                  onCheckedChange={(value) => column.toggleVisibility(Boolean(value))}
                >
                  {column.id}
                </DropdownMenuCheckboxItem>
              ))}
          </DropdownMenuContent>
        </DropdownMenu>

        <DataTableExport table={table} fileName={fileName} />
      </div>
    </div>
  );
}

