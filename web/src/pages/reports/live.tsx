import { Link, useParams } from "@tanstack/react-router";
import { Apple, Check, Globe, Loader2, Monitor, Play, Plus, Search, Server, X } from "lucide-react";
import type { ComponentType, SVGProps } from "react";
import { useEffect, useMemo, useState } from "react";

import { BackLink, PageLead, ShowQueryButton } from "@/components/queries/query-ui";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useHosts, type Host } from "@/hooks/use-hosts";
import { useLabels, type Label } from "@/hooks/use-labels";
import {
  useCreateLiveQuery,
  useLiveQueryStream,
  type LiveQueryCreate,
  type LiveQueryRow,
} from "@/hooks/use-live-queries";
import { useQueryDetail } from "@/hooks/use-queries";

export function ReportLivePage() {
  const { reportId } = useParams({ from: "/_authenticated/reports/$reportId" });
  const report = useQueryDetail(reportId);

  if (report.error) {
    return (
      <div className="p-6">
        <Alert variant="destructive">
          <AlertTitle>Failed to load report</AlertTitle>
          <AlertDescription>{report.error.message}</AlertDescription>
        </Alert>
      </div>
    );
  }
  if (!report.data) {
    return (
      <div className="text-muted-foreground flex items-center gap-2 p-6 text-sm">
        <Loader2 className="size-4 animate-spin" /> Loading report...
      </div>
    );
  }

  return <LiveReportRunner reportId={reportId} name={report.data.name} sql={report.data.query} />;
}

function LiveReportRunner({ reportId, name, sql }: { reportId: string; name: string; sql: string }) {
  const create = useCreateLiveQuery();
  const [selectedLabels, setSelectedLabels] = useState<Label[]>([]);
  const [selectedHosts, setSelectedHosts] = useState<Host[]>([]);
  const [liveQueryId, setLiveQueryId] = useState("");
  const stream = useLiveQueryStream(liveQueryId);
  const targetCount = selectedLabels.length + selectedHosts.length;
  const hasTargets = targetCount > 0;

  async function run() {
    const body: LiveQueryCreate = {
      sql,
      query_id: Number(reportId),
      selected: {
        hosts: selectedHosts.map((host) => host.id),
        labels: selectedLabels.map((label) => label.id),
      },
    };
    const handle = await create.mutateAsync(body);
    setLiveQueryId(String(handle.id));
  }

  return (
    <div className="flex flex-col gap-5 p-6">
      <BackLink to={`/reports/${reportId}`}>Back to report</BackLink>
      <PageLead
        title={`Run ${name}`}
        description="Choose labels or individual hosts, then run this report once against online hosts."
        actions={
          <>
            <ShowQueryButton sql={sql} />
            <Button asChild variant="outline" size="sm">
              <Link to="/reports/$reportId/edit" params={{ reportId }}>
                Edit report
              </Link>
            </Button>
          </>
        }
      />

      {create.error ? (
        <Alert variant="destructive">
          <AlertTitle>Run failed</AlertTitle>
          <AlertDescription>{create.error.message}</AlertDescription>
        </Alert>
      ) : null}

      <div className="grid gap-6 lg:grid-cols-[minmax(20rem,24rem)_1fr]">
        <TargetPicker
          selectedLabels={selectedLabels}
          selectedHosts={selectedHosts}
          onLabelsChange={setSelectedLabels}
          onHostsChange={setSelectedHosts}
        />
        <div className="grid content-start gap-3">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <h2 className="text-base font-semibold">Results</h2>
              <p className="text-muted-foreground text-sm">
                {create.data
                  ? `Targeted ${create.data.resolved_host_count} host${create.data.resolved_host_count === 1 ? "" : "s"}.`
                  : hasTargets
                    ? `${targetCount} target${targetCount === 1 ? "" : "s"} selected.`
                    : "Select a target to run this report."}
              </p>
            </div>
            <Button
              size="sm"
              onClick={() => void run()}
              disabled={!hasTargets || create.isPending || stream.status === "open"}
            >
              {create.isPending || stream.status === "open" ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                <Play className="size-4" />
              )}
              {liveQueryId ? "Run again" : "Run"}
            </Button>
          </div>
          <LiveResults rows={stream.results} status={stream.status} />
        </div>
      </div>
    </div>
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
        <div className="relative">
          <Search className="text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2" />
          <Input
            value={hostSearch}
            onChange={(event) => setHostSearch(event.target.value)}
            placeholder="Search hosts"
            className="pr-8 pl-8"
          />
          {hostSearch ? (
            <button
              type="button"
              onClick={() => setHostSearch("")}
              className="text-muted-foreground hover:text-foreground absolute top-1/2 right-2 -translate-y-1/2 rounded p-0.5"
              aria-label="Clear host search"
            >
              <X className="size-3.5" />
            </button>
          ) : null}
        </div>
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
  icon: Icon,
}: {
  label: string;
  selected: boolean;
  onClick: () => void;
  icon?: ComponentType<SVGProps<SVGSVGElement>>;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      data-selected={selected}
      className="border-input data-[selected=true]:bg-primary data-[selected=true]:text-primary-foreground hover:bg-muted inline-flex h-8 items-center rounded-full border px-3 text-sm transition-colors data-[selected=true]:border-primary"
    >
      {selected ? <Check className="mr-1 size-3.5" /> : <Plus className="mr-1 size-3.5" />}
      {Icon ? <Icon className="mr-1 size-3.5" /> : null}
      {label}
    </button>
  );
}

