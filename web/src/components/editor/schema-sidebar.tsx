import { Check, ChevronsUpDown, ExternalLink, X } from "lucide-react";
import { useMemo, useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Command, CommandEmpty, CommandInput, CommandItem, CommandList } from "@/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { useOsquerySchema, type OsqueryColumn, type OsqueryTable } from "@/hooks/use-osquery-schema";
import { cn } from "@/lib/utils";

import { Markdown } from "./markdown";

const PLATFORM_ORDER = ["darwin", "windows", "linux", "chrome"];
const PLATFORM_LABELS: Record<string, string> = {
  darwin: "macOS",
  windows: "Windows",
  linux: "Linux",
  chrome: "ChromeOS",
};

interface SchemaSidebarProps {
  onClose?: () => void;
  onInsertColumn?: (columnName: string) => void;
  className?: string;
}

export function SchemaSidebar({ onClose, onInsertColumn, className }: SchemaSidebarProps) {
  const schema = useOsquerySchema();
  const tables = useMemo(() => schema.data ?? [], [schema.data]);
  const [selectedName, setSelectedName] = useState<string | null>(null);

  const selected = useMemo(() => {
    if (!tables.length) return null;
    const byName = new Map(tables.map((t) => [t.name, t]));
    if (selectedName && byName.has(selectedName)) return byName.get(selectedName) ?? null;
    return byName.get("users") ?? tables[0];
  }, [tables, selectedName]);

  return (
    <TooltipProvider delayDuration={150}>
      <aside
        className={cn("bg-card fixed top-12 right-0 bottom-0 z-30 flex w-80 flex-col border-l shadow-lg", className)}
      >
        {onClose ? (
          <button
            type="button"
            onClick={onClose}
            aria-label="Close schema panel"
            className="bg-card hover:border-primary hover:text-primary absolute top-10 -left-3 z-10 flex size-6 items-center justify-center rounded-full border shadow-sm"
          >
            <X className="size-3.5" />
          </button>
        ) : null}

        <div className="flex items-center justify-between gap-2 border-b p-4 pb-3">
          <div className="flex items-center gap-2">
            <h2 className="text-sm font-semibold">Tables</h2>
            <Badge variant="secondary" className="rounded-full px-2 text-[11px] font-normal">
              {tables.length}
            </Badge>
          </div>
        </div>

        <div className="border-b px-4 py-3">
          <TableSelector tables={tables} value={selected?.name ?? null} onChange={setSelectedName} />
        </div>

        <div className="flex-1 overflow-y-auto">
          {schema.isLoading ? (
            <div className="text-muted-foreground p-4 text-sm">Loading schema…</div>
          ) : schema.error ? (
            <div className="text-muted-foreground p-4 text-sm">Schema unavailable</div>
          ) : selected ? (
            <TableDetail table={selected} onInsertColumn={onInsertColumn} />
          ) : null}
        </div>
      </aside>
    </TooltipProvider>
  );
}

