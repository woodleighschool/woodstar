import { Link, useNavigate, useParams } from "@tanstack/react-router";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { ChevronDown, ChevronRight, Loader2, PanelRightOpen } from "lucide-react";
import { useRef, useState } from "react";

import { SchemaSidebar } from "@/components/editor/schema-sidebar";
import { SQLEditor } from "@/components/editor/sql-editor";
import { LabelScopeSelector } from "@/components/queries/label-scope-selector";
import { BackLink, PageLead } from "@/components/queries/query-ui";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { useCreateQuery, useQueryDetail, useUpdateQuery, type QueryMutation } from "@/hooks/use-queries";
import { cn } from "@/lib/utils";

const FREQUENCY_OPTIONS: { value: number; label: string }[] = [
  { value: 0, label: "Never" },
  { value: 300, label: "Every 5 minutes" },
  { value: 600, label: "Every 10 minutes" },
  { value: 900, label: "Every 15 minutes" },
  { value: 1800, label: "Every 30 minutes" },
  { value: 3600, label: "Every hour" },
  { value: 21600, label: "Every 6 hours" },
  { value: 43200, label: "Every 12 hours" },
  { value: 86400, label: "Every day" },
  { value: 604800, label: "Every week" },
];

const PLATFORM_OPTIONS = [
  { value: "all", label: "All platforms" },
  { value: "darwin", label: "macOS" },
];

const emptyQuery: QueryMutation = {
  name: "",
  description: "",
  query: "select * from os_version;",
  schedule_interval: 0,
  label_scope: {},
};

export function ReportEditPage({ mode }: { mode: "create" | "edit" }) {
  const params = useParams({ strict: false });
  const reportId = params.reportId ?? "";
  const detail = useQueryDetail(reportId);

  if (mode === "edit") {
    if (detail.error) {
      return (
        <div className="p-6">
          <Alert variant="destructive">
            <AlertTitle>Failed to load report</AlertTitle>
            <AlertDescription>{detail.error.message}</AlertDescription>
          </Alert>
        </div>
      );
    }
    if (!detail.data) {
      return (
        <div className="text-muted-foreground flex items-center gap-2 p-6 text-sm">
          <Loader2 className="size-4 animate-spin" /> Loading...
        </div>
      );
    }
  }

  const initial: QueryMutation =
    mode === "edit" && detail.data
      ? {
          name: detail.data.name,
          description: detail.data.description,
          query: detail.data.query,
          platform: detail.data.platform,
          min_osquery_version: detail.data.min_osquery_version,
          schedule_interval: detail.data.schedule_interval,
          label_scope: detail.data.label_scope ?? {},
        }
      : emptyQuery;

  return <ReportEditForm key={reportId || "new"} mode={mode} reportId={reportId} initial={initial} />;
}

