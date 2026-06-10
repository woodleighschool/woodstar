import { Loader2 } from "lucide-react";
import type { ComponentProps } from "react";

import { Button } from "@/components/ui/button";

// One submit/action-button idiom: spinner-prefix while pending + disabled.
// type defaults to "submit" but is overridable for action buttons (rotate, reorder).
export function SubmitButton({
  pending,
  disabled,
  children,
  ...props
}: ComponentProps<typeof Button> & { pending: boolean }) {
  return (
    <Button type="submit" disabled={pending || disabled} {...props}>
      {pending ? <Loader2 className="animate-spin" data-icon="inline-start" /> : null}
      {children}
    </Button>
  );
}
