import { Link } from "@tanstack/react-router";
import { Check, Loader2, Play, Plus, Square, X } from "lucide-react";
import type { ReactNode } from "react";
import { useEffect, useMemo, useState } from "react";

import { CheckStatusBadge } from "@/components/checks/check-status-badge";
import { DataTableSearch } from "@/components/data-table/data-table-search";
import { PlatformIcon } from "@/components/platform/platform-icons";
import { PageLead, ShowQueryButton } from "@/components/queries/query-ui";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import { PageTabs, PageTabsContent, PageTabsList, PageTabsTrigger } from "@/components/ui/page-tabs";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useHosts, type Host } from "@/hooks/use-hosts";
import { useLabels, type Label } from "@/hooks/use-labels";
import {
  useCreateLiveQuery,
  useLiveQueryStream,
  useLiveQueryTargetCount,
  useStopLiveQuery,
  type LiveQueryCreate,
  type LiveQueryResult,
  type LiveQueryRow,
  type LiveQueryTargetCount,
  type LiveQueryTargetCountBody,
} from "@/hooks/use-live-queries";

type LiveRunKind = "report" | "check";
type LiveRunStep = "targets" | "run";

export function LiveRunner({
  kind,
  itemId,
  name,
  sql,
  editAction,
}: {
  kind: LiveRunKind;
  itemId: string;
  name: string;
  sql: string;
  editAction: ReactNode;
}) {
  const create = useCreateLiveQuery();
  const stop = useStopLiveQuery();
  const [step, setStep] = useState<LiveRunStep>("targets");
  const [selectedLabels, setSelectedLabels] = useState<Label[]>([]);
  const [selectedHosts, setSelectedHosts] = useState<Host[]>([]);
  const [liveQueryId, setLiveQueryId] = useState("");
  const [runTargetCount, setRunTargetCount] = useState(0);
  const [stopRequested, setStopRequested] = useState(false);
  const stream = useLiveQueryStream(liveQueryId);
  const selectedHostIDs = useMemo(() => selectedHosts.map((host) => host.id), [selectedHosts]);
  const selectedLabelIDs = useMemo(() => selectedLabels.map((label) => label.id), [selectedLabels]);
  const targetCount = selectedLabels.length + selectedHosts.length;
  const hasTargets = targetCount > 0;
  const targetSelection = useMemo<LiveQueryTargetCountBody>(
    () => ({
      query_id: Number(itemId),
      selected: {
        hosts: selectedHostIDs,
        labels: selectedLabelIDs,
      },
    }),
    [itemId, selectedHostIDs, selectedLabelIDs],
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
    const body: LiveQueryCreate = {
      sql,
      query_id: targetSelection.query_id,
      selected: targetSelection.selected,
    };
    const handle = await create.mutateAsync(body);
    setRunTargetCount(handle.resolved_host_count);
    setLiveQueryId(String(handle.id));
    setStep("run");
  }

  async function stopRun() {
    if (!liveQueryId) return;
    setStopRequested(true);
    await stop.mutateAsync(Number(liveQueryId));
  }

  function backToTargets() {
    if (liveQueryId && stream.status === "running") {
      void stop.mutateAsync(Number(liveQueryId)).catch(() => undefined);
    }
    setStep("targets");
    setLiveQueryId("");
    setRunTargetCount(0);
    setStopRequested(false);
    create.reset();
    stop.reset();
  }

  const itemLabel = kind === "report" ? "report" : "check";
  const title = step === "targets" ? `Run ${name}` : `${name} live run`;

  return (
    <div className="flex flex-col gap-5 p-6">
      <PageLead
        title={title}
        description={
          step === "targets"
            ? `Choose labels or individual hosts, then run this ${itemLabel} once against online hosts.`
            : `Live ${itemLabel} results from this browser session.`
        }
        actions={
          <>
            <ShowQueryButton sql={sql} />
            {editAction}
          </>
        }
      />

      {create.error ? (
        <Alert variant="destructive">
          <AlertTitle>Run failed</AlertTitle>
          <AlertDescription>{create.error.message}</AlertDescription>
        </Alert>
      ) : null}
      {stop.error ? (
        <Alert variant="destructive">
          <AlertTitle>Stop failed</AlertTitle>
          <AlertDescription>{stop.error.message}</AlertDescription>
        </Alert>
      ) : null}

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
          onStop={() => void stopRun()}
          onRunAgain={() => void run()}
          onChangeTargets={backToTargets}
        />
      )}
    </div>
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
  metrics?: LiveQueryTargetCount;
  isLoading: boolean;
  error?: string;
  canRun: boolean;
  isStarting: boolean;
  onRun: () => void;
}) {
  return (
    <div className="grid content-start gap-4">
      <div>
        <h2 className="text-base font-semibold">Ready to run</h2>
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
  metrics?: LiveQueryTargetCount;
  isLoading: boolean;
  error?: string;
}) {
  if (!hasTargets) {
    return <p className="text-muted-foreground text-sm">Select a target to run.</p>;
  }
  if (isLoading) {
    return <p className="text-muted-foreground text-sm">Counting {selectedCount} selected target groups...</p>;
  }
  if (metrics) {
    const missing =
      metrics.targets_missing_in_action > 0 ? `, ${metrics.targets_missing_in_action} missing in action` : "";
    return (
      <p className="text-muted-foreground text-sm">
        {metrics.targets_count} target host{metrics.targets_count === 1 ? "" : "s"}: {metrics.targets_online} online,{" "}
        {metrics.targets_offline} offline{missing}.
      </p>
    );
  }
  if (error) {
    return <p className="text-destructive text-sm">{error}</p>;
  }
  return (
    <p className="text-muted-foreground text-sm">
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
  onStop: () => void;
  onRunAgain: () => void;
  onChangeTargets: () => void;
}) {
  const isRunning = status === "running";
  const finished = status === "completed";
  return (
    <div className="grid gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3 border-b pb-4">
        <div>
          <h2 className="text-base font-semibold">{runHeading(status, stopped)}</h2>
          <p className="text-muted-foreground text-sm">
            {respondedCount} of {targetCount} online host{targetCount === 1 ? "" : "s"} responded.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {isRunning || isStopping ? (
            <Button size="sm" variant="destructive" onClick={onStop} disabled={isStopping}>
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
              {finished ? "Run again" : "Run"}
            </Button>
          )}
          <Button size="sm" variant="outline" onClick={onChangeTargets}>
            <X data-icon="inline-start" />
            Change targets
          </Button>
        </div>
      </div>
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
  if (status === "error") return "Stream interrupted";
  return "Starting";
}

function ReportRunResults({ rows, running }: { rows: LiveQueryRow[]; running: boolean }) {
  const resultRows = reportResultRows(rows);
  const errorRows = liveErrorRows(rows);
  return (
    <PageTabs defaultValue="results">
      <PageTabsList>
        <PageTabsTrigger value="results">Results</PageTabsTrigger>
        <PageTabsTrigger value="errors" disabled={errorRows.length === 0}>
          Errors{errorRows.length ? ` ${errorRows.length}` : ""}
        </PageTabsTrigger>
      </PageTabsList>
      <PageTabsContent value="results">
        <ReportRowsTable rows={resultRows} running={running} />
      </PageTabsContent>
      <PageTabsContent value="errors">
        <ErrorRowsTable rows={errorRows} />
      </PageTabsContent>
    </PageTabs>
  );
}

function CheckRunResults({ rows, running }: { rows: LiveQueryRow[]; running: boolean }) {
  const hostRows = checkResultRows(rows);
  const errorRows = liveErrorRows(rows);
  const passing = hostRows.filter((row) => row.response === "pass").length;
  const failing = hostRows.filter((row) => row.response === "fail").length;
  return (
    <PageTabs defaultValue="results">
      <PageTabsList>
        <PageTabsTrigger value="results">Results</PageTabsTrigger>
        <PageTabsTrigger value="errors" disabled={errorRows.length === 0}>
          Errors{errorRows.length ? ` ${errorRows.length}` : ""}
        </PageTabsTrigger>
      </PageTabsList>
      <PageTabsContent value="results">
        <div className="grid gap-3">
          {hostRows.length ? (
            <p className="text-muted-foreground text-sm">
              {passing} passing, {failing} failing.
            </p>
          ) : null}
          <CheckRowsTable rows={hostRows} running={running} />
        </div>
      </PageTabsContent>
      <PageTabsContent value="errors">
        <ErrorRowsTable rows={errorRows} />
      </PageTabsContent>
    </PageTabs>
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
  const labels = useLabels({ per_page: 500, order_key: "name", order_direction: "asc" });
  const [hostSearch, setHostSearch] = useState("");
  const hosts = useHosts({ q: hostSearch, per_page: 8, order_key: "display_name", order_direction: "asc" });
  const grouped = useMemo(() => groupLabels(labels.data?.items ?? []), [labels.data?.items]);
  const hostRows = hosts.data?.items ?? [];

  useEffect(() => {
    if (selectedLabels.length > 0 || selectedHosts.length > 0) return;
    const allHosts = grouped.allHosts.at(0);
    if (allHosts) onLabelsChange([allHosts]);
  }, [grouped.allHosts, onLabelsChange, selectedHosts.length, selectedLabels.length]);

  function toggleLabel(label: Label) {
    const isAllHosts = label.label_type === "builtin" && label.name === "All Hosts";
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
    onLabelsChange([...selectedLabels.filter((item) => item.name !== "All Hosts"), label]);
  }

  function addHost(host: Host) {
    if (selectedHosts.some((item) => item.id === host.id)) return;
    onLabelsChange(selectedLabels.filter((item) => item.name !== "All Hosts"));
    onHostsChange([...selectedHosts, host]);
    setHostSearch("");
  }

  return (
    <div className="grid content-start gap-5 rounded-md border p-4">
      <div>
        <h2 className="text-base font-semibold">Select targets</h2>
        <p className="text-muted-foreground text-sm">Use system labels, custom labels, or exact hosts.</p>
      </div>

      <TargetSection title="" labels={grouped.allHosts} selected={selectedLabels} onToggle={toggleLabel} />
      <TargetSection title="Platforms" labels={grouped.platforms} selected={selectedLabels} onToggle={toggleLabel} />
      <TargetSection title="Labels" labels={grouped.other} selected={selectedLabels} onToggle={toggleLabel} />

      <div className="grid gap-2">
        <h3 className="text-sm font-medium">Hosts</h3>
        <DataTableSearch
          value={hostSearch}
          onChange={setHostSearch}
          placeholder="Search hosts"
          label="Search hosts"
          className="max-w-none"
        />
        {hostSearch ? (
          <div className="grid gap-1 rounded-md border p-2">
            {hosts.isFetching ? (
              <div className="text-muted-foreground flex items-center gap-2 px-2 py-2 text-sm">
                <Loader2 className="size-3.5 animate-spin" /> Searching hosts...
              </div>
            ) : hostRows.length ? (
              hostRows.map((host) => (
                <button
                  type="button"
                  key={host.id}
                  className="hover:bg-muted flex items-center justify-between rounded px-2 py-2 text-left text-sm"
                  onClick={() => addHost(host)}
                >
                  <span>{host.display_name || host.hostname || String(host.id)}</span>
                  <span className="text-muted-foreground text-xs">{host.platform || "unknown"}</span>
                </button>
              ))
            ) : (
              <div className="text-muted-foreground px-2 py-2 text-sm">No matching hosts.</div>
            )}
          </div>
        ) : null}
        {selectedHosts.length ? (
          <div className="flex flex-wrap gap-2">
            {selectedHosts.map((host) => (
              <TargetChip
                key={host.id}
                selected
                label={host.display_name || host.hostname || String(host.id)}
                onClick={() => onHostsChange(selectedHosts.filter((item) => item.id !== host.id))}
              />
            ))}
          </div>
        ) : null}
      </div>
    </div>
  );
}

function TargetSection({
  title,
  labels,
  selected,
  onToggle,
}: {
  title: string;
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
            icon={targetIcon(label)}
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
  icon,
}: {
  label: string;
  selected: boolean;
  onClick: () => void;
  icon?: ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      data-selected={selected}
      className="border-input data-[selected=true]:bg-primary data-[selected=true]:text-primary-foreground hover:bg-muted inline-flex h-8 items-center rounded-full border px-3 text-sm transition-colors data-[selected=true]:border-primary"
    >
      {selected ? <Check className="mr-1 size-3.5" /> : <Plus className="mr-1 size-3.5" />}
      {icon ? <span className="mr-1 inline-flex size-3.5 items-center">{icon}</span> : null}
      {label}
    </button>
  );
}

function ReportRowsTable({ rows, running }: { rows: Array<Record<string, string>>; running: boolean }) {
  const columns = reportColumns(rows);
  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Host</TableHead>
            {columns.map((name) => (
              <TableHead key={name}>{name}</TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.length === 0 ? (
            <TableRow>
              <TableCell colSpan={Math.max(columns.length + 1, 1)} className="h-40">
                <RunEmptyState
                  title={running ? "Waiting for results" : "No rows returned"}
                  description={
                    running
                      ? "Results will appear as online hosts respond."
                      : "The report ran successfully but did not return any rows."
                  }
                />
              </TableCell>
            </TableRow>
          ) : (
            rows.map((row) => (
              <TableRow key={`${row.host_id}-${JSON.stringify(row)}`}>
                <TableCell className="font-medium">
                  <Link to="/hosts/$hostId" params={{ hostId: row.host_id }} className="hover:underline">
                    {row.host_name}
                  </Link>
                </TableCell>
                {columns.map((name) => (
                  <TableCell key={name}>{row[name] ?? "-"}</TableCell>
                ))}
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </div>
  );
}

function CheckRowsTable({ rows, running }: { rows: CheckLiveRow[]; running: boolean }) {
  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Host</TableHead>
            <TableHead>Result</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.length === 0 ? (
            <TableRow>
              <TableCell colSpan={2} className="h-40">
                <RunEmptyState
                  title={running ? "Waiting for hosts" : "No host results"}
                  description={
                    running
                      ? "Check results will appear as online hosts respond."
                      : "No hosts returned a check result for this run."
                  }
                />
              </TableCell>
            </TableRow>
          ) : (
            rows.map((row) => (
              <TableRow key={row.host_id}>
                <TableCell className="font-medium">
                  <Link to="/hosts/$hostId" params={{ hostId: String(row.host_id) }} className="hover:underline">
                    {row.host_name ?? row.host_id}
                  </Link>
                </TableCell>
                <TableCell>
                  <CheckStatusBadge response={row.response} />
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </div>
  );
}

function ErrorRowsTable({ rows }: { rows: LiveQueryRow[] }) {
  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Host</TableHead>
            <TableHead>Error</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.length === 0 ? (
            <TableRow>
              <TableCell colSpan={2} className="h-40">
                <RunEmptyState title="No errors" description="No host errors have been reported for this run." />
              </TableCell>
            </TableRow>
          ) : (
            rows.map((row) => (
              <TableRow key={row._seq}>
                <TableCell>{row.host_name ?? row.host_id ?? "?"}</TableCell>
                <TableCell className="text-destructive">{row.error ?? row.status}</TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </div>
  );
}

function RunEmptyState({ title, description }: { title: string; description: string }) {
  return (
    <Empty className="border-0 p-0">
      <EmptyHeader>
        <EmptyTitle>{title}</EmptyTitle>
        <EmptyDescription>{description}</EmptyDescription>
      </EmptyHeader>
    </Empty>
  );
}

function groupLabels(labels: Label[]) {
  return {
    allHosts: labels.filter((label) => label.label_type === "builtin" && label.name === "All Hosts"),
    platforms: labels.filter(isPrimaryPlatformLabel),
    other: labels.filter((label) => label.name !== "All Hosts" && !isPrimaryPlatformLabel(label)),
  };
}

function displayLabel(label: Label) {
  return label.name;
}

function isPrimaryPlatformLabel(label: Label) {
  return (
    label.label_type === "builtin" &&
    (label.name === "macOS" || label.name === "Windows" || label.name === "Linux")
  );
}

function targetIcon(label: Label): ReactNode {
  switch (label.name) {
    case "macOS":
      return <PlatformIcon platform="darwin" className="size-3.5" />;
    case "Windows":
      return <PlatformIcon platform="windows" className="size-3.5" />;
    case "Linux":
      return <PlatformIcon platform="linux" className="size-3.5" />;
    default:
      return undefined;
  }
}

function reportResultRows(rows: LiveQueryRow[]) {
  return rows.flatMap((row) => {
    if (row.status !== "success") return [];
    return liveDataRows(row).map((data) => ({
      host_id: String(row.host_id ?? ""),
      host_name: row.host_name ?? String(row.host_id ?? ""),
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
  return Object.fromEntries(Object.entries(row).map(([key, value]) => [key, formatLiveValue(value)]));
}

function formatLiveValue(value: unknown) {
  if (value === null || value === undefined) return "";
  if (typeof value === "object") return JSON.stringify(value);
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean" || typeof value === "bigint") return String(value);
  return "";
}

function reportColumns(rows: Array<Record<string, string>>) {
  const seen = new Set<string>();
  for (const row of rows) {
    Object.keys(row).forEach((key) => {
      if (key !== "host_id" && key !== "host_name") seen.add(key);
    });
  }
  return [...seen].sort((a, b) => a.localeCompare(b));
}
