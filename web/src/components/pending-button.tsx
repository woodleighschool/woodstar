import type { ReactNode } from "react";

import { Pending } from "@/components/pending";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";

type PendingButtonProps = React.ComponentProps<typeof Button> & {
  isPending?: boolean;
  icon?: ReactNode;
};

export function PendingButton({
  isPending = false,
  disabled,
  icon,
  children,
  ...props
}: PendingButtonProps) {
  return (
    <Pending
      isPending={isPending}
      disabled={disabled}
      render={<Button disabled={disabled} {...props} />}
    >
      {isPending ? <Spinner data-icon="inline-start" /> : icon}
      {children}
    </Pending>
  );
}