function ReportEditForm({
  mode,
  reportId,
  initial,
}: {
  mode: "create" | "edit";
  reportId: string;
  initial: QueryMutation;
}) {
  const navigate = useNavigate();
  const createQuery = useCreateQuery();
  const updateQuery = useUpdateQuery(reportId);
  const [form, setForm] = useState<QueryMutation>(initial);
  const [schemaOpen, setSchemaOpen] = useState(true);
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const editorRef = useRef<ReactCodeMirrorRef>(null);

  const error = createQuery.error ?? updateQuery.error;
  const pending = createQuery.isPending || updateQuery.isPending;

  async function submit() {
    const saved = mode === "create" ? await createQuery.mutateAsync(form) : await updateQuery.mutateAsync(form);
    void navigate({ to: "/reports/$reportId", params: { reportId: String(saved.id) } });
  }

  function insertAtCursor(snippet: string) {
    const view = editorRef.current?.view;
    if (!view) {
      setForm((prev) => ({ ...prev, query: prev.query + " " + snippet }));
      return;
    }
    view.dispatch({ changes: { from: view.state.selection.main.from, insert: snippet } });
  }

  return (
    <div className={cn("flex h-full flex-col transition-[padding] duration-150", schemaOpen && "pr-80")}>
      <form
        className="flex flex-col gap-5 p-6 pb-4"
        onSubmit={(event) => {
          event.preventDefault();
          void submit();
        }}
      >
        <BackLink to={mode === "edit" ? `/reports/${reportId}` : "/reports"}>
          {mode === "edit" ? "Back to report" : "Back to reports"}
        </BackLink>
        <PageLead
          title={mode === "create" ? "New report" : "Edit report"}
          description="Use osquery SQL to gather data about hosts. Add an interval to collect snapshot results on a schedule."
          actions={
            <>
              {!schemaOpen ? (
                <Button type="button" variant="outline" size="sm" onClick={() => setSchemaOpen(true)}>
                  <PanelRightOpen className="size-4" />
                  Schema
                </Button>
              ) : null}
              {mode === "edit" ? (
                <Button asChild type="button" variant="outline" size="sm">
                  <Link to="/reports/$reportId" params={{ reportId }}>
                    Cancel
                  </Link>
                </Button>
              ) : null}
              <Button type="submit" size="sm" disabled={pending}>
                {pending ? "Saving..." : mode === "create" ? "Save report" : "Save"}
              </Button>
            </>
          }
        />
        {error ? (
          <Alert variant="destructive">
            <AlertTitle>Unable to save report</AlertTitle>
            <AlertDescription>{error.message}</AlertDescription>
          </Alert>
        ) : null}
        <div className="grid max-w-3xl gap-4">
          <div className="grid gap-2">
            <Label htmlFor="report-name">Name</Label>
            <Input
              id="report-name"
              required
              value={form.name}
              onChange={(event) => setForm({ ...form, name: event.target.value })}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="report-description">Description</Label>
            <Textarea
              id="report-description"
              rows={3}
              placeholder="What information does this report reveal?"
              value={form.description ?? ""}
              onChange={(event) => setForm({ ...form, description: event.target.value })}
            />
          </div>
        </div>
        <div className="grid gap-4">
          <div className="grid max-w-3xl gap-4 sm:grid-cols-2">
            <div className="grid gap-2">
              <Label htmlFor="report-interval">Interval</Label>
              <Select
                value={String(form.schedule_interval ?? 0)}
                onValueChange={(value) => setForm({ ...form, schedule_interval: Number(value) })}
              >
                <SelectTrigger id="report-interval" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {FREQUENCY_OPTIONS.map((option) => (
                    <SelectItem key={option.value} value={String(option.value)}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-muted-foreground text-xs">This is how often your report collects data.</p>
            </div>
            <div className="grid gap-2">
              <Label htmlFor="report-platform">Targeted platforms</Label>
              <Select
                value={form.platform ?? "all"}
                onValueChange={(value) => setForm({ ...form, platform: value === "all" ? undefined : value })}
              >
                <SelectTrigger id="report-platform" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {PLATFORM_OPTIONS.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <LabelScopeSelector value={form.label_scope} onChange={(label_scope) => setForm({ ...form, label_scope })} />
        </div>

        <div className="grid max-w-3xl gap-3">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="w-fit px-0"
            onClick={() => setAdvancedOpen((open) => !open)}
          >
            {advancedOpen ? <ChevronDown className="size-4" /> : <ChevronRight className="size-4" />}
            Advanced options
          </Button>
          {advancedOpen ? (
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="grid gap-2">
                <Label htmlFor="report-min-version">Minimum osquery version</Label>
                <Input
                  id="report-min-version"
                  value={form.min_osquery_version ?? ""}
                  placeholder="5.18.1"
                  onChange={(event) => setForm({ ...form, min_osquery_version: event.target.value || undefined })}
                />
              </div>
            </div>
          ) : null}
        </div>
      </form>

      <div className="flex flex-1 px-6 pb-6">
        <SQLEditor
          ref={editorRef}
          value={form.query}
          onChange={(query) => setForm({ ...form, query })}
          placeholder="SELECT ..."
          minHeight="28rem"
          className="flex-1"
        />
      </div>
      {schemaOpen ? <SchemaSidebar onClose={() => setSchemaOpen(false)} onInsertColumn={insertAtCursor} /> : null}
    </div>
  );
}
