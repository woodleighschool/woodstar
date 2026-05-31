import { Link, useNavigate } from "@tanstack/react-router";
import { Loader2 } from "lucide-react";
import { useMemo, useState } from "react";
import { z } from "zod";

import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import {
  useCreateMunkiArtifact,
  useCreateMunkiAssignment,
  useCreateMunkiRelease,
  useCreateMunkiSoftwareTitle,
  useMunkiArtifacts,
  useMunkiReleases,
  useMunkiSoftwareTitles,
  type MunkiArtifactMutation,
  type MunkiAssignmentMutation,
  type MunkiReleaseMutation,
  type MunkiSoftwareTitleMutation,
} from "@/hooks/use-munki";
import { fieldErrors, requiredString } from "@/lib/form-validation";

type Intent = MunkiAssignmentMutation["intent"];

const intentOptions: { value: Intent; label: string }[] = [
  { value: "ensure_installed", label: "Install" },
  { value: "ensure_absent", label: "Remove" },
  { value: "update_if_present", label: "Update" },
  { value: "optional", label: "Optional" },
  { value: "featured", label: "Featured" },
];

const softwareTitleSchema = z.object({
  name: requiredString("Name"),
  display_name: z.string().trim(),
  description: z.string().trim(),
  category: z.string().trim(),
  developer: z.string().trim(),
});

const artifactSchema = z.object({
  kind: z.enum(["package", "icon"]),
  display_name: z.string().trim(),
  location: requiredString("Location").refine((value) => isRelativeMunkiPath(value), {
    message: "Location must be a relative Munki path.",
  }),
  content_type: z.string().trim(),
  size_bytes: z.coerce.number().int().min(0, "Size must be zero or greater."),
  sha256: z
    .string()
    .trim()
    .regex(/^[0-9a-f]{64}$/, "SHA-256 must be 64 lowercase hex characters."),
  storage_key: requiredString("Storage key").refine((value) => !value.startsWith("/"), {
    message: "Storage key must be relative.",
  }),
});

const releaseSchema = z.object({
  software_id: z.coerce.number().int().positive("Software is required."),
  name: requiredString("Name"),
  version: requiredString("Version"),
  display_name: z.string().trim(),
  installer_artifact_id: z.coerce.number().int().positive().optional(),
  eligible: z.boolean(),
  pkginfo: requiredString("Pkginfo"),
});

const assignmentSchema = z
  .object({
    release_id: z.coerce.number().int().positive("Release is required."),
    intent: z.enum(["ensure_installed", "ensure_absent", "update_if_present", "optional", "featured"]),
    all_hosts: z.boolean(),
    include_label_ids: z.string().trim(),
    exclude_label_ids: z.string().trim(),
    include_host_ids: z.string().trim(),
    exclude_host_ids: z.string().trim(),
  })
  .superRefine((value, ctx) => {
    for (const key of ["include_label_ids", "exclude_label_ids", "include_host_ids", "exclude_host_ids"] as const) {
      const result = parseIDList(value[key]);
      if (!result.ok) {
        ctx.addIssue({ code: "custom", message: result.message, path: [key] });
      }
    }
    if (
      !value.all_hosts &&
      parseIDList(value.include_label_ids).ids.length === 0 &&
      parseIDList(value.include_host_ids).ids.length === 0
    ) {
      ctx.addIssue({ code: "custom", message: "Assignment scope is required.", path: ["all_hosts"] });
    }
  });

interface SoftwareTitleFormState {
  name: string;
  display_name: string;
  description: string;
  category: string;
  developer: string;
}

interface ArtifactFormState {
  kind: "package" | "icon";
  display_name: string;
  location: string;
  content_type: string;
  size_bytes: string;
  sha256: string;
  storage_key: string;
}

interface ReleaseFormState {
  software_id: string;
  name: string;
  version: string;
  display_name: string;
  installer_artifact_id: string;
  eligible: boolean;
  pkginfo: string;
}

