import { useNavigate } from "@tanstack/react-router";
import { useEffect, useMemo, useState } from "react";
import { z } from "zod";

import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { EditableMunkiIcon } from "@/components/munki/editable-munki-icon";
import { FieldGroup } from "@/components/ui/field";
import { useCreateMunkiArtifact, useCreateMunkiArtifactUpload } from "@/hooks/munki/artifacts";
import {
  useCreateMunkiSoftwareTitle,
  useMunkiSoftwareTitle,
  useMunkiSoftwareTitles,
  useUpdateMunkiSoftwareTitle,
  type MunkiSoftwareTitleMutation,
} from "@/hooks/munki/software-titles";
import { fieldErrors, requiredString } from "@/lib/form-validation";
import { MAX_PAGE_SIZE } from "@/lib/pagination";

import { DatalistTextField, FormActions, MutationError, TextAreaField, TextField } from "./edit-shared";
import { runSubmit, uniqueOptions, uploadSelectedArtifact, useSoftwareIDParam } from "./edit-utils";

const softwareTitleSchema = z.object({
  name: requiredString("Name"),
  display_name: z.string().trim(),
  description: z.string().trim(),
  category: z.string().trim(),
  developer: z.string().trim(),
});

interface SoftwareTitleFormState {
  name: string;
  display_name: string;
  description: string;
  category: string;
  developer: string;
}

export function MunkiSoftwareTitleNewPage() {
  const navigate = useNavigate();
  const create = useCreateMunkiSoftwareTitle();
  const createUpload = useCreateMunkiArtifactUpload();
  const createArtifact = useCreateMunkiArtifact();
  const titles = useMunkiSoftwareTitles({ page_size: MAX_PAGE_SIZE, sort: "name.asc" });
  const categoryOptions = useMemo(
    () => uniqueOptions((titles.data?.items ?? []).map((item) => item.category)),
    [titles.data?.items],
  );
  const developerOptions = useMemo(
    () => uniqueOptions((titles.data?.items ?? []).map((item) => item.developer)),
    [titles.data?.items],
  );
  const [form, setForm] = useState<SoftwareTitleFormState>({
    name: "",
    display_name: "",
    description: "",
    category: "",
    developer: "",
  });
  const [iconFile, setIconFile] = useState<File | null>(null);
  const [showErrors, setShowErrors] = useState(false);
  const parsed = useMemo(() => softwareTitleSchema.safeParse(form), [form]);
  const errors = useMemo(() => fieldErrors(parsed), [parsed]);

  async function submit() {
    const next = softwareTitleSchema.safeParse(form);
    if (!next.success) {
      setShowErrors(true);
      return;
    }
    const iconArtifact = iconFile
      ? await uploadSelectedArtifact(iconFile, "icon", createUpload.mutateAsync, createArtifact.mutateAsync)
      : null;
    const body: MunkiSoftwareTitleMutation = {
      ...next.data,
      icon_artifact_id: iconArtifact?.id,
    };
    const title = await create.mutateAsync(body);
    void navigate({ to: "/munki/software-titles/$softwareId", params: { softwareId: String(title.id) } });
  }

  return (
    <PageShell asChild>
      <form noValidate onSubmit={(event) => runSubmit(event, submit)}>
        <PageHeader
          title="New Software"
          description="Create the software title admins target. Add package versions and assignments after the title exists."
          leading={
            <EditableMunkiIcon
              title="software icon"
              file={iconFile}
              clearable={!!iconFile}
              onFileChange={setIconFile}
              onClear={() => setIconFile(null)}
            />
          }
        />
        <MutationError
          title="Failed to Create Software"
          message={create.error?.message ?? createUpload.error?.message ?? createArtifact.error?.message}
        />
        <FieldGroup className="max-w-3xl">
          <TextField
            id="munki-software-name"
            label="Name"
            required
            description="Stable Munki item name. Use Display Name for spaces, punctuation, and nicer casing."
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
            <DatalistTextField
              id="munki-software-category"
              label="Category"
              value={form.category}
              options={categoryOptions}
              onChange={(category) => setForm({ ...form, category })}
            />
            <DatalistTextField
              id="munki-software-developer"
              label="Developer"
              value={form.developer}
              options={developerOptions}
              onChange={(developer) => setForm({ ...form, developer })}
            />
          </div>
          <FormActions
            pending={create.isPending || createUpload.isPending || createArtifact.isPending}
            cancelTo="/munki/software-titles"
          />
        </FieldGroup>
      </form>
    </PageShell>
  );
}

