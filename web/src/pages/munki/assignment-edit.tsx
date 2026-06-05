import { useNavigate } from "@tanstack/react-router";
import { useEffect, useMemo, useState } from "react";
import { z } from "zod";

import { LabelPicker } from "@/components/labels/label-picker";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  useCreateMunkiAssignment,
  useMunkiAssignment,
  useUpdateMunkiAssignment,
  type MunkiAssignmentMutation,
} from "@/hooks/munki/assignments";
import { useMunkiSoftwareTitle } from "@/hooks/munki/software-titles";
import { fieldErrors } from "@/lib/form-validation";

import { CheckboxField, FormActions, MutationError } from "./edit-shared";
import { runSubmit, useAssignmentIDParam, useSoftwareIDParam } from "./edit-utils";

type AssignmentAction = NonNullable<MunkiAssignmentMutation["action"]>;
type AssignmentEffect = MunkiAssignmentMutation["effect"];
type PackageSelection = NonNullable<MunkiAssignmentMutation["package_selection"]>;

const actionOptions: { value: AssignmentAction; label: string; description: string }[] = [
  { value: "install", label: "Managed Installs", description: "Forces installation by writing managed_installs." },
  { value: "remove", label: "Managed Uninstalls", description: "Forces removal by writing managed_uninstalls." },
  {
    value: "update_if_present",
    label: "Managed Updates",
    description: "Updates installed items by writing managed_updates.",
  },
  {
    value: "none",
    label: "No managed section",
    description: "Only Optional Installs and Featured Items section membership is rendered.",
  },
];

const packageSelectionOptions: { value: PackageSelection; label: string; description: string }[] = [
  {
    value: "latest_eligible",
    label: "Latest compatible",
    description: "Render the Munki name and include all eligible pkginfos for the client to choose from.",
  },
  {
    value: "specific_package",
    label: "Pinned package",
    description: "Render Name--Version and include only that pkginfo candidate.",
  },
];

const assignmentSchema = z
  .object({
    priority: z.number().int("Priority must be a whole number.").positive("Priority starts at 1."),
    effect: z.enum(["include", "exclude"]),
    label_id: z
      .string()
      .trim()
      .refine((value) => value !== "", "Pick a label."),
    package_selection: z.enum(["latest_eligible", "specific_package"]),
    pinned_package_id: z.string().trim(),
    action: z.enum(["install", "remove", "update_if_present", "none"]),
    optional_install: z.boolean(),
    featured_item: z.boolean(),
  })
  .superRefine((value, ctx) => {
    if (value.effect === "exclude") return;
    if (value.package_selection === "specific_package" && !Number(value.pinned_package_id)) {
      ctx.addIssue({ code: "custom", message: "Package is required.", path: ["pinned_package_id"] });
    }
    if (value.featured_item && !value.optional_install) {
      ctx.addIssue({
        code: "custom",
        message: "Featured Items must also be Optional Installs.",
        path: ["featured_item"],
      });
    }
    if (value.action === "remove" && (value.optional_install || value.featured_item)) {
      ctx.addIssue({
        code: "custom",
        message: "Managed Uninstalls cannot also be Optional Installs or Featured Items.",
        path: ["optional_install"],
      });
    }
  });

interface AssignmentFormState {
  priority: number;
  effect: AssignmentEffect;
  label_id: string;
  package_selection: PackageSelection;
  pinned_package_id: string;
  action: AssignmentAction;
  optional_install: boolean;
  featured_item: boolean;
}

