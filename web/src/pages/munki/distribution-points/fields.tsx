import { revalidateLogic, useForm } from "@tanstack/react-form";
import { Trash2 } from "lucide-react";
import { z } from "zod";

import { FormActions } from "@/components/form-actions";
import { FormField } from "@/components/form-field";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Button } from "@/components/ui/button";
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@/components/ui/input-group";
import { Switch } from "@/components/ui/switch";
import { usePageFormExitGuard } from "@/hooks/use-page-form-exit-guard";
import type { MunkiDistributionPointDetail, MunkiDistributionPointMutation } from "@/lib/api";
import { firstErrorMessage, requiredString } from "@/lib/form-validation";
interface StringRow {
  rowID: string;
  value: string;
}
interface DistributionPointFormState {
  name: string;
  enabled: boolean;
  client_base_url: string;
  client_cidrs: StringRow[];
}
const cidrSchema = z.union([z.cidrv4(), z.cidrv6()]);
const distributionPointFormSchema = z
  .object({
    name: requiredString("Name"),
    enabled: z.boolean(),
    client_base_url: z
      .string()
      .trim()
      .refine((value) => value === "" || isHTTPSOrigin(value), "Base URL must be an HTTPS origin."),
    client_cidrs: z
      .array(z.object({ rowID: z.string(), value: z.string() }))
      .refine(
        (rows) => rows.every((row) => cidrSchema.safeParse(row.value.trim()).success),
        "Enter a valid IPv4 or IPv6 CIDR.",
      ),
  })
  .refine((value) => !value.enabled || value.client_base_url.length > 0, {
    path: ["client_base_url"],
    message: "Base URL is required to enable a distribution point.",
  });
