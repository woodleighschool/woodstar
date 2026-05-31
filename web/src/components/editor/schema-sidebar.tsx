import { ExternalLink, PanelRightClose, PanelRightOpen } from "lucide-react";
import { isValidElement, useState } from "react";

import { Badge } from "@/components/ui/badge";
import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { useOsquerySchema, type OsqueryColumn, type OsqueryTable } from "@/hooks/use-osquery-schema";
import { cn } from "@/lib/utils";

import { Markdown } from "./markdown";
import { SQLEditor } from "./sql-editor";

interface SchemaSidebarProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onInsertColumn?: (columnName: string) => void;
  selectedTable?: string | null;
  onSelectedTableChange?: (tableName: string) => void;
}

export function SchemaSidebar({
  open,
  onOpenChange,
  onInsertColumn,
  selectedTable,
  onSelectedTableChange,
}: SchemaSidebarProps) {
  const Icon = open ? PanelRightClose : PanelRightOpen;

  return (
    <TooltipProvider delayDuration={150}>
      <button
        type="button"
        title={open ? "Collapse schema sidebar" : "Expand schema sidebar"}
        onClick={() => onOpenChange(!open)}
        className={cn(
          "bg-card fixed top-20 z-40 rounded-l-md border border-r-0 p-2 shadow-sm md:top-6",
          "transition-[right,color,border-color] duration-200 ease-out hover:bg-accent hover:text-accent-foreground",
          open ? "right-80" : "right-0",
        )}
      >
        <Icon className="size-4" />
      </button>
      <SchemaPanel
        open={open}
        onInsertColumn={onInsertColumn}
        selectedTable={selectedTable}
        onSelectedTableChange={onSelectedTableChange}
      />
    </TooltipProvider>
  );
}

function SchemaPanel({
  open,
  onInsertColumn,
  selectedTable,
  onSelectedTableChange,
}: {
  open: boolean;
  onInsertColumn?: (columnName: string) => void;
  selectedTable?: string | null;
  onSelectedTableChange?: (tableName: string) => void;
}) {
  const schema = useOsquerySchema();
  const tables = schema.data ?? [];
  const [localSelectedName, setLocalSelectedName] = useState<string | null>(null);
  const selectedName = selectedTable ?? localSelectedName;

  function selectTable(tableName: string) {
    setLocalSelectedName(tableName);
    onSelectedTableChange?.(tableName);
  }

  const selected = selectedName ? tables.find((candidate) => candidate.name === selectedName) : undefined;
  const table =
    tables.length > 0 ? (selected ?? tables.find((candidate) => candidate.name === "users") ?? tables[0]) : null;

  return (
    <aside
      className={cn(
        "bg-card fixed top-12 right-0 bottom-0 z-30 flex w-80 flex-col border-l shadow-lg md:top-0",
        "transition-transform duration-200 ease-out",
        open ? "translate-x-0" : "translate-x-full",
      )}
    >
      <div className="flex items-center justify-between gap-2 p-4">
        <div className="flex items-center gap-2">
          <h2 className="text-sm font-semibold">Tables</h2>
          <Badge variant="secondary" className="rounded-full px-2 text-[11px] font-normal">
            {tables.length}
          </Badge>
        </div>
      </div>

      <div className="p-4">
        <TableSelector tables={tables} value={table?.name ?? null} onChange={selectTable} />
      </div>

      <div className="flex-1 overflow-y-auto">
        {schema.isLoading ? (
          <div className="text-muted-foreground p-4 text-sm">Loading Schema…</div>
        ) : schema.error ? (
          <div className="text-muted-foreground p-4 text-sm">Schema unavailable</div>
        ) : table ? (
          <TableDetail table={table} onInsertColumn={onInsertColumn} />
        ) : null}
      </div>
    </aside>
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
  const tableNames = tables.map((table) => table.name);

  return (
    <Combobox
      items={tableNames}
      value={value ?? null}
      onValueChange={(next) => {
        if (next) onChange(next);
      }}
      onInputValueChange={(next) => {
        if (tableNames.includes(next)) onChange(next);
      }}
    >
      <ComboboxInput placeholder="Select a Table" className="w-full text-sm" />
      <ComboboxContent>
        <ComboboxEmpty>No Tables Found.</ComboboxEmpty>
        <ComboboxList>
          {(item: string) => (
            <ComboboxItem key={item} value={item} className="text-sm">
              {item}
            </ComboboxItem>
          )}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  );
}

function TableDetail({ table, onInsertColumn }: { table: OsqueryTable; onInsertColumn?: (name: string) => void }) {
  const badges = [
    table.evented ? (
      <Badge key="evented" variant="outline">
        evented
      </Badge>
    ) : null,
    table.cacheable ? (
      <Badge key="cacheable" variant="outline">
        cacheable
      </Badge>
    ) : null,
  ].filter(Boolean);
  return (
    <div className="space-y-5 p-4">
      {badges.length ? <div className="flex flex-wrap items-center gap-2">{badges}</div> : null}

      {table.description ? (
        <section>
          <Markdown className="text-muted-foreground text-sm">{table.description}</Markdown>
        </section>
      ) : null}

      <ColumnList columns={table.columns} onInsertColumn={onInsertColumn} />

      {table.examples ? (
        <section>
          <SectionHeading>Example</SectionHeading>
          <Markdown className="text-muted-foreground text-sm" components={exampleComponents}>
            {exampleMarkdown(table.examples)}
          </Markdown>
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

const exampleComponents = {
  pre: ({ children }: { children?: React.ReactNode }) => (
    <div className="text-foreground mb-2 last:mb-0">
      <SQLEditor value={codeText(children)} onChange={() => null} readOnly />
    </div>
  ),
};

function codeText(children: React.ReactNode) {
  if (!isValidElement<{ children?: React.ReactNode }>(children)) return "";
  const code = children.props.children;
  return typeof code === "string" ? code.trim() : "";
}

function exampleMarkdown(examples: NonNullable<OsqueryTable["examples"]>) {
  if (typeof examples === "string") return examples;
  return examples
    .map((example) => [example.description, "```sql", example.query, "```"].filter(Boolean).join("\n\n"))
    .join("\n\n");
}

function ColumnList({
  columns,
  onInsertColumn,
}: {
  columns: OsqueryColumn[];
  onInsertColumn?: (name: string) => void;
}) {
  const ordered = [...columns].sort((a, b) => Number(b.required) - Number(a.required) || a.name.localeCompare(b.name));

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
        <span className="truncate text-sm">{column.name}</span>
        {column.required ? <span className="text-destructive text-xs">*</span> : null}
      </span>
      <span className="text-muted-foreground shrink-0 text-[10px] tracking-wide uppercase">{column.type}</span>
    </div>
  );

  if (!onInsert && !tooltip) return <li>{row}</li>;

  const button = onInsert ? (
    <button
      type="button"
      onClick={() => onInsert(column.name)}
      className="hover:bg-muted/60 -mx-2 block w-[calc(100%+1rem)] rounded px-2 text-left"
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
