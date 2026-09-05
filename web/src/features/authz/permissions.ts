import type { Requirement } from "@woodleighschool/authz";

import type { AuthzResource } from "@lib/api-client";

export type { AuthzResource };
export type PermissionRequirement = Requirement<AuthzResource>;
