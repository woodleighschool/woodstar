import type { CellContext, ColumnDef } from "@tanstack/react-table";
import { Play, Square, X } from "lucide-react";
import type { ReactNode } from "react";
import { useMemo, useState } from "react";

import { AsyncButton } from "@components/async-button";
import { ConfirmDialog } from "@components/confirm-dialog";
import { DataTableStatic } from "@components/data-table/data-table-static";
import { encodeSort } from "@components/data-table/use-data-table-search";
import { EnumStatusIndicator } from "@components/enum-status-indicator";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { Button } from "@components/ui/button";
import {
  Combobox,
  ComboboxChip,
  ComboboxChips,
  ComboboxChipsInput,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxItem,
  ComboboxList,
  ComboboxTrigger,
  ComboboxValue,
  useComboboxAnchor,
} from "@components/ui/combobox";
import { Field, FieldGroup, FieldLabel, FieldLegend, FieldSet } from "@components/ui/field";
import { Spinner } from "@components/ui/spinner";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@components/ui/tabs";
import { useAuth } from "@features/auth/queries";
import { useHosts } from "@features/hosts/queries";
import { useLabels } from "@features/labels/queries";
import { CHECK_RESULT_STATUSES } from "@features/osquery/checks/model";
import type { Host, Label } from "@lib/api";
import { MAX_PAGE_SIZE } from "@lib/pagination";

import {
  type LiveQueryResult,
  type LiveQueryRow,
  type OsqueryLiveQueryCreateBody,
  type OsqueryLiveQueryTargetCountBody,
  type OsqueryLiveQueryTargetCountOutputBody,
  useCreateLiveQuery,
  useLiveQueryStream,
  useLiveQueryTargetCount,
  useStopLiveQuery,
} from "./queries";
import { ShowQueryButton } from "./query-actions";
type LiveRunKind = "report" | "check";
type LiveRunStep = "targets" | "run";
type ReportResultRow = Record<string, string>;

function ReportHostCell({ row }: CellContext<ReportResultRow, unknown>) {
  return (
    <Link to="/hosts/$id" params={{ id: row.original.host_id }} className="whitespace-nowrap">
      {row.original.host_name}
    </Link>
  );
}

function ReportValueCell({ row, column }: CellContext<ReportResultRow, unknown>) {
  return <span className="whitespace-nowrap">{row.original[column.id] ?? "-"}</span>;
}

type CheckLiveRow = {
  host_id: number;
  host_name?: string;
  response: "pass" | "fail";
};

const checkResultColumns: ColumnDef<CheckLiveRow>[] = [
  {
    accessorKey: "host_name",
    header: "Host",
    cell: ({ row }) => (
      <Link to="/hosts/$id" params={{ id: String(row.original.host_id) }}>
        {row.original.host_name}
      </Link>
    ),
  },
  {
    accessorKey: "response",
    header: "Result",
    cell: ({ row }) => (
      <EnumStatusIndicator value={row.original.response} metadata={CHECK_RESULT_STATUSES} />
    ),
  },
];

const errorResultColumns: ColumnDef<LiveQueryRow>[] = [
  {
    id: "host",
    header: "Host",
    cell: ({ row }) => row.original.host_name,
  },
  {
    id: "error",
    header: "Error",
    cell: ({ row }) => row.original.error ?? row.original.status,
  },
];

