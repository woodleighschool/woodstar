import { LabelPicker } from "@/components/labels/label-picker";
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
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { cn } from "@/lib/utils";

import type { MunkiAssignmentFormState } from "@/lib/munki-assignment-form";
import {
  MUNKI_ASSIGNMENT_ACTION_OPTIONS,
  MUNKI_PACKAGE_SELECTION_OPTIONS,
  munkiAssignmentActionDescription,
  munkiPackageSelectionDescription,
  type MunkiAssignmentAction,
  type MunkiPackageSelection,
} from "@/lib/munki-software-title";

export interface MunkiAssignmentPackageOption {
  id: number;
  version: string;
  display_name?: string;
  name: string;
}

export function MunkiAssignmentFormFields({
  form,
  packages,
  showErrors,
  errors,
  loadingPackages,
  unavailableLabelIDs = [],
  className,
  onChange,
}: {
  form: MunkiAssignmentFormState;
  packages: MunkiAssignmentPackageOption[];
  showErrors: boolean;
  errors: Record<string, string>;
  loadingPackages: boolean;
  unavailableLabelIDs?: readonly number[];
  className?: string;
  onChange: (next: MunkiAssignmentFormState) => void;
}) {
  return (
    <FieldGroup className={cn("max-w-3xl", className)}>
      <Field data-invalid={showErrors && errors.priority ? true : undefined}>
        <FieldLabel htmlFor="munki-assignment-priority" required>
          Priority
        </FieldLabel>
        <Input
          id="munki-assignment-priority"
          type="number"
          min={1}
          step={1}
          required
          inputMode="numeric"
          aria-invalid={showErrors && errors.priority ? true : undefined}
          value={form.priority}
          onChange={(event) => onChange({ ...form, priority: Number(event.target.value) })}
        />
        {showErrors && errors.priority ? <FieldError>{errors.priority}</FieldError> : null}
      </Field>

      <Field data-invalid={showErrors && errors.label_id ? true : undefined}>
        <FieldLabel required>Label</FieldLabel>
        <LabelPicker
          value={form.label_id === null ? [] : [form.label_id]}
          selectionMode="single"
          includeBuiltins
          required
          placeholder="Select Label"
          emptyMessage="No Labels Found."
          unavailableLabelIDs={unavailableLabelIDs}
          invalid={showErrors && errors.label_id ? true : undefined}
          onChange={(labelIDs) => onChange({ ...form, label_id: labelIDs[0] ?? null })}
        />
        {showErrors && errors.label_id ? <FieldError>{errors.label_id}</FieldError> : null}
      </Field>

      <Field>
        <FieldLabel htmlFor="munki-assignment-selection" required>
          Package Selection
        </FieldLabel>
        <Select
          value={form.package_selection}
          onValueChange={(package_selection) =>
            onChange({
              ...form,
              package_selection: package_selection as MunkiPackageSelection,
              pinned_package_id: package_selection === "latest_eligible" ? "" : form.pinned_package_id,
            })
          }
        >
          <SelectTrigger id="munki-assignment-selection" className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              {MUNKI_PACKAGE_SELECTION_OPTIONS.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
        <FieldDescription>{munkiPackageSelectionDescription(form.package_selection)}</FieldDescription>
      </Field>

      {form.package_selection === "specific_package" ? (
        <Field data-invalid={showErrors && errors.pinned_package_id ? true : undefined}>
          <FieldLabel htmlFor="munki-assignment-package" required>
            Pinned Package
          </FieldLabel>
          <Select
            value={form.pinned_package_id}
            onValueChange={(pinned_package_id) => onChange({ ...form, pinned_package_id })}
          >
            <SelectTrigger id="munki-assignment-package" className="w-full">
              <SelectValue placeholder={loadingPackages ? "Loading..." : "Select Package"} />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {packages.map((pkg) => (
                  <SelectItem key={pkg.id} value={String(pkg.id)}>
                    {pkg.version} · {pkg.display_name ?? pkg.name}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
          <FieldDescription>Rendered as Name--Version in the manifest.</FieldDescription>
          {showErrors && errors.pinned_package_id ? <FieldError>{errors.pinned_package_id}</FieldError> : null}
        </Field>
      ) : null}

      <Field>
        <FieldLabel htmlFor="munki-assignment-action" required>
          Managed Section
        </FieldLabel>
        <Select
          value={form.action}
          onValueChange={(action) =>
            onChange({
              ...form,
              action: action as MunkiAssignmentAction,
              optional_install: action === "remove" ? false : form.optional_install,
              featured_item: action === "remove" ? false : form.featured_item,
            })
          }
        >
          <SelectTrigger id="munki-assignment-action" className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              {MUNKI_ASSIGNMENT_ACTION_OPTIONS.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
        <FieldDescription>{munkiAssignmentActionDescription(form.action)}</FieldDescription>
      </Field>

      <FieldSet>
        <FieldLegend>Managed Software Centre</FieldLegend>
        <FieldDescription>These write the optional_installs and featured_items manifest sections.</FieldDescription>
        <Field orientation="horizontal" className={form.action === "remove" ? "opacity-60" : undefined}>
          <Checkbox
            id="munki-assignment-optional-install"
            checked={form.optional_install}
            disabled={form.action === "remove"}
            onCheckedChange={(checked) =>
              onChange({
                ...form,
                optional_install: checked === true,
                featured_item: checked === true ? form.featured_item : false,
              })
            }
          />
          <FieldContent>
            <FieldLabel htmlFor="munki-assignment-optional-install">Optional Installs</FieldLabel>
            <FieldDescription>Adds this item to optional_installs so it appears in MSC.</FieldDescription>
          </FieldContent>
        </Field>
        <Field orientation="horizontal" className={form.action === "remove" ? "opacity-60" : undefined}>
          <Checkbox
            id="munki-assignment-featured-item"
            checked={form.featured_item}
            disabled={form.action === "remove"}
            onCheckedChange={(checked) =>
              onChange({
                ...form,
                optional_install: checked === true ? true : form.optional_install,
                featured_item: checked === true,
              })
            }
          />
          <FieldContent>
            <FieldLabel htmlFor="munki-assignment-featured-item">Featured Items</FieldLabel>
            <FieldDescription>
              Also adds this item to featured_items. Munki expects featured items to also be optional installs.
            </FieldDescription>
          </FieldContent>
        </Field>
        {showErrors && errors.optional_install ? <FieldError>{errors.optional_install}</FieldError> : null}
        {showErrors && errors.featured_item ? <FieldError>{errors.featured_item}</FieldError> : null}
      </FieldSet>
    </FieldGroup>
  );
}