export function MunkiAssignmentNewPage() {
  const navigate = useNavigate();
  const softwareId = useSoftwareIDParam();
  const software = useMunkiSoftwareTitle(softwareId);
  const create = useCreateMunkiAssignment();
  const packages = software.data?.packages ?? [];
  const [form, setForm] = useState<AssignmentFormState>({
    priority: 1,
    effect: "include",
    label_id: "",
    package_selection: "latest_eligible",
    pinned_package_id: "",
    action: "install",
    optional_install: false,
    featured_item: false,
  });
  const [showErrors, setShowErrors] = useState(false);
  const parsed = useMemo(() => assignmentSchema.safeParse(form), [form]);
  const errors = useMemo(() => fieldErrors(parsed), [parsed]);

  useEffect(() => {
    if (!software.data) return;
    setForm((current) => ({
      ...current,
      priority: current.priority === 1 ? (software.data.assignments?.length ?? 0) + 1 : current.priority,
    }));
  }, [software.data]);

  async function submit() {
    const next = assignmentSchema.safeParse(form);
    if (!next.success || softwareId === null) {
      setShowErrors(true);
      return;
    }
    const body = assignmentBody(softwareId, next.data);
    await create.mutateAsync(body);
    void navigate({ to: "/munki/software-titles/$softwareId", params: { softwareId: String(softwareId) } });
  }

  return (
    <PageShell asChild>
      <form noValidate onSubmit={(event) => runSubmit(event, submit)}>
        <PageHeader title="New Assignment" description="Priority decides which matching row wins for this software." />
        <MutationError title="Failed to Create Assignment" message={create.error?.message ?? software.error?.message} />
        <AssignmentFormFields
          form={form}
          packages={packages}
          showErrors={showErrors}
          errors={errors}
          loadingPackages={software.isLoading}
          onChange={setForm}
        />
        <FormActions
          pending={create.isPending}
          cancelTo="/munki/software-titles/$softwareId"
          cancelParams={{ softwareId: String(softwareId ?? "") }}
        />
      </form>
    </PageShell>
  );
}

export function MunkiAssignmentEditPage() {
  const navigate = useNavigate();
  const softwareId = useSoftwareIDParam();
  const assignmentId = useAssignmentIDParam();
  const software = useMunkiSoftwareTitle(softwareId);
  const assignment = useMunkiAssignment(assignmentId);
  const update = useUpdateMunkiAssignment();
  const packages = software.data?.packages ?? [];
  const [form, setForm] = useState<AssignmentFormState>({
    priority: 1,
    effect: "include",
    label_id: "",
    package_selection: "latest_eligible",
    pinned_package_id: "",
    action: "install",
    optional_install: false,
    featured_item: false,
  });
  const [showErrors, setShowErrors] = useState(false);
  const parsed = useMemo(() => assignmentSchema.safeParse(form), [form]);
  const errors = useMemo(() => fieldErrors(parsed), [parsed]);

  useEffect(() => {
    if (!assignment.data) return;
    setForm({
      priority: assignment.data.priority,
      effect: assignment.data.effect,
      label_id: String(assignment.data.label_id),
      package_selection: assignment.data.package_selection ?? "latest_eligible",
      pinned_package_id: assignment.data.pinned_package_id ? String(assignment.data.pinned_package_id) : "",
      action: assignment.data.action ?? "install",
      optional_install: assignment.data.optional_install,
      featured_item: assignment.data.featured_item,
    });
  }, [assignment.data]);

  async function submit() {
    const next = assignmentSchema.safeParse(form);
    if (!next.success || softwareId === null || assignmentId === null) {
      setShowErrors(true);
      return;
    }
    const body = assignmentBody(softwareId, next.data);
    await update.mutateAsync({ id: assignmentId, body });
    void navigate({ to: "/munki/software-titles/$softwareId", params: { softwareId: String(softwareId) } });
  }

  return (
    <PageShell asChild>
      <form noValidate onSubmit={(event) => runSubmit(event, submit)}>
        <PageHeader title="Edit Assignment" description="Priority 1 is evaluated first." />
        <MutationError
          title="Failed to Update Assignment"
          message={update.error?.message ?? assignment.error?.message ?? software.error?.message}
        />
        <AssignmentFormFields
          form={form}
          packages={packages}
          showErrors={showErrors}
          errors={errors}
          loadingPackages={software.isLoading}
          onChange={setForm}
        />
        <FormActions
          pending={update.isPending || assignment.isLoading}
          cancelTo="/munki/software-titles/$softwareId"
          cancelParams={{ softwareId: String(softwareId ?? "") }}
        />
      </form>
    </PageShell>
  );
}