export function LiveRunner({
  kind,
  sql,
  onCancel,
}: {
  kind: LiveRunKind;
  sql: string;
  onCancel: () => void;
}) {
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const create = useCreateLiveQuery();
  const stop = useStopLiveQuery();
  const [step, setStep] = useState<LiveRunStep>("targets");
  const [selectedLabels, setSelectedLabels] = useState<Label[]>([]);
  const [selectedHosts, setSelectedHosts] = useState<Host[]>([]);
  const [liveQueryId, setLiveQueryId] = useState<number | null>(null);
  const [runTargetCount, setRunTargetCount] = useState(0);
  const [stopRequested, setStopRequested] = useState(false);
  const stream = useLiveQueryStream(liveQueryId);
  const selectedHostIDs = useMemo(() => selectedHosts.map((host) => host.id), [selectedHosts]);
  const selectedLabelIDs = useMemo(() => selectedLabels.map((label) => label.id), [selectedLabels]);
  const hasTargets = selectedLabels.length > 0 || selectedHosts.length > 0;
  const targetSelection = useMemo<OsqueryLiveQueryTargetCountBody>(
    () => ({
      selected: {
        hosts: selectedHostIDs,
        labels: selectedLabelIDs,
      },
    }),
    [selectedHostIDs, selectedLabelIDs],
  );
  const targetMetrics = useLiveQueryTargetCount(targetSelection, hasTargets);
  const isRunning = stream.status === "running";
  const isStarting = create.isPending;
  const isStopping = stop.isPending;
  const respondedCount = stream.results.filter(
    (row) => row.host_id !== undefined && row.status !== "stopped",
  ).length;
  const canRun =
    hasTargets &&
    !targetMetrics.isFetching &&
    (targetMetrics.data?.targets_online ?? 0) > 0 &&
    !isStarting &&
    !isRunning &&
    !isStopping;
  async function run() {
    create.reset();
    stop.reset();
    setStopRequested(false);
    const body: OsqueryLiveQueryCreateBody = {
      sql,
      selected: targetSelection.selected,
    };
    const handle = await create.mutateAsync(body);
    setRunTargetCount(handle.resolved_host_count);
    setLiveQueryId(handle.id);
    setStep("run");
  }
  async function stopRun() {
    if (liveQueryId === null) return;
    setStopRequested(true);
    await stop.mutateAsync(liveQueryId);
  }
  function backToTargets() {
    if (liveQueryId !== null && stream.status === "running") {
      void stop.mutateAsync(liveQueryId).catch(() => undefined);
    }
    setStep("targets");
    setLiveQueryId(null);
    setRunTargetCount(0);
    setStopRequested(false);
    create.reset();
    stop.reset();
  }
  const itemLabel = kind === "report" ? "report" : "check";
  const title = kind === "report" ? "Run Report" : "Run Check";
  if (!isAdmin) {
    return (
      <PageShell>
        <PageHeader
          title={title}
          description={`Live ${itemLabel} execution is admin-only.`}
          actions={<ShowQueryButton sql={sql} />}
        />
        <PanelEmptyState>Live execution is unavailable for this account.</PanelEmptyState>
      </PageShell>
    );
  }
  return (
    <PageShell>
      <PageHeader title={title} actions={<ShowQueryButton sql={sql} />} />

      {step === "targets" ? (
        <>
          <TargetPicker
            selectedLabels={selectedLabels}
            selectedHosts={selectedHosts}
            onLabelsChange={setSelectedLabels}
            onHostsChange={setSelectedHosts}
          />
          <TargetSummary metrics={targetMetrics.data} error={targetMetrics.error?.message} />
          <Field orientation="horizontal" className="max-w-3xl justify-start">
            <AsyncButton
              isPending={isStarting}
              type="button"
              size="sm"
              onClick={() => void run()}
              disabled={!canRun && !isStarting}
              icon={<Play data-icon="inline-start" />}
            >
              Run
            </AsyncButton>
            <Button type="button" variant="outline" size="sm" onClick={onCancel}>
              Cancel
            </Button>
          </Field>
        </>
      ) : (
        <RunResults
          kind={kind}
          rows={stream.results}
          status={stream.status}
          stopped={stopRequested && stream.status === "completed"}
          isStopping={isStopping}
          targetCount={runTargetCount}
          respondedCount={respondedCount}
          onStop={stopRun}
          onRunAgain={() => void run()}
          onChangeTargets={backToTargets}
        />
      )}
    </PageShell>
  );
}
function TargetSummary({
  metrics,
  error,
}: {
  metrics?: OsqueryLiveQueryTargetCountOutputBody;
  error?: string;
}) {
  const targetCount = metrics?.targets_count ?? 0;
  const onlinePercent =
    targetCount === 0 ? 0 : Math.round(((metrics?.targets_online ?? 0) / targetCount) * 100);
  return (
    <div className="flex max-w-3xl flex-col gap-1">
      <p className="text-sm text-muted-foreground">
        <strong className="font-semibold text-foreground">{targetCount}</strong>{" "}
        {targetCount === 1 ? "host" : "hosts"} targeted ({onlinePercent}% online)
      </p>
      {error ? <p className="text-sm text-destructive">{error}</p> : null}
    </div>
  );
}
function RunResults({
  kind,
  rows,
  status,
  stopped,
  isStopping,
  targetCount,
  respondedCount,
  onStop,
  onRunAgain,
  onChangeTargets,
}: {
  kind: LiveRunKind;
  rows: LiveQueryRow[];
  status: string;
  stopped: boolean;
  isStopping: boolean;
  targetCount: number;
  respondedCount: number;
  onStop: () => Promise<void>;
  onRunAgain: () => void;
  onChangeTargets: () => void;
}) {
  const isRunning = status === "running";
  const finished = status === "completed";
  const [stopOpen, setStopOpen] = useState(false);
  async function confirmStop() {
    await onStop();
    setStopOpen(false);
  }
  return (
    <div className="grid min-w-0 gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3 border-b pb-4">
        <div>
          <div className="flex items-center gap-2">
            <h2 className="text-base font-semibold">{runHeading(status, stopped)}</h2>
            {isRunning ? <Spinner className="size-3.5 text-muted-foreground" /> : null}
          </div>
          <p className="text-sm text-muted-foreground">
            {respondedCount} of {targetCount} online host
            {targetCount === 1 ? "" : "s"} responded.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {isRunning || isStopping ? (
            <AsyncButton
              isPending={isStopping}
              size="sm"
              variant="destructive"
              onClick={() => setStopOpen(true)}
              icon={<Square data-icon="inline-start" />}
            >
              Stop
            </AsyncButton>
          ) : (
            <Button size="sm" onClick={onRunAgain}>
              <Play data-icon="inline-start" />
              {finished ? "Run Again" : "Run"}
            </Button>
          )}
          <Button size="sm" variant="outline" onClick={onChangeTargets}>
            <X data-icon="inline-start" />
            Change Targets
          </Button>
        </div>
      </div>
      <ConfirmDialog
        open={stopOpen}
        onOpenChange={setStopOpen}
        title="Stop Live Run?"
        description="Hosts that have not responded yet will be marked stopped for this live run."
        confirmLabel="Stop"
        variant="destructive"
        pending={isStopping}
        onConfirm={() => void confirmStop()}
      />
      {kind === "report" ? (
        <ReportRunResults rows={rows} running={isRunning} />
      ) : (
        <CheckRunResults rows={rows} running={isRunning} />
      )}
    </div>
  );
}
function runHeading(status: string, stopped: boolean) {
  if (status === "running") return "Running";
  if (stopped) return "Stopped";
  if (status === "completed") return "Finished";
  if (status === "error") return "Stream Interrupted";
  return "Starting";
}
function ReportRunResults({ rows, running }: { rows: LiveQueryRow[]; running: boolean }) {
  const resultRows = reportResultRows(rows);
  const errorRows = liveErrorRows(rows);
  return (
    <RunResultsTabs errorRows={errorRows}>
      <ReportRowsTable rows={resultRows} running={running} />
    </RunResultsTabs>
  );
}
function CheckRunResults({ rows, running }: { rows: LiveQueryRow[]; running: boolean }) {
  const hostRows = checkResultRows(rows);
  const errorRows = liveErrorRows(rows);
  const passing = hostRows.filter((row) => row.response === "pass").length;
  const failing = hostRows.filter((row) => row.response === "fail").length;
  return (
    <RunResultsTabs errorRows={errorRows}>
      <div className="grid gap-3">
        {hostRows.length ? (
          <p className="text-sm text-muted-foreground">
            {passing} passing, {failing} failing.
          </p>
        ) : null}
        <CheckRowsTable rows={hostRows} running={running} />
      </div>
    </RunResultsTabs>
  );
}
function RunResultsTabs({
  children,
  errorRows,
}: {
  children: ReactNode;
  errorRows: LiveQueryRow[];
}) {
  return (
    <Tabs defaultValue="results">
      <TabsList>
        <TabsTrigger value="results">Results</TabsTrigger>
        <TabsTrigger value="errors" disabled={errorRows.length === 0}>
          Errors{errorRows.length ? ` ${errorRows.length}` : ""}
        </TabsTrigger>
      </TabsList>
      <TabsContent value="results">{children}</TabsContent>
      <TabsContent value="errors">
        <ErrorRowsTable rows={errorRows} />
      </TabsContent>
    </Tabs>
  );
}
function TargetPicker({
  selectedLabels,
  selectedHosts,
  onLabelsChange,
  onHostsChange,
}: {
  selectedLabels: Label[];
  selectedHosts: Host[];
  onLabelsChange: (labels: Label[]) => void;
  onHostsChange: (hosts: Host[]) => void;
}) {
  const labels = useLabels({
    per_page: MAX_PAGE_SIZE,
    sort: encodeSort("name"),
  });
  const [hostSearch, setHostSearch] = useState("");
  const hosts = useHosts({
    q: hostSearch,
    per_page: 8,
    sort: encodeSort("display_name"),
  });
  const labelRows = labels.data?.items ?? [];
  const hostRows = hosts.data?.items ?? [];
  return (
    <FieldSet className="max-w-3xl">
      <FieldLegend>Targets</FieldLegend>
      <FieldGroup>
        <LabelCombobox labels={labelRows} value={selectedLabels} onChange={onLabelsChange} />
        <HostCombobox
          hosts={hostRows}
          value={selectedHosts}
          inputValue={hostSearch}
          isFetching={hosts.isFetching}
          onInputValueChange={setHostSearch}
          onChange={onHostsChange}
        />
      </FieldGroup>
    </FieldSet>
  );
}
function LabelCombobox({
  labels,
  value,
  onChange,
}: {
  labels: Label[];
  value: Label[];
  onChange: (labels: Label[]) => void;
}) {
  const anchorRef = useComboboxAnchor();
  return (
    <Field>
      <FieldLabel htmlFor="live-label-targets">Labels</FieldLabel>
      <Combobox
        multiple
        items={labels}
        value={value}
        onValueChange={onChange}
        itemToStringLabel={(label) => label.name}
        itemToStringValue={(label) => String(label.id)}
        isItemEqualToValue={(label, selected) => label.id === selected.id}
      >
        <ComboboxChips ref={anchorRef}>
          <ComboboxValue>
            {(selected: Label[]) =>
              selected.map((label) => <ComboboxChip key={label.id}>{label.name}</ComboboxChip>)
            }
          </ComboboxValue>
          <ComboboxChipsInput id="live-label-targets" placeholder="Add label" />
          <ComboboxTrigger className="ml-auto" />
        </ComboboxChips>
        <ComboboxContent anchor={anchorRef}>
          <ComboboxEmpty>No Labels Found.</ComboboxEmpty>
          <ComboboxList>
            {labels.map((label) => (
              <ComboboxItem key={label.id} value={label}>
                {label.name}
              </ComboboxItem>
            ))}
          </ComboboxList>
        </ComboboxContent>
      </Combobox>
    </Field>
  );
}
function HostCombobox({
  hosts,
  value,
  inputValue,
  isFetching,
  onInputValueChange,
  onChange,
}: {
  hosts: Host[];
  value: Host[];
  inputValue: string;
  isFetching: boolean;
  onInputValueChange: (value: string) => void;
  onChange: (hosts: Host[]) => void;
}) {
  const anchorRef = useComboboxAnchor();
  const items = isFetching ? [] : hosts;
  return (
    <Field>
      <FieldLabel htmlFor="live-host-targets">Hosts</FieldLabel>
      <Combobox
        multiple
        items={items}
        filter={null}
        value={value}
        inputValue={inputValue}
        onInputValueChange={onInputValueChange}
        onValueChange={onChange}
        itemToStringLabel={(host) => host.display_name}
        itemToStringValue={(host) => String(host.id)}
        isItemEqualToValue={(host, selected) => host.id === selected.id}
      >
        <ComboboxChips ref={anchorRef}>
          <ComboboxValue>
            {(selected: Host[]) =>
              selected.map((host) => <ComboboxChip key={host.id}>{host.display_name}</ComboboxChip>)
            }
          </ComboboxValue>
          <ComboboxChipsInput id="live-host-targets" placeholder="Add host" />
          <ComboboxTrigger className="ml-auto" />
        </ComboboxChips>
        <ComboboxContent anchor={anchorRef}>
          <ComboboxEmpty>{isFetching ? "Searching hosts…" : "No Hosts Found."}</ComboboxEmpty>
          <ComboboxList>
            {items.map((host) => (
              <ComboboxItem key={host.id} value={host}>
                {host.display_name}
              </ComboboxItem>
            ))}
          </ComboboxList>
        </ComboboxContent>
      </Combobox>
    </Field>
  );
}
function ReportRowsTable({ rows, running }: { rows: ReportResultRow[]; running: boolean }) {
  const resultColumns = reportColumns(rows);
  const columns: ColumnDef<ReportResultRow>[] = [
    {
      accessorKey: "host_name",
      header: "Host",
      cell: ReportHostCell,
    },
    ...resultColumns.map<ColumnDef<ReportResultRow>>((name) => ({
      id: name,
      accessorFn: (row) => row[name] ?? "-",
      header: name,
      cell: ReportValueCell,
    })),
  ];
  return (
    <DataTableStatic
      columns={columns}
      data={rows}
      empty={<RunEmptyState text={running ? "Waiting for results" : "No rows returned"} />}
    />
  );
}
function CheckRowsTable({ rows, running }: { rows: CheckLiveRow[]; running: boolean }) {
  return (
    <DataTableStatic
      columns={checkResultColumns}
      data={rows}
      empty={<RunEmptyState text={running ? "Waiting for hosts" : "No host results yet"} />}
    />
  );
}
function ErrorRowsTable({ rows }: { rows: LiveQueryRow[] }) {
  return (
    <DataTableStatic
      columns={errorResultColumns}
      data={rows}
      empty={<RunEmptyState text="No errors yet" />}
    />
  );
}
function RunEmptyState({ text }: { text: string }) {
  return <PanelEmptyState className="border-0">{text}</PanelEmptyState>;
}
function reportResultRows(rows: LiveQueryRow[]) {
  return rows.flatMap((row) => {
    if (row.status !== "success") return [];
    return liveDataRows(row).map((data) => ({
      host_id: String(row.host_id ?? ""),
      host_name: row.host_name ?? "",
      ...data,
    }));
  });
}
function checkResultRows(rows: LiveQueryRow[]): CheckLiveRow[] {
  return rows.flatMap((row) => {
    if (row.status !== "success" || row.host_id === undefined) return [];
    return [
      {
        host_id: row.host_id,
        host_name: row.host_name,
        response: liveDataRows(row).length > 0 ? "pass" : "fail",
      },
    ];
  });
}
function liveErrorRows(rows: LiveQueryRow[]) {
  return rows.filter((row) => row.status !== "success" && row.status !== "stopped");
}
function liveDataRows(row: LiveQueryResult): Array<Record<string, string>> {
  if (Array.isArray(row.data)) {
    return row.data.filter(isRecord).map(stringRecord);
  }
  if (isRecord(row.data)) {
    return [stringRecord(row.data)];
  }
  return [];
}
function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
function stringRecord(row: Record<string, unknown>) {
  return Object.fromEntries(
    Object.entries(row).map(([key, value]) => [key, formatLiveValue(value)]),
  );
}
function formatLiveValue(value: unknown) {
  if (value === null || value === undefined) return "";
  if (typeof value === "object") return JSON.stringify(value);
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean" || typeof value === "bigint")
    return String(value);
  return "";
}
function reportColumns(rows: Array<Record<string, string>>) {
  const seen = new Set<string>();
  for (const row of rows) {
    Object.keys(row).forEach((key) => {
      if (key !== "host_id" && key !== "host_name") seen.add(key);
    });
  }
  return [...seen].toSorted((a, b) => a.localeCompare(b));
}
