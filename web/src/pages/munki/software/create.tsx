import { encodeSort } from "@/hooks/use-data-table-search";
import { useForm } from "@tanstack/react-form";
import { Link, useNavigate } from "@tanstack/react-router";
import { Info } from "lucide-react";
import { useMemo, useState } from "react";

import { FormField } from "@/components/form-field";
import { MutableResourceTabs } from "@/components/layout/mutable-resource-tabs";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { EditableMunkiIcon } from "@/components/munki/editable-munki-icon";
import { FreeTextCombobox } from "@/components/munki/free-text-combobox";
import { SubmitButton } from "@/components/submit-button";
import { Button } from "@/components/ui/button";
import { FieldError, FieldGroup } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { MAX_PAGE_SIZE } from "@/hooks/use-data-table-search";
import { useUploadMunkiArtifact } from "@/hooks/use-munki-artifacts";
import { useCreateMunkiSoftware, useMunkiSoftware, type MunkiSoftwareMutation } from "@/hooks/use-munki-software";
import { uniqueOptions } from "@/lib/form-validation";
import { emptyMunkiSoftwareForm, munkiSoftwareSchema } from "./fields";

export function MunkiSoftwareCreatePage() {
  const navigate = useNavigate();
  const create = useCreateMunkiSoftware();
  const iconUpload = useUploadMunkiArtifact("icon");
  // Category/developer suggestions are loose helper text; MAX_PAGE_SIZE is enough for this non-managed vocabulary.
  const titles = useMunkiSoftware({ per_page: MAX_PAGE_SIZE, sort: encodeSort("name") });
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
                      <form.Field name="name">
                        {(field) => (
                          <FormField field={field} label="Name" htmlFor="munki-software-name" required>
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
                    </div>
                  </div>
                  <form.Field name="description">
                    {(field) => (
                      <FormField
                        field={field}
                        htmlFor="munki-software-description"
                        label={
                          <>
                            Description
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button variant="ghost" size="icon-xs" type="button">
                                  <Info className="size-3.5" />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>Description is shown in Managed Software Center.</TooltipContent>
                            </Tooltip>
                          </>
                        }
                      >
                        {(control) => (
                          <Textarea
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
                  <div className="grid gap-4 md:grid-cols-2">
                    <form.Field name="category">
                      {(field) => (
                        <FormField field={field} label="Category" htmlFor="munki-software-category">
                          {(control) => (
                            <FreeTextCombobox
                              id={control.id}
                              name={field.name}
                              value={field.state.value}
                              options={categoryOptions}
                              invalid={control["aria-invalid"]}
                              onBlur={field.handleBlur}
                              onChange={field.handleChange}
                            />
                          )}
                        </FormField>
                      )}
                    </form.Field>
                    <form.Field name="developer">
                      {(field) => (
                        <FormField field={field} label="Developer" htmlFor="munki-software-developer">
                          {(control) => (
                            <FreeTextCombobox
                              id={control.id}
                              name={field.name}
                              value={field.state.value}
                              options={developerOptions}
                              invalid={control["aria-invalid"]}
                              onBlur={field.handleBlur}
                              onChange={field.handleChange}
                            />
                          )}
                        </FormField>
                      )}
                    </form.Field>
                  </div>
                </FieldGroup>
              ),
            },
          ]}
        />
        <div className="flex flex-col gap-2 border-t pt-4">
          {create.error || iconUpload.error ? (
            <FieldError>{create.error?.message ?? iconUpload.error?.message}</FieldError>
          ) : null}
          <div className="flex items-center gap-2">
            <SubmitButton pending={create.isPending || iconUpload.isUploading} size="sm">
              Save
            </SubmitButton>
            <Button asChild type="button" variant="outline" size="sm">
              <Link to="/munki/software">Cancel</Link>
            </Button>
          </div>
        </div>
      </form>
    </PageShell>
  );
}