export function MunkiSoftwareTitleEditPage() {
  const navigate = useNavigate();
  const softwareId = useSoftwareIDParam();
  const software = useMunkiSoftwareTitle(softwareId);
  const update = useUpdateMunkiSoftwareTitle();
  const createUpload = useCreateMunkiArtifactUpload();
  const createArtifact = useCreateMunkiArtifact();
  const titles = useMunkiSoftwareTitles({ page_size: MAX_PAGE_SIZE, sort: "name.asc" });
  const categoryOptions = useMemo(
    () => uniqueOptions((titles.data?.items ?? []).map((item) => item.category)),
    [titles.data?.items],
  );
  const developerOptions = useMemo(
    () => uniqueOptions((titles.data?.items ?? []).map((item) => item.developer)),
    [titles.data?.items],
  );
  const [form, setForm] = useState<SoftwareTitleFormState>({
    name: "",
    display_name: "",
    description: "",
    category: "",
    developer: "",
  });
  const [iconFile, setIconFile] = useState<File | null>(null);
  const [iconCleared, setIconCleared] = useState(false);
  const [showErrors, setShowErrors] = useState(false);
  const parsed = useMemo(() => softwareTitleSchema.safeParse(form), [form]);
  const errors = useMemo(() => fieldErrors(parsed), [parsed]);

  useEffect(() => {
    if (!software.data) return;
    setForm({
      name: software.data.name,
      display_name: software.data.display_name,
      description: software.data.description,
      category: software.data.category,
      developer: software.data.developer,
    });
    setIconFile(null);
    setIconCleared(false);
  }, [software.data]);

  async function submit() {
    const next = softwareTitleSchema.safeParse(form);
    if (!next.success || softwareId === null) {
      setShowErrors(true);
      return;
    }
    const iconArtifact = iconFile
      ? await uploadSelectedArtifact(iconFile, "icon", createUpload.mutateAsync, createArtifact.mutateAsync)
      : null;
    const body: MunkiSoftwareTitleMutation = {
      ...next.data,
      icon_artifact_id: iconArtifact?.id ?? (iconCleared ? undefined : software.data?.icon_artifact_id),
    };
    const title = await update.mutateAsync({ id: softwareId, body });
    void navigate({ to: "/munki/software-titles/$softwareId", params: { softwareId: String(title.id) } });
  }

  return (
    <PageShell asChild>
      <form noValidate onSubmit={(event) => runSubmit(event, submit)}>
        <PageHeader
          title="Edit Software"
          description="Edit the software title admins target. Package versions inherit this icon unless they override it."
          leading={
            <EditableMunkiIcon
              title="software icon"
              iconUrl={iconCleared ? undefined : software.data?.icon_url}
              file={iconFile}
              clearable={!!iconFile || (!iconCleared && !!software.data?.icon_artifact_id)}
              onFileChange={(file) => {
                setIconFile(file);
                setIconCleared(false);
              }}
              onClear={() => {
                setIconFile(null);
                setIconCleared(!!software.data?.icon_artifact_id);
              }}
            />
          }
        />
        <MutationError
          title="Failed to Update Software"
          message={
            update.error?.message ??
            createUpload.error?.message ??
            createArtifact.error?.message ??
            software.error?.message
          }
        />
        <FieldGroup className="max-w-3xl">
          <TextField
            id="munki-software-name"
            label="Name"
            required
            description="Stable Munki item name. Use Display Name for spaces, punctuation, and nicer casing."
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
            <DatalistTextField
              id="munki-software-category"
              label="Category"
              value={form.category}
              options={categoryOptions}
              onChange={(category) => setForm({ ...form, category })}
            />
            <DatalistTextField
              id="munki-software-developer"
              label="Developer"
              value={form.developer}
              options={developerOptions}
              onChange={(developer) => setForm({ ...form, developer })}
            />
          </div>
          <FormActions
            pending={update.isPending || createUpload.isPending || createArtifact.isPending || software.isLoading}
            cancelTo="/munki/software-titles/$softwareId"
            cancelParams={{ softwareId: String(softwareId ?? "") }}
          />
        </FieldGroup>
      </form>
    </PageShell>
  );
}
