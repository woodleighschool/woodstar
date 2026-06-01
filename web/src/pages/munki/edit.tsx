import { Link, useNavigate, useParams } from "@tanstack/react-router";
import { Loader2 } from "lucide-react";
import { useEffect, useMemo, useState, type SyntheticEvent } from "react";
import { z } from "zod";

import { LabelPicker } from "@/components/labels/label-picker";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import {
  useCreateMunkiDeployment,
  useCreateMunkiPackage,
  useCreateMunkiSoftwareTitle,
  useMunkiSoftwareTitle,
  type MunkiDeploymentMutation,
  type MunkiPackageMutation,
  type MunkiSoftwareTitleMutation,
} from "@/hooks/use-munki";
import { fieldErrors, requiredString } from "@/lib/form-validation";

type Intent = MunkiDeploymentMutation["intent"];

const intentOptions: { value: Intent; label: string }[] = [
  { value: "ensure_installed", label: "Install and update" },
  { value: "ensure_absent", label: "Remove" },
  { value: "update_if_present", label: "Update if present" },
  { value: "optional", label: "Self Service" },
  { value: "featured", label: "Featured" },
];

const softwareTitleSchema = z.object({
  name: requiredString("Name"),
  display_name: z.string().trim(),
  description: z.string().trim(),
  category: z.string().trim(),
  developer: z.string().trim(),
});

const packageSchema = z.object({
  name: requiredString("Name"),
  version: requiredString("Version"),
  display_name: z.string().trim(),
  description: z.string().trim(),
  category: z.string().trim(),
  developer: z.string().trim(),
  installer_type: z.string().trim(),
  eligible: z.boolean(),
  unattended_install: z.boolean(),
  unattended_uninstall: z.boolean(),
  uninstallable: z.boolean(),
});

const deploymentSchema = z
  .object({
    package_id: z.coerce.number().int().positive("Package is required."),
    intent: z.enum(["ensure_installed", "ensure_absent", "update_if_present", "optional", "featured"]),
    all_hosts: z.boolean(),
    include_label_ids: z.array(z.number().int().positive()),
    exclude_label_ids: z.array(z.number().int().positive()),
  })
  .superRefine((value, ctx) => {
    if (!value.all_hosts && value.include_label_ids.length === 0) {
      ctx.addIssue({ code: "custom", message: "Select at least one target label.", path: ["include_label_ids"] });
    }
  });

interface SoftwareTitleFormState {
  name: string;
  display_name: string;
  description: string;
  category: string;
  developer: string;
}

interface PackageFormState {
  name: string;
  version: string;
  display_name: string;
  description: string;
  category: string;
  developer: string;
  installer_type: string;
  eligible: boolean;
  unattended_install: boolean;
  unattended_uninstall: boolean;
  uninstallable: boolean;
}

interface DeploymentFormState {
  package_id: string;
  intent: Intent;
  all_hosts: boolean;
  include_label_ids: number[];
  exclude_label_ids: number[];
}

export function MunkiSoftwareTitleNewPage() {
  const navigate = useNavigate();
  const create = useCreateMunkiSoftwareTitle();
  const [form, setForm] = useState<SoftwareTitleFormState>({
    name: "",
    display_name: "",
    description: "",
    category: "",
    developer: "",
  });
  const [showErrors, setShowErrors] = useState(false);
  const parsed = useMemo(() => softwareTitleSchema.safeParse(form), [form]);
  const errors = useMemo(() => fieldErrors(parsed), [parsed]);

  async function submit() {
    const next = softwareTitleSchema.safeParse(form);
    if (!next.success) {
      setShowErrors(true);
      return;
    }
    const body: MunkiSoftwareTitleMutation = next.data;
    const title = await create.mutateAsync(body);
    void navigate({ to: "/munki/software-titles/$softwareId", params: { softwareId: String(title.id) } });
  }

  return (
    <PageShell asChild>
      <form noValidate onSubmit={(event) => runSubmit(event, submit)}>
        <PageHeader title="New Munki Software" />
        <MutationError title="Failed to Create Software" message={create.error?.message} />
        <FieldGroup className="max-w-3xl">
          <TextField
            id="munki-software-name"
            label="Name"
            required
            value={form.name}
            error={showErrors ? errors.name : undefined}
            onChange={(name) => setForm({ ...form, name })}
          />
          <TextField
            id="munki-software-display-name"
            label="Display Name"
            value={form.display_name}
            onChange={(display_name) => setForm({ ...form, display_name })}
          />
          <TextAreaField
            id="munki-software-description"
            label="Description"
            value={form.description}
            onChange={(description) => setForm({ ...form, description })}
          />
          <div className="grid gap-4 md:grid-cols-2">
            <TextField
              id="munki-software-category"
              label="Category"
              value={form.category}
              onChange={(category) => setForm({ ...form, category })}
            />
            <TextField
              id="munki-software-developer"
              label="Developer"
              value={form.developer}
              onChange={(developer) => setForm({ ...form, developer })}
            />
          </div>
          <FormActions pending={create.isPending} cancelTo="/munki/software-titles" />
        </FieldGroup>
      </form>
    </PageShell>
  );
}

