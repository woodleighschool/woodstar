import { z } from "zod";

import type { SantaRule, SantaRuleMutation, SantaRulePolicy, SantaRuleType } from "@/hooks/use-santa";
import { optionalText, requiredString, selectedIDArray } from "@/lib/form-validation";
import { santaCELExpressionError } from "@/lib/santa-cel";
import { POLICY_VALUES, RULE_IDENTIFIER_RULES, RULE_TYPE_VALUES } from "@/lib/santa-rules";

const includeSchema = z
  .object({
    id: z.number(),
    policy: z.enum(POLICY_VALUES),
    cel_expression: z.string().trim(),
    label_id: z.number().int("Label selection is invalid.").nullable(),
  })
  .refine((value) => value.label_id !== null, {
    message: "Pick a label.",
    path: ["label_id"],
  })
  .superRefine((value, ctx) => {
    if (value.policy !== "cel") return;
    if (value.cel_expression === "") {
      ctx.addIssue({ code: "custom", message: "CEL policy requires an expression.", path: ["cel_expression"] });
      return;
    }
    const error = santaCELExpressionError(value.cel_expression);
    if (error) {
      ctx.addIssue({ code: "custom", message: error, path: ["cel_expression"] });
    }
  });

export const ruleFormSchema = z
  .object({
    rule_type: z.enum(RULE_TYPE_VALUES),
    identifier: requiredString("Target"),
    name: requiredString("Name"),
    description: z.string().trim(),
    custom_message: z.string().trim(),
    custom_url: z.string().trim(),
    exclude_label_ids: selectedIDArray("Label"),
    includes: z.array(includeSchema),
  })
  .superRefine((value, ctx) => {
    if (value.identifier === "") return;
    const rule = RULE_IDENTIFIER_RULES[value.rule_type];
    if (!rule.pattern.test(value.identifier)) {
      ctx.addIssue({ code: "custom", message: rule.hint, path: ["identifier"] });
    }
  });

export type RuleFormParse = ReturnType<typeof ruleFormSchema.safeParse>;
export type RuleIncludeErrors = { cel_expression?: string; label_id?: string };

export interface RuleIncludeForm {
  id: number;
  policy: SantaRulePolicy;
  cel_expression: string;
  label_id: number | null;
}

export interface RuleFormState {
  rule_type: SantaRuleType;
  identifier: string;
  name: string;
  description: string;
  custom_message: string;
  custom_url: string;
  exclude_label_ids: number[];
  includes: RuleIncludeForm[];
}

export const emptyRuleForm: RuleFormState = {
  rule_type: "binary",
  identifier: "",
  name: "",
  description: "",
  custom_message: "",
  custom_url: "",
  exclude_label_ids: [],
  includes: [],
};

export function formFromSearch(search: Record<string, unknown>): RuleFormState {
  const ruleType = isRuleType(search.rule_type) ? search.rule_type : emptyRuleForm.rule_type;
  return {
    ...emptyRuleForm,
    rule_type: ruleType,
    identifier: typeof search.identifier === "string" ? search.identifier : "",
    name: typeof search.name === "string" ? search.name : "",
    description: "",
  };
}

function isRuleType(value: unknown): value is SantaRuleType {
  return typeof value === "string" && RULE_TYPE_VALUES.includes(value as SantaRuleType);
}

export function formFromRule(rule: SantaRule): RuleFormState {
  return {
    rule_type: rule.rule_type,
    identifier: rule.identifier,
    name: rule.name,
    description: rule.description,
    custom_message: rule.custom_message,
    custom_url: rule.custom_url,
    exclude_label_ids: rule.exclude_label_ids ?? [],
    includes: (rule.includes ?? []).map((include) => ({
      id: include.id,
      policy: include.policy,
      cel_expression: include.cel_expression ?? "",
      label_id: include.label_id,
    })),
  };
}

export function ruleBody(form: RuleFormState): SantaRuleMutation {
  return {
    rule_type: form.rule_type,
    identifier: form.identifier.trim(),
    name: form.name.trim(),
    description: optionalText(form.description),
    custom_message: optionalText(form.custom_message),
    custom_url: optionalText(form.custom_url),
    exclude_label_ids: form.exclude_label_ids,
    includes: form.includes.map(includeBody),
  };
}

function includeBody(include: RuleIncludeForm) {
  if (include.label_id === null) {
    throw new Error("validated include is missing a label");
  }
  return {
    policy: include.policy,
    cel_expression: include.policy === "cel" ? optionalText(include.cel_expression) : undefined,
    label_id: include.label_id,
  };
}

export function includeErrorMap(
  result: RuleFormParse,
  includes: RuleIncludeForm[],
): Partial<Record<number, RuleIncludeErrors>> {
  if (result.success) return {};
  const out: Partial<Record<number, RuleIncludeErrors>> = {};
  for (const issue of result.error.issues) {
    if (issue.path[0] !== "includes") continue;
    const index = issue.path[1];
    if (typeof index !== "number" || index >= includes.length) continue;
    const include = includes[index];
    const entry = out[include.id] ?? {};
    const field = issue.path[2];
    if (field === "cel_expression" && !entry.cel_expression) entry.cel_expression = issue.message;
    if (field === "label_id" && !entry.label_id) entry.label_id = issue.message;
    out[include.id] = entry;
  }
  return out;
}

export function selectedIncludeLabelIDs(includes: RuleIncludeForm[]) {
  return includes.flatMap((include) => (include.label_id === null ? [] : [include.label_id]));
}

export function ruleIdentifierHint(ruleType: SantaRuleType) {
  return RULE_IDENTIFIER_RULES[ruleType].hint;
}

export function identifierErrorFor(result: RuleFormParse): string | undefined {
  if (result.success) return undefined;
  return result.error.issues.find((issue) => issue.path[0] === "identifier")?.message;
}
