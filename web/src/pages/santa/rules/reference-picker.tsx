import { Badge } from "@/components/ui/badge";
import { FreeTextCombobox } from "@/components/free-text-combobox";
import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field";
import {
  type SantaRuleReference,
  type SantaRuleType,
  useSantaRuleReferences,
} from "@/hooks/use-santa-rules";
import { ruleTypeLabel } from "@/lib/santa-rules";

import { type RuleFormState, ruleIdentifierHint } from "./form-state";

export function RuleReferencePicker({
  form,
  identifierError,
  onBlur,
  onChange,
}: {
  form: RuleFormState;
  identifierError?: string;
  onBlur?: () => void;
  onChange: (form: RuleFormState) => void;
}) {
  const identifierInvalid = identifierError !== undefined;
  const references = useSantaRuleReferences({
    q: form.identifier,
    rule_type: form.rule_type,
    limit: 50,
  });
  const rows = references.data ?? [];
  const selected =
    rows.find(
      (candidate) =>
        candidate.rule_type === form.rule_type && candidate.identifier === form.identifier,
    ) ?? null;
  const description = ruleReferenceDescription(selected, form.rule_type);
  const showDescription = identifierError !== description;

  return (
    <Field data-invalid={identifierInvalid || undefined}>
      <FieldLabel htmlFor="santa-rule-identifier" required>
        Identifier
      </FieldLabel>
      <FreeTextCombobox
        id="santa-rule-identifier"
        name="identifier"
        value={form.identifier}
        items={rows}
        placeholder="Enter identifier"
        invalid={identifierInvalid ? true : undefined}
        onBlur={onBlur}
        emptyText={
          references.error
            ? references.error.message
            : references.isLoading
              ? "Loading Suggestions..."
              : "No Suggestions Found."
        }
        noResultsText="No Suggestions Found."
        itemToStringValue={(reference) => reference.identifier}
        freeTextItem={(identifier) => currentRuleReferenceCandidate(identifier, form.rule_type)}
        itemKey={(reference) => `${reference.rule_type}:${reference.identifier}`}
        itemDisabled={(reference) => reference.rule_type === "bundle" && !reference.complete}
        renderItem={ruleReferenceItem}
        onChange={(identifier) => onChange({ ...form, identifier })}
        onSelectItem={(reference) =>
          onChange({
            ...form,
            rule_type: reference.rule_type,
            identifier: reference.identifier,
            name:
              form.name.trim() === "" && reference.display_name
                ? reference.display_name
                : form.name,
          })
        }
      />
      {showDescription ? <FieldDescription>{description}</FieldDescription> : null}
      {identifierError ? <FieldError>{identifierError}</FieldError> : null}
    </Field>
  );
}

function currentRuleReferenceCandidate(
  identifier: string,
  ruleType: SantaRuleType,
): SantaRuleReference {
  return {
    rule_type: ruleType,
    identifier,
    rule_count: 0,
    complete: true,
  };
}

function ruleReferenceItem(reference: SantaRuleReference) {
  const disabled = reference.rule_type === "bundle" && !reference.complete;
  const secondary = ruleReferenceSecondary(reference);
  return (
    <>
      <span className="min-w-0 flex-1">
        <span className="block leading-snug break-all">{reference.identifier}</span>
        {secondary ? (
          <span className="block truncate text-xs text-muted-foreground">{secondary}</span>
        ) : null}
      </span>
      <span className="flex shrink-0 items-center gap-2">
        {reference.rule_type === "bundle" ? (
          <span className="text-xs text-muted-foreground tabular-nums">
            {reference.collected_binary_count ?? 0}/{reference.binary_count ?? 0}
          </span>
        ) : null}
        <Badge variant={disabled ? "outline" : "secondary"} className="font-normal">
          {ruleTypeLabel(reference.rule_type)}
        </Badge>
      </span>
    </>
  );
}

function ruleReferenceSecondary(reference: SantaRuleReference) {
  if (reference.rule_type === "teamid") return undefined;
  if (!reference.display_name) return undefined;
  return reference.display_name;
}

function ruleReferenceDescription(reference: SantaRuleReference | null, ruleType: SantaRuleType) {
  if (!reference) return ruleIdentifierHint(ruleType);
  if (reference.rule_type === "bundle" && !reference.complete) {
    return "Bundle reference is incomplete.";
  }
  const description = ruleReferenceDescriptionParts(reference).join(" | ");
  return description || ruleIdentifierHint(reference.rule_type);
}

function ruleReferenceDescriptionParts(reference: SantaRuleReference) {
  const parts: string[] = [];
  if (reference.rule_type === "teamid") {
    if (reference.certificate_organization) parts.push(reference.certificate_organization);
    return parts;
  }
  if (reference.rule_type === "certificate" && reference.certificate_common_name) {
    parts.push(reference.certificate_common_name);
  }
  if (reference.rule_type === "certificate" && reference.certificate_organization) {
    parts.push(reference.certificate_organization);
  }
  if (reference.bundle_identifier) parts.push(reference.bundle_identifier);
  if (reference.version) parts.push(reference.version);
  if (reference.path) parts.push(reference.path);
  return parts;
}
