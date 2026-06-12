import type { User } from "@/hooks/use-users";
import { enumLabel, type EnumMetadataMap, enumOptions } from "@/lib/enum-metadata";

export type UserRole = NonNullable<User["role"]>;
export type UserAccessRole = UserRole | "none";

export const USER_ROLES = {
  admin: {
    name: "Admin",
    description: "Can manage Woodstar settings, users, enrollments, and data.",
    variant: "default",
  },
  viewer: {
    name: "Viewer",
    description: "Can view Woodstar data without making administrative changes.",
    variant: "secondary",
  },
} satisfies EnumMetadataMap<UserRole>;

export const USER_ACCESS_ROLES = {
  ...USER_ROLES,
  none: {
    name: "No Access",
    description: "Cannot sign in until a role is assigned.",
    variant: "outline",
  },
} satisfies EnumMetadataMap<UserAccessRole>;

export const USER_ROLE_OPTIONS = enumOptions(USER_ROLES);
export const USER_ACCESS_ROLE_OPTIONS = enumOptions(USER_ACCESS_ROLES);

export function userRoleLabel(value: string | null | undefined) {
  return enumLabel(USER_ROLES, value);
}

export function userAccessRole(value: User["role"]): UserAccessRole {
  return value ?? "none";
}

export function userMutationRole(value: UserAccessRole): User["role"] {
  return value === "none" ? undefined : value;
}
