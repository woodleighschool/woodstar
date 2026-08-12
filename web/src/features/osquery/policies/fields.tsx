import { revalidateLogic, useForm } from "@tanstack/react-form";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { Play } from "lucide-react";
import { useCallback, useLayoutEffect, useRef, useState } from "react";
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
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from "@components/ui/field";
import { Input } from "@components/ui/input";
import { Switch } from "@components/ui/switch";
import { TabsContent } from "@components/ui/tabs";
import { Textarea } from "@components/ui/textarea";
import { ValidatedFormField } from "@components/validated-form-field";
import { usePageFormExitGuard } from "@hooks/use-page-form-exit-guard";
import type {
  OsqueryPolicy,
  OsqueryPolicyMutation,
  OsqueryPolicyRemediationMutation,
} from "@lib/api";
import { firstErrorMessage, requiredString } from "@lib/form-validation";
import { sqlSyntaxError } from "@lib/sql-validation";
import { emptyLabelTargetSet, labelTargetSetSchema, normalizeLabelTargetSet } from "@lib/targeting";
type PolicyFormValue = Omit<OsqueryPolicyMutation, "remediation"> & {
  remediation: OsqueryPolicyRemediationMutation;
};

export const emptyPolicy: PolicyFormValue = {
  name: "",
  description: "",
  resolution: "",
  query: "select 1;",
  targets: emptyLabelTargetSet(),
  remediation: { script: "", automatic: false },
};
export function policyFromDetail(detail: OsqueryPolicy, script: string): PolicyFormValue {
  return {
    name: detail.name,
    description: detail.description,
    resolution: detail.resolution,
    query: detail.query,
    targets: normalizeLabelTargetSet(detail.targets),
    remediation: { script, automatic: detail.remediation.automatic },
  };
}
const policyFormSchema = z.object({
  name: requiredString("Name"),
  description: z.string().optional(),
  resolution: z.string().optional(),
  query: requiredString("Query"),
  targets: labelTargetSetSchema,
  remediation: z
    .object({
      script: z.string(),
      automatic: z.boolean(),
    })
    .superRefine((remediation, ctx) => {
      if (remediation.automatic && !remediation.script.trim()) {
        ctx.addIssue({
          code: "custom",
          path: ["script"],
          message: "A remediation script is required when automatic remediation is enabled.",
        });
      }
    }),
});
const policyFormTabs = [
  { value: "options", fields: ["name", "description", "resolution", "query"] },
  { value: "targets", fields: ["targets"] },
  { value: "remediation", fields: ["remediation"] },
] as const satisfies readonly FormTabDefinition[];
const noOp = () => undefined;
function policyFormValue(value: OsqueryPolicyMutation): PolicyFormValue {
  return {
    ...value,
    remediation: value.remediation ?? { script: "", automatic: false },
  };
}
function policyMutationFromForm(value: PolicyFormValue): OsqueryPolicyMutation {
  const scriptConfigured = value.remediation.script.trim().length > 0;
  return {
    name: value.name.trim(),
    description: value.description?.trim() ?? "",
    resolution: value.resolution?.trim() ?? "",
    query: value.query.trim(),
    targets: normalizeLabelTargetSet(value.targets),
    remediation: scriptConfigured
      ? {
          script: value.remediation.script,
          automatic: value.remediation.automatic,
        }
      : undefined,
  };
}
export function PolicyForm({
  initial,
  draft,
  title,
  submitLabel,
  activeTab,
  onActiveTabChange,
  onSubmit,
  onSuccess,
  onCancel,
  onRunLive,
  confirmResultReset = false,
}: {
  initial: PolicyFormValue;
  draft?: OsqueryPolicyMutation;
  title: string;
  submitLabel: string;
  activeTab: string;
  onActiveTabChange: (value: string) => void;
  onSubmit: (value: OsqueryPolicyMutation) => Promise<number | undefined>;
  onSuccess?: (id: number | undefined) => unknown;
  onCancel?: () => unknown;
  onRunLive: (value: OsqueryPolicyMutation) => Promise<void>;
  confirmResultReset?: boolean;
}) {
  const [schemaOpen, setSchemaOpen] = useSchemaSidebar();
  const [selectedSchemaTable, setSelectedSchemaTable] = useState<string | null>(null);
  const [liveQueryRequired, setLiveQueryRequired] = useState(false);
  const [pendingResultReset, setPendingResultReset] = useState<OsqueryPolicyMutation | null>(null);
  const [resultResetPending, setResultResetPending] = useState(false);
  const editorRef = useRef<ReactCodeMirrorRef>(null);
  const form = useForm({
    defaultValues: initial,
    validationLogic: revalidateLogic({
      mode: "submit",
      modeAfterSubmission: "change",
    }),
    validators: { onDynamic: policyFormSchema },
    onSubmit: async ({ value, formApi }) => {
      const next = policyMutationFromForm(value);
      const queryChanged = next.query !== initial.query.trim();
      if (confirmResultReset && queryChanged) {
        setPendingResultReset(next);
        return;
      }
      const id = await onSubmit(next);
      formApi.reset(policyFormValue(next));
      await onSuccess?.(id);
    },
  });
  useLayoutEffect(() => {
    if (draft) form.reset(policyFormValue(draft), { keepDefaultValues: true });
  }, [draft, form]);
  const exitGuard = usePageFormExitGuard({ form, onDiscard: onCancel ?? noOp });
  async function confirmPendingResultReset() {
    if (!pendingResultReset) return;
    setResultResetPending(true);
    try {
      const id = await onSubmit(pendingResultReset);
      form.reset(policyFormValue(pendingResultReset));
      setPendingResultReset(null);
      await onSuccess?.(id);
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
  async function runLive() {
    if (!form.getFieldValue("query").trim()) {
      setLiveQueryRequired(true);
      onActiveTabChange("options");
      return;
    }
    await exitGuard.runWithoutPrompt(() => onRunLive(policyMutationFromForm(form.state.values)));
  }
  return (
    <div className="flex min-h-full w-full min-w-0">
      <PageShell className="h-full min-w-0 flex-1">
        <PageHeader title={title} />

        <ScrollableTabs value={activeTab} onValueChange={onActiveTabChange}>
          <ScrollableTabsList>
            <FormTabTrigger form={form} tab={policyFormTabs[0]}>
              Options
            </FormTabTrigger>
            <FormTabTrigger form={form} tab={policyFormTabs[1]}>
              Targets
            </FormTabTrigger>
            <FormTabTrigger form={form} tab={policyFormTabs[2]}>
              Remediation
            </FormTabTrigger>
          </ScrollableTabsList>

          <TabsContent value="options" keepMounted className="data-inactive:hidden">
            <div className="flex max-w-3xl flex-col gap-6">
              <FieldGroup>
                <form.Field name="name">
                  {(field) => (
                    <ValidatedFormField field={field} label="Name" htmlFor="policy-name" required>
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
                      htmlFor="policy-description"
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

                <form.Field name="resolution">
                  {(field) => (
                    <ValidatedFormField
                      field={field}
                      label="Resolution"
                      htmlFor="policy-resolution"
                      description="Instructions someone can follow to resolve a failing result."
                    >
                      {(control) => (
                        <Textarea
                          {...control}
                          name={field.name}
                          rows={4}
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

          <TabsContent value="remediation" keepMounted className="data-inactive:hidden">
            <div className="flex max-w-3xl flex-col gap-6">
              <form.Field name="remediation.script">
                {(field) => (
                  <ValidatedFormField
                    field={field}
                    label="Script"
                    htmlFor="policy-remediation-script"
                    description="Optional script sent to eligible Orbit hosts. Include a shebang."
                  >
                    {(control) => (
                      <Textarea
                        {...control}
                        name={field.name}
                        rows={14}
                        className="font-mono text-xs"
                        spellCheck={false}
                        placeholder={"#!/bin/zsh\n\n# Repair the failing condition."}
                        value={field.state.value}
                        onBlur={field.handleBlur}
                        onChange={(event) => field.handleChange(event.target.value)}
                      />
                    )}
                  </ValidatedFormField>
                )}
              </form.Field>

              <form.Field name="remediation.automatic">
                {(field) => (
                  <Field orientation="horizontal">
                    <FieldContent>
                      <FieldLabel htmlFor="policy-remediation-automatic">
                        Automatic remediation
                      </FieldLabel>
                      <FieldDescription>
                        Run once when an eligible host newly becomes failing. Enabling this does not
                        run against existing failures.
                      </FieldDescription>
                    </FieldContent>
                    <Switch
                      id="policy-remediation-automatic"
                      checked={field.state.value}
                      onCheckedChange={field.handleChange}
                    />
                  </Field>
                )}
              </form.Field>
            </div>
          </TabsContent>
        </ScrollableTabs>

        <FormActions
          form={form}
          submitLabel={submitLabel}
          onSubmit={async () => {
            await form.handleSubmit();
            revealFirstInvalidFormTab(form, policyFormTabs, onActiveTabChange);
          }}
          onCancel={onCancel ? exitGuard.requestDiscard : undefined}
        >
          <Button type="button" variant="ghost" size="sm" onClick={() => void runLive()}>
            <Play data-icon="inline-start" />
            Run Live
          </Button>
        </FormActions>

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
                Changing this policy&apos;s SQL will delete its previous results, since the existing
                results do not reflect the updated SQL.
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
