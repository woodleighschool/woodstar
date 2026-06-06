import { useForm } from "@tanstack/react-form";
import { Link, useNavigate, useParams } from "@tanstack/react-router";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { Loader2 } from "lucide-react";
import { useCallback, useRef, useState } from "react";
import { z } from "zod";

import { SchemaSidebar } from "@/components/editor/schema-sidebar";
import { SQLEditor } from "@/components/editor/sql-editor";
import { DerivedSelector, HostSelector } from "@/components/labels/label-membership-selectors";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { useCreateLabel, useLabel, useUpdateLabel, type LabelMutation } from "@/hooks/use-labels";
import { useSchemaSidebar } from "@/hooks/use-schema-sidebar";
import { fieldErrors, requiredString, selectedIDArray } from "@/lib/form-validation";
import {
  LABEL_DERIVED_ATTRIBUTE_OPTIONS,
  LABEL_DERIVED_ATTRIBUTE_VALUES,
  LABEL_MEMBERSHIP_OPTIONS,
  LABEL_MEMBERSHIP_TYPES,
  LABEL_MEMBERSHIP_VALUES,
  labelDerivedAttributeSelectorLabel,
  type LabelDerivedAttribute,
  type LabelMembershipType,
} from "@/lib/labels";
import { sqlSyntaxError } from "@/lib/sql-validation";
import { cn } from "@/lib/utils";

interface FormState {
  name: string;
  description: string;
  query: string;
  host_ids: number[];
  derived_attribute: LabelDerivedAttribute;
  derived_values: string[];
  label_membership_type: LabelMembershipType;
}

const empty: FormState = {
  name: "",
  description: "",
  query: "select 1 from os_version where major >= 13;",
  host_ids: [],
  derived_attribute: "user_department",
  derived_values: [],
  label_membership_type: "dynamic",
};

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
        ctx.addIssue({ code: "custom", message: query.error.issues[0]?.message ?? "Invalid query.", path: ["query"] });
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

export function LabelMutationPage({ mode }: { mode: "create" | "edit" }) {
  const params = useParams({ strict: false });
  const labelId = params.labelId ?? "";
  const labelID = mode === "edit" ? Number(labelId) : null;
  const detail = useLabel(labelID);

  if (mode === "edit") {
    if (detail.error) {
      return (
        <PageShell>
          <Alert variant="destructive">
            <AlertTitle>Failed to Load Label</AlertTitle>
            <AlertDescription>{detail.error.message}</AlertDescription>
          </Alert>
        </PageShell>
      );
    }
    if (!detail.data) {
      return (
        <PageShell className="text-muted-foreground flex-row items-center gap-2 text-sm">
          <Loader2 className="size-4 animate-spin" /> Loading Label...
        </PageShell>
      );
    }
    if (detail.data.label_type === "builtin") {
      return (
        <PageShell>
          <Alert>
            <AlertTitle>Built-In Label</AlertTitle>
            <AlertDescription>Built-in labels are managed by Woodstar and cannot be edited.</AlertDescription>
          </Alert>
        </PageShell>
      );
    }
  }

  const initial: FormState =
    mode === "edit" && detail.data
      ? {
          name: detail.data.name,
          description: detail.data.description,
          query: detail.data.query ?? empty.query,
          host_ids: detail.data.host_ids ?? [],
          derived_attribute: derivedAttributeFromString(detail.data.criteria?.attribute),
          derived_values: detail.data.criteria?.values ?? [],
          label_membership_type: membershipFromString(detail.data.label_membership_type),
        }
      : empty;

  return <LabelEditForm key={labelId || "new"} mode={mode} labelId={labelID} initial={initial} />;
}

export function LabelNewPage() {
  return <LabelMutationPage mode="create" />;
}

export function LabelEditPage() {
  return <LabelMutationPage mode="edit" />;
}