function TableSelector({
  tables,
  value,
  onChange,
}: {
  tables: OsqueryTable[];
  value: string | null;
  onChange: (name: string) => void;
}) {
  const [open, setOpen] = useState(false);
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className="w-full justify-between font-mono text-sm"
        >
          <span className="truncate">{value ?? "Select a table"}</span>
          <ChevronsUpDown className="text-muted-foreground size-4 shrink-0 opacity-70" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[var(--radix-popover-trigger-width)] p-0" align="start">
        <Command>
          <CommandInput placeholder="Search tables…" />
          <CommandList>
            <CommandEmpty>No tables found.</CommandEmpty>
            {tables.map((table) => (
              <CommandItem
                key={table.name}
                value={table.name}
                onSelect={() => {
                  onChange(table.name);
                  setOpen(false);
                }}
              >
                <Check className={cn("size-4", value === table.name ? "opacity-100" : "opacity-0")} />
                <span className="font-mono text-sm">{table.name}</span>
              </CommandItem>
            ))}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

function TableDetail({ table, onInsertColumn }: { table: OsqueryTable; onInsertColumn?: (name: string) => void }) {
  const exampleMarkdown = typeof table.examples === "string" ? table.examples : null;
  return (
    <div className="space-y-5 p-4">
      <div className="flex flex-wrap items-center gap-2">
        <h3 className="font-mono text-base font-semibold">{table.name}</h3>
        {table.evented ? <Badge variant="outline">evented</Badge> : null}
        {table.cacheable ? <Badge variant="outline">cacheable</Badge> : null}
      </div>

      {table.description ? (
        <section>
          <Markdown className="text-muted-foreground text-sm">{table.description}</Markdown>
        </section>
      ) : null}

      {table.platforms?.length ? <PlatformList platforms={table.platforms} /> : null}

      <ColumnList columns={table.columns} onInsertColumn={onInsertColumn} />

      {exampleMarkdown ? (
        <section>
          <SectionHeading>Example</SectionHeading>
          <Markdown className="text-muted-foreground">{exampleMarkdown}</Markdown>
        </section>
      ) : null}

      {table.notes ? (
        <section>
          <SectionHeading>Notes</SectionHeading>
          <Markdown className="text-muted-foreground">{table.notes}</Markdown>
        </section>
      ) : null}

      {table.url ? (
        <a
          href={table.url}
          target="_blank"
          rel="noreferrer"
          className="text-primary inline-flex items-center gap-1 text-sm hover:underline"
        >
          Source <ExternalLink className="size-3.5" />
        </a>
      ) : null}
    </div>
  );
}

function SectionHeading({ children }: { children: React.ReactNode }) {
  return <h4 className="mb-2 text-xs font-semibold tracking-wide uppercase">{children}</h4>;
}

function PlatformList({ platforms }: { platforms: string[] }) {
  const sorted = [...platforms]
    .filter((p) => PLATFORM_LABELS[p])
    .sort((a, b) => PLATFORM_ORDER.indexOf(a) - PLATFORM_ORDER.indexOf(b));
  if (!sorted.length) return null;
  return (
    <section>
      <SectionHeading>Compatible with</SectionHeading>
      <ul className="flex flex-wrap gap-1.5">
        {sorted.map((p) => (
          <li key={p}>
            <Badge variant="secondary" className="font-normal">
              {PLATFORM_LABELS[p]}
            </Badge>
          </li>
        ))}
      </ul>
    </section>
  );
}

function ColumnList({
  columns,
  onInsertColumn,
}: {
  columns: OsqueryColumn[];
  onInsertColumn?: (name: string) => void;
}) {
  const ordered = useMemo(() => {
    const required = columns.filter((c) => c.required).sort((a, b) => a.name.localeCompare(b.name));
    const rest = columns.filter((c) => !c.required).sort((a, b) => a.name.localeCompare(b.name));
    return [...required, ...rest];
  }, [columns]);

  return (
    <section>
      <SectionHeading>Columns</SectionHeading>
      <ul className="divide-border divide-y">
        {ordered.map((column) => (
          <ColumnRow key={column.name} column={column} onInsert={onInsertColumn} />
        ))}
      </ul>
    </section>
  );
}

function ColumnRow({ column, onInsert }: { column: OsqueryColumn; onInsert?: (name: string) => void }) {
  const tooltip = renderColumnTooltip(column);
  const row = (
    <div className="flex items-baseline justify-between gap-2 py-1.5">
      <span className="flex min-w-0 items-baseline gap-1">
        <span className="truncate font-mono text-sm">{column.name}</span>
        {column.required ? <span className="text-destructive text-xs">*</span> : null}
      </span>
      <span className="text-muted-foreground shrink-0 font-mono text-[10px] tracking-wide uppercase">
        {column.type}
      </span>
    </div>
  );

  if (!onInsert && !tooltip) return <li>{row}</li>;

  const button = onInsert ? (
    <button
      type="button"
      onClick={() => onInsert(column.name)}
      className="hover:bg-muted/60 -mx-2 block w-[calc(100%+1rem)] rounded px-2 text-left"
      aria-label={`Insert column ${column.name}`}
    >
      {row}
    </button>
  ) : (
    row
  );

  if (!tooltip) return <li>{button}</li>;

  return (
    <li>
      <Tooltip>
        <TooltipTrigger asChild>{button}</TooltipTrigger>
        <TooltipContent side="left" className="max-w-xs whitespace-normal text-xs">
          {tooltip}
        </TooltipContent>
      </Tooltip>
    </li>
  );
}

function renderColumnTooltip(column: OsqueryColumn): React.ReactNode | null {
  const lines: { key: string; text: string }[] = [];
  if (column.description) lines.push({ key: "desc", text: column.description });
  if (column.required) lines.push({ key: "req", text: "Required in WHERE clause." });
  if (column.platforms?.length) lines.push({ key: "plat", text: `Only on ${column.platforms.join(", ")}.` });
  if (column.hidden) lines.push({ key: "hide", text: "Not returned by SELECT *." });
  if (!lines.length) return null;
  return (
    <div className="space-y-1">
      {lines.map((line) => (
        <div key={line.key}>{line.text}</div>
      ))}
    </div>
  );
}
