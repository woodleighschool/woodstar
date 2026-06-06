import { useForm } from "@tanstack/react-form";
import { useNavigate, useParams } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { Loader2 } from "lucide-react";
import { useCallback, useRef, useState } from "react";

import { DataTable, DataTableColumnHeader } from "@/components/data-table";
import { SchemaSidebar } from "@/components/editor/schema-sidebar";
import { SQLEditor } from "@/components/editor/sql-editor";
import { MutableResourceTabs } from "@/components/layout/mutable-resource-tabs";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { LiveRunButton } from "@/components/queries/query-ui";
import { LabelScopeEditor } from "@/components/targeting/label-scope-editor";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import {
  useCreateReport,
  useReport,
  useReportResults,
  useUpdateReport,
  type ReportMutation,
} from "@/hooks/use-reports";
import { useSchemaSidebar } from "@/hooks/use-schema-sidebar";
import { firstErrorMessage, requiredString } from "@/lib/form-validation";
import {
  reportRows,
  reportTableColumns,
  resultColumnNames,
  resultValue,
  type ReportTableRow,
} from "@/lib/query-results";
import { invalidSQLSyntaxMessage, validSQLSyntax } from "@/lib/sql-validation";
import { cn } from "@/lib/utils";

const FREQUENCY_OPTIONS: { value: number; label: string }[] = [
  { value: 0, label: "Off" },
  { value: 300, label: "5 Minutes" },
  { value: 600, label: "10 Minutes" },
  { value: 900, label: "15 Minutes" },
  { value: 1800, label: "30 Minutes" },
  { value: 3600, label: "1 Hour" },
  { value: 21600, label: "6 Hours" },
  { value: 43200, label: "12 Hours" },
  { value: 86400, label: "1 Day" },
  { value: 604800, label: "1 Week" },
];

const emptyReport: ReportMutation = {
  name: "",
  description: "",
  query: "select * from os_version;",
  schedule_interval: 0,
  targets: [],
};

const reportQuerySchema = requiredString("Query").refine(validSQLSyntax, { message: invalidSQLSyntaxMessage });

export function ReportMutationPage({ mode }: { mode: "create" | "edit" }) {
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
          <Loader2 className="animate-spin" /> Loading...
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
          targets: detail.data.targets ?? [],
        }
      : emptyReport;

  return (
    <ReportForm key={reportId || "new"} mode={mode} reportId={reportID} reportParam={reportId} initial={initial} />
  );
}

export function ReportNewPage() {
  return <ReportMutationPage mode="create" />;
}

export function ReportResourcePage() {
  return <ReportMutationPage mode="edit" />;
}

