import { revalidateLogic, useForm } from "@tanstack/react-form";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { Play } from "lucide-react";
import { useCallback, useLayoutEffect, useRef, useState } from "react";
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
import { Button } from "@components/ui/button";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@components/ui/field";
import { Input } from "@components/ui/input";
import { TabsContent } from "@components/ui/tabs";
import { Textarea } from "@components/ui/textarea";
import { ValidatedFormField } from "@components/validated-form-field";
import { usePageFormExitGuard } from "@hooks/use-page-form-exit-guard";
import type { OsqueryCheck, OsqueryCheckMutation } from "@lib/api";
import { firstErrorMessage, requiredString } from "@lib/form-validation";
import { sqlSyntaxError } from "@lib/sql-validation";
import { emptyLabelTargetSet, labelTargetSetSchema, normalizeLabelTargetSet } from "@lib/targeting";
export const emptyCheck: OsqueryCheckMutation = {
  name: "",
  description: "",
  query: "select 1;",
  targets: emptyLabelTargetSet(),
};
export function checkFromDetail(detail: OsqueryCheck): OsqueryCheckMutation {
  return {
    name: detail.name,
    description: detail.description,
    query: detail.query,
    targets: normalizeLabelTargetSet(detail.targets),
  };
}
const checkFormSchema = z.object({
  name: requiredString("Name"),
  description: z.string().optional(),
  query: requiredString("Query"),
  targets: labelTargetSetSchema,
});
const checkFormTabs = [
  { value: "options", fields: ["name", "description", "query"] },
  { value: "targets", fields: ["targets"] },
] as const satisfies readonly FormTabDefinition[];
const noOp = () => undefined;
function trimCheck(value: OsqueryCheckMutation): OsqueryCheckMutation {
  return {
    ...value,
    name: value.name.trim(),
    description: value.description?.trim() ?? "",
    query: value.query.trim(),
    targets: normalizeLabelTargetSet(value.targets),
  };
}
export function CheckForm({
  initial,
  draft,
  title,
  submitLabel,
  onSubmit,
  onSuccess,
  onCancel,
  onRunLive,
}: {
  initial: OsqueryCheckMutation;
  draft?: OsqueryCheckMutation;
  title: string;
  submitLabel: string;
  onSubmit: (value: OsqueryCheckMutation) => Promise<number | undefined>;
  onSuccess?: (id: number | undefined) => unknown;
  onCancel?: () => unknown;
  onRunLive: (value: OsqueryCheckMutation) => Promise<void>;
}) {
  const [schemaOpen, setSchemaOpen] = useSchemaSidebar();
  const [activeTab, setActiveTab] = useState("options");
  const [selectedSchemaTable, setSelectedSchemaTable] = useState<string | null>(null);
  const [liveQueryRequired, setLiveQueryRequired] = useState(false);
  const editorRef = useRef<ReactCodeMirrorRef>(null);
  const form = useForm({
    defaultValues: initial,
    validationLogic: revalidateLogic({
      mode: "submit",
      modeAfterSubmission: "change",
    }),
    validators: { onDynamic: checkFormSchema },
    onSubmit: async ({ value, formApi }) => {
      const id = await onSubmit(trimCheck(value));
      formApi.reset(value);
      await onSuccess?.(id);
    },
  });
  useLayoutEffect(() => {
    if (draft) form.reset(draft, { keepDefaultValues: true });
  }, [draft, form]);
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
  async function runLive() {
    if (!form.getFieldValue("query").trim()) {
      setLiveQueryRequired(true);
      setActiveTab("options");
      return;
    }
    await exitGuard.runWithoutPrompt(() => onRunLive(form.state.values));
  }
  return (
    <div className="flex min-h-full w-full min-w-0">
      <PageShell className="h-full min-w-0 flex-1">
        <PageHeader title={title} />

        <ScrollableTabs value={activeTab} onValueChange={setActiveTab}>
          <ScrollableTabsList>
            <FormTabTrigger form={form} tab={checkFormTabs[0]}>
              Options
            </FormTabTrigger>
            <FormTabTrigger form={form} tab={checkFormTabs[1]}>
              Targets
            </FormTabTrigger>
          </ScrollableTabsList>

          <TabsContent value="options" keepMounted className="data-inactive:hidden">
            <div className="flex max-w-3xl flex-col gap-6">
              <FieldGroup>
                <form.Field name="name">
                  {(field) => (
                    <ValidatedFormField field={field} label="Name" htmlFor="check-name" required>
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
                      htmlFor="check-description"
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
              </FieldGroup>

              <form.Field name="query">
                {(field) => {
                  const error =
                    (liveQueryRequired ? "Query is required." : undefined) ??
                    firstErrorMessage(field.state.meta.errors) ??
                    sqlSyntaxError(field.state.value);
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
                        onChange={(value) => {
                          setLiveQueryRequired(false);
                          field.handleChange(value);
                        }}
                        onTableMetaClick={selectSchemaTable}
                        placeholder="SELECT ..."
                        invalid={error ? true : undefined}
                      />
                      <FieldDescription>
                        One or more returned rows is a pass; no rows is a fail. Changing a saved
                        query clears its existing results.
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
            revealFirstInvalidFormTab(form, checkFormTabs, setActiveTab);
          }}
          onCancel={onCancel ? exitGuard.requestDiscard : undefined}
        >
          <Button type="button" variant="ghost" size="sm" onClick={() => void runLive()}>
            <Play data-icon="inline-start" />
            Run Live
          </Button>
        </FormActions>

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
