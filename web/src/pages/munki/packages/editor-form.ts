import { revalidateLogic, useForm } from "@tanstack/react-form";

import { packageFormSchema, type PackageFormState } from "./form-state";

export function usePackageEditorForm(
  initial: PackageFormState,
  onSubmit: (value: PackageFormState) => Promise<boolean>,
  onSuccess: () => void,
) {
  return useForm({
    defaultValues: initial,
    validationLogic: revalidateLogic({ mode: "submit", modeAfterSubmission: "change" }),
    validators: {
      onDynamic: packageFormSchema(),
    },
    onSubmit: async ({ value, formApi }) => {
      if (!(await onSubmit(value))) return;
      formApi.reset(value);
      onSuccess();
    },
  });
}

export type PackageEditorForm = ReturnType<typeof usePackageEditorForm>;
