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
        <div className="contents">
          <DialogHeader>
            <DialogTitle>{link ? "Edit Link" : "Add Link"}</DialogTitle>
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
                  label="Open in Browser"
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
            onSubmit={async () => {
              await form.handleSubmit();
              if (!form.state.isValid) focusFirstInvalidField();
            }}
            onCancel={onClose}
            className="justify-end"
          />
        </div>
      </DialogContent>
    </Dialog>
  );
}
