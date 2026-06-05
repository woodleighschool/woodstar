import {
  DndContext,
  KeyboardSensor,
  MouseSensor,
  TouchSensor,
  closestCenter,
  useSensor,
  useSensors,
  type DragEndEvent,
  type UniqueIdentifier,
} from "@dnd-kit/core";
import { restrictToVerticalAxis } from "@dnd-kit/modifiers";
import { SortableContext, arrayMove, useSortable, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { useNavigate } from "@tanstack/react-router";
import {
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
  type Cell,
  type Column,
  type ColumnDef,
  type OnChangeFn,
  type PaginationState,
  type RowSelectionState,
  type SortingState,
  type Row as TanStackRow,
  type Table as TanStackTable,
  type Updater,
  type VisibilityState,
} from "@tanstack/react-table";
import {
  ArrowDown,
  ArrowUp,
  Check,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
  ChevronsUpDown,
  Columns3,
  Download,
  GripVertical,
  PlusCircle,
  SearchIcon,
  XIcon,
} from "lucide-react";
import {
  createContext,
  useContext,
  useId,
  useMemo,
  useState,
  type CSSProperties,
  type ComponentProps,
  type ReactNode,
} from "react";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from "@/components/ui/command";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { InputGroup, InputGroupAddon, InputGroupButton, InputGroupInput } from "@/components/ui/input-group";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { PAGE_SIZE_OPTIONS } from "@/lib/pagination";
import { cn } from "@/lib/utils";

const CONTROL_COLUMN_IDS = new Set(["__select", "__drag", "drag"]);
const DEFAULT_GET_ROW_ID = (row: unknown): string => String((row as { id: string | number }).id);

interface WoodstarColumnMeta {
  label?: string;
  headClassName?: string;
  cellClassName?: string;
}

function columnMeta(meta: unknown): WoodstarColumnMeta {
  return typeof meta === "object" && meta !== null ? meta : {};
}

interface DataTableProps<TData, TValue> {
  columns: ColumnDef<TData, TValue>[];
  data: TData[];
  totalCount: number;
  pagination: PaginationState;
  sorting: SortingState;
  onPaginationChange: OnChangeFn<PaginationState>;
  onSortingChange: OnChangeFn<SortingState>;
  isLoading?: boolean;
  enableRowSelection?: boolean;
  selectedRowIds?: string[];
  onSelectedRowIdsChange?: (ids: string[]) => void;
  bulkActions?: ReactNode;
  rowHref?: (row: TData) => DataTableRowHref;
  onRowClick?: (row: TData) => void;
  getRowId?: (row: TData) => string;
  toolbar?: ReactNode | ((table: TanStackTable<TData>, exportButton: ReactNode) => ReactNode);
  empty: ReactNode;
  emptyClassName?: string;
  footer?: ReactNode;
  skeletonRows?: number;
  perPageOptions?: readonly number[];
  onRowReorder?: (rows: TData[]) => void;
  rowReorderDisabled?: boolean;
  clientSort?: boolean;
  showExport?: boolean;
  exportFilename?: string;
}

interface DataTableRowHref {
  to: string;
  params?: Record<string, string>;
  search?: Record<string, string | number | boolean | undefined>;
}

export function DataTable<TData, TValue>({
  columns,
  data,
  totalCount,
  pagination,
  sorting,
  onPaginationChange,
  onSortingChange,
  isLoading = false,
  enableRowSelection = false,
  selectedRowIds = [],
  onSelectedRowIdsChange,
  bulkActions,
  rowHref,
  onRowClick,
  getRowId,
  toolbar,
  empty,
  emptyClassName,
  footer,
  skeletonRows = 8,
  perPageOptions,
  onRowReorder,
  rowReorderDisabled = false,
  clientSort = false,
  showExport = false,
  exportFilename = "table.csv",
}: DataTableProps<TData, TValue>) {
  const sortableID = useId();
  const sensors = useSensors(useSensor(MouseSensor, {}), useSensor(TouchSensor, {}), useSensor(KeyboardSensor, {}));
  const [localSorting, setLocalSorting] = useState<SortingState>([]);
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});
  const sortingState = clientSort ? localSorting : sorting;

  const rowSelection: RowSelectionState = useMemo(
    () => Object.fromEntries(selectedRowIds.map((id) => [id, true])),
    [selectedRowIds],
  );

  function handleSortingChange(updater: Updater<SortingState>) {
    const next = typeof updater === "function" ? updater(sortingState) : updater;
    setLocalSorting(next);
  }

  function handleRowSelectionChange(updater: Updater<RowSelectionState>) {
    const next = typeof updater === "function" ? updater(rowSelection) : updater;
    onSelectedRowIdsChange?.(Object.keys(next).filter((id) => next[id]));
  }

  const tableColumns = useMemo(
    () => (enableRowSelection ? [selectionColumn<TData>(), ...columns] : columns),
    [columns, enableRowSelection],
  );

  const table = useReactTable({
    data,
    columns: tableColumns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: clientSort ? getSortedRowModel() : undefined,
    manualPagination: !clientSort,
    manualSorting: !clientSort,
    enableRowSelection,
    rowCount: totalCount,
    state: { pagination, sorting: sortingState, columnVisibility, rowSelection },
    onPaginationChange,
    onSortingChange: clientSort ? handleSortingChange : onSortingChange,
    onColumnVisibilityChange: setColumnVisibility,
    onRowSelectionChange: handleRowSelectionChange,
    getRowId: getRowId ?? DEFAULT_GET_ROW_ID,
  });

  const showSkeleton = isLoading && data.length === 0;
  const showEmpty = !showSkeleton && data.length === 0;
  const visibleRows = table.getRowModel().rows;
  const rowIds = visibleRows.map((row) => row.id);
  const dataIDs: UniqueIdentifier[] = rowIds;
  const skeletonRowIds = Array.from({ length: skeletonRows }, (_, row) => `skeleton-row-${row}`);
  const canReorderRows =
    onRowReorder !== undefined && !rowReorderDisabled && visibleRows.length > 1 && !showSkeleton && !showEmpty;
  const exportButton = showExport ? (
    <Button
      type="button"
      variant="outline"
      size="sm"
      disabled={showSkeleton || showEmpty}
      onClick={() => exportTableAsCSV(table, exportFilename)}
    >
      <Download data-icon="inline-start" />
      Export
    </Button>
  ) : null;
  const toolbarContent = typeof toolbar === "function" ? toolbar(table, exportButton) : toolbar;
  const defaultToolbarContent = exportButton ? <div className="flex justify-end">{exportButton}</div> : null;

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event;
    if (!over || active.id === over.id) return;

    const oldIndex = rowIds.indexOf(String(active.id));
    const newIndex = rowIds.indexOf(String(over.id));
    if (oldIndex < 0 || newIndex < 0) return;

    onRowReorder?.(
      arrayMove(
        visibleRows.map((row) => row.original),
        oldIndex,
        newIndex,
      ),
    );
  }

  const tableContent = (
    <div className="overflow-hidden rounded-lg border">
      <Table className="w-max min-w-full">
        <TableHeader className="bg-muted sticky top-0 z-10">
          {table.getHeaderGroups().map((group) => (
            <TableRow key={group.id}>
              {group.headers.map((header) => (
                <TableHead
                  key={header.id}
                  colSpan={header.colSpan}
                  className={columnMeta(header.column.columnDef.meta).headClassName}
                >
                  {header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}
                </TableHead>
              ))}
            </TableRow>
          ))}
        </TableHeader>
        {showEmpty ? null : (
          <TableBody className="**:data-[slot=table-cell]:first:w-8">
            {showSkeleton ? (
              skeletonRowIds.map((rowId) => (
                <TableRow key={rowId}>
                  {table.getVisibleLeafColumns().map((col) => (
                    <TableCell key={col.id}>
                      <Skeleton className="h-4 w-3/4" />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : canReorderRows ? (
              <SortableContext items={dataIDs} strategy={verticalListSortingStrategy}>
                {visibleRows.map((row) => (
                  <SortableDataTableRow key={row.id} row={row} rowHref={rowHref} onRowClick={onRowClick} />
                ))}
              </SortableContext>
            ) : (
              visibleRows.map((row) => (
                <DataTableBodyRow key={row.id} row={row} rowHref={rowHref} onRowClick={onRowClick} />
              ))
            )}
          </TableBody>
        )}
      </Table>
      {showEmpty ? (
        <div
          className={cn("flex min-h-72 items-center justify-center whitespace-normal p-4 text-center", emptyClassName)}
        >
          {empty}
        </div>
      ) : null}
    </div>
  );

  return (
    <div className="flex min-w-0 flex-col gap-4">
      {toolbarContent ?? defaultToolbarContent}
      {canReorderRows ? (
        <DndContext
          id={sortableID}
          sensors={sensors}
          collisionDetection={closestCenter}
          modifiers={[restrictToVerticalAxis]}
          onDragEnd={handleDragEnd}
        >
          {tableContent}
        </DndContext>
      ) : (
        tableContent
      )}
      {clientSort ? null : (
        <DataTablePagination
          table={table}
          totalCount={totalCount}
          selectedCount={selectedRowIds.length}
          bulkActions={bulkActions}
          perPageOptions={perPageOptions}
        />
      )}
      {footer}
    </div>
  );
}

function exportTableAsCSV<TData>(table: TanStackTable<TData>, filename: string) {
  const columns = table.getVisibleLeafColumns().filter((column) => !CONTROL_COLUMN_IDS.has(column.id));
  const rows = table.getRowModel().rows;
  const csv = [
    columns.map((column) => csvCell(headerLabel(column))).join(","),
    ...rows.map((row) =>
      row
        .getVisibleCells()
        .filter((cell) => !CONTROL_COLUMN_IDS.has(cell.column.id))
        .map((cell) => csvCell(cellValue(cell)))
        .join(","),
    ),
  ].join("\n");

  const blob = new Blob([csv], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = csvFilename(filename);
  document.body.append(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

function headerLabel<TData>(column: Column<TData, unknown>) {
  const header = column.columnDef.header;
  return columnMeta(column.columnDef.meta).label ?? (typeof header === "string" ? header : column.id);
}

function cellValue<TData>(cell: Cell<TData, unknown>) {
  const value = cell.getValue();
  if (value == null) return "";
  if (value instanceof Date) return value.toISOString();
  if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") return String(value);
  return JSON.stringify(value);
}

function csvCell(value: string) {
  return `"${value.replaceAll('"', '""')}"`;
}

function csvFilename(filename: string) {
  const trimmed = filename.trim() || "table.csv";
  return trimmed.toLowerCase().endsWith(".csv") ? trimmed : `${trimmed}.csv`;
}

interface DataTableBodyRowProps<TData> {
  row: TanStackRow<TData>;
  rowHref?: (row: TData) => DataTableRowHref;
  onRowClick?: (row: TData) => void;
  isDragging?: boolean;
}

function DataTableBodyRow<TData>({
  row,
  rowHref,
  onRowClick,
  isDragging,
  ref,
  style,
}: DataTableBodyRowProps<TData> & Pick<ComponentProps<"tr">, "ref" | "style">) {
  const navigate = useNavigate();
  const linkProps = rowHref?.(row.original);
  const hasRowAction = linkProps !== undefined || onRowClick !== undefined;
  const firstDataCellID = row.getVisibleCells().find((cell) => !CONTROL_COLUMN_IDS.has(cell.column.id))?.id;

  return (
    <TableRow
      ref={ref}
      style={style}
      data-state={row.getIsSelected() && "selected"}
      data-dragging={isDragging}
      className={cn(
        "relative z-0 data-[dragging=true]:z-10 data-[dragging=true]:opacity-80",
        hasRowAction && "cursor-pointer",
      )}
      onClick={
        hasRowAction
          ? (event) => {
              const target = event.target as HTMLElement;
              if (target.closest(INTERACTIVE_SELECTOR)) return;
              if (window.getSelection()?.toString()) return;
              if (linkProps !== undefined) {
                void navigate({ to: linkProps.to, params: linkProps.params, search: linkProps.search });
              } else {
                onRowClick?.(row.original);
              }
            }
          : undefined
      }
    >
      {row.getVisibleCells().map((cell) => (
        <TableCell
          key={cell.id}
          className={cn(
            columnMeta(cell.column.columnDef.meta).cellClassName,
            cell.id === firstDataCellID && "font-medium",
          )}
        >
          {flexRender(cell.column.columnDef.cell, cell.getContext())}
        </TableCell>
      ))}
    </TableRow>
  );
}

const INTERACTIVE_SELECTOR = [
  "a",
  "button",
  "input",
  "label",
  "select",
  "textarea",
  "[data-slot=dropdown-menu-content]",
].join(", ");

type RowDragHandleContextValue = Pick<
  ReturnType<typeof useSortable>,
  "attributes" | "listeners" | "setActivatorNodeRef"
> | null;

const RowDragHandleContext = createContext<RowDragHandleContextValue>(null);

export function DataTableRowDragHandle() {
  const drag = useContext(RowDragHandleContext);

  return (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      className={cn(
        "size-7 text-muted-foreground hover:bg-transparent",
        drag ? "cursor-grab active:cursor-grabbing" : "cursor-default opacity-50",
      )}
      disabled={!drag}
      ref={drag?.setActivatorNodeRef}
      {...drag?.attributes}
      {...drag?.listeners}
    >
      <GripVertical />
    </Button>
  );
}

function SortableDataTableRow<TData>({
  row,
  rowHref,
  onRowClick,
}: Pick<DataTableBodyRowProps<TData>, "row" | "rowHref" | "onRowClick">) {
  const { attributes, listeners, setActivatorNodeRef, setNodeRef, transform, transition, isDragging } = useSortable({
    id: row.id,
  });
  const style: CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  return (
    <RowDragHandleContext.Provider value={{ attributes, listeners, setActivatorNodeRef }}>
      <DataTableBodyRow
        row={row}
        rowHref={rowHref}
        onRowClick={onRowClick}
        ref={setNodeRef}
        style={style}
        isDragging={isDragging}
      />
    </RowDragHandleContext.Provider>
  );
}

function selectionColumn<TData>(): ColumnDef<TData, unknown> {
  return {
    id: "__select",
    enableSorting: false,
    enableHiding: false,
    header: ({ table }) => (
      <div className="flex items-center justify-center">
        <Checkbox
          checked={table.getIsAllPageRowsSelected() || (table.getIsSomePageRowsSelected() ? "indeterminate" : false)}
          onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
        />
      </div>
    ),
    cell: ({ row }) => (
      <div className="flex items-center justify-center">
        <Checkbox checked={row.getIsSelected()} onCheckedChange={(value) => row.toggleSelected(!!value)} />
      </div>
    ),
    meta: {
      headClassName: "w-10",
      cellClassName: "w-10",
    },
  };
}

interface DataTablePaginationProps<TData> {
  table: TanStackTable<TData>;
  totalCount: number;
  selectedCount: number;
  bulkActions?: ReactNode;
  perPageOptions?: readonly number[];
}

function DataTablePagination<TData>({
  table,
  totalCount,
  selectedCount,
  bulkActions,
  perPageOptions = PAGE_SIZE_OPTIONS,
}: DataTablePaginationProps<TData>) {
  const { pageIndex, pageSize } = table.getState().pagination;
  const pageCount = Math.max(1, table.getPageCount());
  const visibleCount = table.getRowModel().rows.length;
  const fromIndex = totalCount === 0 ? 0 : pageIndex * pageSize + 1;
  const toIndex = pageIndex * pageSize + visibleCount;
  const rangeLabel = totalCount === 0 ? "No Results" : `${fromIndex}-${toIndex} of ${totalCount}`;

  return (
    <div className="flex flex-col gap-3 px-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="text-muted-foreground flex min-h-8 flex-1 items-center gap-3 text-sm tabular-nums">
        <span>{selectedCount > 0 ? `${selectedCount} selected` : rangeLabel}</span>
        {selectedCount > 0 && bulkActions ? <div className="flex items-center gap-2">{bulkActions}</div> : null}
      </div>
      <div className="flex w-full items-center gap-8 sm:w-fit">
        <div className="hidden items-center gap-2 lg:flex">
          <span className="text-sm font-medium">Rows per page</span>
          <Select value={String(pageSize)} onValueChange={(value) => table.setPageSize(Number(value))}>
            <SelectTrigger size="sm" className="w-20">
              <SelectValue />
            </SelectTrigger>
            <SelectContent side="top">
              <SelectGroup>
                {perPageOptions.map((n) => (
                  <SelectItem key={n} value={String(n)}>
                    {n}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>
        <div className="flex w-fit items-center justify-center text-sm font-medium">
          Page {pageIndex + 1} of {pageCount}
        </div>
        <div className="ml-auto flex items-center gap-2 sm:ml-0">
          <Button
            type="button"
            variant="outline"
            size="icon"
            className="hidden size-8 lg:flex"
            disabled={!table.getCanPreviousPage()}
            onClick={() => table.firstPage()}
          >
            <ChevronsLeft />
          </Button>
          <Button
            type="button"
            variant="outline"
            size="icon"
            className="size-8"
            disabled={!table.getCanPreviousPage()}
            onClick={() => table.previousPage()}
          >
            <ChevronLeft />
          </Button>
          <Button
            type="button"
            variant="outline"
            size="icon"
            className="size-8"
            disabled={!table.getCanNextPage()}
            onClick={() => table.nextPage()}
          >
            <ChevronRight />
          </Button>
          <Button
            type="button"
            variant="outline"
            size="icon"
            className="hidden size-8 lg:flex"
            disabled={!table.getCanNextPage()}
            onClick={() => table.lastPage()}
          >
            <ChevronsRight />
          </Button>
        </div>
      </div>
    </div>
  );
}

interface DataTableColumnHeaderProps<TData, TValue> extends ComponentProps<"div"> {
  column: Column<TData, TValue>;
  title: string;
  align?: "left" | "right";
}

export function DataTableColumnHeader<TData, TValue>({
  column,
  title,
  align = "left",
  className,
}: DataTableColumnHeaderProps<TData, TValue>) {
  if (!column.getCanSort()) {
    return <span className={cn(align === "right" && "block text-right", className)}>{title}</span>;
  }

  const sorted = column.getIsSorted();
  const Icon = sorted === "asc" ? ArrowUp : sorted === "desc" ? ArrowDown : ChevronsUpDown;

  return (
    <Button
      type="button"
      variant="ghost"
      size="sm"
      onClick={() => column.toggleSorting(sorted === "asc")}
      className={cn("-ml-2 h-8 px-2 data-[state=open]:bg-accent", align === "right" && "ml-auto", className)}
    >
      {title}
      <Icon className={cn(sorted ? "text-foreground" : "text-muted-foreground/60")} />
    </Button>
  );
}

interface DataTableColumnToggleProps<TData> {
  table: TanStackTable<TData>;
  variant?: ComponentProps<typeof Button>["variant"];
}

export function DataTableColumnToggle<TData>({ table, variant = "outline" }: DataTableColumnToggleProps<TData>) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button type="button" variant={variant} size="sm">
          <Columns3 data-icon="inline-start" />
          Columns
          <ChevronDown data-icon="inline-end" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56">
        <DropdownMenuLabel>Toggle columns</DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuGroup>
          {table
            .getAllColumns()
            .filter((column) => column.getCanHide())
            .map((column) => (
              <DropdownMenuCheckboxItem
                key={column.id}
                checked={column.getIsVisible()}
                onCheckedChange={(value) => column.toggleVisibility(!!value)}
                onSelect={(event) => event.preventDefault()}
              >
                {columnMeta(column.columnDef.meta).label ?? column.id}
              </DropdownMenuCheckboxItem>
            ))}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export interface FacetedFilterOption {
  value: string;
  label: string;
  icon?: React.ComponentType<{ className?: string }>;
}

interface DataTableFacetedFilterProps {
  title: string;
  options: FacetedFilterOption[];
  selected: string[];
  onChange: (next: string[]) => void;
  singleSelect?: boolean;
}

export function DataTableFacetedFilter({
  title,
  options,
  selected,
  onChange,
  singleSelect,
}: DataTableFacetedFilterProps) {
  const selectedSet = new Set(selected);

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button type="button" variant="outline" size="sm" className="border-dashed">
          <PlusCircle data-icon="inline-start" />
          {title}
          {selectedSet.size > 0 ? (
            <>
              <Separator orientation="vertical" className="mx-1 h-4" />
              <Badge variant="secondary" className="px-1.5 font-normal lg:hidden">
                {selectedSet.size}
              </Badge>
              <div className="hidden gap-1 lg:flex">
                {selectedSet.size > 2 ? (
                  <Badge variant="secondary" className="px-1.5 font-normal">
                    {selectedSet.size} Selected
                  </Badge>
                ) : (
                  options
                    .filter((option) => selectedSet.has(option.value))
                    .map((option) => (
                      <Badge variant="secondary" key={option.value} className="px-1.5 font-normal">
                        {option.label}
                      </Badge>
                    ))
                )}
              </div>
            </>
          ) : null}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[220px] p-0" align="start">
        <Command>
          <CommandInput placeholder={title} />
          <CommandList>
            <CommandEmpty>No Results Found.</CommandEmpty>
            <CommandGroup>
              {options.map((option) => {
                const isSelected = selectedSet.has(option.value);
                return (
                  <CommandItem
                    key={option.value}
                    onSelect={() => {
                      if (singleSelect) {
                        onChange(isSelected ? [] : [option.value]);
                        return;
                      }
                      const next = new Set(selectedSet);
                      if (isSelected) next.delete(option.value);
                      else next.add(option.value);
                      onChange(Array.from(next));
                    }}
                  >
                    <div
                      className={cn(
                        "border-primary mr-2 flex size-4 items-center justify-center rounded-sm border",
                        isSelected ? "bg-primary text-primary-foreground" : "[&_svg]:invisible opacity-50",
                      )}
                    >
                      <Check className="size-3" />
                    </div>
                    {option.icon ? <option.icon className="text-muted-foreground" /> : null}
                    <span>{option.label}</span>
                  </CommandItem>
                );
              })}
            </CommandGroup>
            {selectedSet.size > 0 ? (
              <>
                <CommandSeparator />
                <CommandGroup>
                  <CommandItem onSelect={() => onChange([])} className="justify-center text-center">
                    Clear Filters
                  </CommandItem>
                </CommandGroup>
              </>
            ) : null}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

interface DataTableSearchProps {
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  className?: string;
}

export function DataTableSearch({ value, onChange, placeholder, className }: DataTableSearchProps) {
  return (
    <InputGroup className={cn("max-w-md flex-1", className)}>
      <InputGroupInput value={value} onChange={(event) => onChange(event.target.value)} placeholder={placeholder} />
      <InputGroupAddon align="inline-start">
        <SearchIcon />
      </InputGroupAddon>
      {value ? (
        <InputGroupAddon align="inline-end">
          <InputGroupButton size="icon-xs" onClick={() => onChange("")}>
            <XIcon />
          </InputGroupButton>
        </InputGroupAddon>
      ) : null}
    </InputGroup>
  );
}

interface DataTableEmptyStateProps {
  icon: ReactNode;
  title: ReactNode;
  description: ReactNode;
}

export function DataTableEmptyState({ icon, title, description }: DataTableEmptyStateProps) {
  return (
    <Empty>
      <EmptyHeader>
        <EmptyMedia variant="icon">{icon}</EmptyMedia>
        <EmptyTitle>{title}</EmptyTitle>
        <EmptyDescription>{description}</EmptyDescription>
      </EmptyHeader>
    </Empty>
  );
}

export interface BulkDeleteDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  count: number;
  noun: string;
  description?: string;
  pending?: boolean;
  onConfirm: () => void;
}

export function BulkDeleteDialog({
  open,
  onOpenChange,
  count,
  noun,
  description,
  pending = false,
  onConfirm,
}: BulkDeleteDialogProps) {
  const label = count === 1 ? noun : `${noun}s`;

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            Delete {count} Selected {label}?
          </AlertDialogTitle>
          {description ? <AlertDialogDescription>{description}</AlertDialogDescription> : null}
        </AlertDialogHeader>

        <AlertDialogFooter>
          <AlertDialogCancel variant="ghost" size="sm" disabled={pending}>
            Cancel
          </AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            size="sm"
            disabled={pending || count === 0}
            onClick={(event) => {
              event.preventDefault();
              onConfirm();
            }}
          >
            Delete
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
