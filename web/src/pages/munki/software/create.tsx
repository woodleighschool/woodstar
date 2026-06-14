import { encodeSort, MAX_PAGE_SIZE } from "@/hooks/use-data-table-search";
import { useNavigate } from "@tanstack/react-router";
import { useMemo, useState } from "react";

import { FormActions } from "@/components/form-actions";
import { ScrollableTabs, ScrollableTabsList } from "@/components/layout/scrollable-tabs";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { TabsContent, TabsTrigger } from "@/components/ui/tabs";
import { useUploadMunkiIcon } from "@/hooks/use-munki-uploads";
import { useCreateMunkiSoftware, useMunkiSoftware } from "@/hooks/use-munki-software";
import { uniqueOptions } from "@/lib/form-validation";

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
  const form = useMunkiSoftwareForm(emptyMunkiSoftwareForm(), async (value) => {
    const data = munkiSoftwareSchema.parse(value);
    const title = await create.mutateAsync({ ...data, targets: { include: [], exclude: [] } });
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
            file: iconFile,
            clearable: !!iconFile,
            onFileChange: setIconFile,
            onClear: () => setIconFile(null),
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
            <TabsContent key={tab.value} value={tab.value} className="min-w-0">
              {tab.content}
            </TabsContent>
          ))}
        </ScrollableTabs>
        <FormActions
          form={form}
          requireDirty={false}
          onCancel={() => void navigate({ to: "/munki/software" })}
        />
      </form>
    </PageShell>
  );
}
