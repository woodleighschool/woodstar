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
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { cn } from "@/lib/utils";

import type { MunkiSoftwareTargetFormState } from "@/lib/munki-software-target-form";
import {
  MUNKI_PACKAGE_STRATEGY_OPTIONS,
  MUNKI_SOFTWARE_STATE_OPTIONS,
  munkiPackageStrategyDescription,
  munkiSoftwareStateDescription,
  type MunkiPackageStrategy,
  type MunkiSoftwareState,
} from "@/lib/munki-software";

export interface MunkiSoftwareTargetPackageOption {
  id: number;
  version: string;
  software_name: string;
}

export function MunkiSoftwareTargetFormFields({
  form,
  packages,
  showErrors,
  errors,
  loadingPackages,
  className,
  onChange,
}: {
  form: MunkiSoftwareTargetFormState;
  packages: MunkiSoftwareTargetPackageOption[];
  showErrors: boolean;
  errors: Partial<Record<string, string>>;
  loadingPackages: boolean;
  className?: string;
  onChange: (next: MunkiSoftwareTargetFormState) => void;
}) {
  return (
    <FieldGroup className={cn("max-w-3xl", className)}>
      <Field>
        <FieldLabel htmlFor="munki-target-package-strategy" required>
          Package Selection
        </FieldLabel>
        <Select
          value={form.strategy}
          onValueChange={(strategy) =>
            onChange({
              ...form,
              strategy: strategy as MunkiPackageStrategy,
              package_id: strategy === "latest" ? "" : form.package_id,
            })
          }
        >
          <SelectTrigger id="munki-target-package-strategy" className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              {MUNKI_PACKAGE_STRATEGY_OPTIONS.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
        <FieldDescription>{munkiPackageStrategyDescription(form.strategy)}</FieldDescription>
      </Field>

      {form.strategy === "specific" ? (
        <Field data-invalid={showErrors && errors.package_id ? true : undefined}>
          <FieldLabel htmlFor="munki-target-package" required>
            Pinned Package
          </FieldLabel>
          <Select value={form.package_id} onValueChange={(package_id) => onChange({ ...form, package_id })}>
            <SelectTrigger id="munki-target-package" className="w-full">
              <SelectValue placeholder={loadingPackages ? "Loading..." : "Select Package"} />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {packages.map((pkg) => (
                  <SelectItem key={pkg.id} value={String(pkg.id)}>
                    {pkg.version} · {pkg.software_name}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
          <FieldDescription>Rendered as that package ID in the manifest.</FieldDescription>
          {showErrors && errors.package_id ? <FieldError>{errors.package_id}</FieldError> : null}
        </Field>
      ) : null}

      <Field>
        <FieldLabel htmlFor="munki-target-state" required>
          Managed Section
        </FieldLabel>
        <Select
          value={form.state}
          onValueChange={(state) =>
            onChange({
              ...form,
              state: state as MunkiSoftwareState,
              featured: state === "optional_install" ? form.featured : false,
            })
          }
        >
          <SelectTrigger id="munki-target-state" className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              {MUNKI_SOFTWARE_STATE_OPTIONS.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
        <FieldDescription>{munkiSoftwareStateDescription(form.state)}</FieldDescription>
      </Field>

      <FieldSet>
        <FieldLegend>Managed Software Centre</FieldLegend>
        <FieldDescription>Featured items are available only for Optional Installs.</FieldDescription>
        <Field orientation="horizontal" className={form.state !== "optional_install" ? "opacity-60" : undefined}>
          <Checkbox
            id="munki-target-featured"
            checked={form.featured}
            disabled={form.state !== "optional_install"}
            onCheckedChange={(checked) =>
              onChange({
                ...form,
                featured: checked === true,
              })
            }
          />
          <FieldContent>
            <FieldLabel htmlFor="munki-target-featured">Featured Items</FieldLabel>
            <FieldDescription>
              Also adds this item to Munki's featured list. Featured items must also be optional installs.
            </FieldDescription>
          </FieldContent>
        </Field>
        {showErrors && errors.featured ? <FieldError>{errors.featured}</FieldError> : null}
      </FieldSet>
    </FieldGroup>
  );
}