export function MunkiPackageNewPage() {
  const navigate = useNavigate();
  const softwareId = useSoftwareIDParam();
  const software = useMunkiSoftwareTitle(softwareId);
  const create = useCreateMunkiPackage();
  const [form, setForm] = useState<PackageFormState>({
    name: "",
    version: "",
    display_name: "",
    description: "",
    category: "",
    developer: "",
    installer_type: "pkg",
    eligible: true,
    unattended_install: true,
    unattended_uninstall: true,
    uninstallable: false,
  });
  const [showErrors, setShowErrors] = useState(false);
  const parsed = useMemo(() => packageSchema.safeParse(form), [form]);
  const errors = useMemo(() => fieldErrors(parsed), [parsed]);

  useEffect(() => {
    if (!software.data) return;
    setForm((current) => ({
      ...current,
      name: current.name || software.data.name,
      display_name: current.display_name || software.data.display_name,
      description: current.description || software.data.description,
      category: current.category || software.data.category,
      developer: current.developer || software.data.developer,
    }));
  }, [software.data]);

  async function submit() {
    const next = packageSchema.safeParse(form);
    if (!next.success || softwareId === null) {
      setShowErrors(true);
      return;
    }
    const body: MunkiPackageMutation = {
      software_id: softwareId,
      name: next.data.name,
      version: next.data.version,
      display_name: optionalText(next.data.display_name),
      description: optionalText(next.data.description),
      category: optionalText(next.data.category),
      developer: optionalText(next.data.developer),
      eligible: next.data.eligible,
      metadata: {
        installer_type: optionalText(next.data.installer_type),
        unattended_install: next.data.unattended_install,
        unattended_uninstall: next.data.unattended_uninstall,
        uninstallable: next.data.uninstallable,
      },
    };
    await create.mutateAsync(body);
    void navigate({ to: "/munki/software-titles/$softwareId", params: { softwareId: String(softwareId) } });
  }

  return (
    <PageShell asChild>
      <form noValidate onSubmit={(event) => runSubmit(event, submit)}>
        <PageHeader title="New Package" />
        <MutationError title="Failed to Create Package" message={create.error?.message ?? software.error?.message} />
        <FieldGroup className="max-w-3xl">
          <TextField
            id="munki-package-name"
            label="Name"
            required
            value={form.name}
            error={showErrors ? errors.name : undefined}
            placeholder={software.data?.name}
            onChange={(name) => setForm({ ...form, name })}
          />
          <div className="grid gap-4 md:grid-cols-2">
            <TextField
              id="munki-package-version"
              label="Version"
              required
              value={form.version}
              error={showErrors ? errors.version : undefined}
              onChange={(version) => setForm({ ...form, version })}
            />
            <TextField
              id="munki-package-display-name"
              label="Display Name"
              value={form.display_name}
              placeholder={software.data?.display_name}
              onChange={(display_name) => setForm({ ...form, display_name })}
            />
          </div>
          <TextAreaField
            id="munki-package-description"
            label="Description"
            value={form.description}
            placeholder={software.data?.description}
            onChange={(description) => setForm({ ...form, description })}
          />
          <div className="grid gap-4 md:grid-cols-3">
            <TextField
              id="munki-package-category"
              label="Category"
              value={form.category}
              placeholder={software.data?.category}
              onChange={(category) => setForm({ ...form, category })}
            />
            <TextField
              id="munki-package-developer"
              label="Developer"
              value={form.developer}
              placeholder={software.data?.developer}
              onChange={(developer) => setForm({ ...form, developer })}
            />
            <TextField
              id="munki-package-installer-type"
              label="Installer Type"
              value={form.installer_type}
              onChange={(installer_type) => setForm({ ...form, installer_type })}
            />
          </div>
          <CheckboxField
            id="munki-package-eligible"
            label="Available for deployment"
            checked={form.eligible}
            onChange={(eligible) => setForm({ ...form, eligible })}
          />
          <div className="grid gap-3 md:grid-cols-3">
            <CheckboxField
              id="munki-package-unattended-install"
              label="Unattended install"
              checked={form.unattended_install}
              onChange={(unattended_install) => setForm({ ...form, unattended_install })}
            />
            <CheckboxField
              id="munki-package-unattended-uninstall"
              label="Unattended uninstall"
              checked={form.unattended_uninstall}
              onChange={(unattended_uninstall) => setForm({ ...form, unattended_uninstall })}
            />
            <CheckboxField
              id="munki-package-uninstallable"
              label="Uninstallable"
              checked={form.uninstallable}
              onChange={(uninstallable) => setForm({ ...form, uninstallable })}
            />
          </div>
          <FormActions
            pending={create.isPending}
            cancelTo="/munki/software-titles/$softwareId"
            cancelParams={{ softwareId: String(softwareId ?? "") }}
          />
        </FieldGroup>
      </form>
    </PageShell>
  );
}

