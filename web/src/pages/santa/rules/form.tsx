import { useForm } from "@tanstack/react-form";
import { useNavigate, useParams, useSearch } from "@tanstack/react-router";
import { Loader2, Plus } from "lucide-react";
import { useState } from "react";

import { LabelPicker } from "@/components/labels/label-picker";
import { MutableResourceTabs } from "@/components/layout/mutable-resource-tabs";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { CELDialog, IncludeTargetsTable, RuleReferencePicker } from "@/components/santa/rules/rule-form-fields";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Textarea } from "@/components/ui/textarea";
import { useCreateSantaRule, useSantaRule, useUpdateSantaRule, type SantaRuleType } from "@/hooks/use-santa";
import { firstErrorMessage } from "@/lib/form-validation";
import {
  formFromRule,
  formFromSearch,
  identifierErrorFor,
  includeErrorMap,
  labelIDsFromRefs,
  labelRefsFromIDs,
  ruleBody,
  ruleFormSchema,
  selectedIncludeLabelIDs,
  type RuleFormState,
  type RuleIncludeForm,
} from "@/lib/santa-rule-form";
import { RULE_TYPE_OPTIONS } from "@/lib/santa-rules";

export function SantaRuleResourcePage({ mode }: { mode: "create" | "edit" }) {
  const params = useParams({ strict: false });
  const search = useSearch({ strict: false });
  const ruleId = params.ruleId ?? "";
  const ruleID = mode === "edit" ? Number(ruleId) : null;
  const detail = useSantaRule(ruleID);

  if (mode === "edit") {
    if (detail.error) {
      return (
        <PageShell>
          <Alert variant="destructive">
            <AlertTitle>Failed to Load Rule</AlertTitle>
            <AlertDescription>{detail.error.message}</AlertDescription>
          </Alert>
        </PageShell>
      );
    }
    if (!detail.data) {
      return (
        <PageShell className="text-muted-foreground flex-row items-center gap-2 text-sm">
          <Loader2 className="animate-spin" /> Loading Rule...
        </PageShell>
      );
    }
  }

  const initial = mode === "edit" && detail.data ? formFromRule(detail.data) : formFromSearch(search);

  return <RuleForm key={ruleId || "new"} mode={mode} ruleId={ruleID} initial={initial} />;
}

