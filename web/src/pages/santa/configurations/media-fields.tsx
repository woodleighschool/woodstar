import { Checkbox } from "@/components/ui/checkbox";
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
  FieldTitle,
} from "@/components/ui/field";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { firstErrorMessage } from "@/lib/form-validation";
import {
  isSantaMediaAction,
  MEDIA_ACTIONS,
  MEDIA_ACTION_OPTIONS,
  REMOUNT_FLAG_OPTIONS,
  type SantaMediaAction,
  type SantaRemountFlag,
} from "@/lib/santa-configurations";

import type { ConfigurationEditorForm } from "./fields";

export function ConfigurationMediaFields({ form }: { form: ConfigurationEditorForm }) {
  return (
    <FieldSet>
      <FieldLegend>Removable Media</FieldLegend>
      <FieldDescription>
        Controls how Santa handles newly mounted USB mass-storage volumes.
      </FieldDescription>
      <FieldGroup>
        <form.Field name="removable_media_action">
          {(actionField) => (
            <form.Field name="removable_media_remount_flags">
              {(flagsField) => (
                <MediaActionField
                  id="santa-removable-media"
                  name={actionField.name}
                  label="Unencrypted"
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
            </form.Field>
          )}
        </form.Field>
        <form.Field name="encrypted_removable_media_action">
          {(actionField) => (
            <form.Field name="encrypted_removable_media_remount_flags">
              {(flagsField) => (
                <MediaActionField
                  id="santa-encrypted-removable-media"
                  name={actionField.name}
                  label="Encrypted"
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
            </form.Field>
          )}
        </form.Field>
      </FieldGroup>
    </FieldSet>
  );
}

function MediaActionField({
  id,
  name,
  label,
  action,
  flags,
  flagsError,
  onActionChange,
  onFlagsChange,
}: {
  id: string;
  name: string;
  label: string;
  action: SantaMediaAction;
  flags: SantaRemountFlag[];
  flagsError?: string;
  onActionChange: (value: SantaMediaAction) => void;
  onFlagsChange: (value: SantaRemountFlag[]) => void;
}) {
  return (
    <FieldSet data-invalid={flagsError ? true : undefined}>
      <FieldLegend variant="label">{label}</FieldLegend>
      <RadioGroup
        name={name}
        value={action}
        className="grid grid-cols-2 gap-2 sm:grid-cols-4"
        onValueChange={(value) => {
          if (isSantaMediaAction(value)) onActionChange(value);
        }}
      >
        {MEDIA_ACTION_OPTIONS.map((option) => (
          <FieldLabel key={option.value} htmlFor={`${id}-${option.value}`}>
            <Field orientation="horizontal">
              <RadioGroupItem id={`${id}-${option.value}`} value={option.value} />
              <FieldContent>
                <FieldTitle>{option.label}</FieldTitle>
                <FieldDescription>{MEDIA_ACTIONS[option.value].description}</FieldDescription>
              </FieldContent>
            </Field>
          </FieldLabel>
        ))}
      </RadioGroup>
      {action === "remount" ? (
        <FieldSet aria-invalid={flagsError ? true : undefined}>
          <FieldLegend variant="label">
            Mount Flags <span className="text-destructive">*</span>
          </FieldLegend>
          <FieldDescription>
            macOS mount options applied when Santa remounts the volume.
          </FieldDescription>
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
    </FieldSet>
  );
}

function toggleRemountFlag(flags: SantaRemountFlag[], flag: SantaRemountFlag, checked: boolean) {
  if (checked) return flags.includes(flag) ? flags : [...flags, flag];
  return flags.filter((value) => value !== flag);
}
