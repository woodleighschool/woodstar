import type { Label } from "@/hooks/use-labels";
import { enumLabel, type EnumMetadataMap, enumOptions } from "@/lib/enum-metadata";

export type LabelMembershipType = Label["label_membership_type"];
export type LabelBuiltinKey = NonNullable<Label["builtin_key"]>;
export type LabelDerivedAttribute = "user_department" | "directory_group" | "user";

export const BUILTIN_KEY_ALL_HOSTS = "all-hosts" as const satisfies LabelBuiltinKey;

export const LABEL_MEMBERSHIP_VALUES = [
  "dynamic",
  "manual",
  "derived",
] as const satisfies readonly LabelMembershipType[];

export const LABEL_MEMBERSHIP_TYPES = {
  dynamic: {
    name: "Dynamic",
    description: "Membership comes from an osquery SQL query evaluated on each host.",
  },
  manual: {
    name: "Manual",
    description: "Membership is managed directly in Woodstar.",
  },
  derived: {
    name: "Derived",
    description: "Membership is computed from linked users, groups, or other host attributes.",
  },
} satisfies EnumMetadataMap<LabelMembershipType>;

export const LABEL_MEMBERSHIP_OPTIONS = enumOptions(LABEL_MEMBERSHIP_TYPES);

export const LABEL_DERIVED_ATTRIBUTE_VALUES = [
  "user_department",
  "directory_group",
  "user",
] as const satisfies readonly LabelDerivedAttribute[];

export const LABEL_DERIVED_ATTRIBUTES = {
  user_department: { name: "User Department" },
  directory_group: { name: "Group" },
  user: { name: "User" },
} satisfies EnumMetadataMap<LabelDerivedAttribute>;

export const LABEL_DERIVED_ATTRIBUTE_OPTIONS = enumOptions(LABEL_DERIVED_ATTRIBUTES);

export function labelMembershipLabel(value: LabelMembershipType) {
  return enumLabel(LABEL_MEMBERSHIP_TYPES, value);
}

export function labelDerivedAttributeSelectorLabel(attribute: LabelDerivedAttribute) {
  switch (attribute) {
    case "user_department":
      return "Departments";
    case "directory_group":
      return "Groups";
    case "user":
      return "Users";
  }
}

export function isAllHostsLabel(label: Pick<Label, "builtin_key">) {
  return label.builtin_key === BUILTIN_KEY_ALL_HOSTS;
}
