import { useNavigate } from "@tanstack/react-router";
import { useMemo, useState } from "react";

import { FormActions } from "@/components/form-actions";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { ScrollableTabs, ScrollableTabsList } from "@/components/layout/scrollable-tabs";
import { TabsContent, TabsTrigger } from "@/components/ui/tabs";
import { encodeSort } from "@/hooks/use-data-table-search";
import { useCreateMunkiSoftware, useMunkiSoftware } from "@/hooks/use-munki-software";
import { useUploadMunkiIcon } from "@/hooks/use-munki-uploads";
import { uniqueOptions } from "@/lib/form-validation";
import { MAX_PAGE_SIZE } from "@/lib/pagination";

import {
  emptyMunkiSoftwareForm,
  MunkiSoftwareOptionsFields,
  munkiSoftwareSchema,
  useMunkiSoftwareForm,
} from "./fields";

export function MunkiSoftwareCreatePage() {
  const navigate = useNavigate();
  const create = useCreateMunkiSoftware();
  const iconUpload = useUploadMunkiIcon();
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
  const [pickedIcon, setPickedIcon] = useState<{ id: number; url: string } | null>(null);
  const form = useMunkiSoftwareForm(emptyMunkiSoftwareForm(), async (value) => {
    const data = munkiSoftwareSchema.parse(value);
    const title = await create.mutateAsync({
      ...data,
      icon_object_id: pickedIcon?.id,
      targets: { include: [], exclude: [] },
    });
    if (iconFile) {
      await iconUpload.upload({ softwareId: title.id, file: iconFile });
    }
    void navigate({ to: "/munki/software/$softwareId", params: { softwareId: String(title.id) } });
  });
  const tabs = [
    {
      value: "options",
      label: "Options",
      content: (
        <MunkiSoftwareOptionsFields
          form={form}
          categoryOptions={categoryOptions}
          developerOptions={developerOptions}
          icon={{
            iconUrl: pickedIcon?.url,
            file: iconFile,
            clearable: !!iconFile || !!pickedIcon,
            onFileChange: (file) => {
              setIconFile(file);
              setPickedIcon(null);
            },
            onPickExisting: (object) => {
              setPickedIcon(object);
              setIconFile(null);
            },
            onClear: () => {
              setIconFile(null);
              setPickedIcon(null);
            },
          }}
        />
      ),
    },
  ];

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
        <ScrollableTabs defaultValue="options">
          <ScrollableTabsList>
            {tabs.map((tab) => (
              <TabsTrigger key={tab.value} value={tab.value}>
                {tab.label}
              </TabsTrigger>
            ))}
          </ScrollableTabsList>
          {tabs.map((tab) => (
            <TabsContent key={tab.value} value={tab.value}>
              {tab.content}
            </TabsContent>
          ))}
        </ScrollableTabs>
        <FormActions
          form={form}
          submitLabel="Create"
          onCancel={() => void navigate({ to: "/munki/software" })}
        />
      </form>
    </PageShell>
  );
}
