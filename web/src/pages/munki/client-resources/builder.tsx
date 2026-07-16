import { RotateCcw } from "lucide-react";
import { useState } from "react";

import { ConfirmDialog } from "@/components/confirm-dialog";
import { FormActions } from "@/components/form-actions";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { QueryGate } from "@/components/query-gate";
import { Button } from "@/components/ui/button";
import { useFormExitGuard } from "@/hooks/use-form-exit-guard";
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
  const initialDraft = clientResourcesDraft(resource);
  const saveResource = useSaveMunkiClientResources();
  const uploadAndSave = useUploadAndSaveMunkiClientResourcesBanner();
  const deleteResource = useDeleteMunkiClientResources();
  const form = useClientResourcesForm(initialDraft, save);
  const exitGuard = useFormExitGuard({ form, onDiscard: cancel });
  async function save(draft: typeof initialDraft) {
    const banner = draft.banner.asset;
    if (!banner) throw new Error("Validated client resources are missing a banner.");
    const body = clientResourcesMutation(draft);
    if (banner.file) {
      await uploadAndSave.upload({ file: banner.file, body });
      return;
    }
    if (banner.objectID === null) {
      throw new Error("The selected banner has no upload or stored object.");
    }
    await saveResource.mutateAsync({ ...body, banner_object_id: banner.objectID });
  }
  function cancel() {
    uploadAndSave.cancel();
    form.reset(initialDraft);
  }
  return (
    <PageShell
      className="min-h-0 flex-1"
      render={
        <form
          noValidate
          onSubmit={(event) => {
            event.preventDefault();
            void form.handleSubmit();
          }}
        />
      }
    >
      <PageHeader
        title="Client Resources"
        description="Configure Managed Software Center branding."
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
            bannerUploading={uploadAndSave.isUploading}
          />
        )}
      </form.Subscribe>

      <FormActions
        form={form}
        submitLabel="Save"
        onCancel={exitGuard.requestDiscard}
        canCancelWhileSubmitting
      />

      {exitGuard.dialog}

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
    </PageShell>
  );
}
