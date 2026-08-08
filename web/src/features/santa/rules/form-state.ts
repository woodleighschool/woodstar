import { z } from "zod";

import type { LabelRef, SantaRule, SantaRuleMutation } from "@lib/api";
import { requiredString } from "@lib/form-validation";
import { isOneOf, nonEmpty } from "@lib/utils";

import { santaCELExpressionError } from "./cel";
import {
  POLICY_VALUES,
  RULE_IDENTIFIER_RULES,
  RULE_TYPE_VALUES,
  type SantaRulePolicy,
  type SantaRuleType,
} from "./metadata";

const labelRefSchema = z.object({
  label_id: z.number().int("Label selection is invalid.").positive("Pick a label."),
});

export const ruleFormSchema = z
  .object({
    configuration_id: z
      .number()
      .int("Configuration selection is invalid.")
      .positive("Pick a configuration."),
    rule_type: z.enum(RULE_TYPE_VALUES),
    identifier: requiredString("Identifier"),
    name: requiredString("Name"),
    description: z.string().trim(),
    policy: z.enum(POLICY_VALUES),
    cel_expression: z.string().trim(),
    custom_message: z.string().trim(),
    custom_url: z
      .string()
      .trim()
      .refine((value) => value === "" || isHTTPSURL(value), "Custom URL must be an HTTPS URL."),
    targets: z.object({
      include: z.array(labelRefSchema),
      exclude: z.array(labelRefSchema),
    }),
  })
  .superRefine((value, ctx) => {
    const identifierRule = RULE_IDENTIFIER_RULES[value.rule_type];
    if (value.identifier !== "" && !identifierRule.pattern.test(value.identifier)) {
      ctx.addIssue({
        code: "custom",
        message: "Identifier is invalid for the selected rule type.",
        path: ["identifier"],
      });
    }
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

export interface RuleFormState {
  configuration_id: number;
  rule_type: SantaRuleType;
  identifier: string;
  name: string;
  description: string;
  policy: SantaRulePolicy;
  cel_expression: string;
  custom_message: string;
  custom_url: string;
  targets: {
    include: LabelRef[];
    exclude: LabelRef[];
  };
}

const emptyRuleForm: RuleFormState = {
  configuration_id: 0,
  rule_type: "signingid",
  identifier: "",
  name: "",
  description: "",
  policy: "blocklist",
  cel_expression: "",
  custom_message: "",
  custom_url: "",
  targets: {
    include: [],
    exclude: [],
  },
};

export function formFromSearch(
  search: Record<string, unknown>,
  configurationID: number,
): RuleFormState {
  const ruleType = isRuleType(search.rule_type) ? search.rule_type : emptyRuleForm.rule_type;
  return {
    ...emptyRuleForm,
    configuration_id: configurationID,
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
    configuration_id: rule.configuration_id,
    rule_type: rule.rule_type,
    identifier: rule.identifier,
    name: rule.name,
    description: rule.description,
    policy: rule.policy,
    cel_expression: rule.cel_expression ?? "",
    custom_message: rule.custom_message,
    custom_url: rule.custom_url,
    targets: {
      include: rule.targets.include,
      exclude: rule.targets.exclude,
    },
  };
}

export function ruleBody(form: RuleFormState): SantaRuleMutation {
  return {
    configuration_id: form.configuration_id,
    rule_type: form.rule_type,
    identifier: form.identifier.trim(),
    name: form.name.trim(),
    description: nonEmpty(form.description),
    policy: form.policy,
    cel_expression: form.policy === "cel" ? nonEmpty(form.cel_expression) : undefined,
    custom_message: nonEmpty(form.custom_message),
    custom_url: nonEmpty(form.custom_url),
    targets: {
      include: form.targets.include,
      exclude: form.targets.exclude,
    },
  };
}

export function ruleIdentifierPlaceholder(ruleType: SantaRuleType) {
  return RULE_IDENTIFIER_RULES[ruleType].placeholder;
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
