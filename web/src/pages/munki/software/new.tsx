import { useForm } from "@tanstack/react-form";
import { Link, useNavigate } from "@tanstack/react-router";
import { Info, Loader2 } from "lucide-react";
import { useMemo, useState } from "react";

import { MutableResourceTabs } from "@/components/layout/mutable-resource-tabs";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { EditableMunkiIcon } from "@/components/munki/editable-munki-icon";
import { FreeTextCombobox } from "@/components/munki/free-text-combobox";
import { MutationError } from "@/components/mutation-error";
import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useUploadMunkiArtifact } from "@/hooks/munki/artifacts";
import { useCreateMunkiSoftware, useMunkiSoftware, type MunkiSoftwareMutation } from "@/hooks/munki/software";
import { uniqueOptions } from "@/lib/form-validation";
import { emptyMunkiSoftwareForm, munkiSoftwareSchema } from "@/lib/munki-software-form";
import { MAX_PAGE_SIZE } from "@/lib/pagination";

export function MunkiSoftwareNewPage() {
  const navigate = useNavigate();
  const create = useCreateMunkiSoftware();
  const iconUpload = useUploadMunkiArtifact("icon");
  // Category/developer suggestions are loose helper text; MAX_PAGE_SIZE is enough for this non-managed vocabulary.
  const titles = useMunkiSoftware({ page_size: MAX_PAGE_SIZE, sort: "name.asc" });
  const categoryOptions = useMemo(
    () => uniqueOptions((titles.data?.items ?? []).map((item) => item.category)),
    [titles.data?.items],
  );
  const developerOptions = useMemo(
    () => uniqueOptions((titles.data?.items ?? []).map((item) => item.developer)),
    [titles.data?.items],
  );
  const [iconFile, setIconFile] = useState<File | null>(null);
  const form = useForm({
    defaultValues: emptyMunkiSoftwareForm(),
    validators: {
      onSubmit: munkiSoftwareSchema,
    },
    onSubmit: async ({ value }) => {
      const data = munkiSoftwareSchema.parse(value);
      const iconArtifact = iconFile ? await iconUpload.upload(iconFile) : null;
      const body: MunkiSoftwareMutation = {
        ...data,
        icon_artifact_id: iconArtifact?.id,
        targets: { include: [], exclude: [] },
      };
      const title = await create.mutateAsync(body);
      void navigate({ to: "/munki/software/$softwareId", params: { softwareId: String(title.id) } });
    },
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
        <PageHeader title="New Software" />
        <MutationError title="Failed to Create Software" message={create.error?.message ?? iconUpload.error?.message} />
        <MutableResourceTabs
          tabs={[
            {
              value: "options",
              label: "Options",
              content: (
                <FieldGroup className="max-w-3xl">
                  <div className="flex items-start gap-4">
                    <EditableMunkiIcon
                      title="software icon"
                      file={iconFile}
                      clearable={!!iconFile}
                      onFileChange={setIconFile}
                      onClear={() => setIconFile(null)}
                    />
                    <div className="min-w-0 flex-1">
                      <form.Field
                        name="name"
                        children={(field) => {
                          const invalid = field.state.meta.isTouched && !field.state.meta.isValid;
                          return (
                            <Field data-invalid={invalid}>
                              <FieldLabel htmlFor="munki-software-name" required>
                                Name
                              </FieldLabel>
                              <Input
                                id="munki-software-name"
                                name={field.name}
                                value={field.state.value}
                                aria-invalid={invalid}
                                onBlur={field.handleBlur}
                                onChange={(event) => field.handleChange(event.target.value)}
                              />
                              {invalid ? <FieldError errors={field.state.meta.errors} /> : null}
                            </Field>
                          );
                        }}
                      />
                    </div>
                  </div>
                  <form.Field
                    name="description"
                    children={(field) => {
                      const invalid = field.state.meta.isTouched && !field.state.meta.isValid;
                      return (
                        <Field data-invalid={invalid}>
                          <FieldLabel htmlFor="munki-software-description">
                            Description
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <button
                                  type="button"
                                  className="text-muted-foreground hover:text-foreground inline-flex"
                                >
                                  <Info className="size-3.5" />
                                </button>
                              </TooltipTrigger>
                              <TooltipContent>Description is shown in Managed Software Center.</TooltipContent>
                            </Tooltip>
                          </FieldLabel>
                          <Textarea
                            id="munki-software-description"
                            name={field.name}
                            value={field.state.value}
                            aria-invalid={invalid}
                            onBlur={field.handleBlur}
                            onChange={(event) => field.handleChange(event.target.value)}
                          />
                          {invalid ? <FieldError errors={field.state.meta.errors} /> : null}
                        </Field>
                      );
                    }}
                  />
                  <div className="grid gap-4 md:grid-cols-2">
                    <form.Field
                      name="category"
                      children={(field) => {
                        const invalid = field.state.meta.isTouched && !field.state.meta.isValid;
                        return (
                          <Field data-invalid={invalid}>
                            <FieldLabel htmlFor="munki-software-category">Category</FieldLabel>
                            <FreeTextCombobox
                              id="munki-software-category"
                              name={field.name}
                              value={field.state.value}
                              options={categoryOptions}
                              invalid={invalid}
                              onBlur={field.handleBlur}
                              onChange={field.handleChange}
                            />
                            {invalid ? <FieldError errors={field.state.meta.errors} /> : null}
                          </Field>
                        );
                      }}
                    />
                    <form.Field
                      name="developer"
                      children={(field) => {
                        const invalid = field.state.meta.isTouched && !field.state.meta.isValid;
                        return (
                          <Field data-invalid={invalid}>
                            <FieldLabel htmlFor="munki-software-developer">Developer</FieldLabel>
                            <FreeTextCombobox
                              id="munki-software-developer"
                              name={field.name}
                              value={field.state.value}
                              options={developerOptions}
                              invalid={invalid}
                              onBlur={field.handleBlur}
                              onChange={field.handleChange}
                            />
                            {invalid ? <FieldError errors={field.state.meta.errors} /> : null}
                          </Field>
                        );
                      }}
                    />
                  </div>
                </FieldGroup>
              ),
            },
          ]}
        />
        <div className="flex items-center gap-2 border-t pt-4">
          <Button type="submit" size="sm" disabled={create.isPending || iconUpload.isUploading}>
            {create.isPending || iconUpload.isUploading ? (
              <Loader2 data-icon="inline-start" className="animate-spin" />
            ) : null}
            Save
          </Button>
          <Button asChild type="button" variant="outline" size="sm">
            <Link to="/munki/software">Cancel</Link>
          </Button>
        </div>
      </form>
    </PageShell>
  );
}