function LiveResults({ rows, status }: { rows: LiveQueryRow[]; status: string }) {
  const columnNames = liveColumnNames(rows);
  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Host</TableHead>
            <TableHead>Status</TableHead>
            {columnNames.map((name) => (
              <TableHead key={name}>{name}</TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.length === 0 ? (
            <TableRow>
              <TableCell
                colSpan={Math.max(2 + columnNames.length, 2)}
                className="text-muted-foreground h-28 text-center"
              >
                {status === "open" ? "Waiting for hosts..." : "Run this report to see live results."}
              </TableCell>
            </TableRow>
          ) : (
            rows.map((row) => (
              <TableRow key={row._seq}>
                <TableCell>{row.host_name ?? row.host_id ?? "?"}</TableCell>
                <TableCell>
                  <Badge variant={statusVariant(row.status)}>{row.status}</Badge>
                  {row.error ? <div className="text-destructive mt-1 text-xs">{row.error}</div> : null}
                </TableCell>
                {columnNames.map((name) => (
                  <TableCell key={name}>{liveValue(row, name)}</TableCell>
                ))}
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </div>
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
    (label.name === "macOS" || label.name === "Windows" || label.name === "Linux" || label.name === "ChromeOS")
  );
}

function targetIcon(label: Label): ComponentType<SVGProps<SVGSVGElement>> | undefined {
  switch (label.name) {
    case "macOS":
      return Apple;
    case "Windows":
      return Monitor;
    case "Linux":
      return Server;
    case "ChromeOS":
      return Globe;
    default:
      return undefined;
  }
}

function liveColumnNames(rows: LiveQueryRow[]) {
  const seen = new Set<string>();
  for (const row of rows) {
    const values = liveData(row);
    Object.keys(values).forEach((key) => seen.add(key));
  }
  return [...seen].sort((a, b) => a.localeCompare(b));
}

function liveValue(row: LiveQueryRow, name: string) {
  return liveData(row)[name] ?? "-";
}

function liveData(row: LiveQueryRow): Record<string, string> {
  if (!row.data || typeof row.data !== "object" || Array.isArray(row.data)) return {};
  return Object.fromEntries(Object.entries(row.data).map(([key, value]) => [key, String(value)]));
}

function statusVariant(status: string): "default" | "secondary" | "destructive" | "outline" {
  switch (status) {
    case "success":
      return "default";
    case "error":
    case "timeout":
      return "destructive";
    default:
      return "outline";
  }
}
