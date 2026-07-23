import { revalidateLogic, useForm } from "@tanstack/react-form";
import type { ColumnDef } from "@tanstack/react-table";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { type ReactNode, useCallback, useRef, useState } from "react";
import { z } from "zod";

import { DataTableStatic } from "@/components/data-table/data-table-static";
import { SchemaSidebar } from "@/components/editor/schema-sidebar";
import { SQLEditor } from "@/components/editor/sql-editor";
import { EmptyPanel } from "@/components/empty-panel";
import { FormActions } from "@/components/form-actions";
import { FormField } from "@/components/form-field";
import {
  type FormTabDefinition,
  FormTabTrigger,
  revealFirstInvalidFormTab,
} from "@/components/form-tabs";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { ScrollableTabs, ScrollableTabsList } from "@/components/layout/scrollable-tabs";
import {
  reportRows,
  reportTableColumns,
  type ReportTableRow,
  resultColumnNames,
  resultValue,
} from "@/components/reports/query-results";
import { LabelTargetSetEditor } from "@/components/targeting/label-target-set-editor";
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
import { TabsContent, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { usePageFormExitGuard } from "@/hooks/use-page-form-exit-guard";
import { useReportResults } from "@/hooks/use-reports";
import { useSchemaSidebar } from "@/hooks/use-schema-sidebar";
import type { OsqueryReport, OsqueryReportMutation } from "@/lib/api";
import { firstErrorMessage, requiredString } from "@/lib/form-validation";
import { invalidSQLSyntaxMessage, validSQLSyntax } from "@/lib/sql-validation";
import {
  emptyLabelTargetSet,
  labelTargetSetSchema,
  normalizeLabelTargetSet,
} from "@/lib/targeting";
import { cn, nonEmpty } from "@/lib/utils";
const FREQUENCY_OPTIONS: {
  value: number;
  label: string;
}[] = [
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
export const emptyReport: OsqueryReportMutation = {
  name: "",
  description: "",
  query: "select * from os_version;",
  schedule_interval: 0,
  targets: emptyLabelTargetSet(),
};
export function reportFromDetail(detail: OsqueryReport): OsqueryReportMutation {
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
const reportFormSchema = z.object({
  name: requiredString("Name"),
  description: z.string().optional(),
  query: reportQuerySchema,
  min_osquery_version: z.string().optional(),
  schedule_interval: z.number().int().min(0).max(2147483647).optional(),
  targets: labelTargetSetSchema,
});
const reportFormTabs = [
  {
    value: "options",
    fields: ["name", "description", "query", "min_osquery_version", "schedule_interval"],
  },
  { value: "targets", fields: ["targets"] },
] as const satisfies readonly FormTabDefinition[];
const noOp = () => undefined;
function trimReport(value: OsqueryReportMutation): OsqueryReportMutation {
  return {
    ...value,
    name: value.name.trim(),
    description: value.description?.trim() ?? "",
    query: value.query.trim(),
    min_osquery_version: nonEmpty(value.min_osquery_version),
    targets: normalizeLabelTargetSet(value.targets),
  };
}
export function ReportForm({
  initial,
  title,
  submitLabel,
  onSubmit,
  onSuccess,
  onCancel,
  headerActions,
  resultsReportId,
}: {
  initial: OsqueryReportMutation;
  title?: string;
  submitLabel: string;
  onSubmit: (value: OsqueryReportMutation) => Promise<number | undefined>;
  onSuccess?: (id: number | undefined) => void;
  onCancel?: () => void;
  headerActions?: ReactNode;
  resultsReportId?: number;
}) {
  const [schemaOpen, setSchemaOpen] = useSchemaSidebar();
  const [activeTab, setActiveTab] = useState("options");
  const [selectedSchemaTable, setSelectedSchemaTable] = useState<string | null>(null);
  const editorRef = useRef<ReactCodeMirrorRef>(null);
  const form = useForm({
    defaultValues: initial,
    validationLogic: revalidateLogic({ mode: "submit", modeAfterSubmission: "change" }),
    validators: { onDynamic: reportFormSchema },
    onSubmit: async ({ value, formApi }) => {
      const id = await onSubmit(trimReport(value));
      formApi.reset(value);
      onSuccess?.(id);
    },
  });
  const exitGuard = usePageFormExitGuard({ form, onDiscard: onCancel ?? noOp });
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
      className={cn("h-full transition-[padding] duration-200 ease-out", schemaOpen && `pr-84`)}
      render={
        <form
          noValidate
          onSubmit={(event) => {
            event.preventDefault();
            void form.handleSubmit().then(() => {
              revealFirstInvalidFormTab(form, reportFormTabs, setActiveTab);
              return undefined;
            });
          }}
        />
      }
    >
      <form.Subscribe selector={(state) => state.values.name}>
        {(name) => <PageHeader title={title ?? (name || "Report")} actions={headerActions} />}
      </form.Subscribe>

      <ScrollableTabs value={activeTab} onValueChange={setActiveTab}>
        <ScrollableTabsList>
          <FormTabTrigger form={form} tab={reportFormTabs[0]}>
            Options
          </FormTabTrigger>
          <FormTabTrigger form={form} tab={reportFormTabs[1]}>
            Targets
          </FormTabTrigger>
          {resultsReportId !== undefined ? (
            <TabsTrigger value="results">Results</TabsTrigger>
          ) : null}
        </ScrollableTabsList>

        <TabsContent value="options" keepMounted className="data-inactive:hidden">
          <div className="flex max-w-3xl flex-col gap-6">
            <FieldGroup>
              <form.Field name="name">
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
                          onChange={(event) => field.handleChange(event.target.value || undefined)}
                        />
                      )}
                    </FormField>
                  )}
                </form.Field>
              </div>
            </FieldGroup>

            <form.Field name="query">
              {(field) => {
                const error = firstErrorMessage(field.state.meta.errors);
                return (
                  <Field data-invalid={error ? true : undefined}>
                    <FieldLabel>
                      Query
                      <span className="text-destructive" aria-hidden="true">
                        *
                      </span>
                    </FieldLabel>
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
        </TabsContent>

        <TabsContent value="targets" keepMounted className="data-inactive:hidden">
          <form.Field name="targets">
            {(field) => (
              <FormField field={field}>
                {(control) => (
                  <div {...control} tabIndex={-1}>
                    <LabelTargetSetEditor
                      value={normalizeLabelTargetSet(field.state.value)}
                      onChange={field.handleChange}
                    />
                  </div>
                )}
              </FormField>
            )}
          </form.Field>
        </TabsContent>

        {resultsReportId !== undefined ? (
          <TabsContent value="results">
            <form.Subscribe selector={(state) => state.values.name}>
              {() => <ReportResults reportId={resultsReportId} />}
            </form.Subscribe>
          </TabsContent>
        ) : null}
      </ScrollableTabs>

      <FormActions
        form={form}
        submitLabel={submitLabel}
        onCancel={onCancel ? exitGuard.requestDiscard : undefined}
      />

      <SchemaSidebar
        open={schemaOpen}
        onOpenChange={setSchemaOpen}
        onInsertColumn={insertAtCursor}
        selectedTable={selectedSchemaTable}
        onSelectedTableChange={setSelectedSchemaTable}
      />
      {exitGuard.dialog}
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
