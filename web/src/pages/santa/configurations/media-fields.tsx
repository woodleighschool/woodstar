import { Checkbox } from "@/components/ui/checkbox";
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from "@/components/ui/field";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { firstErrorMessage } from "@/lib/form-validation";
import {
  isSantaMediaAction,
  MEDIA_ACTION_OPTIONS,
  REMOUNT_FLAG_OPTIONS,
  type SantaMediaAction,
  type SantaRemountFlag,
} from "@/lib/santa-configurations";

import type { ConfigurationEditorForm } from "./fields";

export function ConfigurationMediaFields({ form }: { form: ConfigurationEditorForm }) {
  return (
    <>
      <form.Field
        name="removable_media_action"
        children={(actionField) => (
          <form.Field
            name="removable_media_remount_flags"
            children={(flagsField) => (
              <MediaActionField
                id="santa-removable-media"
                label="Removable Media"
                action={actionField.state.value}
                flags={flagsField.state.value}
                flagsError={firstErrorMessage(flagsField.state.meta.errors)}
                onActionChange={(action) => {
                  actionField.handleChange(action);
                  if (action !== "remount") flagsField.handleChange([]);
                }}
                onFlagsChange={flagsField.handleChange}
              />
            )}
          />
        )}
      />
      <form.Field
        name="encrypted_removable_media_action"
        children={(actionField) => (
          <form.Field
            name="encrypted_removable_media_remount_flags"
            children={(flagsField) => (
              <MediaActionField
                id="santa-encrypted-removable-media"
                label="Encrypted Removable Media"
                action={actionField.state.value}
                flags={flagsField.state.value}
                flagsError={firstErrorMessage(flagsField.state.meta.errors)}
                onActionChange={(action) => {
                  actionField.handleChange(action);
                  if (action !== "remount") flagsField.handleChange([]);
                }}
                onFlagsChange={flagsField.handleChange}
              />
            )}
          />
        )}
      />
    </>
  );
}

function MediaActionField({
  id,
  label,
  description,
  action,
  flags,
  flagsError,
  onActionChange,
  onFlagsChange,
}: {
  id: string;
  label: string;
  description?: string;
  action: SantaMediaAction;
  flags: SantaRemountFlag[];
  flagsError?: string;
  onActionChange: (value: SantaMediaAction) => void;
  onFlagsChange: (value: SantaRemountFlag[]) => void;
}) {
  return (
    <Field data-invalid={flagsError ? true : undefined}>
      <FieldLabel>{label}</FieldLabel>
      <ToggleGroup
        value={[action]}
        variant="outline"
        className="flex-wrap"
        onValueChange={(value) => {
          const nextAction = value[0];
          if (nextAction && isSantaMediaAction(nextAction)) onActionChange(nextAction);
        }}
      >
        {MEDIA_ACTION_OPTIONS.map((option) => (
          <ToggleGroupItem key={option.value} value={option.value}>
            {option.label}
          </ToggleGroupItem>
        ))}
      </ToggleGroup>
      {description ? <FieldDescription>{description}</FieldDescription> : null}
      {action === "remount" ? (
        <FieldSet aria-invalid={flagsError ? true : undefined}>
          <FieldLegend variant="label">
            Mount Flags <span className="text-destructive">*</span>
          </FieldLegend>
          <FieldGroup data-slot="checkbox-group" className="grid gap-3 sm:grid-cols-2">
            {REMOUNT_FLAG_OPTIONS.map((option) => (
              <Field key={option.value} orientation="horizontal">
                <Checkbox
                  id={`${id}-flag-${option.value}`}
                  checked={flags.includes(option.value)}
                  onCheckedChange={(checked) =>
                    onFlagsChange(toggleRemountFlag(flags, option.value, checked))
                  }
                />
                <FieldLabel htmlFor={`${id}-flag-${option.value}`}>{option.label}</FieldLabel>
              </Field>
            ))}
          </FieldGroup>
        </FieldSet>
      ) : null}
      {flagsError ? <FieldError>{flagsError}</FieldError> : null}
    </Field>
  );
}

function toggleRemountFlag(flags: SantaRemountFlag[], flag: SantaRemountFlag, checked: boolean) {
  if (checked) return flags.includes(flag) ? flags : [...flags, flag];
  return flags.filter((value) => value !== flag);
}