function LabelEditForm({
  mode,
  labelId,
  initial,
}: {
  mode: "create" | "edit";
  labelId: number | null;
  initial: FormState;
}) {
  const navigate = useNavigate();
  const createLabel = useCreateLabel();
  const updateLabel = useUpdateLabel(labelId);
  const [schemaOpen, setSchemaOpen] = useSchemaSidebar();
  const [selectedSchemaTable, setSelectedSchemaTable] = useState<string | null>(null);
  const editorRef = useRef<ReactCodeMirrorRef>(null);
  const form = useForm({
    defaultValues: initial,
    validators: {
      onSubmit: labelFormSchema,
    },
    onSubmit: async ({ value }) => {
      const cleaned = labelFormSchema.parse(value);
      const body: LabelMutation = {
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
      if (mode === "create") {
        await createLabel.mutateAsync(body);
      } else {
        await updateLabel.mutateAsync(body);
      }
      void navigate({ to: "/labels" });
    },
  });
  const pending = createLabel.isPending || updateLabel.isPending;

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
    <PageShell asChild className={cn("h-full transition-[padding] duration-200 ease-out", schemaOpen && "pr-[21rem]")}>
      <form
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit();
        }}
      >
        <PageHeader title={mode === "create" ? "New Label" : "Edit Label"} />
        <form.Subscribe selector={(state) => ({ values: state.values, submissionAttempts: state.submissionAttempts })}>
          {({ values, submissionAttempts }) => {
            const errors = fieldErrors(labelFormSchema.safeParse(values));
            const showErrors = submissionAttempts > 0;
            const isDynamic = values.label_membership_type === "dynamic";
            const isManual = values.label_membership_type === "manual";
            const isDerived = values.label_membership_type === "derived";
            const memberOption = LABEL_MEMBERSHIP_TYPES[values.label_membership_type];

            return (
              <>
                <FieldGroup className="max-w-5xl">
                  <form.Field
                    name="name"
                    children={(field) => (
                      <Field data-invalid={showErrors && errors.name ? true : undefined}>
                        <FieldLabel htmlFor="label-name" required>
                          Name
                        </FieldLabel>
                        <Input
                          id="label-name"
                          name={field.name}
                          required
                          aria-invalid={showErrors && errors.name ? true : undefined}
                          value={field.state.value}
                          onBlur={field.handleBlur}
                          onChange={(event) => field.handleChange(event.target.value)}
                        />
                        {showErrors && errors.name ? <FieldError>{errors.name}</FieldError> : null}
                      </Field>
                    )}
                  />

                  <form.Field
                    name="description"
                    children={(field) => (
                      <Field>
                        <FieldLabel htmlFor="label-description">Description</FieldLabel>
                        <Textarea
                          id="label-description"
                          name={field.name}
                          rows={3}
                          placeholder="Why this label exists"
                          value={field.state.value}
                          onBlur={field.handleBlur}
                          onChange={(event) => field.handleChange(event.target.value)}
                        />
                      </Field>
                    )}
                  />

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
                        <Field data-invalid={showErrors && errors.host_ids ? true : undefined}>
                          <FieldLabel>Hosts</FieldLabel>
                          <HostSelector value={field.state.value} onChange={field.handleChange} />
                          {showErrors && errors.host_ids ? <FieldError>{errors.host_ids}</FieldError> : null}
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
                          <Field data-invalid={showErrors && errors.derived_values ? true : undefined}>
                            <FieldLabel required>
                              {labelDerivedAttributeSelectorLabel(values.derived_attribute)}
                            </FieldLabel>
                            <DerivedSelector
                              attribute={values.derived_attribute}
                              value={field.state.value}
                              onChange={field.handleChange}
                            />
                            <FieldDescription>Matches linked users and groups.</FieldDescription>
                            {showErrors && errors.derived_values ? (
                              <FieldError>{errors.derived_values}</FieldError>
                            ) : null}
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
                      <Field data-invalid={showErrors && errors.query ? true : undefined} className="max-w-3xl">
                        <FieldLabel required>Query</FieldLabel>
                        <SQLEditor
                          ref={editorRef}
                          value={field.state.value}
                          onChange={field.handleChange}
                          onTableMetaClick={selectSchemaTable}
                          placeholder="SELECT ..."
                          invalid={showErrors && errors.query ? true : undefined}
                        />
                        {showErrors && errors.query ? <FieldError>{errors.query}</FieldError> : null}
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

                <div className="flex max-w-5xl items-center gap-2 border-t pt-4">
                  <Button type="submit" size="sm" disabled={pending}>
                    {pending ? "Saving..." : "Save"}
                  </Button>
                  {mode === "edit" ? (
                    <Button asChild type="button" variant="ghost" size="sm">
                      <Link to="/labels">Cancel</Link>
                    </Button>
                  ) : null}
                </div>
              </>
            );
          }}
        </form.Subscribe>
      </form>
    </PageShell>
  );
}

function membershipFromString(value: string | undefined): LabelMembershipType {
  switch (value) {
    case "manual":
    case "derived":
      return value;
    default:
      return "dynamic";
  }
}

function derivedAttributeFromString(value: string | undefined): LabelDerivedAttribute {
  switch (value) {
    case "directory_group":
    case "user":
    case "user_department":
      return value;
    default:
      return "user_department";
  }
}
