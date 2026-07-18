import { useState } from "react";

import { FormActions } from "@/components/form-actions";
import { FormTabTrigger, revealFirstInvalidFormTab } from "@/components/form-tabs";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { ScrollableTabs, ScrollableTabsList } from "@/components/layout/scrollable-tabs";
import { TabsContent } from "@/components/ui/tabs";
import { usePageFormExitGuard } from "@/hooks/use-page-form-exit-guard";
import type { MunkiPackage, MunkiSoftware } from "@/lib/api";

import type { PackageEditorForm } from "./editor-form";
import { PackageEditorTabContent, packageFormTabs } from "./package-form-tabs";
import type { SoftwareInfo } from "./package-reference-editors";
export type { SoftwareInfo } from "./package-reference-editors";
type PackageFormProps = {
  form: PackageEditorForm;
  title: string;
  submitLabel: string;
  softwareInfo: SoftwareInfo | null;
  softwareOptions?: MunkiSoftware[];
  softwareLoading?: boolean;
  packageOptions: MunkiPackage[];
  installerMetadata?: MunkiPackage["installer_file"];
  onCancel: () => void;
  canCancelWhileSubmitting?: boolean;
};
export function PackageForm({
  form,
  title,
  submitLabel,
  softwareInfo,
  softwareOptions,
  softwareLoading,
  packageOptions,
  installerMetadata,
  onCancel,
  canCancelWhileSubmitting,
}: PackageFormProps) {
  const [activeTab, setActiveTab] = useState("basic");
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
