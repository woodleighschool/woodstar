import { revalidateLogic, useForm } from "@tanstack/react-form";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { useCallback, useRef, useState } from "react";
import { z } from "zod";

import { SchemaSidebar } from "@components/editor/schema-sidebar";
import { SQLEditor } from "@components/editor/sql-editor";
import { useSchemaSidebar } from "@components/editor/use-schema-sidebar";
import { FormActions } from "@components/form-actions";
import {
  type FormTabDefinition,
  FormTabTrigger,
  revealFirstInvalidFormTab,
} from "@components/form-tabs";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { ScrollableTabs, ScrollableTabsList } from "@components/layout/scrollable-tabs";
import { LabelTargetSetEditor } from "@components/targeting/label-target-set-editor";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@components/ui/field";
import { Input } from "@components/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@components/ui/select";
import { TabsContent } from "@components/ui/tabs";
import { Textarea } from "@components/ui/textarea";
import { ValidatedFormField } from "@components/validated-form-field";
import { usePageFormExitGuard } from "@hooks/use-page-form-exit-guard";
import type { OsqueryReport, OsqueryReportMutation } from "@lib/api";
import { firstErrorMessage, requiredString } from "@lib/form-validation";
import { invalidSQLSyntaxMessage, validSQLSyntax } from "@lib/sql-validation";
import { emptyLabelTargetSet, labelTargetSetSchema, normalizeLabelTargetSet } from "@lib/targeting";
import { nonEmpty } from "@lib/utils";
const FREQUENCY_OPTIONS: {
  value: string;
  label: string;
}[] = [
  { value: "0", label: "Off" },
  { value: "300", label: "5 Minutes" },
  { value: "600", label: "10 Minutes" },
  { value: "900", label: "15 Minutes" },
  { value: "1800", label: "30 Minutes" },
  { value: "3600", label: "1 Hour" },
  { value: "21600", label: "6 Hours" },
  { value: "43200", label: "12 Hours" },
  { value: "86400", label: "1 Day" },
  { value: "604800", label: "1 Week" },
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
}: {
  initial: OsqueryReportMutation;
  title: string;
  submitLabel: string;
  onSubmit: (value: OsqueryReportMutation) => Promise<number | undefined>;
  onSuccess?: (id: number | undefined) => void;
  onCancel?: () => void;
}) {
  const [schemaOpen, setSchemaOpen] = useSchemaSidebar();
  const [activeTab, setActiveTab] = useState("options");
  const [selectedSchemaTable, setSelectedSchemaTable] = useState<string | null>(null);
  const editorRef = useRef<ReactCodeMirrorRef>(null);
  const form = useForm({
    defaultValues: initial,
    validationLogic: revalidateLogic({
      mode: "submit",
      modeAfterSubmission: "change",
    }),
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
    view.dispatch({
      changes: { from: view.state.selection.main.from, insert: snippet },
    });
  }
  const selectSchemaTable = useCallback(
    (tableName: string) => {
      setSelectedSchemaTable(tableName);
      setSchemaOpen(true);
    },
    [setSchemaOpen],
  );
  return (
    <div className="flex min-h-full w-full min-w-0">
      <PageShell className="h-full min-w-0 flex-1">
        <PageHeader title={title} />

        <ScrollableTabs value={activeTab} onValueChange={setActiveTab}>
          <ScrollableTabsList>
            <FormTabTrigger form={form} tab={reportFormTabs[0]}>
              Options
            </FormTabTrigger>
            <FormTabTrigger form={form} tab={reportFormTabs[1]}>
              Targets
            </FormTabTrigger>
          </ScrollableTabsList>

          <TabsContent value="options" keepMounted className="data-inactive:hidden">
            <div className="flex max-w-3xl flex-col gap-6">
              <FieldGroup>
                <form.Field name="name">
                  {(field) => (
                    <ValidatedFormField field={field} label="Name" htmlFor="report-name" required>
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
                    </ValidatedFormField>
                  )}
                </form.Field>

                <form.Field name="description">
                  {(field) => (
                    <ValidatedFormField
                      field={field}
                      label="Description"
                      htmlFor="report-description"
                    >
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
                    </ValidatedFormField>
                  )}
                </form.Field>

                <div className="grid gap-4 md:grid-cols-2">
                  <form.Field name="schedule_interval">
                    {(field) => (
                      <ValidatedFormField
                        field={field}
                        label="Interval"
                        htmlFor="report-interval"
                        description="Runs the query on targeted hosts at this cadence. Off keeps it out of the schedule."
                      >
                        {(control) => (
                          <Select
                            items={FREQUENCY_OPTIONS}
                            value={String(field.state.value ?? 0)}
                            onValueChange={(value) => field.handleChange(Number(value))}
                          >
                            <SelectTrigger {...control} className="w-full">
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectGroup>
                                {FREQUENCY_OPTIONS.map((option) => (
                                  <SelectItem key={option.value} value={option.value}>
                                    {option.label}
                                  </SelectItem>
                                ))}
                              </SelectGroup>
                            </SelectContent>
                          </Select>
                        )}
                      </ValidatedFormField>
                    )}
                  </form.Field>

                  <form.Field name="min_osquery_version">
                    {(field) => (
                      <ValidatedFormField
                        field={field}
                        label="Minimum Osquery Version"
                        htmlFor="report-min-version"
                        description="Runs only on hosts with this osquery version or newer."
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
                      </ValidatedFormField>
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
                      <FieldDescription>
                        Stores the latest result rows returned by each targeted host. Changing a
                        saved query clears its existing results.
                      </FieldDescription>
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
                <ValidatedFormField field={field}>
                  {(control) => (
                    <div {...control} tabIndex={-1}>
                      <LabelTargetSetEditor
                        value={normalizeLabelTargetSet(field.state.value)}
                        onChange={field.handleChange}
                      />
                    </div>
                  )}
                </ValidatedFormField>
              )}
            </form.Field>
          </TabsContent>
        </ScrollableTabs>

        <FormActions
          form={form}
          submitLabel={submitLabel}
          onSubmit={async () => {
            await form.handleSubmit();
            revealFirstInvalidFormTab(form, reportFormTabs, setActiveTab);
          }}
          onCancel={onCancel ? exitGuard.requestDiscard : undefined}
        />

        {exitGuard.dialog}
      </PageShell>
      <SchemaSidebar
        open={schemaOpen}
        onOpenChange={setSchemaOpen}
        onInsertColumn={insertAtCursor}
        selectedTable={selectedSchemaTable}
        onSelectedTableChange={setSelectedSchemaTable}
      />
    </div>
  );
}
