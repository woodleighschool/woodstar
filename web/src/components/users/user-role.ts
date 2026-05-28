import type { User } from "@/hooks/use-users";
import { enumLabel, enumOptions, type EnumMetadataMap } from "@/lib/enum-metadata";

export type UserRole = User["role"];

export const USER_ROLE_VALUES = ["admin", "viewer"] as const satisfies readonly UserRole[];

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

export const USER_ROLE_OPTIONS = enumOptions(USER_ROLES);

export function userRoleLabel(value: string | null | undefined) {
  return enumLabel(USER_ROLES, value);
}
