import { z } from "zod";

import type { LabelRef, SantaRule, SantaRuleMutation } from "@/lib/api";
import { requiredString } from "@/lib/form-validation";
import {
  POLICY_VALUES,
  RULE_IDENTIFIER_RULES,
  RULE_TYPE_VALUES,
  type SantaRulePolicy,
  type SantaRuleType,
} from "@/lib/santa-rules";
import { isOneOf, nonEmpty } from "@/lib/utils";

import { santaCELExpressionError } from "./cel";

export const santaRuleIncludeSchema = z
  .object({
    id: z.number(),
    policy: z.enum(POLICY_VALUES),
    cel_expression: z.string().trim(),
    label_id: z.number().int("Label selection is invalid.").positive("Pick a label.").nullable(),
  })
  .refine((value) => value.label_id !== null, {
    message: "Pick a label.",
    path: ["label_id"],
  })
  .superRefine((value, ctx) => {
    if (value.policy !== "cel") return;
    if (value.cel_expression === "") {
      ctx.addIssue({
        code: "custom",
        message: "CEL policy requires an expression.",
        path: ["cel_expression"],
      });
      return;
    }
    const error = santaCELExpressionError(value.cel_expression);
    if (error) {
      ctx.addIssue({
        code: "custom",
        message: error,
        path: ["cel_expression"],
      });
    }
  });

export const ruleFormSchema = z
  .object({
    rule_type: z.enum(RULE_TYPE_VALUES),
    identifier: requiredString("Identifier"),
    name: requiredString("Name"),
    description: z.string().trim(),
    custom_message: z.string().trim(),
    custom_url: z
      .string()
      .trim()
      .refine((value) => value === "" || isHTTPSURL(value), "Custom URL must be an HTTPS URL."),
    targets: z.object({
      include: z.array(santaRuleIncludeSchema),
      exclude: z.array(
        z.object({
          label_id: z.number().int("Label selection is invalid.").positive("Pick a label."),
        }),
      ),
    }),
  })
  .superRefine((value, ctx) => {
    if (value.identifier === "") return;
    const rule = RULE_IDENTIFIER_RULES[value.rule_type];
    if (!rule.pattern.test(value.identifier)) {
      ctx.addIssue({
        code: "custom",
        message: rule.hint,
        path: ["identifier"],
      });
    }
  });

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
  targets: {
    include: RuleIncludeForm[];
    exclude: LabelRef[];
  };
}

export const emptyRuleForm: RuleFormState = {
  rule_type: "binary",
  identifier: "",
  name: "",
  description: "",
  custom_message: "",
  custom_url: "",
  targets: {
    include: [],
    exclude: [],
  },
};

export function formFromSearch(search: Record<string, unknown>): RuleFormState {
  const ruleType = isRuleType(search.rule_type) ? search.rule_type : emptyRuleForm.rule_type;
  return {
    ...emptyRuleForm,
    rule_type: ruleType,
    identifier: typeof search.identifier === "string" ? search.identifier : "",
    name: typeof search.name === "string" ? search.name : "",
  };
}

function isRuleType(value: unknown): value is SantaRuleType {
  return isOneOf(value, RULE_TYPE_VALUES);
}

export function formFromRule(rule: SantaRule): RuleFormState {
  return {
    rule_type: rule.rule_type,
    identifier: rule.identifier,
    name: rule.name,
    description: rule.description,
    custom_message: rule.custom_message,
    custom_url: rule.custom_url,
    targets: {
      include: rule.targets.include.map((include, index) => ({
        id: index + 1,
        policy: include.policy,
        cel_expression: include.cel_expression ?? "",
        label_id: include.label_id,
      })),
      exclude: rule.targets.exclude,
    },
  };
}

export function ruleBody(form: RuleFormState): SantaRuleMutation {
  return {
    rule_type: form.rule_type,
    identifier: form.identifier.trim(),
    name: form.name.trim(),
    description: nonEmpty(form.description),
    custom_message: nonEmpty(form.custom_message),
    custom_url: nonEmpty(form.custom_url),
    targets: {
      include: form.targets.include.map(includeBody),
      exclude: form.targets.exclude,
    },
  };
}

function includeBody(include: RuleIncludeForm) {
  if (include.label_id === null) {
    throw new Error("validated include is missing a label");
  }
  return {
    policy: include.policy,
    cel_expression: include.policy === "cel" ? nonEmpty(include.cel_expression) : undefined,
    label_id: include.label_id,
  };
}

export function selectedIncludeLabelIDs(includeRows: RuleIncludeForm[]) {
  return includeRows.flatMap((include) => (include.label_id === null ? [] : [include.label_id]));
}

export function ruleIdentifierHint(ruleType: SantaRuleType) {
  return RULE_IDENTIFIER_RULES[ruleType].hint;
}

function isHTTPSURL(value: string) {
  try {
    const url = new URL(value);
    return (
      url.protocol === "https:" && url.host !== "" && url.username === "" && url.password === ""
    );
  } catch {
    return false;
  }
}
