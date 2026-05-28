import { useNavigate, useParams } from "@tanstack/react-router";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { ChevronDown, ChevronRight, Loader2 } from "lucide-react";
import { useRef, useState } from "react";

import { SchemaSidebar } from "@/components/editor/schema-sidebar";
import { SQLEditor } from "@/components/editor/sql-editor";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { LabelScopeSelector } from "@/components/queries/label-scope-selector";
import { LiveRunButton } from "@/components/queries/query-ui";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { useCreateReport, useReport, useUpdateReport, type ReportMutation } from "@/hooks/use-reports";
import { useSchemaSidebar } from "@/hooks/use-schema-sidebar";
import { cn } from "@/lib/utils";

const FREQUENCY_OPTIONS: { value: number; label: string }[] = [
  { value: 0, label: "Never" },
  { value: 300, label: "Every 5 Minutes" },
  { value: 600, label: "Every 10 Minutes" },
  { value: 900, label: "Every 15 Minutes" },
  { value: 1800, label: "Every 30 Minutes" },
  { value: 3600, label: "Every Hour" },
  { value: 21600, label: "Every 6 Hours" },
  { value: 43200, label: "Every 12 Hours" },
  { value: 86400, label: "Every Day" },
  { value: 604800, label: "Every Week" },
];

const emptyReport: ReportMutation = {
  name: "",
  description: "",
  query: "select * from os_version;",
  schedule_interval: 0,
  label_scope: {},
};

export function ReportEditPage({ mode }: { mode: "create" | "edit" }) {
  const params = useParams({ strict: false });
  const reportId = params.reportId ?? "";
  const reportID = mode === "edit" ? Number(reportId) : null;
  const detail = useReport(reportID);

  if (mode === "edit") {
    if (detail.error) {
      return (
        <PageShell>
          <Alert variant="destructive">
            <AlertTitle>Failed to Load Report</AlertTitle>
            <AlertDescription>{detail.error.message}</AlertDescription>
          </Alert>
        </PageShell>
      );
    }
    if (!detail.data) {
      return (
        <PageShell className="text-muted-foreground flex-row items-center gap-2 text-sm">
          <Loader2 className="size-4 animate-spin" /> Loading...
        </PageShell>
      );
    }
  }

  const initial: ReportMutation =
    mode === "edit" && detail.data
      ? {
          name: detail.data.name,
          description: detail.data.description,
          query: detail.data.query,
          min_osquery_version: detail.data.min_osquery_version,
          schedule_interval: detail.data.schedule_interval,
          label_scope: detail.data.label_scope ?? {},
        }
      : emptyReport;

  return (
    <ReportEditForm key={reportId || "new"} mode={mode} reportId={reportID} reportParam={reportId} initial={initial} />
  );
}

export function ReportCreatePage() {
  return <ReportEditPage mode="create" />;
}

export function ReportUpdatePage() {
  return <ReportEditPage mode="edit" />;
}

function ReportEditForm({
  mode,
  reportId,
  reportParam,
  initial,
}: {
  mode: "create" | "edit";
  reportId: number | null;
  reportParam: string;
  initial: ReportMutation;
}) {
  const navigate = useNavigate();
  const createReport = useCreateReport();
  const updateReport = useUpdateReport(reportId);
  const [form, setForm] = useState<ReportMutation>(initial);
  const [schemaOpen, setSchemaOpen] = useSchemaSidebar();
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const editorRef = useRef<ReactCodeMirrorRef>(null);

  const pending = createReport.isPending || updateReport.isPending;

  async function submit() {
    const payload = trimReport(form);
    const saved = mode === "create" ? await createReport.mutateAsync(payload) : await updateReport.mutateAsync(payload);
    void navigate({ to: "/osquery/reports/$reportId", params: { reportId: String(saved.id) } });
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
    <PageShell asChild className={cn("h-full transition-[padding] duration-200 ease-out", schemaOpen && "pr-[21rem]")}>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          void submit();
        }}
      >
        <PageHeader title={mode === "create" ? "New Report" : "Edit Report"} />
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor="report-name">Name</FieldLabel>
            <Input
              id="report-name"
              required
              value={form.name}
              onChange={(event) => setForm({ ...form, name: event.target.value })}
            />
          </Field>

          <Field>
            <FieldLabel htmlFor="report-description">Description</FieldLabel>
            <Textarea
              id="report-description"
              rows={3}
              placeholder="What this report reveals"
              value={form.description ?? ""}
              onChange={(event) => setForm({ ...form, description: event.target.value })}
            />
          </Field>

          <Field>
            <FieldLabel htmlFor="report-interval">Interval</FieldLabel>
            <Select
              value={String(form.schedule_interval ?? 0)}
              onValueChange={(value) => setForm({ ...form, schedule_interval: Number(value) })}
            >
              <SelectTrigger id="report-interval" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {FREQUENCY_OPTIONS.map((option) => (
                    <SelectItem key={option.value} value={String(option.value)}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
            <FieldDescription>Snapshot collection schedule.</FieldDescription>
          </Field>
        </FieldGroup>

        <LabelScopeSelector value={form.label_scope} onChange={(label_scope) => setForm({ ...form, label_scope })} />

        <div className="grid gap-2">
          <Label>Query</Label>
          <SQLEditor
            ref={editorRef}
            value={form.query}
            onChange={(query) => setForm({ ...form, query })}
            placeholder="SELECT ..."
          />
        </div>

        <FieldGroup>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="w-fit px-0"
            onClick={() => setAdvancedOpen((open) => !open)}
          >
            {advancedOpen ? <ChevronDown data-icon="inline-start" /> : <ChevronRight data-icon="inline-start" />}
            Advanced Options
          </Button>
          {advancedOpen ? (
            <Field>
              <FieldLabel htmlFor="report-min-version">Minimum Osquery Version</FieldLabel>
              <Input
                id="report-min-version"
                value={form.min_osquery_version ?? ""}
                placeholder="5.18.1"
                onChange={(event) => setForm({ ...form, min_osquery_version: event.target.value || undefined })}
              />
            </Field>
          ) : null}
        </FieldGroup>

        <div className="flex items-center gap-2 border-t pt-4">
          <Button type="submit" size="sm" disabled={pending}>
            {pending ? "Saving..." : "Save"}
          </Button>
          {mode === "edit" ? (
            <LiveRunButton to="/osquery/reports/$reportId/live" params={{ reportId: reportParam }} />
          ) : null}
        </div>
        <SchemaSidebar open={schemaOpen} onOpenChange={setSchemaOpen} onInsertColumn={insertAtCursor} />
      </form>
    </PageShell>
  );
}

function trimReport(form: ReportMutation): ReportMutation {
  const min = form.min_osquery_version?.trim() ?? "";
  return {
    ...form,
    name: form.name.trim(),
    description: form.description?.trim() ?? "",
    query: form.query.trim(),
    min_osquery_version: min === "" ? undefined : min,
  };
}