interface AssignmentFormState {
  release_id: string;
  intent: Intent;
  all_hosts: boolean;
  include_label_ids: string;
  exclude_label_ids: string;
  include_host_ids: string;
  exclude_host_ids: string;
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
    await create.mutateAsync(body);
    void navigate({ to: "/munki/software-titles" });
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
          <Field>
            <FieldLabel htmlFor="munki-software-description">Description</FieldLabel>
            <Textarea
              id="munki-software-description"
              rows={3}
              value={form.description}
              onChange={(event) => setForm({ ...form, description: event.target.value })}
            />
          </Field>
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
        </FieldGroup>
        <FormActions pending={create.isPending} cancelTo="/munki/software-titles" />
      </form>
    </PageShell>
  );
}

export function MunkiArtifactNewPage() {
  const navigate = useNavigate();
  const create = useCreateMunkiArtifact();
  const [form, setForm] = useState<ArtifactFormState>({
    kind: "package",
    display_name: "",
    location: "",
    content_type: "application/octet-stream",
    size_bytes: "0",
    sha256: "",
    storage_key: "",
  });
  const [showErrors, setShowErrors] = useState(false);
  const parsed = useMemo(() => artifactSchema.safeParse(form), [form]);
  const errors = useMemo(() => fieldErrors(parsed), [parsed]);

  async function submit() {
    const next = artifactSchema.safeParse(form);
    if (!next.success) {
      setShowErrors(true);
      return;
    }
    const body: MunkiArtifactMutation = next.data;
    await create.mutateAsync(body);
    void navigate({ to: "/munki/artifacts" });
  }

  return (
    <PageShell asChild>
      <form noValidate onSubmit={(event) => runSubmit(event, submit)}>
        <PageHeader title="New Munki Artifact" />
        <MutationError title="Failed to Create Artifact" message={create.error?.message} />
        <FieldGroup className="max-w-3xl">
          <Field>
            <FieldLabel>Kind</FieldLabel>
            <Select
              value={form.kind}
              onValueChange={(kind) => setForm({ ...form, kind: kind as ArtifactFormState["kind"] })}
            >
              <SelectTrigger className="w-48">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="package">Package</SelectItem>
                <SelectItem value="icon">Icon</SelectItem>
              </SelectContent>
            </Select>
          </Field>
          <TextField
            id="munki-artifact-display-name"
            label="Display Name"
            value={form.display_name}
            onChange={(display_name) => setForm({ ...form, display_name })}
          />
          <div className="grid gap-4 md:grid-cols-2">
            <TextField
              id="munki-artifact-location"
              label="Location"
              required
              value={form.location}
              error={showErrors ? errors.location : undefined}
              onChange={(location) => setForm({ ...form, location })}
            />
            <TextField
              id="munki-artifact-content-type"
              label="Content Type"
              value={form.content_type}
              onChange={(content_type) => setForm({ ...form, content_type })}
            />
          </div>
          <TextField
            id="munki-artifact-size"
            label="Size Bytes"
            required
            inputMode="numeric"
            value={form.size_bytes}
            error={showErrors ? errors.size_bytes : undefined}
            onChange={(size_bytes) => setForm({ ...form, size_bytes })}
          />
          <TextField
            id="munki-artifact-sha256"
            label="SHA-256"
            required
            value={form.sha256}
            error={showErrors ? errors.sha256 : undefined}
            onChange={(sha256) => setForm({ ...form, sha256 })}
          />
          <TextField
            id="munki-artifact-storage-key"
            label="Storage Key"
            required
            value={form.storage_key}
            error={showErrors ? errors.storage_key : undefined}
            onChange={(storage_key) => setForm({ ...form, storage_key })}
          />
        </FieldGroup>
        <FormActions pending={create.isPending} cancelTo="/munki/artifacts" />
      </form>
    </PageShell>
  );
}