function RuleForm({
  mode,
  ruleId,
  initial,
}: {
  mode: "create" | "edit";
  ruleId: number | null;
  initial: RuleFormState;
}) {
  const navigate = useNavigate();
  const create = useCreateSantaRule();
  const update = useUpdateSantaRule();
  const [celDialogID, setCELDialogID] = useState<number | null>(null);
  const form = useForm({
    defaultValues: initial,
    validators: {
      onSubmit: ruleFormSchema,
    },
    onSubmit: async ({ value }) => {
      const data = ruleFormSchema.parse(value);
      const saved =
        mode === "create"
          ? await create.mutateAsync(ruleBody(data))
          : await update.mutateAsync({ id: ruleId ?? 0, body: ruleBody(data) });
      void navigate({ to: "/santa/rules/$ruleId", params: { ruleId: String(saved.id) } });
    },
  });
  const pending = create.isPending || update.isPending;

  function setRuleReference(next: RuleFormState) {
    form.setFieldValue("rule_type", next.rule_type);
    form.setFieldValue("identifier", next.identifier);
    form.setFieldValue("name", next.name);
  }

  function updateInclude(values: RuleFormState, id: number, next: Partial<RuleIncludeForm>) {
    if (next.policy && next.policy !== "cel" && celDialogID === id) {
      setCELDialogID(null);
    }
    form.setFieldValue("targets", {
      ...values.targets,
      include: values.targets.include.map((include) => (include.id === id ? { ...include, ...next } : include)),
    });
  }

  return (
    <PageShell asChild>
      <form
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit();
        }}
      >
        <form.Subscribe selector={(state) => ({ values: state.values, submissionAttempts: state.submissionAttempts })}>
          {({ values, submissionAttempts }) => {
            const parsed = ruleFormSchema.safeParse(values);
            const showErrors = submissionAttempts > 0;
            const identifierError = identifierErrorFor(parsed);
            const identifierInvalid = identifierError !== undefined && (showErrors || values.identifier.trim() !== "");
            const includeErrors = includeErrorMap(parsed, values.targets.include);
            const includeLabelIDs = selectedIncludeLabelIDs(values.targets.include);

            return (
              <>
                <PageHeader title={mode === "create" ? "New Rule" : values.name || "Rule"} />

                <MutableResourceTabs
                  tabs={[
                    {
                      value: "options",
                      label: "Options",
                      content: (
                        <FieldGroup className="max-w-3xl">
                          <form.Field
                            name="name"
                            children={(field) => {
                              const error = firstErrorMessage(field.state.meta.errors);
                              return (
                                <Field data-invalid={error ? true : undefined}>
                                  <FieldLabel htmlFor="santa-rule-name" required>
                                    Name
                                  </FieldLabel>
                                  <Input
                                    id="santa-rule-name"
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
                                <FieldLabel htmlFor="santa-rule-description">Description</FieldLabel>
                                <Textarea
                                  id="santa-rule-description"
                                  name={field.name}
                                  rows={3}
                                  value={field.state.value}
                                  onBlur={field.handleBlur}
                                  onChange={(event) => field.handleChange(event.target.value)}
                                />
                              </Field>
                            )}
                          />
                          <form.Field
                            name="rule_type"
                            children={(field) => (
                              <Field>
                                <FieldLabel htmlFor="santa-rule-type">Rule Type</FieldLabel>
                                <Select
                                  value={field.state.value}
                                  onValueChange={(ruleType) => {
                                    field.handleChange(ruleType as SantaRuleType);
                                    form.setFieldValue("identifier", "");
                                  }}
                                >
                                  <SelectTrigger id="santa-rule-type" className="w-full">
                                    <SelectValue />
                                  </SelectTrigger>
                                  <SelectContent>
                                    <SelectGroup>
                                      {RULE_TYPE_OPTIONS.map((type) => (
                                        <SelectItem key={type.value} value={type.value}>
                                          {type.label}
                                        </SelectItem>
                                      ))}
                                    </SelectGroup>
                                  </SelectContent>
                                </Select>
                              </Field>
                            )}
                          />
                          <RuleReferencePicker
                            form={values}
                            identifierError={identifierError}
                            identifierInvalid={identifierInvalid}
                            onChange={setRuleReference}
                          />
                          <form.Field
                            name="custom_url"
                            children={(field) => (
                              <Field>
                                <FieldLabel htmlFor="santa-rule-custom-url">Custom URL</FieldLabel>
                                <Input
                                  id="santa-rule-custom-url"
                                  name={field.name}
                                  value={field.state.value}
                                  onBlur={field.handleBlur}
                                  onChange={(event) => field.handleChange(event.target.value)}
                                />
                              </Field>
                            )}
                          />
                          <form.Field
                            name="custom_message"
                            children={(field) => (
                              <Field>
                                <FieldLabel htmlFor="santa-rule-custom-message">Custom Message</FieldLabel>
                                <Textarea
                                  id="santa-rule-custom-message"
                                  name={field.name}
                                  rows={3}
                                  value={field.state.value}
                                  onBlur={field.handleBlur}
                                  onChange={(event) => field.handleChange(event.target.value)}
                                />
                              </Field>
                            )}
                          />
                        </FieldGroup>
                      ),
                    },
                    {
                      value: "targets",
                      label: "Targets",
                      content: (
                        <FieldGroup>
                          <Field>
                            <div className="flex flex-wrap items-center justify-between gap-2">
                              <FieldLabel>Targets</FieldLabel>
                              <Button
                                type="button"
                                variant="outline"
                                size="sm"
                                onClick={() =>
                                  form.setFieldValue("targets", {
                                    ...values.targets,
                                    include: [
                                      ...values.targets.include,
                                      { id: Date.now(), policy: "allowlist", cel_expression: "", label_id: null },
                                    ],
                                  })
                                }
                              >
                                <Plus data-icon="inline-start" />
                                Add Target
                              </Button>
                            </div>
                            {values.targets.include.length === 0 ? (
                              <div className="text-muted-foreground rounded-md border border-dashed px-4 py-6 text-sm">
                                None
                              </div>
                            ) : (
                              <IncludeTargetsTable
                                includeRows={values.targets.include}
                                showErrors={showErrors}
                                includeErrors={includeErrors}
                                excludedLabelIDs={labelIDsFromRefs(values.targets.exclude)}
                                onChange={(includeRows) =>
                                  form.setFieldValue("targets", { ...values.targets, include: includeRows })
                                }
                                onUpdate={(id, include) => updateInclude(values, id, include)}
                                onEditCEL={setCELDialogID}
                                onDelete={(id) => {
                                  if (celDialogID === id) setCELDialogID(null);
                                  form.setFieldValue("targets", {
                                    ...values.targets,
                                    include: values.targets.include.filter((item) => item.id !== id),
                                  });
                                }}
                              />
                            )}
                          </Field>
                          <Separator />
                          <form.Field
                            name="targets"
                            children={(field) => (
                              <Field>
                                <FieldLabel>Exclusions</FieldLabel>
                                <LabelPicker
                                  value={labelIDsFromRefs(field.state.value.exclude)}
                                  unavailableLabelIDs={includeLabelIDs}
                                  onChange={(labelIDs) =>
                                    field.handleChange({ ...field.state.value, exclude: labelRefsFromIDs(labelIDs) })
                                  }
                                />
                              </Field>
                            )}
                          />
                        </FieldGroup>
                      ),
                    },
                  ]}
                />

                <div className="flex items-center gap-2 border-t pt-4">
                  <Button type="submit" size="sm" disabled={pending}>
                    {pending ? <Loader2 data-icon="inline-start" className="animate-spin" /> : null}
                    Save
                  </Button>
                  {mode === "create" ? (
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => void navigate({ to: "/santa/rules" })}
                    >
                      Cancel
                    </Button>
                  ) : null}
                </div>

                <CELDialog
                  include={values.targets.include.find((include) => include.id === celDialogID)}
                  error={celDialogID !== null ? includeErrors[celDialogID]?.cel_expression : undefined}
                  showRequiredError={showErrors}
                  onOpenChange={(open) => {
                    if (!open) setCELDialogID(null);
                  }}
                  onChange={(celExpression) => {
                    if (celDialogID !== null) updateInclude(values, celDialogID, { cel_expression: celExpression });
                  }}
                />
              </>
            );
          }}
        </form.Subscribe>
      </form>
    </PageShell>
  );
}
