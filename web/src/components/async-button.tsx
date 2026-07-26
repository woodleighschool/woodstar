import type { ReactNode } from "react";

import { Pending } from "@components/pending";
import { Button } from "@components/ui/button";
import { Spinner } from "@components/ui/spinner";

type AsyncButtonProps = React.ComponentProps<typeof Button> & {
  isPending?: boolean;
  icon?: ReactNode;
};

export function AsyncButton({
  isPending = false,
  disabled,
  icon,
  children,
  ...props
}: AsyncButtonProps) {
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
