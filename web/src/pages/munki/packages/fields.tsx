import { revalidateLogic, useForm } from "@tanstack/react-form";
import { useState } from "react";

import { FormActions } from "@/components/form-actions";
import { FormTabTrigger, revealFirstInvalidFormTab } from "@/components/form-tabs";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { ScrollableTabs, ScrollableTabsList } from "@/components/layout/scrollable-tabs";
import { TabsContent } from "@/components/ui/tabs";
import { usePageFormExitGuard } from "@/hooks/use-page-form-exit-guard";
import type { MunkiPackage, MunkiSoftware } from "@/lib/api";

import {
  type PackageFormInput,
  type PackageFormMutation,
  type PackageFormOutput,
  packageFormSchema,
  packageMutationFromForm,
} from "./form-state";
import { PackageEditorTabContent, packageFormTabs } from "./package-form-tabs";
import type { SoftwareInfo } from "./package-reference-editors";
export type { SoftwareInfo } from "./package-reference-editors";
export { emptyPackageForm, packageFormFromPackage } from "./form-state";

const schema = packageFormSchema();

export type PackageFormSubmission = {
  softwareID: PackageFormOutput["software_id"];
  installerFile: PackageFormOutput["installer_file"];
  mutation: PackageFormMutation;
};

function usePackageEditorForm(
  initial: PackageFormInput,
  onSubmit: (submission: PackageFormSubmission) => Promise<boolean>,
  onSuccess: () => void,
) {
  return useForm({
    defaultValues: initial,
    validationLogic: revalidateLogic({ mode: "submit", modeAfterSubmission: "change" }),
    validators: { onDynamic: schema },
    onSubmit: async ({ value, formApi }) => {
      const parsed = schema.parse(value);
      const saved = await onSubmit({
        softwareID: parsed.software_id,
        installerFile: parsed.installer_file,
        mutation: packageMutationFromForm(parsed),
      });
      if (!saved) return;
      formApi.reset(value);
      onSuccess();
    },
  });
}

export type PackageEditorForm = ReturnType<typeof usePackageEditorForm>;

type PackageFormProps = {
  initial: PackageFormInput;
  title: string;
  submitLabel: string;
  softwareInfo: SoftwareInfo | null;
  softwareOptions?: MunkiSoftware[];
  softwareLoading?: boolean;
  packageOptions: MunkiPackage[];
  installerMetadata?: MunkiPackage["installer_file"];
  onSubmit: (submission: PackageFormSubmission) => Promise<boolean>;
  onSuccess: () => void;
  onCancel: () => void;
  canCancelWhileSubmitting?: boolean;
};
export function PackageForm({
  initial,
  title,
  submitLabel,
  softwareInfo,
  softwareOptions,
  softwareLoading,
  packageOptions,
  installerMetadata,
  onSubmit,
  onSuccess,
  onCancel,
  canCancelWhileSubmitting,
}: PackageFormProps) {
  const [activeTab, setActiveTab] = useState("basic");
  const form = usePackageEditorForm(initial, onSubmit, onSuccess);
  const exitGuard = usePageFormExitGuard({ form, onDiscard: onCancel });
  return (
    <PageShell
      render={
        <form
          noValidate
          onSubmit={(event) => {
            event.preventDefault();
            void form.handleSubmit().then(() => {
              revealFirstInvalidFormTab(form, packageFormTabs, setActiveTab);
              return undefined;
            });
          }}
        />
      }
    >
      <PageHeader title={title} />
      <PackageEditorTabs
        form={form}
        activeTab={activeTab}
        onActiveTabChange={setActiveTab}
        softwareInfo={softwareInfo}
        softwareOptions={softwareOptions}
        softwareLoading={softwareLoading}
        packageOptions={packageOptions}
        installerMetadata={installerMetadata}
      />
      <FormActions
        form={form}
        submitLabel={submitLabel}
        onCancel={exitGuard.requestDiscard}
        canCancelWhileSubmitting={canCancelWhileSubmitting}
      />
      {exitGuard.dialog}
    </PageShell>
  );
}
export function PackageEditorTabs({
  form,
  activeTab,
  onActiveTabChange,
  softwareInfo,
  softwareOptions,
  softwareLoading,
  packageOptions,
  installerMetadata,
}: {
  form: PackageEditorForm;
  activeTab: string;
  onActiveTabChange: (value: string) => void;
  softwareInfo: SoftwareInfo | null;
  softwareOptions?: MunkiSoftware[];
  softwareLoading?: boolean;
  packageOptions: MunkiPackage[];
  installerMetadata?: MunkiPackage["installer_file"];
}) {
  return (
    <ScrollableTabs value={activeTab} onValueChange={onActiveTabChange} className="max-w-3xl">
      <ScrollableTabsList>
        {packageFormTabs.map((tab) => (
          <FormTabTrigger key={tab.value} form={form} tab={tab}>
            {tab.label}
          </FormTabTrigger>
        ))}
      </ScrollableTabsList>
      {packageFormTabs.map((tab) => (
        <TabsContent key={tab.value} value={tab.value} keepMounted className="data-inactive:hidden">
          <PackageEditorTabContent
            tab={tab.value}
            form={form}
            softwareInfo={softwareInfo}
            softwareOptions={softwareOptions}
            softwareLoading={softwareLoading}
            packageOptions={packageOptions}
            installerMetadata={installerMetadata}
          />
        </TabsContent>
      ))}
    </ScrollableTabs>
  );
}
