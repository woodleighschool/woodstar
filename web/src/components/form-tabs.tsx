import type { AnyFormApi } from "@tanstack/react-form";
import { useSelector } from "@tanstack/react-store";
import type { ReactNode } from "react";

import { TabsTrigger } from "@/components/ui/tabs";

export interface FormTabDefinition {
  value: string;
  fields: readonly string[];
}

interface FieldMetaWithErrors {
  errors: readonly unknown[];
}

function fieldMatches(fieldName: string, configuredName: string) {
  return (
    fieldName === configuredName ||
    fieldName.startsWith(`${configuredName}.`) ||
    fieldName.startsWith(`${configuredName}[`)
  );
}

export function formTabHasErrors(
  fieldMeta: Record<string, FieldMetaWithErrors | undefined>,
  tab: FormTabDefinition,
) {
  return Object.entries(fieldMeta).some(
    ([fieldName, meta]) =>
      meta?.errors.length &&
      tab.fields.some((configuredName) => fieldMatches(fieldName, configuredName)),
  );
}

export function firstInvalidFormTab(form: AnyFormApi, tabs: readonly FormTabDefinition[]) {
  if (form.state.submissionAttempts === 0) return undefined;
  return tabs.find((tab) => formTabHasErrors(form.state.fieldMeta, tab))?.value;
}

export function focusFirstInvalidField() {
  requestAnimationFrame(() => {
    requestAnimationFrame(() => {
      const invalid = document.querySelector<HTMLElement>(
        '[aria-invalid="true"]:not([aria-hidden="true"]):not([data-slot="tabs-trigger"])',
      );
      if (!invalid) return;
      const focusable = invalid.matches(
        "input, button, select, textarea, [contenteditable=true], [tabindex]",
      )
        ? invalid
        : invalid.querySelector<HTMLElement>(
            "input, button, select, textarea, [contenteditable=true], [tabindex]",
          );
      focusable?.focus();
    });
  });
}

export function revealFirstInvalidFormTab(
  form: AnyFormApi,
  tabs: readonly FormTabDefinition[],
  setActiveTab: (value: string) => void,
) {
  const tab = firstInvalidFormTab(form, tabs);
  if (tab) setActiveTab(tab);
  focusFirstInvalidField();
}

export function FormTabTrigger({
  form,
  tab,
  children,
}: {
  form: AnyFormApi;
  tab: FormTabDefinition;
  children: ReactNode;
}) {
  const invalid = useSelector(
    form.store,
    (state) => state.submissionAttempts > 0 && formTabHasErrors(state.fieldMeta, tab),
  );

  return (
    <TabsTrigger
      value={tab.value}
      aria-invalid={invalid || undefined}
      data-invalid={invalid || undefined}
    >
      {children}
      {invalid ? (
        <>
          <span aria-hidden className="size-1.5 rounded-full bg-destructive" />
          <span className="sr-only">Contains errors</span>
        </>
      ) : null}
    </TabsTrigger>
  );
}
