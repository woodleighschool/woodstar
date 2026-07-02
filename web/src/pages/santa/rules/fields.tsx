import { revalidateLogic, useForm } from "@tanstack/react-form";

import { FormActions } from "@/components/form-actions";
import { FormField } from "@/components/form-field";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { ScrollableTabs, ScrollableTabsList } from "@/components/layout/scrollable-tabs";
import { LabelAssignmentList } from "@/components/targeting/label-assignment-list";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
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
import { TabsContent, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import type { SantaRuleMutation } from "@/lib/api";
import { RULE_TYPE_OPTIONS, type SantaRuleType } from "@/lib/santa-rules";

import {
  ruleBody,
  ruleFormSchema,
  type RuleFormState,
  ruleIdentifierHint,
  selectedIncludeLabelIDs,
} from "./form-state";
import { SantaIncludeTargets } from "./include-targets";

export function RuleForm({
  initial,
  title,
  submitLabel,
  onSubmit,
  onCancel,
}: {
  initial: RuleFormState;
  title?: string;
  submitLabel: string;
  onSubmit: (body: SantaRuleMutation) => Promise<void> | void;
  onCancel?: () => void;
}) {
  const form = useForm({
    defaultValues: initial,
    validationLogic: revalidateLogic(),
    validators: { onDynamic: ruleFormSchema },
    onSubmit: async ({ value }) => onSubmit(ruleBody(ruleFormSchema.parse(value))),
  });

  return (
    <PageShell asChild>
      <form
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit();
        }}
      >
        <form.Subscribe selector={(state) => state.values}>
          {(values) => (
            <>
              <PageHeader title={title ?? (values.name || "Rule")} />

              <ScrollableTabs defaultValue="options">
                <ScrollableTabsList>
                  <TabsTrigger value="options">Options</TabsTrigger>
                  <TabsTrigger value="targets">Targets</TabsTrigger>
                </ScrollableTabsList>

                <TabsContent value="options">
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
                    <form.Field name="identifier">
                      {(field) => (
                        <FormField
                          field={field}
                          label="Identifier"
                          htmlFor="santa-rule-identifier"
                          required
                          description={ruleIdentifierHint(values.rule_type)}
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
                </TabsContent>

                <TabsContent value="targets">
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
                </TabsContent>
              </ScrollableTabs>
            </>
          )}
        </form.Subscribe>
        <FormActions form={form} submitLabel={submitLabel} onCancel={onCancel} />
      </form>
    </PageShell>
  );
}