export const emptyDistributionPointForm: DistributionPointFormState = {
  name: "",
  enabled: true,
  client_base_url: "",
  client_cidrs: [],
};
export function DistributionPointForm({
  initial,
  title,
  submitLabel,
  onSubmit,
  onSuccess,
  onCancel,
}: {
  initial: DistributionPointFormState;
  title?: string;
  submitLabel: string;
  onSubmit: (body: MunkiDistributionPointMutation) => Promise<number | undefined>;
  onSuccess?: (id: number | undefined) => void;
  onCancel?: () => void;
}) {
  const form = useForm({
    defaultValues: initial,
    validationLogic: revalidateLogic({ mode: "submit", modeAfterSubmission: "change" }),
    validators: { onDynamic: distributionPointFormSchema },
    onSubmit: async ({ value, formApi }) => {
      const id = await onSubmit(distributionPointBody(distributionPointFormSchema.parse(value)));
      formApi.reset(value);
      onSuccess?.(id);
    },
  });
  const exitGuard = usePageFormExitGuard({
    form,
    onDiscard: onCancel ?? (() => form.reset(initial)),
  });
  return (
    <>
      <PageShell
        render={
          <form
            noValidate
            onSubmit={(event) => {
              event.preventDefault();
              void form.handleSubmit();
            }}
          />
        }
      >
        <form.Subscribe selector={(state) => state.values.name}>
          {(name) => <PageHeader title={title ?? (name || "Distribution Point")} />}
        </form.Subscribe>

        <FieldGroup className="max-w-3xl">
          <form.Field name="name">
            {(field) => (
              <FormField
                field={field}
                label="Name"
                htmlFor="munki-distribution-point-name"
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
          <form.Field name="enabled">
            {(field) => (
              <BoolField
                id="munki-distribution-point-enabled"
                label="Enabled"
                value={field.state.value}
                onChange={field.handleChange}
              />
            )}
          </form.Field>
          <form.Field name="client_base_url">
            {(field) => (
              <FormField field={field} label="Base URL" htmlFor="munki-distribution-point-base-url">
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
          </form.Field>
          <form.Field name="client_cidrs" mode="array">
            {(field) => {
              const error = firstErrorMessage(field.state.meta.errors);
              return (
                <FieldSet
                  className="gap-4 data-[invalid=true]:text-destructive"
                  data-invalid={error ? true : undefined}
                >
                  <FieldLegend variant="label">Client CIDRs</FieldLegend>
                  <FieldGroup className="gap-2">
                    <StringArrayRows
                      rows={field.state.value}
                      invalid={Boolean(error)}
                      onReplace={(index, row) => field.replaceValue(index, row)}
                      onRemove={(index) => field.removeValue(index)}
                    />
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      className="w-fit"
                      onClick={() => field.pushValue(emptyStringRow())}
                    >
                      Add CIDR
                    </Button>
                  </FieldGroup>
                  <FieldDescription>
                    Clients in these ranges redirect to this distribution point. Empty matches
                    nothing.
                  </FieldDescription>
                  {error ? <FieldError>{error}</FieldError> : null}
                </FieldSet>
              );
            }}
          </form.Field>
        </FieldGroup>

        <FormActions
          form={form}
          submitLabel={submitLabel}
          onCancel={onCancel ? exitGuard.requestDiscard : undefined}
        />
      </PageShell>
      {exitGuard.dialog}
    </>
  );
}
function BoolField({
  id,
  label,
  description,
  value,
  onChange,
}: {
  id: string;
  label: string;
  description?: string;
  value: boolean;
  onChange: (value: boolean) => void;
}) {
  return (
    <Field orientation="horizontal">
      <FieldContent>
        <FieldLabel htmlFor={id}>{label}</FieldLabel>
        {description ? <FieldDescription>{description}</FieldDescription> : null}
      </FieldContent>
      <Switch id={id} checked={value} onCheckedChange={onChange} />
    </Field>
  );
}
function StringArrayRows({
  rows,
  invalid,
  onReplace,
  onRemove,
}: {
  rows: StringRow[];
  invalid?: boolean;
  onReplace: (index: number, row: StringRow) => void;
  onRemove: (index: number) => void;
}) {
  return (
    <>
      {rows.map((row, index) => (
        <InputGroup key={row.rowID}>
          <InputGroupInput
            aria-invalid={invalid ? true : undefined}
            value={row.value}
            onChange={(event) => onReplace(index, { ...row, value: event.target.value })}
          />
          <InputGroupAddon align="inline-end">
            <InputGroupButton
              type="button"
              variant="ghost"
              size="icon-xs"
              onClick={() => onRemove(index)}
            >
              <Trash2 />
            </InputGroupButton>
          </InputGroupAddon>
        </InputGroup>
      ))}
    </>
  );
}
export function formFromDistributionPoint(
  point: MunkiDistributionPointDetail,
): DistributionPointFormState {
  return {
    name: point.name,
    enabled: point.enabled,
    client_base_url: point.client_base_url,
    client_cidrs: stringRows(point.client_cidrs),
  };
}
function distributionPointBody(form: DistributionPointFormState): MunkiDistributionPointMutation {
  return {
    name: form.name.trim(),
    enabled: form.enabled,
    client_base_url: form.client_base_url.trim(),
    client_cidrs: cleanStringRows(form.client_cidrs),
  };
}
function emptyStringRow(): StringRow {
  return { rowID: rowID(), value: "" };
}
function stringRows(values: string[]): StringRow[] {
  return values.map((value) => ({ rowID: rowID(), value }));
}
function cleanStringRows(rows: StringRow[]): string[] {
  return rows.map((row) => row.value.trim()).filter(Boolean);
}
function rowID(): string {
  return crypto.randomUUID();
}
function isHTTPSOrigin(value: string): boolean {
  try {
    const url = new URL(value);
    return (
      url.protocol === "https:" &&
      url.host !== "" &&
      url.username === "" &&
      url.password === "" &&
      url.pathname === "/" &&
      url.search === "" &&
      url.hash === "" &&
      !value.includes("?") &&
      !value.includes("#")
    );
  } catch {
    return false;
  }
}