function AssignmentFormFields({
  form,
  packages,
  showErrors,
  errors,
  loadingPackages,
  onChange,
}: {
  form: AssignmentFormState;
  packages: Array<{ id: number; version: string; display_name?: string; name: string }>;
  showErrors: boolean;
  errors: Record<string, string>;
  loadingPackages: boolean;
  onChange: (next: AssignmentFormState) => void;
}) {
  const include = form.effect === "include";
  return (
    <FieldGroup className="max-w-3xl">
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
          value={form.label_id ? [Number(form.label_id)] : []}
          onChange={(labelIDs) => onChange({ ...form, label_id: labelIDs[0] ? String(labelIDs[0]) : "" })}
          selectionMode="single"
          includeBuiltins
          placeholder="Select label"
          required
          invalid={showErrors && errors.label_id ? true : undefined}
        />
        {showErrors && errors.label_id ? <FieldError>{errors.label_id}</FieldError> : null}
      </Field>

      <Field>
        <FieldLabel htmlFor="munki-assignment-effect" required>
          Effect
        </FieldLabel>
        <Select
          value={form.effect}
          onValueChange={(effect) =>
            onChange({
              ...form,
              effect: effect as AssignmentEffect,
              optional_install: effect === "exclude" ? false : form.optional_install,
              featured_item: effect === "exclude" ? false : form.featured_item,
            })
          }
        >
          <SelectTrigger id="munki-assignment-effect" className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              <SelectItem value="include">Include</SelectItem>
              <SelectItem value="exclude">Exclude</SelectItem>
            </SelectGroup>
          </SelectContent>
        </Select>
      </Field>

      {include ? (
        <>
          <Field>
            <FieldLabel htmlFor="munki-assignment-selection" required>
              Package Selection
            </FieldLabel>
            <Select
              value={form.package_selection}
              onValueChange={(package_selection) =>
                onChange({
                  ...form,
                  package_selection: package_selection as PackageSelection,
                  pinned_package_id: package_selection === "latest_eligible" ? "" : form.pinned_package_id,
                })
              }
            >
              <SelectTrigger id="munki-assignment-selection" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {packageSelectionOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
            <FieldDescription>{packageSelectionDescription(form.package_selection)}</FieldDescription>
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
                  action: action as AssignmentAction,
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
                  {actionOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
            <FieldDescription>{actionDescription(form.action)}</FieldDescription>
          </Field>

          <FieldSet>
            <FieldLegend>Managed Software Centre</FieldLegend>
            <FieldDescription>These write the optional_installs and featured_items manifest sections.</FieldDescription>
            <CheckboxField
              id="munki-assignment-optional-install"
              label="Optional Installs"
              description="Adds this item to optional_installs so it appears in MSC."
              checked={form.optional_install}
              disabled={form.action === "remove"}
              onChange={(optional_install) =>
                onChange({
                  ...form,
                  optional_install,
                  featured_item: optional_install ? form.featured_item : false,
                })
              }
            />
            <CheckboxField
              id="munki-assignment-featured-item"
              label="Featured Items"
              description="Also adds this item to featured_items. Munki expects featured items to also be optional installs."
              checked={form.featured_item}
              disabled={form.action === "remove"}
              onChange={(featured_item) =>
                onChange({
                  ...form,
                  optional_install: featured_item ? true : form.optional_install,
                  featured_item,
                })
              }
            />
            {showErrors && errors.optional_install ? <FieldError>{errors.optional_install}</FieldError> : null}
            {showErrors && errors.featured_item ? <FieldError>{errors.featured_item}</FieldError> : null}
          </FieldSet>
        </>
      ) : null}
    </FieldGroup>
  );
}

function assignmentBody(softwareId: number, form: AssignmentFormState): MunkiAssignmentMutation {
  const body: MunkiAssignmentMutation = {
    software_id: softwareId,
    priority: form.priority,
    effect: form.effect,
    label_id: Number(form.label_id),
  };
  if (form.effect === "exclude") {
    return body;
  }
  return {
    ...body,
    action: form.action,
    optional_install: form.optional_install,
    featured_item: form.featured_item,
    package_selection: form.package_selection,
    pinned_package_id: form.package_selection === "specific_package" ? Number(form.pinned_package_id) : undefined,
  };
}

function packageSelectionDescription(value: PackageSelection) {
  return packageSelectionOptions.find((option) => option.value === value)?.description;
}

function actionDescription(value: AssignmentAction) {
  return actionOptions.find((option) => option.value === value)?.description;
}
