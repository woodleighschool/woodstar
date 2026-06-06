import { useNavigate } from "@tanstack/react-router";
import { useMemo, useState } from "react";

import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { EditableMunkiIcon } from "@/components/munki/editable-munki-icon";
import { FieldGroup } from "@/components/ui/field";
import { useCreateMunkiArtifact, useCreateMunkiArtifactUpload } from "@/hooks/munki/artifacts";
import {
  useCreateMunkiSoftwareTitle,
  useMunkiSoftwareTitles,
  type MunkiSoftwareTitleMutation,
} from "@/hooks/munki/software-titles";
import { fieldErrors } from "@/lib/form-validation";
import { MAX_PAGE_SIZE } from "@/lib/pagination";

import { DatalistTextField, FormActions, MutationError, TextAreaField, TextField } from "./edit-shared";
import { runSubmit, uniqueOptions, uploadSelectedArtifact } from "./edit-utils";
import { emptySoftwareTitleForm, softwareTitleSchema, type SoftwareTitleFormState } from "./software-title-form";

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
  const [form, setForm] = useState<SoftwareTitleFormState>(() => emptySoftwareTitleForm());
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
