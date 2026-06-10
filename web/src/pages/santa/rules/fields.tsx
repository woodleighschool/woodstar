import { useForm } from "@tanstack/react-form";
import { Plus } from "lucide-react";
import { useState } from "react";

import { FormField } from "@/components/form-field";
import { LabelPicker } from "@/components/labels/label-picker";
import { MutableResourceTabs } from "@/components/layout/mutable-resource-tabs";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { SubmitButton } from "@/components/submit-button";
import { TargetSection } from "@/components/targeting/target-section";
import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Textarea } from "@/components/ui/textarea";
import type { SantaRuleMutation, SantaRuleType } from "@/hooks/use-santa-rules";
import { RULE_TYPE_OPTIONS } from "@/lib/santa-rules";

import {
  identifierErrorFor,
  includeErrorMap,
  labelIDsFromRefs,
  labelRefsFromIDs,
  ruleBody,
  ruleFormSchema,
  selectedIncludeLabelIDs,
  type RuleFormState,
  type RuleIncludeForm,
} from "./form-state";
import { CELDialog, IncludeTargetsTable, RuleReferencePicker } from "./rule-form-fields";

export function RuleForm({
  initial,
  title,
  submitLabel,
  pending,
  error,
  onSubmit,
  onCancel,
}: {
  initial: RuleFormState;
  title?: string;
  submitLabel: string;
  pending: boolean;
  error?: { message?: string } | null;
  onSubmit: (body: SantaRuleMutation) => Promise<void> | void;
  onCancel?: () => void;
}) {
  const [celDialogID, setCELDialogID] = useState<number | null>(null);
  const form = useForm({
    defaultValues: initial,
    validators: {
      onSubmit: ruleFormSchema,
    },
    onSubmit: async ({ value }) => onSubmit(ruleBody(ruleFormSchema.parse(value))),
  });

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
                <PageHeader title={title ?? (values.name || "Rule")} />

                <MutableResourceTabs
                  tabs={[
                    {
                      value: "options",
                      label: "Options",
                      content: (
                        <FieldGroup className="max-w-3xl">
                          <form.Field name="name">
                            {(field) => (
                              <FormField field={field} label="Name" htmlFor="santa-rule-name" required>
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
                          <TargetSection
                            title="Include"
                            action={
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
                                Add Include
                              </Button>
                            }
                          >
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
                          </TargetSection>
                          <Separator />
                          <TargetSection title="Exclude">
                            <LabelPicker
                              value={labelIDsFromRefs(values.targets.exclude)}
                              unavailableLabelIDs={includeLabelIDs}
                              onChange={(labelIDs) =>
                                form.setFieldValue("targets", {
                                  ...values.targets,
                                  exclude: labelRefsFromIDs(labelIDs),
                                })
                              }
                              placeholder="Add Exclude"
                            />
                          </TargetSection>
                        </FieldGroup>
                      ),
                    },
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
