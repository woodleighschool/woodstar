import { revalidateLogic, useForm } from "@tanstack/react-form";
import { useSelector } from "@tanstack/react-store";
import { Brush, Trash2, Upload } from "lucide-react";
import { useState } from "react";

import { ConfirmDialog } from "@/components/confirm-dialog";
import { FormActions } from "@/components/form-actions";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { PendingButton } from "@/components/pending-button";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { usePageFormExitGuard } from "@/hooks/use-page-form-exit-guard";
import type { MunkiObjectView } from "@/lib/api";
import type { UploadProgress } from "@/lib/upload";

import { ClientResourcesArchiveField } from "./archive-field";
import { ClientResourcesEditor } from "./editor";
import {
  type ClientResourcesFormInput,
  type ClientResourcesFormOutput,
  clientResourcesFormSchema,
} from "./form-schema";
import { useClientResourceAssetLifecycle } from "./use-client-resource-asset";

function useClientResourcesForm(
  initial: ClientResourcesFormInput,
  onSubmit: (value: ClientResourcesFormOutput) => Promise<boolean>,
) {
  return useForm({
    defaultValues: initial,
    validationLogic: revalidateLogic({ mode: "submit", modeAfterSubmission: "change" }),
    validators: { onDynamic: clientResourcesFormSchema },
    onSubmit: async ({ value, formApi }) => {
      const parsed = clientResourcesFormSchema.parse(value);
      if (!(await onSubmit(parsed))) return;
      formApi.reset(value);
    },
  });
}

export type ClientResourcesForm = ReturnType<typeof useClientResourcesForm>;

export function MunkiClientResourcesForm({
  initial,
  deployed,
  archiveMetadata,
  archiveUploading,
  archiveProgress,
  archiveError,
  bannerUploading,
  undeploying,
  onSubmit,
  onCancel,
  onUndeploy,
}: {
  initial: ClientResourcesFormInput;
  deployed: boolean;
  archiveMetadata?: MunkiObjectView;
  archiveUploading: boolean;
  archiveProgress: UploadProgress | null;
  archiveError: Error | null;
  bannerUploading: boolean;
  undeploying: boolean;
  onSubmit: (value: ClientResourcesFormOutput) => Promise<boolean>;
  onCancel: () => void;
  onUndeploy: () => Promise<void>;
}) {
  const [confirmUndeploy, setConfirmUndeploy] = useState(false);
  const form = useClientResourcesForm(initial, onSubmit);
  const formSubmitting = useSelector(form.store, (state) => state.isSubmitting);
  const bannerAsset = useSelector(form.store, (state) => state.values.banner.asset);
  useClientResourceAssetLifecycle(bannerAsset);

  function discard() {
    onCancel();
    form.reset(initial);
  }

  const exitGuard = usePageFormExitGuard({ form, onDiscard: discard });

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
      <form.Subscribe selector={(state) => [state.values.custom, state.isSubmitting] as const}>
        {([custom, submitting]) => (
          <PageHeader
            title="Client Resources"
            description="Configure Managed Software Center branding."
            context={<DeploymentBadge deployed={deployed} />}
            actions={
              <>
                {deployed ? (
                  <PendingButton
                    type="button"
                    variant="destructive"
                    size="sm"
                    isPending={undeploying}
                    disabled={submitting}
                    icon={<Trash2 data-icon="inline-start" />}
                    onClick={() => setConfirmUndeploy(true)}
                  >
                    Undeploy
                  </PendingButton>
                ) : null}
                <Button
                  type="button"
                  size="sm"
                  disabled={submitting || undeploying}
                  onClick={() => form.setFieldValue("custom", !custom)}
                >
                  {custom ? (
                    <Brush data-icon="inline-start" />
                  ) : (
                    <Upload data-icon="inline-start" />
                  )}
                  {custom ? "Use Builder" : "Upload Custom ZIP"}
                </Button>
              </>
            }
          />
        )}
      </form.Subscribe>

      <fieldset disabled={undeploying || formSubmitting} className="contents">
        <form.Subscribe selector={(state) => state.values}>
          {(values) =>
            values.custom ? (
              <ClientResourcesArchiveField
                form={form}
                metadata={archiveMetadata}
                uploading={archiveUploading}
                progress={archiveProgress}
                error={archiveError}
              />
            ) : (
              <ClientResourcesEditor form={form} draft={values} bannerUploading={bannerUploading} />
            )
          }
        </form.Subscribe>

        <FormActions form={form} submitLabel="Save" onCancel={exitGuard.requestDiscard} />
      </fieldset>

      {exitGuard.dialog}

      <ConfirmDialog
        open={confirmUndeploy}
        onOpenChange={setConfirmUndeploy}
        title="Undeploy client resources?"
        description={
          form.state.isDefaultValue
            ? "Woodstar will stop serving this archive. Munki clients will use their built-in resources."
            : "Woodstar will stop serving this archive and discard your unsaved changes. Munki clients will use their built-in resources."
        }
        confirmLabel="Undeploy"
        variant="destructive"
        pending={undeploying}
        onConfirm={() => {
          void onUndeploy().then(() => setConfirmUndeploy(false));
        }}
      />
    </PageShell>
  );
}

function DeploymentBadge({ deployed }: { deployed: boolean }) {
  const badge = (
    <Badge variant={deployed ? "default" : "secondary"}>
      {deployed ? "Deployed" : "Undeployed"}
    </Badge>
  );
  return (
    <Tooltip>
      <TooltipTrigger render={badge} />
      <TooltipContent>
        {deployed
          ? "Woodstar is serving this archive to Munki clients."
          : "Woodstar is not serving client resources. Munki clients use their built-in resources."}
      </TooltipContent>
    </Tooltip>
  );
}
