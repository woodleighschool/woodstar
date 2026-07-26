import { revalidateLogic, useForm } from "@tanstack/react-form";

import { FormActions } from "@components/form-actions";
import { focusFirstInvalidField } from "@components/form-tabs";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@components/ui/dialog";
import { Input } from "@components/ui/input";
import { Switch } from "@components/ui/switch";
import { ValidatedFormField } from "@components/validated-form-field";

import {
  type ClientResourceLink,
  clientResourceLinkSchema,
  emptyClientResourceLink,
} from "./form-schema";

export function LinkDialog({
  open,
  onOpenChange,
  link,
  onSave,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  link: ClientResourceLink | null;
  onSave: (link: ClientResourceLink) => void;
}) {
  if (!open) return null;

  return (
    <LinkDialogForm
      key={link?.id ?? "new"}
      link={link}
      onClose={() => onOpenChange(false)}
      onSave={(value) => {
        onSave(value);
        onOpenChange(false);
      }}
    />
  );
}

function LinkDialogForm({
  link,
  onClose,
  onSave,
}: {
  link: ClientResourceLink | null;
  onClose: () => void;
  onSave: (link: ClientResourceLink) => void;
}) {
  const form = useForm({
    defaultValues: link ?? emptyClientResourceLink(),
    validationLogic: revalidateLogic({ mode: "submit", modeAfterSubmission: "change" }),
    validators: { onDynamic: clientResourceLinkSchema },
    onSubmit: ({ value }) => onSave(clientResourceLinkSchema.parse(value)),
  });
  return (
    <Dialog
      open
      onOpenChange={(nextOpen) => {
        if (!nextOpen) onClose();
      }}
    >
      <DialogContent className="max-w-xl">
        <form
          noValidate
          className="contents"
          onSubmit={(event) => {
            event.preventDefault();
            event.stopPropagation();
            void form.handleSubmit().then(() => {
              if (!form.state.isValid) focusFirstInvalidField();
              return undefined;
            });
          }}
        >
          <DialogHeader>
            <DialogTitle>{link ? "Edit link" : "Add link"}</DialogTitle>
            <DialogDescription>Use an HTTP URL, email address, or Munki route.</DialogDescription>
          </DialogHeader>

          <div className="flex flex-col gap-4">
            <form.Field name="label">
              {(field) => (
                <ValidatedFormField
                  field={field}
                  label="Label"
                  htmlFor="client-resources-link-label"
                  required
                >
                  {(control) => (
                    <Input
                      {...control}
                      value={field.state.value}
                      onBlur={field.handleBlur}
                      onChange={(event) => field.handleChange(event.target.value)}
                    />
                  )}
                </ValidatedFormField>
              )}
            </form.Field>

            <form.Field name="target">
              {(field) => (
                <ValidatedFormField
                  field={field}
                  label="Target"
                  htmlFor="client-resources-link-target"
                  required
                >
                  {(control) => (
                    <Input
                      {...control}
                      value={field.state.value}
                      onBlur={field.handleBlur}
                      onChange={(event) => field.handleChange(event.target.value)}
                    />
                  )}
                </ValidatedFormField>
              )}
            </form.Field>

            <form.Field name="openInBrowser">
              {(field) => (
                <ValidatedFormField
                  field={field}
                  label="Open in browser"
                  htmlFor="client-resources-link-browser"
                >
                  {(control) => (
                    <Switch
                      {...control}
                      checked={field.state.value}
                      onBlur={field.handleBlur}
                      onCheckedChange={field.handleChange}
                    />
                  )}
                </ValidatedFormField>
              )}
            </form.Field>
          </div>

          <FormActions
            form={form}
            submitLabel={link ? "Save" : "Add"}
            onCancel={onClose}
            className="justify-end"
          />
        </form>
      </DialogContent>
    </Dialog>
  );
}