export function MunkiDeploymentNewPage() {
  const navigate = useNavigate();
  const softwareId = useSoftwareIDParam();
  const software = useMunkiSoftwareTitle(softwareId);
  const create = useCreateMunkiDeployment();
  const packages = software.data?.packages ?? [];
  const [form, setForm] = useState<DeploymentFormState>({
    package_id: "",
    intent: "ensure_installed",
    all_hosts: true,
    include_label_ids: [],
    exclude_label_ids: [],
  });
  const [showErrors, setShowErrors] = useState(false);
  const parsed = useMemo(
    () =>
      deploymentSchema.safeParse({
        ...form,
        package_id: form.package_id,
      }),
    [form],
  );
  const errors = useMemo(() => fieldErrors(parsed), [parsed]);

  useEffect(() => {
    const firstPackage = software.data?.packages?.[0];
    if (!firstPackage) return;
    setForm((current) => (current.package_id ? current : { ...current, package_id: String(firstPackage.id) }));
  }, [software.data?.packages]);

  async function submit() {
    const next = deploymentSchema.safeParse(form);
    if (!next.success || softwareId === null) {
      setShowErrors(true);
      return;
    }
    const body: MunkiDeploymentMutation = {
      package_id: next.data.package_id,
      intent: next.data.intent,
      all_hosts: next.data.all_hosts,
      include_label_ids: next.data.all_hosts ? [] : next.data.include_label_ids,
      exclude_label_ids: next.data.exclude_label_ids,
    };
    await create.mutateAsync(body);
    void navigate({ to: "/munki/software-titles/$softwareId", params: { softwareId: String(softwareId) } });
  }

  return (
    <PageShell asChild>
      <form noValidate onSubmit={(event) => runSubmit(event, submit)}>
        <PageHeader title="New Deployment" />
        <MutationError title="Failed to Create Deployment" message={create.error?.message ?? software.error?.message} />
        <FieldGroup className="max-w-3xl">
          <Field data-invalid={showErrors && errors.package_id ? true : undefined}>
            <FieldLabel htmlFor="munki-deployment-package" required>
              Package
            </FieldLabel>
            <Select value={form.package_id} onValueChange={(package_id) => setForm({ ...form, package_id })}>
              <SelectTrigger id="munki-deployment-package" className="w-full">
                <SelectValue placeholder={software.isLoading ? "Loading..." : "Select Package"} />
              </SelectTrigger>
              <SelectContent>
                {packages.map((pkg) => (
                  <SelectItem key={pkg.id} value={String(pkg.id)}>
                    {pkg.version}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {showErrors && errors.package_id ? <FieldError>{errors.package_id}</FieldError> : null}
          </Field>

          <Field>
            <FieldLabel htmlFor="munki-deployment-intent" required>
              Intent
            </FieldLabel>
            <Select value={form.intent} onValueChange={(intent) => setForm({ ...form, intent: intent as Intent })}>
              <SelectTrigger id="munki-deployment-intent" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {intentOptions.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>

          <CheckboxField
            id="munki-deployment-all-hosts"
            label="All devices"
            checked={form.all_hosts}
            onChange={(all_hosts) => setForm({ ...form, all_hosts })}
          />

          {form.all_hosts ? null : (
            <Field data-invalid={showErrors && errors.include_label_ids ? true : undefined}>
              <FieldLabel required>Target Labels</FieldLabel>
              <LabelPicker
                value={form.include_label_ids}
                selectionMode="multiple"
                includeBuiltins
                unavailableLabelIDs={form.exclude_label_ids}
                invalid={showErrors && errors.include_label_ids ? true : undefined}
                onChange={(include_label_ids) => setForm({ ...form, include_label_ids })}
              />
              {showErrors && errors.include_label_ids ? <FieldError>{errors.include_label_ids}</FieldError> : null}
            </Field>
          )}

          <Field>
            <FieldLabel>Excluded Labels</FieldLabel>
            <LabelPicker
              value={form.exclude_label_ids}
              selectionMode="multiple"
              includeBuiltins
              unavailableLabelIDs={form.include_label_ids}
              onChange={(exclude_label_ids) => setForm({ ...form, exclude_label_ids })}
            />
          </Field>

          <FormActions
            pending={create.isPending}
            cancelTo="/munki/software-titles/$softwareId"
            cancelParams={{ softwareId: String(softwareId ?? "") }}
          />
        </FieldGroup>
      </form>
    </PageShell>
  );
}

function useSoftwareIDParam() {
  const params = useParams({ strict: false });
  const id = Number(params.softwareId);
  return Number.isFinite(id) && id > 0 ? id : null;
}

function TextField({
  id,
  label,
  required,
  value,
  error,
  placeholder,
  onChange,
}: {
  id: string;
  label: string;
  required?: boolean;
  value: string;
  error?: string;
  placeholder?: string;
  onChange: (value: string) => void;
}) {
  return (
    <Field data-invalid={error ? true : undefined}>
      <FieldLabel htmlFor={id} required={required}>
        {label}
      </FieldLabel>
      <Input id={id} value={value} placeholder={placeholder} onChange={(event) => onChange(event.target.value)} />
      {error ? <FieldError>{error}</FieldError> : null}
    </Field>
  );
}

function TextAreaField({
  id,
  label,
  value,
  placeholder,
  onChange,
}: {
  id: string;
  label: string;
  value: string;
  placeholder?: string;
  onChange: (value: string) => void;
}) {
  return (
    <Field>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Textarea id={id} value={value} placeholder={placeholder} onChange={(event) => onChange(event.target.value)} />
    </Field>
  );
}

function CheckboxField({
  id,
  label,
  checked,
  onChange,
}: {
  id: string;
  label: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
}) {
  return (
    <Field orientation="horizontal">
      <Checkbox id={id} checked={checked} onCheckedChange={(value) => onChange(value === true)} />
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
    </Field>
  );
}

function FormActions({
  pending,
  cancelTo,
  cancelParams,
}: {
  pending: boolean;
  cancelTo: string;
  cancelParams?: Record<string, string>;
}) {
  return (
    <div className="flex items-center gap-2">
      <Button type="submit" size="sm" disabled={pending}>
        {pending ? <Loader2 data-icon="inline-start" className="animate-spin" /> : null}
        Save
      </Button>
      <Button asChild type="button" variant="outline" size="sm">
        <Link to={cancelTo} params={cancelParams}>
          Cancel
        </Link>
      </Button>
    </div>
  );
}

function MutationError({ title, message }: { title: string; message?: string }) {
  if (!message) return null;
  return (
    <Alert variant="destructive">
      <AlertTitle>{title}</AlertTitle>
      <AlertDescription>{message}</AlertDescription>
    </Alert>
  );
}

function optionalText(value: string) {
  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
}

function runSubmit(event: SyntheticEvent<HTMLFormElement>, submit: () => Promise<void>) {
  event.preventDefault();
  void submit();
}