function ReportForm({
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
  const [schemaOpen, setSchemaOpen] = useSchemaSidebar();
  const [selectedSchemaTable, setSelectedSchemaTable] = useState<string | null>(null);
  const editorRef = useRef<ReactCodeMirrorRef>(null);
  const form = useForm({
    defaultValues: initial,
    onSubmit: async ({ value }) => {
      const payload = trimReport(value);
      const saved =
        mode === "create" ? await createReport.mutateAsync(payload) : await updateReport.mutateAsync(payload);
      void navigate({ to: "/osquery/reports/$reportId", params: { reportId: String(saved.id) } });
    },
  });
  const pending = createReport.isPending || updateReport.isPending;

  function insertAtCursor(snippet: string) {
    const view = editorRef.current?.view;
    if (!view) {
      form.setFieldValue("query", (current) => `${current} ${snippet}`);
      return;
    }
    view.dispatch({ changes: { from: view.state.selection.main.from, insert: snippet } });
  }

  const selectSchemaTable = useCallback(
    (tableName: string) => {
      setSelectedSchemaTable(tableName);
      setSchemaOpen(true);
    },
    [setSchemaOpen],
  );

  return (
    <PageShell asChild className={cn("h-full transition-[padding] duration-200 ease-out", schemaOpen && "pr-[21rem]")}>
      <form
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit();
        }}
      >
        <form.Subscribe selector={(state) => state.values.name}>
          {(name) => (
            <PageHeader
              title={mode === "create" ? "New Report" : name || "Report"}
              actions={
                mode === "edit" ? (
                  <LiveRunButton to="/osquery/reports/$reportId/live" params={{ reportId: reportParam }} />
                ) : null
              }
            />
          )}
        </form.Subscribe>

        <MutableResourceTabs
          tabs={[
            {
              value: "options",
              label: "Options",
              content: (
                <div className="flex max-w-5xl flex-col gap-6">
                  <FieldGroup className="max-w-3xl">
                    <form.Field
                      name="name"
                      validators={{ onSubmit: requiredString("Name") }}
                      children={(field) => {
                        const error = firstErrorMessage(field.state.meta.errors);
                        return (
                          <Field data-invalid={error ? true : undefined}>
                            <FieldLabel htmlFor="report-name" required>
                              Name
                            </FieldLabel>
                            <Input
                              id="report-name"
                              name={field.name}
                              required
                              aria-invalid={error ? true : undefined}
                              value={field.state.value}
                              onBlur={field.handleBlur}
                              onChange={(event) => field.handleChange(event.target.value)}
                            />
                            {error ? <FieldError>{error}</FieldError> : null}
                          </Field>
                        );
                      }}
                    />

                    <form.Field
                      name="description"
                      children={(field) => (
                        <Field>
                          <FieldLabel htmlFor="report-description">Description</FieldLabel>
                          <Textarea
                            id="report-description"
                            name={field.name}
                            rows={3}
                            value={field.state.value ?? ""}
                            onBlur={field.handleBlur}
                            onChange={(event) => field.handleChange(event.target.value)}
                          />
                        </Field>
                      )}
                    />

                    <div className="grid gap-4 md:grid-cols-2">
                      <form.Field
                        name="schedule_interval"
                        children={(field) => (
                          <Field>
                            <FieldLabel htmlFor="report-interval">Interval</FieldLabel>
                            <Select
                              value={String(field.state.value ?? 0)}
                              onValueChange={(value) => field.handleChange(Number(value))}
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
                          </Field>
                        )}
                      />

                      <form.Field
                        name="min_osquery_version"
                        children={(field) => (
                          <Field>
                            <FieldLabel htmlFor="report-min-version">Minimum Osquery Version</FieldLabel>
                            <Input
                              id="report-min-version"
                              name={field.name}
                              value={field.state.value ?? ""}
                              placeholder="5.18.1"
                              onBlur={field.handleBlur}
                              onChange={(event) => field.handleChange(event.target.value || undefined)}
                            />
                          </Field>
                        )}
                      />
                    </div>
                  </FieldGroup>

                  <form.Field
                    name="query"
                    validators={{ onSubmit: reportQuerySchema }}
                    children={(field) => {
                      const error = firstErrorMessage(field.state.meta.errors);
                      return (
                        <Field data-invalid={error ? true : undefined}>
                          <FieldLabel required>Query</FieldLabel>
                          <SQLEditor
                            ref={editorRef}
                            value={field.state.value}
                            onChange={field.handleChange}
                            onTableMetaClick={selectSchemaTable}
                            placeholder="SELECT ..."
                            invalid={error ? true : undefined}
                          />
                          {error ? <FieldError>{error}</FieldError> : null}
                        </Field>
                      );
                    }}
                  />
                </div>
              ),
            },
            {
              value: "scope",
              label: "Scope",
              content: (
                <form.Field
                  name="targets"
                  children={(field) => (
                    <LabelScopeEditor value={field.state.value ?? []} onChange={field.handleChange} />
                  )}
                />
              ),
            },
            ...(mode === "edit"
              ? [
                  {
                    value: "results",
                    label: "Results",
                    content: (
                      <form.Subscribe selector={(state) => state.values.name}>
                        {(name) => <ReportResults reportId={reportId} reportName={name} />}
                      </form.Subscribe>
                    ),
                  },
                ]
              : []),
          ]}
        />

        <div className="flex items-center gap-2 border-t pt-4">
          <Button type="submit" size="sm" disabled={pending}>
            {pending ? "Saving..." : "Save"}
          </Button>
          {mode === "create" ? (
            <Button type="button" variant="outline" size="sm" onClick={() => void navigate({ to: "/osquery/reports" })}>
              Cancel
            </Button>
          ) : null}
        </div>
        <SchemaSidebar
          open={schemaOpen}
          onOpenChange={setSchemaOpen}
          onInsertColumn={insertAtCursor}
          selectedTable={selectedSchemaTable}
          onSelectedTableChange={setSelectedSchemaTable}
        />
      </form>
    </PageShell>
  );
}

function ReportResults({ reportId, reportName }: { reportId: number | null; reportName: string }) {
  const results = useReportResults(reportId);
  const rows = reportRows(results.data);
  const resultColumns: ColumnDef<ReportTableRow>[] = resultColumnNames(rows).map((name) => ({
    id: name,
    accessorFn: (row) => row.columns[name] ?? "",
    header: ({ column }) => <DataTableColumnHeader column={column} title={name} />,
    cell: ({ row }) => resultValue(row.original.columns[name]),
  }));
  const columns = [...reportTableColumns({ linkHosts: true }), ...resultColumns];

  return (
    <DataTable
      columns={columns}
      data={rows}
      isLoading={results.isLoading}
      showExport
      exportFilename={`${reportName || "report"}-results.csv`}
      totalCount={rows.length}
      pagination={{ pageIndex: 0, pageSize: rows.length || 50 }}
      sorting={[]}
      clientSort
      onPaginationChange={() => null}
      onSortingChange={() => null}
      empty={
        <Empty>
          <EmptyHeader>
            <EmptyTitle>No Results</EmptyTitle>
            <EmptyDescription>Run the report to collect rows.</EmptyDescription>
          </EmptyHeader>
        </Empty>
      }
    />
  );
}

function trimReport(form: ReportMutation): ReportMutation {
  return {
    ...form,
    name: form.name.trim(),
    description: form.description?.trim() ?? "",
    query: form.query.trim(),
    min_osquery_version: nonEmptyText(form.min_osquery_version),
  };
}

function nonEmptyText(value: string | undefined) {
  const trimmed = value?.trim();
  return trimmed === "" ? undefined : trimmed;
}
