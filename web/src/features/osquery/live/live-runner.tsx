import { Play, Square, X } from "lucide-react";
import { useMemo, useState } from "react";

import { AsyncButton } from "@components/async-button";
import { ConfirmDialog } from "@components/confirm-dialog";
import { DataTableStatic } from "@components/data-table/data-table-static";
import { encodeSort } from "@components/data-table/use-data-table-search";
import { PageHeader, PageShell } from "@components/layout/page-layout";
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
import { useAuth } from "@features/auth/queries";
import { useHosts } from "@features/hosts/queries";
import { useLabels } from "@features/labels/queries";
import {
  createCheckResultColumns,
  type CheckResultRow,
} from "@features/osquery/checks/query-results";
import {
  createReportResultColumns,
  type ReportResultRow,
  resultColumnNames,
  SnapshotResultRows,
} from "@features/osquery/reports/query-results";
import type { Host, Label } from "@lib/api";
import { MAX_PAGE_SIZE } from "@lib/pagination";

import {
  type LiveQuerySnapshot,
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
const liveReportResultColumns = createReportResultColumns({
  timestamp: "reported",
  includeError: true,
});
const liveCheckResultColumns = createCheckResultColumns({
  timestampHeader: "Last Evaluated",
  includeError: true,
});

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
  const respondedCount = stream.snapshots.filter(
    (snapshot) => snapshot.status === "collected" || snapshot.status === "error",
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
          snapshots={stream.snapshots}
          status={stream.status}
          error={stream.error}
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
  snapshots,
  status,
  error,
  stopped,
  isStopping,
  targetCount,
  respondedCount,
  onStop,
  onRunAgain,
  onChangeTargets,
}: {
  kind: LiveRunKind;
  snapshots: LiveQuerySnapshot[];
  status: string;
  error?: string;
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
          {error ? <p className="text-sm text-destructive">{error}</p> : null}
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
        <ReportRunResults snapshots={snapshots} />
      ) : (
        <CheckRunResults snapshots={snapshots} />
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
function ReportRunResults({ snapshots }: { snapshots: LiveQuerySnapshot[] }) {
  const rows = useMemo(() => snapshots.map(reportResultFromSnapshot), [snapshots]);
  const columnNames = useMemo(() => resultColumnNames(rows.flatMap((row) => row.rows)), [rows]);
  return (
    <DataTableStatic
      columns={liveReportResultColumns}
      data={rows}
      getRowCanExpand={(row) => row.original.rows.length > 0}
      getRowId={(row) => String(row.host_id)}
      renderSubRow={(row) => (
        <SnapshotResultRows rows={row.original.rows} columnNames={columnNames} />
      )}
      empty={<RunEmptyState text="No targeted hosts" />}
    />
  );
}
function CheckRunResults({ snapshots }: { snapshots: LiveQuerySnapshot[] }) {
  const rows = useMemo(() => snapshots.map(checkResultFromSnapshot), [snapshots]);
  return (
    <DataTableStatic
      columns={liveCheckResultColumns}
      data={rows}
      getRowId={(row) => String(row.host_id)}
      empty={<RunEmptyState text="No targeted hosts" />}
    />
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
function RunEmptyState({ text }: { text: string }) {
  return <PanelEmptyState className="border-0">{text}</PanelEmptyState>;
}

function reportResultFromSnapshot(snapshot: LiveQuerySnapshot): ReportResultRow {
  return {
    host_id: snapshot.host_id,
    host_name: snapshot.host_name,
    status: snapshot.status,
    rows: snapshot.rows,
    result_row_count: snapshot.rows.length,
    returned_row_count: snapshot.rows.length,
    updated_at:
      snapshot.status === "collected" || snapshot.status === "error"
        ? snapshot.updated_at
        : undefined,
    error: snapshot.error,
  };
}

function checkResultFromSnapshot(snapshot: LiveQuerySnapshot): CheckResultRow {
  return {
    host_id: snapshot.host_id,
    host_name: snapshot.host_name,
    status:
      snapshot.status === "collected"
        ? snapshot.rows.length > 0
          ? "pass"
          : "fail"
        : snapshot.status,
    updated_at:
      snapshot.status === "collected" || snapshot.status === "error"
        ? snapshot.updated_at
        : undefined,
    error: snapshot.error,
  };
}
