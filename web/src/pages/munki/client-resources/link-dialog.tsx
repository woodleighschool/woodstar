import { useState } from "react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Field, FieldError, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";

import {
  type ClientResourceLink,
  clientResourceLinkErrors,
  emptyClientResourceLink,
} from "./client-resources";

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
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl">
        {open ? (
          <LinkDialogContent
            key={link?.id ?? "new"}
            link={link}
            onCancel={() => onOpenChange(false)}
            onSave={(value) => {
              onSave(value);
              onOpenChange(false);
            }}
          />
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function LinkDialogContent({
  link,
  onCancel,
  onSave,
}: {
  link: ClientResourceLink | null;
  onCancel: () => void;
  onSave: (link: ClientResourceLink) => void;
}) {
  const [value, setValue] = useState<ClientResourceLink>(() => link ?? emptyClientResourceLink());
  const [showErrors, setShowErrors] = useState(false);
  const errors = clientResourceLinkErrors(value);

  function save() {
    if (Object.keys(errors).length > 0) {
      setShowErrors(true);
      return;
    }
    onSave(value);
  }

  return (
    <>
      <DialogHeader>
        <DialogTitle>{link ? "Edit link" : "Add link"}</DialogTitle>
        <DialogDescription>Use an HTTP URL, email address, or Munki route.</DialogDescription>
      </DialogHeader>

      <div className="flex flex-col gap-4">
        <Field data-invalid={showErrors && errors.label ? true : undefined}>
          <FieldLabel htmlFor="client-resources-link-label" required>
            Label
          </FieldLabel>
          <Input
            id="client-resources-link-label"
            aria-invalid={showErrors && errors.label ? true : undefined}
            maxLength={80}
            value={value.label}
            onChange={(event) => setValue({ ...value, label: event.target.value })}
          />
          {showErrors && errors.label ? <FieldError>{errors.label}</FieldError> : null}
        </Field>

        <Field data-invalid={showErrors && errors.target ? true : undefined}>
          <FieldLabel htmlFor="client-resources-link-target" required>
            Target
          </FieldLabel>
          <Input
            id="client-resources-link-target"
            aria-invalid={showErrors && errors.target ? true : undefined}
            maxLength={2048}
            value={value.target}
            onChange={(event) => setValue({ ...value, target: event.target.value })}
          />
          {showErrors && errors.target ? <FieldError>{errors.target}</FieldError> : null}
        </Field>

        <Field
          orientation="horizontal"
          data-invalid={showErrors && errors.openInBrowser ? true : undefined}
        >
          <FieldLabel htmlFor="client-resources-link-browser">Open in browser</FieldLabel>
          <Switch
            id="client-resources-link-browser"
            aria-invalid={showErrors && errors.openInBrowser ? true : undefined}
            checked={value.openInBrowser}
            onCheckedChange={(openInBrowser) => setValue({ ...value, openInBrowser })}
          />
          {showErrors && errors.openInBrowser ? (
            <FieldError>{errors.openInBrowser}</FieldError>
          ) : null}
        </Field>
      </div>

      <DialogFooter>
        <Button type="button" variant="outline" size="sm" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="button" size="sm" onClick={save}>
          {link ? "Save" : "Add"}
        </Button>
      </DialogFooter>
    </>
  );
}
