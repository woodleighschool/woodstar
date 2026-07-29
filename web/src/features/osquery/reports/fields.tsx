import { revalidateLogic, useForm } from "@tanstack/react-form";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { ChevronDownIcon } from "lucide-react";
import { useCallback, useRef, useState } from "react";
import { z } from "zod";

import { ConfirmDialog } from "@components/confirm-dialog";
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
import { Button } from "@components/ui/button";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@components/ui/collapsible";
import { Field, FieldError, FieldGroup, FieldLabel } from "@components/ui/field";
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
import { sqlSyntaxError } from "@lib/sql-validation";
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

const ALL_OSQUERY_VERSIONS = "__all_osquery_versions__";
const OSQUERY_VERSION_VALUES = [
  "5.23.1",
  "5.23.0",
  "5.22.1",
  "5.21.0",
  "5.20.0",
  "5.19.0",
  "5.18.1",
  "5.18.0",
  "5.17.0",
  "5.16.0",
  "5.15.0",
  "5.14.1",
  "5.13.1",
  "5.12.2",
  "5.12.1",
  "5.11.0",
  "5.10.2",
  "5.9.1",
  "5.8.2",
  "5.8.1",
  "5.7.0",
  "5.6.0",
  "5.5.1",
  "5.4.0",
  "5.3.0",
  "5.2.3",
  "5.2.2",
  "5.2.1",
  "5.2.0",
  "5.1.0",
  "5.0.1",
  "5.0.0",
  "4.9.0",
  "4.8.0",
  "4.7.0",
  "4.6.0",
  "4.5.1",
  "4.5.0",
  "4.4.0",
  "4.3.0",
  "4.2.0",
  "4.1.2",
  "4.1.1",
  "4.1.0",
  "4.0.2",
  "4.0.1",
  "4.0.0",
  "3.4.0",
  "3.3.2",
  "3.3.1",
  "3.2.6",
  "2.2.1",
  "2.2.0",
  "2.1.2",
  "2.1.1",
  "2.0.0",
  "1.8.2",
  "1.8.1",
] as const;
const OSQUERY_VERSION_OPTIONS = [
  { value: ALL_OSQUERY_VERSIONS, label: "All versions" },
  ...OSQUERY_VERSION_VALUES.map((value) => ({ value, label: value })),
];
const osqueryVersionSchema = z
  .string()
  .regex(/^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/, "Minimum osquery version must use X.Y.Z.")
  .optional();

