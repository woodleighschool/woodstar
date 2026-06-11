import { Badge } from "@/components/ui/badge";
import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox";
import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field";
import { useSantaRuleReferences, type SantaRuleReference, type SantaRuleType } from "@/hooks/use-santa-rules";
import { ruleTypeLabel } from "@/lib/santa-rules";

import { ruleIdentifierHint, type RuleFormState } from "./form-state";

export function RuleReferencePicker({
  form,
  identifierError,
  identifierInvalid,
  onChange,
}: {
  form: RuleFormState;
  identifierError?: string;
  identifierInvalid: boolean;
  onChange: (form: RuleFormState) => void;
}) {
  const references = useSantaRuleReferences({ rule_type: form.rule_type, limit: 50 });
  const rows = references.data ?? [];
  const selected =
    rows.find((candidate) => candidate.rule_type === form.rule_type && candidate.identifier === form.identifier) ??
    currentRuleReferenceCandidate(form);
  const items =
    selected &&
    !rows.some(
      (candidate) => candidate.rule_type === selected.rule_type && candidate.identifier === selected.identifier,
    )
      ? [selected, ...rows]
      : rows;

  return (
    <Field data-invalid={identifierInvalid}>
      <FieldLabel htmlFor="santa-rule-reference" required>
        Identifier
      </FieldLabel>
      <Combobox
        items={items}
        value={selected}
        itemToStringLabel={referenceLabel}
        itemToStringValue={(candidate) => candidate.identifier}
        onValueChange={(candidate) => {
          if (!candidate) {
            onChange({ ...form, identifier: "" });
            return;
          }
          onChange({
            ...form,
            rule_type: candidate.rule_type,
            identifier: candidate.identifier,
            name: form.name.trim() === "" && candidate.display_name ? candidate.display_name : form.name,
          });
        }}
      >
        <ComboboxInput
          id="santa-rule-reference"
          required
          aria-invalid={identifierInvalid ? true : undefined}
          showClear
          disabled={references.isLoading}
          placeholder={references.isLoading ? "Loading References" : "Select Identifier"}
        />
        <ComboboxContent>
          <ComboboxEmpty>
            {references.error
              ? references.error.message
              : references.isLoading
                ? "Loading References..."
                : "No References Found."}
          </ComboboxEmpty>
          <ComboboxList>{ruleReferenceItem}</ComboboxList>
        </ComboboxContent>
      </Combobox>
      <FieldDescription>{ruleReferenceDescription(selected, form.rule_type)}</FieldDescription>
      {identifierInvalid && identifierError ? <FieldError>{identifierError}</FieldError> : null}
    </Field>
  );
}

function currentRuleReferenceCandidate(form: RuleFormState): SantaRuleReference | null {
  const identifier = form.identifier.trim();
  if (identifier === "") return null;
  return {
    rule_type: form.rule_type,
    identifier,
    rule_count: 0,
    complete: true,
  };
}

function referenceLabel(reference: SantaRuleReference) {
  return reference.identifier;
}

function ruleReferenceItem(reference: SantaRuleReference) {
  const disabled = reference.rule_type === "bundle" && !reference.complete;
  const secondary = ruleReferenceSecondary(reference);
  return (
    <ComboboxItem key={`${reference.rule_type}:${reference.identifier}`} value={reference} disabled={disabled}>
      <span className="min-w-0 flex-1">
        <span className="block break-all leading-snug">{referenceLabel(reference)}</span>
        {secondary ? <span className="text-muted-foreground block truncate text-xs">{secondary}</span> : null}
      </span>
      <span className="flex shrink-0 items-center gap-2">
        {reference.rule_type === "bundle" ? (
          <span className="text-muted-foreground text-xs tabular-nums">
            {reference.collected_binary_count ?? 0}/{reference.binary_count ?? 0}
          </span>
        ) : null}
        <Badge variant={disabled ? "outline" : "secondary"} className="font-normal">
          {ruleTypeLabel(reference.rule_type)}
        </Badge>
      </span>
    </ComboboxItem>
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
