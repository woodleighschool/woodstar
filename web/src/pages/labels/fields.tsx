import { useForm } from "@tanstack/react-form";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { useCallback, useRef, useState } from "react";
import { z } from "zod";

import { SchemaSidebar } from "@/components/editor/schema-sidebar";
import { SQLEditor } from "@/components/editor/sql-editor";
import { FormField } from "@/components/form-field";
import { DerivedSelector, HostSelector } from "@/components/labels/label-membership-selectors";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { SubmitButton } from "@/components/submit-button";
import { Button } from "@/components/ui/button";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { useSchemaSidebar } from "@/hooks/use-schema-sidebar";
import type { Label, LabelMutation } from "@/lib/api";
import { requiredString, selectedIDArray } from "@/lib/form-validation";
import {
  LABEL_DERIVED_ATTRIBUTE_OPTIONS,
  LABEL_DERIVED_ATTRIBUTE_VALUES,
  LABEL_MEMBERSHIP_OPTIONS,
  LABEL_MEMBERSHIP_TYPES,
  LABEL_MEMBERSHIP_VALUES,
  type LabelDerivedAttribute,
  labelDerivedAttributeSelectorLabel,
  type LabelMembershipType,
} from "@/lib/labels";
import { sqlSyntaxError } from "@/lib/sql-validation";
import { cn } from "@/lib/utils";

interface LabelFormValue {
  name: string;
  description: string;
  query: string;
  host_ids: number[];
  derived_attribute: LabelDerivedAttribute;
  derived_values: string[];
  label_membership_type: LabelMembershipType;
}

export const emptyLabel: LabelFormValue = {
  name: "",
  description: "",
  query: "select 1 from os_version where major >= 13;",
  host_ids: [],
  derived_attribute: "user_department",
  derived_values: [],
  label_membership_type: "dynamic",
};

export function labelFromDetail(detail: Label): LabelFormValue {
  return {
    name: detail.name,
    description: detail.description,
    query: detail.query ?? emptyLabel.query,
    host_ids: detail.host_ids ?? [],
    derived_attribute: derivedAttributeFromString(detail.criteria?.attribute),
    derived_values: detail.criteria?.values ?? [],
    label_membership_type: membershipFromString(detail.label_membership_type),
  };
}

const queryRequiredSchema = requiredString("Query");

const labelFormSchema = z
  .object({
    name: requiredString("Name"),
    description: z.string().trim(),
    query: z.string().trim(),
    host_ids: selectedIDArray("Host"),
    derived_attribute: z.enum(LABEL_DERIVED_ATTRIBUTE_VALUES),
    derived_values: z.array(requiredString("Derived value")),
    label_membership_type: z.enum(LABEL_MEMBERSHIP_VALUES),
  })
  .superRefine((value, ctx) => {
    if (value.label_membership_type === "dynamic") {
      const query = queryRequiredSchema.safeParse(value.query);
      if (!query.success) {
        ctx.addIssue({
          code: "custom",
          message: query.error.issues[0]?.message ?? "Invalid query.",
          path: ["query"],
        });
      } else {
        const syntaxError = sqlSyntaxError(value.query);
        if (syntaxError) {
          ctx.addIssue({ code: "custom", message: syntaxError, path: ["query"] });
        }
      }
    }
    if (value.label_membership_type === "derived" && value.derived_values.length === 0) {
      ctx.addIssue({
        code: "custom",
        message: "Derived labels need at least one selected item.",
        path: ["derived_values"],
      });
    }
  });

function toBody(value: LabelFormValue): LabelMutation {
  const cleaned = labelFormSchema.parse(value);
  return {
    name: cleaned.name,
    description: cleaned.description,
    label_membership_type: cleaned.label_membership_type,
    query: cleaned.label_membership_type === "dynamic" ? cleaned.query : undefined,
    host_ids: cleaned.label_membership_type === "manual" ? cleaned.host_ids : undefined,
    criteria:
      cleaned.label_membership_type === "derived"
        ? { attribute: cleaned.derived_attribute, values: cleaned.derived_values }
        : undefined,
  };
}

