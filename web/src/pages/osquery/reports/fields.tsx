import { useForm } from "@tanstack/react-form";
import type { ColumnDef } from "@tanstack/react-table";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { type ReactNode, useCallback, useRef, useState } from "react";

import { DataTableStatic } from "@/components/data-table/data-table-static";
import { SchemaSidebar } from "@/components/editor/schema-sidebar";
import { SQLEditor } from "@/components/editor/sql-editor";
import { EmptyPanel } from "@/components/empty-panel";
import { FormField } from "@/components/form-field";
import { ScrollableTabs } from "@/components/layout/scrollable-tabs";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import {
  reportRows,
  reportTableColumns,
  type ReportTableRow,
  resultColumnNames,
  resultValue,
} from "@/components/reports/query-results";
import { SubmitButton } from "@/components/submit-button";
import { LabelTargetSetEditor } from "@/components/targeting/label-target-set-editor";
import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { useReportResults } from "@/hooks/use-reports";
import { useSchemaSidebar } from "@/hooks/use-schema-sidebar";
import type { Report, ReportMutation } from "@/lib/api";
import { firstErrorMessage, requiredString } from "@/lib/form-validation";
import { invalidSQLSyntaxMessage, validSQLSyntax } from "@/lib/sql-validation";
import { emptyLabelTargetSet, normalizeLabelTargetSet } from "@/lib/targeting";
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

export const emptyReport: ReportMutation = {
  name: "",
  description: "",
  query: "select * from os_version;",
  schedule_interval: 0,
  targets: emptyLabelTargetSet(),
};

export function reportFromDetail(detail: Report): ReportMutation {
  return {
    name: detail.name,
    description: detail.description,
    query: detail.query,
    min_osquery_version: detail.min_osquery_version,
    schedule_interval: detail.schedule_interval,
    targets: normalizeLabelTargetSet(detail.targets),
  };
}

const reportQuerySchema = requiredString("Query").refine(validSQLSyntax, {
  message: invalidSQLSyntaxMessage,
});

function trimReport(value: ReportMutation): ReportMutation {
  return {
    ...value,
    name: value.name.trim(),
    description: value.description?.trim() ?? "",
    query: value.query.trim(),
    min_osquery_version: nonEmptyText(value.min_osquery_version),
    targets: normalizeLabelTargetSet(value.targets),
  };
}

function nonEmptyText(value: string | undefined) {
  const trimmed = value?.trim();
  return trimmed === "" ? undefined : trimmed;
}

