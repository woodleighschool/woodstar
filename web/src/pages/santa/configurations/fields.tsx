import { revalidateLogic, useForm } from "@tanstack/react-form";
import { useState } from "react";

import { FormActions } from "@/components/form-actions";
import { FormField } from "@/components/form-field";
import {
  type FormTabDefinition,
  FormTabTrigger,
  revealFirstInvalidFormTab,
} from "@/components/form-tabs";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { ScrollableTabs, ScrollableTabsList } from "@/components/layout/scrollable-tabs";
import { LabelTargetSetEditor } from "@/components/targeting/label-target-set-editor";
import { TabsContent } from "@/components/ui/tabs";
import { usePageFormExitGuard } from "@/hooks/use-page-form-exit-guard";
import type { SantaConfigurationMutation } from "@/lib/api";

import { configurationBody } from "./form-adapter";
import { type ConfigurationFormInput, configurationFormSchema } from "./form-schema";
import { ConfigurationOptionsFields } from "./options-fields";

export { emptyConfigurationForm, formFromConfiguration } from "./form-adapter";

const configurationFormTabs = [
  {
    value: "options",
    fields: [
      "name",
      "description",
      "client_mode",
      "enable_bundles",
      "enable_transitive_rules",
      "enable_all_event_upload",
      "disable_unknown_event_upload",
      "override_file_access_action",
      "full_sync_interval_seconds",
      "batch_size",
      "allowed_path_regex",
      "blocked_path_regex",
      "event_detail_url",
      "event_detail_text",
      "removable_media_action",
      "removable_media_remount_flags",
      "encrypted_removable_media_action",
      "encrypted_removable_media_remount_flags",
    ],
  },
  { value: "targets", fields: ["targets"] },
] as const satisfies readonly FormTabDefinition[];

const noOp = () => undefined;

function useConfigurationEditorForm(
  initial: ConfigurationFormInput,
  onSubmit: (body: SantaConfigurationMutation) => Promise<number | undefined>,
  onSuccess?: (id: number | undefined) => void,
) {
  return useForm({
    defaultValues: initial,
    validationLogic: revalidateLogic({ mode: "submit", modeAfterSubmission: "change" }),
    validators: { onDynamic: configurationFormSchema },
    onSubmit: async ({ value, formApi }) => {
      const id = await onSubmit(configurationBody(configurationFormSchema.parse(value)));
      formApi.reset(value);
      onSuccess?.(id);
    },
  });
}

export type ConfigurationEditorForm = ReturnType<typeof useConfigurationEditorForm>;

export function ConfigurationForm({
  initial,
  title,
  submitLabel,
  onSubmit,
  onSuccess,
  onCancel,
}: {
  initial: ConfigurationFormInput;
  title: string;
  submitLabel: string;
  onSubmit: (body: SantaConfigurationMutation) => Promise<number | undefined>;
  onSuccess?: (id: number | undefined) => void;
  onCancel?: () => void;
}) {
  const [activeTab, setActiveTab] = useState("options");
  const form = useConfigurationEditorForm(initial, onSubmit, onSuccess);
  const exitGuard = usePageFormExitGuard({ form, onDiscard: onCancel ?? noOp });

  return (
    <PageShell
      render={
        <form
          noValidate
          onSubmit={(event) => {
            event.preventDefault();
            void form.handleSubmit().then(() => {
              revealFirstInvalidFormTab(form, configurationFormTabs, setActiveTab);
              return undefined;
            });
          }}
        />
      }
    >
      <PageHeader title={title} />

      <ScrollableTabs value={activeTab} onValueChange={setActiveTab}>
        <ScrollableTabsList>
          <FormTabTrigger form={form} tab={configurationFormTabs[0]}>
            Options
          </FormTabTrigger>
          <FormTabTrigger form={form} tab={configurationFormTabs[1]}>
            Targets
          </FormTabTrigger>
        </ScrollableTabsList>

        <TabsContent value="options" keepMounted className="data-inactive:hidden">
          <ConfigurationOptionsFields form={form} />
        </TabsContent>

        <TabsContent value="targets" keepMounted className="data-inactive:hidden">
          <form.Field name="targets">
            {(field) => (
              <FormField field={field}>
                {(control) => (
                  <div {...control} tabIndex={-1}>
                    <LabelTargetSetEditor value={field.state.value} onChange={field.handleChange} />
                  </div>
                )}
              </FormField>
            )}
          </form.Field>
        </TabsContent>
      </ScrollableTabs>

      <FormActions
        form={form}
        submitLabel={submitLabel}
        onCancel={onCancel ? exitGuard.requestDiscard : undefined}
      />
      {exitGuard.dialog}
    </PageShell>
  );
}