export function LabelForm({
  initial,
  title,
  submitLabel,
  pending,
  error,
  onSubmit,
  onCancel,
}: {
  initial: LabelFormValue;
  title: string;
  submitLabel: string;
  pending: boolean;
  error?: { message?: string } | null;
  onSubmit: (body: LabelMutation) => Promise<void> | void;
  onCancel?: () => void;
}) {
  const [schemaOpen, setSchemaOpen] = useSchemaSidebar();
  const [selectedSchemaTable, setSelectedSchemaTable] = useState<string | null>(null);
  const editorRef = useRef<ReactCodeMirrorRef>(null);
  const form = useForm({
    defaultValues: initial,
    validators: { onSubmit: labelFormSchema },
    onSubmit: async ({ value }) => onSubmit(toBody(value)),
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
        <PageHeader title={title} />
        <form.Subscribe selector={(state) => state.values}>
          {(values) => {
            const isDynamic = values.label_membership_type === "dynamic";
            const isManual = values.label_membership_type === "manual";
            const isDerived = values.label_membership_type === "derived";
            const memberOption = LABEL_MEMBERSHIP_TYPES[values.label_membership_type];

            return (
              <>
                <FieldGroup className="max-w-5xl">
                  <form.Field name="name">
                    {(field) => (
                      <FormField field={field} label="Name" htmlFor="label-name" required>
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
                      <FormField field={field} label="Description" htmlFor="label-description">
                        {(control) => (
                          <Textarea
                            {...control}
                            name={field.name}
                            rows={3}
                            placeholder="Why this label exists"
                            value={field.state.value}
                            onBlur={field.handleBlur}
                            onChange={(event) => field.handleChange(event.target.value)}
                          />
                        )}
                      </FormField>
                    )}
                  </form.Field>

                  <form.Field
                    name="label_membership_type"
                    children={(field) => (
                      <Field>
                        <FieldLabel>Type</FieldLabel>
                        <ToggleGroup
                          type="single"
                          value={field.state.value}
                          onValueChange={(value) => {
                            if (!value) return;
                            const membershipType = value as LabelMembershipType;
                            field.handleChange(membershipType);
                            if (membershipType !== "dynamic") setSchemaOpen(false);
                          }}
                          variant="outline"
                          size="sm"
                          className="flex-wrap"
                        >
                          {LABEL_MEMBERSHIP_OPTIONS.map((option) => (
                            <ToggleGroupItem key={option.value} value={option.value}>
                              {option.label}
                            </ToggleGroupItem>
                          ))}
                        </ToggleGroup>
                        {memberOption.description ? (
                          <FieldDescription>{memberOption.description}</FieldDescription>
                        ) : null}
                      </Field>
                    )}
                  />

                  {isManual ? (
                    <form.Field
                      name="host_ids"
                      children={(field) => (
                        <Field data-invalid={field.state.meta.errors.length > 0 ? true : undefined}>
                          <FieldLabel>Hosts</FieldLabel>
                          <HostSelector value={field.state.value} onChange={field.handleChange} />
                          <FieldError errors={field.state.meta.errors} />
                        </Field>
                      )}
                    />
                  ) : null}

                  {isDerived ? (
                    <FieldGroup>
                      <form.Field
                        name="derived_attribute"
                        children={(field) => (
                          <Field>
                            <FieldLabel htmlFor="label-derived-attribute">Attribute</FieldLabel>
                            <Select
                              value={field.state.value}
                              onValueChange={(value) => {
                                field.handleChange(value as LabelDerivedAttribute);
                                form.setFieldValue("derived_values", []);
                              }}
                            >
                              <SelectTrigger id="label-derived-attribute" className="w-full">
                                <SelectValue />
                              </SelectTrigger>
                              <SelectContent>
                                <SelectGroup>
                                  {LABEL_DERIVED_ATTRIBUTE_OPTIONS.map((option) => (
                                    <SelectItem key={option.value} value={option.value}>
                                      {option.label}
                                    </SelectItem>
                                  ))}
                                </SelectGroup>
                              </SelectContent>
                            </Select>
                          </Field>
                        )}
                      />
                      <form.Field
                        name="derived_values"
                        children={(field) => (
                          <Field
                            data-invalid={field.state.meta.errors.length > 0 ? true : undefined}
                          >
                            <FieldLabel required>
                              {labelDerivedAttributeSelectorLabel(values.derived_attribute)}
                            </FieldLabel>
                            <DerivedSelector
                              attribute={values.derived_attribute}
                              value={field.state.value}
                              onChange={field.handleChange}
                            />
                            <FieldDescription>Matches linked users and groups.</FieldDescription>
                            <FieldError errors={field.state.meta.errors} />
                          </Field>
                        )}
                      />
                    </FieldGroup>
                  ) : null}
                </FieldGroup>

                {isDynamic ? (
                  <form.Field
                    name="query"
                    children={(field) => (
                      <Field
                        data-invalid={field.state.meta.errors.length > 0 ? true : undefined}
                        className="max-w-3xl"
                      >
                        <FieldLabel required>Query</FieldLabel>
                        <SQLEditor
                          ref={editorRef}
                          value={field.state.value}
                          onChange={field.handleChange}
                          onTableMetaClick={selectSchemaTable}
                          placeholder="SELECT ..."
                          invalid={field.state.meta.errors.length > 0 ? true : undefined}
                        />
                        <FieldError errors={field.state.meta.errors} />
                      </Field>
                    )}
                  />
                ) : null}

                {isDynamic ? (
                  <SchemaSidebar
                    open={schemaOpen}
                    onOpenChange={setSchemaOpen}
                    onInsertColumn={insertAtCursor}
                    selectedTable={selectedSchemaTable}
                    onSelectedTableChange={setSelectedSchemaTable}
                  />
                ) : null}
              </>
            );
          }}
        </form.Subscribe>

        <div className="flex max-w-5xl flex-col gap-2 border-t pt-4">
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
      </form>
    </PageShell>
  );
}

function membershipFromString(value: string | undefined): LabelMembershipType {
  switch (value) {
    case undefined:
      return "dynamic";
    case "manual":
    case "derived":
      return value;
    default:
      return "dynamic";
  }
}

function derivedAttributeFromString(value: string | undefined): LabelDerivedAttribute {
  switch (value) {
    case undefined:
      return "user_department";
    case "directory_group":
    case "user":
    case "user_department":
      return value;
    default:
      return "user_department";
  }
}
