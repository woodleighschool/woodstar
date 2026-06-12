import { encodeSort, MAX_PAGE_SIZE } from "@/hooks/use-data-table-search";
import { Link, useNavigate } from "@tanstack/react-router";
import { useMemo, useState } from "react";

import { MutableResourceTabs } from "@/components/layout/mutable-resource-tabs";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Button } from "@/components/ui/button";
import { useUploadMunkiArtifact } from "@/hooks/use-munki-artifacts";
import {
  type MunkiSoftwareMutation,
  useCreateMunkiSoftware,
  useMunkiSoftware,
} from "@/hooks/use-munki-software";
import { uniqueOptions } from "@/lib/form-validation";

import {
  emptyMunkiSoftwareForm,
  MunkiSoftwareFormActions,
  MunkiSoftwareOptionsFields,
  munkiSoftwareSchema,
  useMunkiSoftwareForm,
} from "./fields";

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
  const form = useMunkiSoftwareForm(emptyMunkiSoftwareForm(), async (value) => {
    const data = munkiSoftwareSchema.parse(value);
    const iconArtifact = iconFile ? await iconUpload.upload(iconFile) : null;
    const body: MunkiSoftwareMutation = {
      ...data,
      icon_artifact_id: iconArtifact?.id,
      targets: { include: [], exclude: [] },
    };
    const title = await create.mutateAsync(body);
    void navigate({ to: "/munki/software/$softwareId", params: { softwareId: String(title.id) } });
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
                <MunkiSoftwareOptionsFields
                  form={form}
                  categoryOptions={categoryOptions}
                  developerOptions={developerOptions}
                  icon={{
                    file: iconFile,
                    clearable: !!iconFile,
                    onFileChange: setIconFile,
                    onClear: () => setIconFile(null),
                  }}
                />
              ),
            },
          ]}
        />
        <MunkiSoftwareFormActions
          pending={create.isPending || iconUpload.isUploading}
          error={create.error?.message ?? iconUpload.error?.message}
          cancel={
            <Button asChild type="button" variant="outline" size="sm">
              <Link to="/munki/software">Cancel</Link>
            </Button>
          }
        />
      </form>
    </PageShell>
  );
}