export function ReportForm({
  initial,
  title,
  submitLabel,
  pending,
  error,
  onSubmit,
  onCancel,
  headerActions,
  resultsReportId,
}: {
  initial: ReportMutation;
  title?: string;
  submitLabel: string;
  pending: boolean;
  error?: { message?: string } | null;
  onSubmit: (value: ReportMutation) => Promise<void> | void;
  onCancel?: () => void;
  headerActions?: ReactNode;
  resultsReportId?: number;
}) {
  const [schemaOpen, setSchemaOpen] = useSchemaSidebar();
  const [selectedSchemaTable, setSelectedSchemaTable] = useState<string | null>(null);
  const editorRef = useRef<ReactCodeMirrorRef>(null);
  const form = useForm({
    defaultValues: initial,
    onSubmit: async ({ value }) => onSubmit(trimReport(value)),
  });

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
    <PageShell
      asChild
      className={cn(
        "h-full transition-[padding] duration-200 ease-out",
        schemaOpen && "pr-[21rem]",
      )}
    >
      <form
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit();
        }}
      >
        <form.Subscribe selector={(state) => state.values.name}>
          {(name) => <PageHeader title={title ?? (name || "Report")} actions={headerActions} />}
        </form.Subscribe>

        <ScrollableTabs
          tabs={[
            {
              value: "options",
              label: "Options",
              content: (
                <div className="flex max-w-5xl flex-col gap-6">
                  <FieldGroup className="max-w-3xl">
                    <form.Field name="name" validators={{ onSubmit: requiredString("Name") }}>
                      {(field) => (
                        <FormField field={field} label="Name" htmlFor="report-name" required>
                          {(control) => (
                            <Input
                              {...control}
                              name={field.name}
                              required
                              value={field.state.value}
                              onBlur={field.handleBlur}
                              onChange={(event) => field.handleChange(event.target.value)}
                            />
                          )}
                        </FormField>
                      )}
                    </form.Field>

                    <form.Field name="description">
                      {(field) => (
                        <FormField field={field} label="Description" htmlFor="report-description">
                          {(control) => (
                            <Textarea
                              {...control}
                              name={field.name}
                              rows={3}
                              value={field.state.value ?? ""}
                              onBlur={field.handleBlur}
                              onChange={(event) => field.handleChange(event.target.value)}
                            />
                          )}
                        </FormField>
                      )}
                    </form.Field>

                    <div className="grid gap-4 md:grid-cols-2">
                      <form.Field name="schedule_interval">
                        {(field) => (
                          <FormField field={field} label="Interval" htmlFor="report-interval">
                            {(control) => (
                              <Select
                                value={String(field.state.value ?? 0)}
                                onValueChange={(value) => field.handleChange(Number(value))}
                              >
                                <SelectTrigger {...control} className="w-full">
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
                            )}
                          </FormField>
                        )}
                      </form.Field>

                      <form.Field name="min_osquery_version">
                        {(field) => (
                          <FormField
                            field={field}
                            label="Minimum Osquery Version"
                            htmlFor="report-min-version"
                          >
                            {(control) => (
                              <Input
                                {...control}
                                name={field.name}
                                value={field.state.value ?? ""}
                                placeholder="5.18.1"
                                onBlur={field.handleBlur}
                                onChange={(event) =>
                                  field.handleChange(event.target.value || undefined)
                                }
                              />
                            )}
                          </FormField>
                        )}
                      </form.Field>
                    </div>
                  </FieldGroup>

                  <form.Field name="query" validators={{ onSubmit: reportQuerySchema }}>
                    {(field) => {
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
                  </form.Field>
                </div>
              ),
            },
            {
              value: "targets",
              label: "Targets",
              content: (
                <form.Field name="targets">
                  {(field) => (
                    <LabelTargetSetEditor
                      value={normalizeLabelTargetSet(field.state.value)}
                      onChange={field.handleChange}
                    />
                  )}
                </form.Field>
              ),
            },
            ...(resultsReportId !== undefined
              ? [
                  {
                    value: "results",
                    label: "Results",
                    content: (
                      <form.Subscribe selector={(state) => state.values.name}>
                        {() => <ReportResults reportId={resultsReportId} />}
                      </form.Subscribe>
                    ),
                  },
                ]
              : []),
          ]}
        />

        <div className="flex flex-col gap-2 border-t pt-4">
          <div className="flex items-center gap-2">
            <SubmitButton pending={pending} size="sm">
              {submitLabel}
            </SubmitButton>
            {onCancel ? (
              <Button type="button" variant="outline" size="sm" onClick={onCancel}>
                Cancel
              </Button>
            ) : null}
          </div>
          {error ? <FieldError>{error.message}</FieldError> : null}
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

function ReportResults({ reportId }: { reportId: number }) {
  const results = useReportResults(reportId);
  const rows = reportRows(results.data);
  const resultColumns: ColumnDef<ReportTableRow>[] = resultColumnNames(rows).map((name) => ({
    id: name,
    accessorFn: (row) => row.columns[name] ?? "",
    header: () => name,
    cell: ({ row }) => resultValue(row.original.columns[name]),
  }));
  const columns = [...reportTableColumns(), ...resultColumns];

  if (results.isLoading) return null;
  if (rows.length === 0) return <EmptyPanel>No results yet</EmptyPanel>;

  return <DataTableStatic columns={columns} data={rows} />;
}
