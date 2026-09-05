import { createAuthz } from "@woodleighschool/authz/react";
import type { ReactNode } from "react";

import { useAccount } from "@features/account/queries";
import type { AuthzResource } from "@features/authz/permissions";

const { AuthzProvider: Provider, useCan, Can } = createAuthz<AuthzResource>();
export { useCan, Can };

export function AuthzProvider({ children }: { children: ReactNode }) {
  const account = useAccount();
  return <Provider permissions={account.data?.effective_permissions}>{children}</Provider>;
}
