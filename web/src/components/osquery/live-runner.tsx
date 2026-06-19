import { Link } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Check, Loader2, Play, Plus, Square, X } from "lucide-react";
import type { ReactNode } from "react";
import { useEffect, useMemo, useState } from "react";

import { DataTableStatic } from "@/components/data-table/data-table-static";
import { EmptyPanel } from "@/components/empty-panel";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { CheckStatusBadge } from "@/components/osquery/checks/check-status-badge";
import { ShowQueryButton } from "@/components/queries/query-ui";
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
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useAuth } from "@/hooks/use-auth";
import { encodeSort } from "@/hooks/use-data-table-search";
import { type Host, useHosts } from "@/hooks/use-hosts";
import { type Label, useLabels } from "@/hooks/use-labels";
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
} from "@/hooks/use-live-queries";
import { isAllHostsLabel } from "@/lib/labels";
import { MAX_PAGE_SIZE } from "@/lib/pagination";

type LiveRunKind = "report" | "check";
type LiveRunStep = "targets" | "run";
type ReportResultRow = Record<string, string>;

export function LiveRunner({
  kind,
  itemId,
  name,
  sql,
  editAction,
}: {
  kind: LiveRunKind;
  itemId: number;
  name: string;
  sql: string;
  editAction: ReactNode;
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
  const targetCount = selectedLabels.length + selectedHosts.length;
  const hasTargets = targetCount > 0;
  const targetSelection = useMemo<OsqueryLiveQueryTargetCountBody>(
    () => ({
      report_id: kind === "report" ? itemId : undefined,
      selected: {
        hosts: selectedHostIDs,
        labels: selectedLabelIDs,
      },
    }),
    [itemId, kind, selectedHostIDs, selectedLabelIDs],
  );
  const targetMetrics = useLiveQueryTargetCount(targetSelection, hasTargets);
  const isRunning = stream.status === "running";
  const isStarting = create.isPending;
  const isStopping = stop.isPending;
  const respondedCount = stream.results.filter((row) => row.status !== "stopped").length;
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
      report_id: targetSelection.report_id,
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
  const title = step === "targets" ? `Run ${name}` : `${name} Live Run`;

  if (!isAdmin) {
    return (
      <PageShell>
        <PageHeader
          title={name}
          description={`Live ${itemLabel} execution is admin-only.`}
          actions={<ShowQueryButton sql={sql} />}
        />
        <EmptyPanel>Live execution is unavailable for this account.</EmptyPanel>
      </PageShell>
    );
  }

  return (
    <PageShell>
      <PageHeader
        title={title}
        description={
          step === "targets"
            ? `Run this ${itemLabel} against online hosts.`
            : `Live ${itemLabel} results.`
        }
        actions={
          <>
            <ShowQueryButton sql={sql} />
            {editAction}
          </>
        }
      />

      {step === "targets" ? (
        <div className="grid gap-6 lg:grid-cols-[minmax(20rem,24rem)_1fr]">
          <TargetPicker
            selectedLabels={selectedLabels}
            selectedHosts={selectedHosts}
            onLabelsChange={setSelectedLabels}
            onHostsChange={setSelectedHosts}
          />
          <TargetRunPanel
            selectedCount={targetCount}
            hasTargets={hasTargets}
            metrics={targetMetrics.data}
            isLoading={targetMetrics.isFetching}
            error={targetMetrics.error?.message}
            canRun={canRun}
            isStarting={isStarting}
            onRun={() => void run()}
          />
        </div>
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

function TargetRunPanel({
  selectedCount,
  hasTargets,
  metrics,
  isLoading,
  error,
  canRun,
  isStarting,
  onRun,
}: {
  selectedCount: number;
  hasTargets: boolean;
  metrics?: OsqueryLiveQueryTargetCountOutputBody;
  isLoading: boolean;
  error?: string;
  canRun: boolean;
  isStarting: boolean;
  onRun: () => void;
}) {
  return (
    <div className="grid content-start gap-4">
      <div>
        <h2 className="text-base font-semibold">Ready to Run</h2>
        <TargetSummary
          selectedCount={selectedCount}
          hasTargets={hasTargets}
          metrics={metrics}
          isLoading={isLoading}
          error={error}
        />
      </div>
      <div className="rounded-md border p-4">
        <Button onClick={onRun} disabled={!canRun}>
          {isStarting ? (
            <Loader2 data-icon="inline-start" className="animate-spin" />
          ) : (
            <Play data-icon="inline-start" />
          )}
          {isStarting ? "Starting" : "Run"}
        </Button>
      </div>
    </div>
  );
}

function TargetSummary({
  selectedCount,
  hasTargets,
  metrics,
  isLoading,
  error,
}: {
  selectedCount: number;
  hasTargets: boolean;
  metrics?: OsqueryLiveQueryTargetCountOutputBody;
  isLoading: boolean;
  error?: string;
}) {
  if (!hasTargets) {
    return <p className="text-sm text-muted-foreground">Select a target to run.</p>;
  }
  if (isLoading) {
    return (
      <p className="text-sm text-muted-foreground">
        Counting {selectedCount} selected target groups...
      </p>
    );
  }
  if (metrics) {
    return (
      <p className="text-sm text-muted-foreground">
        {metrics.targets_count} target host{metrics.targets_count === 1 ? "" : "s"}:{" "}
        {metrics.targets_online} online, {metrics.targets_offline} offline.
      </p>
    );
  }
  if (error) {
    return <p className="text-sm text-destructive">{error}</p>;
  }
  return (
    <p className="text-sm text-muted-foreground">
      {selectedCount} target{selectedCount === 1 ? "" : "s"} selected.
    </p>
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
          <h2 className="text-base font-semibold">{runHeading(status, stopped)}</h2>
          <p className="text-sm text-muted-foreground">
            {respondedCount} of {targetCount} online host{targetCount === 1 ? "" : "s"} responded.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {isRunning || isStopping ? (
            <Button
              size="sm"
              variant="destructive"
              onClick={() => setStopOpen(true)}
              disabled={isStopping}
            >
              {isStopping ? (
                <Loader2 data-icon="inline-start" className="animate-spin" />
              ) : (
                <Square data-icon="inline-start" />
              )}
              {isStopping ? "Stopping" : "Stop"}
            </Button>
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
      <AlertDialog open={stopOpen} onOpenChange={setStopOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Stop Live Run?</AlertDialogTitle>
            <AlertDialogDescription>
              Hosts that have not responded yet will be marked stopped for this live run.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="ghost" size="sm" disabled={isStopping}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              size="sm"
              disabled={isStopping}
              onClick={(event) => {
                event.preventDefault();
                void confirmStop();
              }}
            >
              Stop
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
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
  const labels = useLabels({ per_page: MAX_PAGE_SIZE, sort: encodeSort("name") });
  const [hostSearch, setHostSearch] = useState("");
  const hosts = useHosts({ q: hostSearch, per_page: 8, sort: encodeSort("display_name") });
  const grouped = useMemo(() => groupLabels(labels.data?.items ?? []), [labels.data?.items]);
  const hostRows = hosts.data?.items ?? [];

  useEffect(() => {
    if (selectedLabels.length > 0 || selectedHosts.length > 0) return;
    const allHosts = grouped.allHosts.at(0);
    if (allHosts) onLabelsChange([allHosts]);
  }, [grouped.allHosts, onLabelsChange, selectedHosts.length, selectedLabels.length]);

  function toggleLabel(label: Label) {
    const isAllHosts = isAllHostsLabel(label);
    const alreadySelected = selectedLabels.some((item) => item.id === label.id);
    if (alreadySelected) {
      onLabelsChange(selectedLabels.filter((item) => item.id !== label.id));
      return;
    }
    if (isAllHosts) {
      onHostsChange([]);
      onLabelsChange([label]);
      return;
    }
    onLabelsChange([...selectedLabels.filter((item) => !isAllHostsLabel(item)), label]);
  }

  function addHost(host: Host) {
    if (selectedHosts.some((item) => item.id === host.id)) return;
    onLabelsChange(selectedLabels.filter((item) => !isAllHostsLabel(item)));
    onHostsChange([...selectedHosts, host]);
    setHostSearch("");
  }

  return (
    <div className="grid content-start gap-5 rounded-md border p-4">
      <div>
        <h2 className="text-base font-semibold">Select Targets</h2>
      </div>

      <LabelChipGroup labels={grouped.allHosts} selected={selectedLabels} onToggle={toggleLabel} />
      <LabelChipGroup
        title="Labels"
        labels={grouped.other}
        selected={selectedLabels}
        onToggle={toggleLabel}
      />

      <div className="grid gap-2">
        <h3 className="text-sm font-medium">Hosts</h3>
        <Input
          value={hostSearch}
          onChange={(event) => setHostSearch(event.target.value)}
          placeholder="Search hosts"
          className="h-8"
        />
        {hostSearch ? (
          <div className="grid gap-1 rounded-md border p-2">
            {hosts.isFetching ? (
              <div className="flex items-center gap-2 px-2 py-2 text-sm text-muted-foreground">
                <Loader2 className="size-3.5 animate-spin" /> Searching Hosts...
              </div>
            ) : hostRows.length ? (
              hostRows.map((host) => (
                <button
                  type="button"
                  key={host.id}
                  className="flex items-center justify-between rounded-sm px-2 py-2 text-left text-sm hover:bg-muted"
                  onClick={() => addHost(host)}
                >
                  <span>{host.display_name}</span>
                </button>
              ))
            ) : (
              <div className="px-2 py-2 text-sm text-muted-foreground">No Hosts Found.</div>
            )}
          </div>
        ) : null}
        {selectedHosts.length ? (
          <div className="flex flex-wrap gap-2">
            {selectedHosts.map((host) => (
              <TargetChip
                key={host.id}
                selected
                label={host.display_name}
                onClick={() => onHostsChange(selectedHosts.filter((item) => item.id !== host.id))}
              />
            ))}
          </div>
        ) : null}
      </div>
    </div>
  );
}

function LabelChipGroup({
  title,
  labels,
  selected,
  onToggle,
}: {
  title?: string;
  labels: Label[];
  selected: Label[];
  onToggle: (label: Label) => void;
}) {
  if (labels.length === 0) return null;
  return (
    <div className="grid gap-2">
      {title ? <h3 className="text-sm font-medium">{title}</h3> : null}
      <div className="flex flex-wrap gap-2">
        {labels.map((label) => (
          <TargetChip
            key={label.id}
            selected={selected.some((item) => item.id === label.id)}
            label={displayLabel(label)}
            onClick={() => onToggle(label)}
          />
        ))}
      </div>
    </div>
  );
}

function TargetChip({
  label,
  selected,
  onClick,
}: {
  label: string;
  selected: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      data-selected={selected}
      className="inline-flex h-8 items-center rounded-full border border-input px-3 text-sm transition-colors hover:bg-muted data-[selected=true]:border-primary data-[selected=true]:bg-primary data-[selected=true]:text-primary-foreground"
    >
      {selected ? <Check className="mr-1 size-3.5" /> : <Plus className="mr-1 size-3.5" />}
      {label}
    </button>
  );
}

function ReportRowsTable({ rows, running }: { rows: ReportResultRow[]; running: boolean }) {
  const resultColumns = reportColumns(rows);
  const columns: ColumnDef<ReportResultRow>[] = [
    {
      accessorKey: "host_name",
      header: "Host",
      cell: ({ row }) => (
        <Link
          to="/hosts/$hostId"
          params={{ hostId: row.original.host_id }}
          className="whitespace-nowrap hover:underline"
        >
          {row.original.host_name}
        </Link>
      ),
    },
    ...resultColumns.map<ColumnDef<ReportResultRow>>((name) => ({
      id: name,
      accessorFn: (row) => row[name] ?? "-",
      header: name,
      cell: ({ row }) => <span className="whitespace-nowrap">{row.original[name] ?? "-"}</span>,
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
  const columns: ColumnDef<CheckLiveRow>[] = [
    {
      accessorKey: "host_name",
      header: "Host",
      cell: ({ row }) => (
        <Link
          to="/hosts/$hostId"
          params={{ hostId: String(row.original.host_id) }}
          className="hover:underline"
        >
          {row.original.host_name}
        </Link>
      ),
    },
    {
      accessorKey: "response",
      header: "Result",
      cell: ({ row }) => <CheckStatusBadge response={row.original.response} />,
    },
  ];
  return (
    <DataTableStatic
      columns={columns}
      data={rows}
      empty={<RunEmptyState text={running ? "Waiting for hosts" : "No host results yet"} />}
    />
  );
}

function ErrorRowsTable({ rows }: { rows: LiveQueryRow[] }) {
  const columns: ColumnDef<LiveQueryRow>[] = [
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
  return (
    <DataTableStatic columns={columns} data={rows} empty={<RunEmptyState text="No errors yet" />} />
  );
}

function RunEmptyState({ text }: { text: string }) {
  return <EmptyPanel className="border-0">{text}</EmptyPanel>;
}

function groupLabels(labels: Label[]) {
  return {
    allHosts: labels.filter(isAllHostsLabel),
    other: labels.filter((label) => !isAllHostsLabel(label)),
  };
}

function displayLabel(label: Label) {
  return label.name;
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

type CheckLiveRow = {
  host_id: number;
  host_name?: string;
  response: "pass" | "fail";
};

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
  return rows.filter((row) => row.status === "error");
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