export function MunkiReleaseNewPage() {
  const navigate = useNavigate();
  const create = useCreateMunkiRelease();
  const titles = useMunkiSoftwareTitles({ page_size: 1000 });
  const artifacts = useMunkiArtifacts({ page_size: 1000 });
  const [form, setForm] = useState<ReleaseFormState>({
    software_id: "",
    name: "",
    version: "",
    display_name: "",
    installer_artifact_id: "none",
    eligible: true,
    pkginfo: '{\n  "name": "",\n  "version": ""\n}',
  });
  const [showErrors, setShowErrors] = useState(false);
  const [pkginfoError, setPkginfoError] = useState<string | undefined>();
  const parsed = useMemo(() => releaseSchema.safeParse(normalizeReleaseForm(form)), [form]);
  const errors = useMemo(() => fieldErrors(parsed), [parsed]);
  const packageArtifacts = (artifacts.data?.items ?? []).filter((artifact) => artifact.kind === "package");

  async function submit() {
    const next = releaseSchema.safeParse(normalizeReleaseForm(form));
    if (!next.success) {
      setShowErrors(true);
      return;
    }
    let decoded: unknown;
    try {
      decoded = JSON.parse(next.data.pkginfo) as unknown;
    } catch {
      setPkginfoError("Pkginfo must be valid JSON.");
      setShowErrors(true);
      return;
    }
    if (decoded === null || Array.isArray(decoded) || typeof decoded !== "object") {
      setPkginfoError("Pkginfo must be a JSON object.");
      setShowErrors(true);
      return;
    }
    const pkginfo = decoded as Record<string, unknown>;
    setPkginfoError(undefined);
    const body: MunkiReleaseMutation = {
      software_id: next.data.software_id,
      name: next.data.name,
      version: next.data.version,
      display_name: next.data.display_name || undefined,
      installer_artifact_id: next.data.installer_artifact_id,
      pkginfo,
      eligible: next.data.eligible,
    };
    await create.mutateAsync(body);
    void navigate({ to: "/munki/releases" });
  }

  return (
    <PageShell asChild>
      <form noValidate onSubmit={(event) => runSubmit(event, submit)}>
        <PageHeader title="New Munki Release" />
        <MutationError title="Failed to Create Release" message={create.error?.message} />
        <LoadError message={titles.error?.message ?? artifacts.error?.message} />
        <FieldGroup className="max-w-4xl">
          <Field data-invalid={showErrors && errors.software_id ? true : undefined}>
            <FieldLabel required>Software</FieldLabel>
            <Select
              value={form.software_id || undefined}
              onValueChange={(software_id) => setForm({ ...form, software_id })}
            >
              <SelectTrigger className="w-full max-w-md">
                <SelectValue placeholder={titles.isLoading ? "Loading..." : "Select software"} />
              </SelectTrigger>
              <SelectContent>
                {(titles.data?.items ?? []).map((title) => (
                  <SelectItem key={title.id} value={String(title.id)}>
                    {title.display_name || title.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {showErrors && errors.software_id ? <FieldError>{errors.software_id}</FieldError> : null}
          </Field>
          <div className="grid gap-4 md:grid-cols-2">
            <TextField
              id="munki-release-name"
              label="Name"
              required
              value={form.name}
              error={showErrors ? errors.name : undefined}
              onChange={(name) => setForm({ ...form, name })}
            />
            <TextField
              id="munki-release-version"
              label="Version"
              required
              value={form.version}
              error={showErrors ? errors.version : undefined}
              onChange={(version) => setForm({ ...form, version })}
            />
          </div>
          <TextField
            id="munki-release-display-name"
            label="Display Name"
            value={form.display_name}
            onChange={(display_name) => setForm({ ...form, display_name })}
          />
          <Field>
            <FieldLabel>Installer Artifact</FieldLabel>
            <Select
              value={form.installer_artifact_id}
              onValueChange={(installer_artifact_id) => setForm({ ...form, installer_artifact_id })}
            >
              <SelectTrigger className="w-full max-w-md">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">None</SelectItem>
                {packageArtifacts.map((artifact) => (
                  <SelectItem key={artifact.id} value={String(artifact.id)}>
                    {artifact.display_name || artifact.location}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>
          <BooleanField
            id="munki-release-eligible"
            label="Eligible"
            checked={form.eligible}
            onChange={(eligible) => setForm({ ...form, eligible })}
          />
          <Field data-invalid={showErrors && (errors.pkginfo || pkginfoError) ? true : undefined}>
            <FieldLabel htmlFor="munki-release-pkginfo" required>
              Pkginfo
            </FieldLabel>
            <Textarea
              id="munki-release-pkginfo"
              rows={10}
              className="font-mono text-sm"
              value={form.pkginfo}
              onChange={(event) => {
                setPkginfoError(undefined);
                setForm({ ...form, pkginfo: event.target.value });
              }}
            />
            {showErrors && (errors.pkginfo || pkginfoError) ? (
              <FieldError>{errors.pkginfo || pkginfoError}</FieldError>
            ) : null}
          </Field>
        </FieldGroup>
        <FormActions pending={create.isPending} cancelTo="/munki/releases" />
      </form>
    </PageShell>
  );
}

export function MunkiAssignmentNewPage() {
  const navigate = useNavigate();
  const create = useCreateMunkiAssignment();
  const releases = useMunkiReleases({ page_size: 1000 });
  const [form, setForm] = useState<AssignmentFormState>({
    release_id: "",
    intent: "ensure_installed",
    all_hosts: true,
    include_label_ids: "",
    exclude_label_ids: "",
    include_host_ids: "",
    exclude_host_ids: "",
  });
  const [showErrors, setShowErrors] = useState(false);
  const parsed = useMemo(() => assignmentSchema.safeParse(form), [form]);
  const errors = useMemo(() => fieldErrors(parsed), [parsed]);

  async function submit() {
    const next = assignmentSchema.safeParse(form);
    if (!next.success) {
      setShowErrors(true);
      return;
    }
    const body: MunkiAssignmentMutation = {
      release_id: next.data.release_id,
      intent: next.data.intent,
      all_hosts: next.data.all_hosts,
      include_label_ids: parseIDList(next.data.include_label_ids).ids,
      exclude_label_ids: parseIDList(next.data.exclude_label_ids).ids,
      include_host_ids: parseIDList(next.data.include_host_ids).ids,
      exclude_host_ids: parseIDList(next.data.exclude_host_ids).ids,
    };
    await create.mutateAsync(body);
    void navigate({ to: "/munki/assignments" });
  }

  return (
    <PageShell asChild>
      <form noValidate onSubmit={(event) => runSubmit(event, submit)}>
        <PageHeader title="New Munki Assignment" />
        <MutationError title="Failed to Create Assignment" message={create.error?.message} />
        <LoadError message={releases.error?.message} />
        <FieldGroup className="max-w-4xl">
          <Field data-invalid={showErrors && errors.release_id ? true : undefined}>
            <FieldLabel required>Release</FieldLabel>
            <Select
              value={form.release_id || undefined}
              onValueChange={(release_id) => setForm({ ...form, release_id })}
            >
              <SelectTrigger className="w-full max-w-md">
                <SelectValue placeholder={releases.isLoading ? "Loading..." : "Select release"} />
              </SelectTrigger>
              <SelectContent>
                {(releases.data?.items ?? []).map((release) => (
                  <SelectItem key={release.id} value={String(release.id)}>
                    {release.display_name || release.name} {release.version}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {showErrors && errors.release_id ? <FieldError>{errors.release_id}</FieldError> : null}
          </Field>
          <Field>
            <FieldLabel>Intent</FieldLabel>
            <Select value={form.intent} onValueChange={(intent) => setForm({ ...form, intent: intent as Intent })}>
              <SelectTrigger className="w-56">
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
          <BooleanField
            id="munki-assignment-all-hosts"
            label="All Hosts"
            checked={form.all_hosts}
            error={showErrors ? errors.all_hosts : undefined}
            onChange={(all_hosts) => setForm({ ...form, all_hosts })}
          />
          <div className="grid gap-4 md:grid-cols-2">
            <IDListField
              id="munki-assignment-include-labels"
              label="Include Label IDs"
              value={form.include_label_ids}
              error={showErrors ? errors.include_label_ids : undefined}
              onChange={(include_label_ids) => setForm({ ...form, include_label_ids })}
            />
            <IDListField
              id="munki-assignment-exclude-labels"
              label="Exclude Label IDs"
              value={form.exclude_label_ids}
              error={showErrors ? errors.exclude_label_ids : undefined}
              onChange={(exclude_label_ids) => setForm({ ...form, exclude_label_ids })}
            />
            <IDListField
              id="munki-assignment-include-hosts"
              label="Include Host IDs"
              value={form.include_host_ids}
              error={showErrors ? errors.include_host_ids : undefined}
              onChange={(include_host_ids) => setForm({ ...form, include_host_ids })}
            />
            <IDListField
              id="munki-assignment-exclude-hosts"
              label="Exclude Host IDs"
              value={form.exclude_host_ids}
              error={showErrors ? errors.exclude_host_ids : undefined}
              onChange={(exclude_host_ids) => setForm({ ...form, exclude_host_ids })}
            />
          </div>
        </FieldGroup>
        <FormActions pending={create.isPending} cancelTo="/munki/assignments" />
      </form>
    </PageShell>
  );
}

function TextField({
  id,
  label,
  value,
  onChange,
  error,
  required,
  inputMode,
}: {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  error?: string;
  required?: boolean;
  inputMode?: "numeric";
}) {
  return (
    <Field data-invalid={error ? true : undefined}>
      <FieldLabel htmlFor={id} required={required}>
        {label}
      </FieldLabel>
      <Input
        id={id}
        required={required}
        aria-invalid={error ? true : undefined}
        inputMode={inputMode}
        value={value}
        onChange={(event) => onChange(event.target.value)}
      />
      {error ? <FieldError>{error}</FieldError> : null}
    </Field>
  );
}

function BooleanField({
  id,
  label,
  checked,
  onChange,
  error,
}: {
  id: string;
  label: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
  error?: string;
}) {
  return (
    <Field data-invalid={error ? true : undefined}>
      <div className="flex items-center gap-2">
        <Checkbox id={id} checked={checked} onCheckedChange={(value) => onChange(value === true)} />
        <FieldLabel htmlFor={id}>{label}</FieldLabel>
      </div>
      {error ? <FieldError>{error}</FieldError> : null}
    </Field>
  );
}

function IDListField({
  id,
  label,
  value,
  onChange,
  error,
}: {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  error?: string;
}) {
  return (
    <Field data-invalid={error ? true : undefined}>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Input id={id} value={value} onChange={(event) => onChange(event.target.value)} />
      {error ? <FieldError>{error}</FieldError> : null}
    </Field>
  );
}

function FormActions({ pending, cancelTo }: { pending: boolean; cancelTo: string }) {
  return (
    <div className="flex items-center gap-2 border-t pt-4">
      <Button type="submit" size="sm" disabled={pending}>
        {pending ? (
          <>
            <Loader2 className="size-4 animate-spin" />
            Saving
          </>
        ) : (
          "Save"
        )}
      </Button>
      <Button type="button" size="sm" variant="ghost" asChild>
        <Link to={cancelTo}>Cancel</Link>
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

function LoadError({ message }: { message?: string }) {
  if (!message) return null;
  return (
    <Alert variant="destructive">
      <AlertTitle>Failed to Load Form Data</AlertTitle>
      <AlertDescription>{message}</AlertDescription>
    </Alert>
  );
}

function runSubmit(event: { preventDefault: () => void }, submit: () => Promise<void>) {
  event.preventDefault();
  void submit();
}

function normalizeReleaseForm(form: ReleaseFormState) {
  return {
    ...form,
    installer_artifact_id: form.installer_artifact_id === "none" ? undefined : form.installer_artifact_id,
  };
}

function isRelativeMunkiPath(value: string) {
  const trimmed = value.trim();
  if (trimmed === "" || trimmed.startsWith("/") || trimmed.includes("\\")) return false;
  return trimmed.split("/").every((segment) => segment !== "" && segment !== "." && segment !== "..");
}

function parseIDList(value: string): { ok: true; ids: number[] } | { ok: false; ids: number[]; message: string } {
  const trimmed = value.trim();
  if (trimmed === "") return { ok: true, ids: [] };
  const ids: number[] = [];
  for (const part of trimmed.split(",")) {
    const id = Number(part.trim());
    if (!Number.isInteger(id) || id <= 0) {
      return { ok: false, ids: [], message: "Use comma-separated positive IDs." };
    }
    ids.push(id);
  }
  return { ok: true, ids };
}
