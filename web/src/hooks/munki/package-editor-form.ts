import { useForm } from "@tanstack/react-form";

import { packageIdentitySchema, validatePackageForm, type PackageFormState } from "@/lib/munki-package-form";

export function usePackageEditorForm(initial: PackageFormState, onSubmit: (value: PackageFormState) => Promise<void>) {
  return useForm({
    defaultValues: initial,
    validators: {
      onSubmit: validatePackageForm,
    },
    onSubmit: ({ value }) => {
      if (!packageIdentitySchema.safeParse(value).success) return;
      return onSubmit(value);
    },
  });
}

export type PackageEditorForm = ReturnType<typeof usePackageEditorForm>;
