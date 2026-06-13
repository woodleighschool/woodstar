import { useForm } from "@tanstack/react-form";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { type ReactNode, useCallback, useRef, useState } from "react";

import { SchemaSidebar } from "@/components/editor/schema-sidebar";
import { SQLEditor } from "@/components/editor/sql-editor";
import { FormField } from "@/components/form-field";
import { ScrollableTabs, ScrollableTabsList } from "@/components/layout/scrollable-tabs";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { FormActions } from "@/components/form-actions";
import { LabelTargetSetEditor } from "@/components/targeting/label-target-set-editor";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { TabsContent, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { useSchemaSidebar } from "@/hooks/use-schema-sidebar";
import type { Check, CheckMutation } from "@/lib/api";
import { firstErrorMessage, requiredString } from "@/lib/form-validation";
import { invalidSQLSyntaxMessage, validSQLSyntax } from "@/lib/sql-validation";
import { emptyLabelTargetSet, normalizeLabelTargetSet } from "@/lib/targeting";
import { cn } from "@/lib/utils";

export const emptyCheck: CheckMutation = {
  name: "",
  description: "",
  query: "select 1;",
  targets: emptyLabelTargetSet(),
};

export function checkFromDetail(detail: Check): CheckMutation {
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

function trimCheck(value: CheckMutation): CheckMutation {
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
  pending,
  error,
  onSubmit,
  onCancel,
  headerContext,
  headerActions,
}: {
  initial: CheckMutation;
  title?: string;
  submitLabel: string;
  pending: boolean;
  error?: { message?: string } | null;
  onSubmit: (value: CheckMutation) => Promise<void> | void;
  onCancel?: () => void;
  headerContext?: ReactNode;
  headerActions?: ReactNode;
}) {
  const [schemaOpen, setSchemaOpen] = useSchemaSidebar();
  const [selectedSchemaTable, setSelectedSchemaTable] = useState<string | null>(null);
  const editorRef = useRef<ReactCodeMirrorRef>(null);
  const form = useForm({
    defaultValues: initial,
    onSubmit: async ({ value }) => onSubmit(trimCheck(value)),
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
          {(name) => (
            <PageHeader
              title={title ?? (name || "Check")}
              context={headerContext}
              actions={headerActions}
            />
          )}
        </form.Subscribe>

        <ScrollableTabs defaultValue="options">
          <ScrollableTabsList>
            <TabsTrigger value="options">Options</TabsTrigger>
            <TabsTrigger value="targets">Targets</TabsTrigger>
          </ScrollableTabsList>

          <TabsContent value="options" className="min-w-0">
            <div className="flex max-w-5xl flex-col gap-6">
              <FieldGroup className="max-w-3xl">
                <form.Field name="name" validators={{ onSubmit: requiredString("Name") }}>
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

              <form.Field name="query" validators={{ onSubmit: checkQuerySchema }}>
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
          </TabsContent>

          <TabsContent value="targets" className="min-w-0">
            <form.Field name="targets">
              {(field) => (
                <LabelTargetSetEditor
                  value={normalizeLabelTargetSet(field.state.value)}
                  onChange={field.handleChange}
                />
              )}
            </form.Field>
          </TabsContent>
        </ScrollableTabs>

        <FormActions
          pending={pending}
          error={error?.message}
          submitLabel={submitLabel}
          onCancel={onCancel}
        />

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