function osqueryVersionOptions(current: string | undefined) {
  if (current === undefined || OSQUERY_VERSION_VALUES.some((value) => value === current)) {
    return OSQUERY_VERSION_OPTIONS;
  }
  return [
    OSQUERY_VERSION_OPTIONS[0],
    { value: current, label: current },
    ...OSQUERY_VERSION_OPTIONS.slice(1),
  ];
}

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
const reportFormSchema = z.object({
  name: requiredString("Name"),
  description: z.string().optional(),
  query: requiredString("Query"),
  min_osquery_version: osqueryVersionSchema,
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
  confirmResultReset = false,
}: {
  initial: OsqueryReportMutation;
  title: string;
  submitLabel: string;
  onSubmit: (value: OsqueryReportMutation) => Promise<number | undefined>;
  onSuccess?: (id: number | undefined) => void;
  onCancel?: () => void;
  confirmResultReset?: boolean;
}) {
  const [schemaOpen, setSchemaOpen] = useSchemaSidebar();
  const [activeTab, setActiveTab] = useState("options");
  const [selectedSchemaTable, setSelectedSchemaTable] = useState<string | null>(null);
  const [pendingResultReset, setPendingResultReset] = useState<{
    value: OsqueryReportMutation;
    sqlOnly: boolean;
  } | null>(null);
  const [resultResetPending, setResultResetPending] = useState(false);
  const editorRef = useRef<ReactCodeMirrorRef>(null);
  const form = useForm({
    defaultValues: initial,
    validationLogic: revalidateLogic({
      mode: "submit",
      modeAfterSubmission: "change",
    }),
    validators: { onDynamic: reportFormSchema },
    onSubmit: async ({ value, formApi }) => {
      const next = trimReport(value);
      const queryChanged = next.query !== initial.query.trim();
      const minVersionChanged = next.min_osquery_version !== nonEmpty(initial.min_osquery_version);
      if (confirmResultReset && (queryChanged || minVersionChanged)) {
        setPendingResultReset({
          value: next,
          sqlOnly: queryChanged && !minVersionChanged,
        });
        return;
      }
      const id = await onSubmit(next);
      formApi.reset(next);
      onSuccess?.(id);
    },
  });
  const exitGuard = usePageFormExitGuard({ form, onDiscard: onCancel ?? noOp });
  async function confirmPendingResultReset() {
    if (!pendingResultReset) return;
    setResultResetPending(true);
    try {
      const id = await onSubmit(pendingResultReset.value);
      form.reset(pendingResultReset.value);
      setPendingResultReset(null);
      onSuccess?.(id);
    } catch {
      return;
    } finally {
      setResultResetPending(false);
    }
  }
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
                </div>
              </FieldGroup>

              <form.Field name="query">
                {(field) => {
                  const error =
                    firstErrorMessage(field.state.meta.errors) ?? sqlSyntaxError(field.state.value);
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

              <Collapsible>
                <CollapsibleTrigger
                  render={
                    <Button type="button" variant="ghost" size="sm" className="-ml-2 w-fit" />
                  }
                >
                  Advanced options
                  <ChevronDownIcon className="text-muted-foreground" />
                </CollapsibleTrigger>
                <CollapsibleContent className="h-(--collapsible-panel-height) overflow-hidden opacity-100 transition-[height,opacity] duration-200 ease-out data-ending-style:h-0 data-ending-style:opacity-0 data-starting-style:h-0 data-starting-style:opacity-0 motion-reduce:transition-none">
                  <div className="pt-3">
                    <form.Field name="min_osquery_version">
                      {(field) => {
                        const options = osqueryVersionOptions(field.state.value);
                        return (
                          <ValidatedFormField
                            field={field}
                            label="Minimum Osquery Version"
                            htmlFor="report-min-version"
                            description="Runs only on hosts with this osquery version or newer."
                          >
                            {(control) => (
                              <Select
                                items={options}
                                value={field.state.value ?? ALL_OSQUERY_VERSIONS}
                                onValueChange={(value) =>
                                  field.handleChange(
                                    value === null || value === ALL_OSQUERY_VERSIONS
                                      ? undefined
                                      : value,
                                  )
                                }
                              >
                                <SelectTrigger {...control} className="w-full max-w-sm">
                                  <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                  <SelectGroup>
                                    {options.map((option) => (
                                      <SelectItem key={option.value} value={option.value}>
                                        {option.label}
                                      </SelectItem>
                                    ))}
                                  </SelectGroup>
                                </SelectContent>
                              </Select>
                            )}
                          </ValidatedFormField>
                        );
                      }}
                    </form.Field>
                  </div>
                </CollapsibleContent>
              </Collapsible>
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
        <ConfirmDialog
          open={pendingResultReset !== null}
          onOpenChange={(open) => {
            if (!open && !resultResetPending) setPendingResultReset(null);
          }}
          title="Save changes?"
          description={
            <>
              <span className="block">
                {pendingResultReset?.sqlOnly
                  ? "Changing this report's SQL will delete its previous results, since the existing results do not reflect the updated SQL."
                  : "The changes you are making to this report will delete its previous results."}
              </span>
              <span className="mt-3 block">You cannot undo this action.</span>
            </>
          }
          confirmLabel="Save"
          pending={resultResetPending}
          onConfirm={() => void confirmPendingResultReset()}
        />
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
