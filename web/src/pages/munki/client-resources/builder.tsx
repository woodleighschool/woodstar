import { RotateCcw } from "lucide-react";
import { useState } from "react";

import { ConfirmDialog } from "@/components/confirm-dialog";
import { FormActions } from "@/components/form-actions";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { QueryGate } from "@/components/query-gate";
import { Button } from "@/components/ui/button";
import {
  useDeleteMunkiClientResources,
  useMunkiClientResources,
  useSaveMunkiClientResources,
  useUploadAndSaveMunkiClientResourcesBanner,
} from "@/hooks/use-munki-client-resources";
import type { MunkiClientResources } from "@/lib/api";

import {
  clientResourcesDraft,
  clientResourcesMutation,
  useClientResourcesForm,
} from "./client-resources";
import { ClientResourcesEditor } from "./editor";
import { useClientResourceAsset } from "./use-client-resource-asset";

export function MunkiClientResourcesPage() {
  const query = useMunkiClientResources();

  if (query.isPending) return null;
  if (query.error) {
    return (
      <QueryGate
        title="Failed to load client resources"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }

  const resource = query.data ?? null;
  return (
    <ClientResourcesBuilder key={resource?.updated_at ?? "unconfigured"} resource={resource} />
  );
}

function ClientResourcesBuilder({ resource }: { resource: MunkiClientResources | null }) {
  const [confirmDefaults, setConfirmDefaults] = useState(false);
  const [bannerRequiredError, setBannerRequiredError] = useState<string | null>(null);
  const initialDraft = clientResourcesDraft(resource);
  const banner = useClientResourceAsset(
    resource
      ? {
          name: resource.banner.filename,
          url: resource.banner.content_url,
          objectID: resource.banner.id,
          file: null,
        }
      : null,
  );
  const saveResource = useSaveMunkiClientResources();
  const uploadAndSave = useUploadAndSaveMunkiClientResourcesBanner();
  const deleteResource = useDeleteMunkiClientResources();
  const form = useClientResourcesForm(initialDraft, save);

  async function save(draft: typeof initialDraft) {
    if (!banner.asset) {
      setBannerRequiredError("Choose a banner image.");
      return;
    }

    setBannerRequiredError(null);
    const body = clientResourcesMutation(draft);
    if (banner.asset.file) {
      await uploadAndSave.upload({ file: banner.asset.file, body });
      return;
    }
    if (banner.asset.objectID === null) {
      throw new Error("The selected banner has no upload or stored object.");
    }
    await saveResource.mutateAsync({ ...body, banner_object_id: banner.asset.objectID });
  }

  function cancel() {
    form.reset(initialDraft);
    banner.reset();
    setBannerRequiredError(null);
  }

  return (
    <PageShell asChild className="min-h-0 flex-1">
      <form
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit();
        }}
      >
        <PageHeader
          title="Client Resources"
          description="Compose the branding Munki displays in Managed Software Center."
          actions={
            resource ? (
              <Button type="button" variant="outline" onClick={() => setConfirmDefaults(true)}>
                <RotateCcw />
                Use Munki defaults
              </Button>
            ) : null
          }
        />

        <form.Subscribe selector={(state) => state.values}>
          {(draft) => (
            <ClientResourcesEditor
              form={form}
              draft={draft}
              banner={banner.asset}
              bannerError={banner.error ?? bannerRequiredError}
              bannerUploading={uploadAndSave.isUploading}
              onBannerReject={setBannerRequiredError}
              onBannerChange={(file) => {
                if (!banner.replace(file)) return;
                setBannerRequiredError(null);
              }}
            />
          )}
        </form.Subscribe>

        <FormActions form={form} submitLabel="Save" onCancel={cancel} />

        <ConfirmDialog
          open={confirmDefaults}
          onOpenChange={setConfirmDefaults}
          title="Use Munki defaults?"
          description="This removes the published client resources archive. Munki clients will use their built-in resources."
          confirmLabel="Use defaults"
          variant="destructive"
          pending={deleteResource.isPending}
          onConfirm={() => {
            deleteResource.mutate(undefined, {
              onSuccess: () => setConfirmDefaults(false),
            });
          }}
        />
      </form>
    </PageShell>
  );
}
