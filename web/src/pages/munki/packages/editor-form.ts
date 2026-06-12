import { useForm } from "@tanstack/react-form";

import { type PackageFormState, validatePackageForm } from "./form-state";

export function usePackageEditorForm(
  initial: PackageFormState,
  onSubmit: (value: PackageFormState) => Promise<void>,
) {
  return useForm({
    defaultValues: initial,
    validators: {
      onSubmit: validatePackageForm,
    },
    onSubmit: ({ value }) => onSubmit(value),
  });
}

export type PackageEditorForm = ReturnType<typeof usePackageEditorForm>;
