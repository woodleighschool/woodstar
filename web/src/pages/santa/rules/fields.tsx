import { revalidateLogic, useForm } from "@tanstack/react-form";
import { useState } from "react";

import { FormActions } from "@/components/form-actions";
import { FormField } from "@/components/form-field";
import {
  type FormTabDefinition,
  FormTabTrigger,
  revealFirstInvalidFormTab,
} from "@/components/form-tabs";
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
import { TabsContent } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { usePageFormExitGuard } from "@/hooks/use-page-form-exit-guard";
import type { SantaRuleMutation } from "@/lib/api";
import { RULE_TYPE_OPTIONS, RULE_TYPE_VALUES } from "@/lib/santa-rules";
import { isOneOf } from "@/lib/utils";

import {
  ruleBody,
  ruleFormSchema,
  type RuleFormState,
  ruleIdentifierHint,
  selectedIncludeLabelIDs,
} from "./form-state";
import { SantaIncludeTargets } from "./include-targets";
const ruleFormTabs = [
  {
    value: "options",
    fields: ["name", "description", "rule_type", "identifier", "custom_url", "custom_message"],
  },
  { value: "targets", fields: ["targets"] },
] as const satisfies readonly FormTabDefinition[];
const noOp = () => undefined;
export function RuleForm({
  initial,
  title,
  submitLabel,
  onSubmit,
  onSuccess,
  onCancel,
}: {
  initial: RuleFormState;
  title?: string;
  submitLabel: string;
  onSubmit: (body: SantaRuleMutation) => Promise<number | undefined>;
  onSuccess?: (id: number | undefined) => void;
  onCancel?: () => void;
}) {
  const [activeTab, setActiveTab] = useState("options");
  const form = useForm({
    defaultValues: initial,
    validationLogic: revalidateLogic({ mode: "submit", modeAfterSubmission: "change" }),
    validators: { onDynamic: ruleFormSchema },
    onSubmit: async ({ value, formApi }) => {
      const id = await onSubmit(ruleBody(ruleFormSchema.parse(value)));
      formApi.reset(value);
      onSuccess?.(id);
    },
  });
  const exitGuard = usePageFormExitGuard({ form, onDiscard: onCancel ?? noOp });
  return (
    <PageShell
      render={
        <form
          noValidate
          onSubmit={(event) => {
            event.preventDefault();
            void form.handleSubmit().then(() => {
              revealFirstInvalidFormTab(form, ruleFormTabs, setActiveTab);
              return undefined;
            });
          }}
        />
      }
    >
      <form.Subscribe selector={(state) => state.values}>
        {(values) => (
          <>
            <PageHeader title={title ?? (values.name || "Rule")} />

            <ScrollableTabs value={activeTab} onValueChange={setActiveTab}>
              <ScrollableTabsList>
                <FormTabTrigger form={form} tab={ruleFormTabs[0]}>
                  Options
                </FormTabTrigger>
                <FormTabTrigger form={form} tab={ruleFormTabs[1]}>
                  Targets
                </FormTabTrigger>
              </ScrollableTabsList>

              <TabsContent value="options" keepMounted className="data-inactive:hidden">
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
                      <FormField field={field} label="Description" htmlFor="santa-rule-description">
                        {(control) => (
                          <Textarea
                            {...control}
                            name={field.name}
                            rows={3}
                            value={field.state.value}
                            onBlur={field.handleBlur}
                            onChange={(event) => field.handleChange(event.target.value)}
                          />
                        )}
                      </FormField>
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
                            if (!isOneOf(ruleType, RULE_TYPE_VALUES)) return;
                            field.handleChange(ruleType);
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
                      <FormField field={field} label="Custom URL" htmlFor="santa-rule-custom-url">
                        {(control) => (
                          <Input
                            {...control}
                            name={field.name}
                            value={field.state.value}
                            onBlur={field.handleBlur}
                            onChange={(event) => field.handleChange(event.target.value)}
                          />
                        )}
                      </FormField>
                    )}
                  />
                  <form.Field
                    name="custom_message"
                    children={(field) => (
                      <FormField
                        field={field}
                        label="Custom Message"
                        htmlFor="santa-rule-custom-message"
                      >
                        {(control) => (
                          <Textarea
                            {...control}
                            name={field.name}
                            rows={3}
                            value={field.state.value}
                            onBlur={field.handleBlur}
                            onChange={(event) => field.handleChange(event.target.value)}
                          />
                        )}
                      </FormField>
                    )}
                  />
                </FieldGroup>
              </TabsContent>

              <TabsContent value="targets" keepMounted className="data-inactive:hidden">
                <form.Field name="targets">
                  {(field) => (
                    <FormField field={field}>
                      {(control) => (
                        <FieldGroup {...control} tabIndex={-1}>
                          <SantaIncludeTargets
                            include={field.state.value.include}
                            excludeLabelIDs={field.state.value.exclude.map((ref) => ref.label_id)}
                            onChange={(include) =>
                              field.handleChange({ ...field.state.value, include })
                            }
                          />
                          <Separator />
                          <LabelAssignmentList
                            title="Exclude"
                            addLabel="Add Exclude"
                            emptyText="No excludes yet"
                            rows={field.state.value.exclude}
                            crossListLabelIDs={selectedIncludeLabelIDs(field.state.value.include)}
                            includeBuiltins={false}
                            onChange={(exclude) =>
                              field.handleChange({ ...field.state.value, exclude })
                            }
                          />
                        </FieldGroup>
                      )}
                    </FormField>
                  )}
                </form.Field>
              </TabsContent>
            </ScrollableTabs>
          </>
        )}
      </form.Subscribe>
      <FormActions
        form={form}
        submitLabel={submitLabel}
        onCancel={onCancel ? exitGuard.requestDiscard : undefined}
      />
      {exitGuard.dialog}
    </PageShell>
  );
}
