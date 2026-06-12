import { useForm } from "@tanstack/react-form";

import { FormField } from "@/components/form-field";
import { MutableResourceTabs } from "@/components/layout/mutable-resource-tabs";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { SubmitButton } from "@/components/submit-button";
import { LabelAssignmentList } from "@/components/targeting/label-assignment-list";
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
import { Separator } from "@/components/ui/separator";
import { Textarea } from "@/components/ui/textarea";
import type { SantaRuleMutation, SantaRuleType } from "@/hooks/use-santa-rules";
import { RULE_TYPE_OPTIONS } from "@/lib/santa-rules";

import {
  identifierErrorFor,
  ruleBody,
  ruleFormSchema,
  type RuleFormState,
  selectedIncludeLabelIDs,
} from "./form-state";
import { SantaIncludeTargets } from "./include-targets";
import { RuleReferencePicker } from "./rule-form-fields";

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

  return (
    <PageShell asChild>
      <form
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit();
        }}
      >
        <form.Subscribe
          selector={(state) => ({
            values: state.values,
            submissionAttempts: state.submissionAttempts,
          })}
        >
          {({ values, submissionAttempts }) => {
            const parsed = ruleFormSchema.safeParse(values);
            const showErrors = submissionAttempts > 0;
            const identifierError = identifierErrorFor(parsed);
            const identifierInvalid =
              identifierError !== undefined && (showErrors || values.identifier.trim() !== "");

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
                              <FormField
                                field={field}
                                label="Name"
                                htmlFor="santa-rule-name"
                                required
                              >
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
                                <FieldLabel htmlFor="santa-rule-description">
                                  Description
                                </FieldLabel>
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
                                <FieldLabel htmlFor="santa-rule-custom-message">
                                  Custom Message
                                </FieldLabel>
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
                          <SantaIncludeTargets
                            include={values.targets.include}
                            excludeLabelIDs={values.targets.exclude.map((ref) => ref.label_id)}
                            onChange={(include) =>
                              form.setFieldValue("targets", { ...values.targets, include })
                            }
                          />
                          <Separator />
                          <LabelAssignmentList
                            title="Exclude"
                            addLabel="Add Exclude"
                            emptyText="No excludes yet"
                            rows={values.targets.exclude}
                            crossListLabelIDs={selectedIncludeLabelIDs(values.targets.include)}
                            includeBuiltins={false}
                            onChange={(exclude) =>
                              form.setFieldValue("targets", { ...values.targets, exclude })
                            }
                          />
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
              </>
            );
          }}
        </form.Subscribe>
      </form>
    </PageShell>
  );
}
