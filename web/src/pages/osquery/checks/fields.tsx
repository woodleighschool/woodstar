import { revalidateLogic, useForm } from "@tanstack/react-form";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { type ReactNode, useCallback, useRef, useState } from "react";
import { z } from "zod";

import { SchemaSidebar } from "@/components/editor/schema-sidebar";
import { SQLEditor } from "@/components/editor/sql-editor";
import { FormActions } from "@/components/form-actions";
import { FormField } from "@/components/form-field";
import {
  type FormTabDefinition,
  FormTabTrigger,
  revealFirstInvalidFormTab,
} from "@/components/form-tabs";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { ScrollableTabs, ScrollableTabsList } from "@/components/layout/scrollable-tabs";
import { LabelTargetSetEditor } from "@/components/targeting/label-target-set-editor";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { TabsContent } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { usePageFormExitGuard } from "@/hooks/use-page-form-exit-guard";
import { useSchemaSidebar } from "@/hooks/use-schema-sidebar";
import type { OsqueryCheck, OsqueryCheckMutation } from "@/lib/api";
import { firstErrorMessage, requiredString } from "@/lib/form-validation";
import { invalidSQLSyntaxMessage, validSQLSyntax } from "@/lib/sql-validation";
import {
  emptyLabelTargetSet,
  labelTargetSetSchema,
  normalizeLabelTargetSet,
} from "@/lib/targeting";
import { cn } from "@/lib/utils";
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
const checkQuerySchema = requiredString("Query").refine(validSQLSyntax, {
  message: invalidSQLSyntaxMessage,
});
const checkFormSchema = z.object({
  name: requiredString("Name"),
  description: z.string().optional(),
  query: checkQuerySchema,
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
  title,
  submitLabel,
  onSubmit,
  onSuccess,
  onCancel,
  headerContext,
  headerActions,
}: {
  initial: OsqueryCheckMutation;
  title: string;
  submitLabel: string;
  onSubmit: (value: OsqueryCheckMutation) => Promise<number | undefined>;
  onSuccess?: (id: number | undefined) => void;
  onCancel?: () => void;
  headerContext?: ReactNode;
  headerActions?: ReactNode;
}) {
  const [schemaOpen, setSchemaOpen] = useSchemaSidebar();
  const [activeTab, setActiveTab] = useState("options");
  const [selectedSchemaTable, setSelectedSchemaTable] = useState<string | null>(null);
  const editorRef = useRef<ReactCodeMirrorRef>(null);
  const form = useForm({
    defaultValues: initial,
    validationLogic: revalidateLogic({ mode: "submit", modeAfterSubmission: "change" }),
    validators: { onDynamic: checkFormSchema },
    onSubmit: async ({ value, formApi }) => {
      const id = await onSubmit(trimCheck(value));
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
              revealFirstInvalidFormTab(form, checkFormTabs, setActiveTab);
              return undefined;
            });
          }}
        />
      }
    >
      <PageHeader title={title} context={headerContext} actions={headerActions} />

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
                  <FormField field={field} label="Name" htmlFor="check-name" required>
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
                  <FormField field={field} label="Description" htmlFor="check-description">
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
                      One or more returned rows is a pass; no rows is a fail.
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
