import { useRef } from "react";

import { QueryGate } from "@/components/query-gate";
import {
  useDeleteMunkiClientResources,
  useMunkiClientResources,
  useSaveMunkiClientResourcesBuilder,
  useUploadAndPublishMunkiClientResourcesArchive,
  useUploadAndSaveMunkiClientResourcesBanner,
} from "@/hooks/use-munki-client-resources";
import type { MunkiClientResources } from "@/lib/api";

import { MunkiClientResourcesForm } from "./fields";
import { clientResourcesBuilderMutation, clientResourcesFormFromResource } from "./form-adapter";
import type { ClientResourcesFormOutput } from "./form-schema";

export function MunkiClientResourcesEditPage() {
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
    <MunkiClientResourcesEditForm key={resource?.updated_at ?? "undeployed"} resource={resource} />
  );
}

function MunkiClientResourcesEditForm({ resource }: { resource: MunkiClientResources | null }) {
  const saveBuilder = useSaveMunkiClientResourcesBuilder();
  const uploadBanner = useUploadAndSaveMunkiClientResourcesBanner();
  const uploadArchive = useUploadAndPublishMunkiClientResourcesArchive();
  const undeploy = useDeleteMunkiClientResources();
  const saveBuilderAbort = useRef<AbortController | null>(null);
  const initial = clientResourcesFormFromResource(resource);

  async function save(form: ClientResourcesFormOutput) {
    if (form.custom) {
      if (form.archive_file) {
        await uploadArchive.upload({ file: form.archive_file });
      }
      return true;
    }

    const banner = form.banner.asset;
    if (!banner) throw new Error("Validated client resources are missing a banner.");
    const body = clientResourcesBuilderMutation(form);
    if (banner.file) {
      await uploadBanner.upload({ file: banner.file, body });
      return true;
    }
    if (banner.objectID === null) {
      throw new Error("The selected banner has no upload or stored object.");
    }
    const abortController = new AbortController();
    saveBuilderAbort.current = abortController;
    try {
      await saveBuilder.mutateAsync({
        body: { ...body, banner_object_id: banner.objectID },
        signal: abortController.signal,
      });
    } finally {
      if (saveBuilderAbort.current === abortController) saveBuilderAbort.current = null;
    }
    return true;
  }

  return (
    <MunkiClientResourcesForm
      initial={initial}
      deployed={resource !== null}
      archiveMetadata={resource?.custom ? resource.archive : undefined}
      archiveUploading={uploadArchive.isUploading}
      archiveProgress={uploadArchive.progress}
      archiveError={uploadArchive.error}
      bannerUploading={uploadBanner.isUploading}
      undeploying={undeploy.isPending}
      onSubmit={save}
      onCancel={() => {
        uploadArchive.cancel();
        uploadArchive.reset();
        uploadBanner.cancel();
        uploadBanner.reset();
        saveBuilderAbort.current?.abort();
      }}
      onUndeploy={async () => {
        uploadArchive.cancel();
        uploadBanner.cancel();
        saveBuilderAbort.current?.abort();
        await undeploy.mutateAsync();
      }}
    />
  );
}
