import { revalidateLogic, useForm } from "@tanstack/react-form";

import { type PackageFormState, validatePackageForm } from "./form-state";

export function usePackageEditorForm(
  initial: PackageFormState,
  onSubmit: (value: PackageFormState) => Promise<void>,
  options: { hasInstallerFile: boolean },
) {
  return useForm({
    defaultValues: initial,
    validationLogic: revalidateLogic(),
    validators: {
      onDynamic: ({ value }) =>
        validatePackageForm({ value, hasInstallerFile: options.hasInstallerFile }),
    },
    onSubmit: ({ value }) => onSubmit(value),
  });
}

export type PackageEditorForm = ReturnType<typeof usePackageEditorForm>;
