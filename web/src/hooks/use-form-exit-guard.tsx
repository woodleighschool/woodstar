import type { AnyFormApi } from "@tanstack/react-form";
import { useBlocker } from "@tanstack/react-router";
import { useSelector } from "@tanstack/react-store";
import { useCallback, useRef, useState } from "react";

import { ConfirmDialog } from "@/components/confirm-dialog";

export function useFormExitGuard({
  form,
  onDiscard,
  blockNavigation = true,
}: {
  form: AnyFormApi;
  onDiscard: () => void;
  blockNavigation?: boolean;
}) {
  const [discardRequested, setDiscardRequested] = useState(false);
  const allowExit = useRef(false);
  const isDefaultValue = useSelector(form.store, (state) => state.isDefaultValue);
  const shouldBlock = !isDefaultValue;

  const blocker = useBlocker({
    shouldBlockFn: () => shouldBlock && !allowExit.current,
    enableBeforeUnload: shouldBlock,
    withResolver: true,
    disabled: !blockNavigation,
  });

  const leave = useCallback((callback: () => void) => {
    allowExit.current = true;
    callback();
    queueMicrotask(() => {
      allowExit.current = false;
    });
  }, []);

  const requestDiscard = useCallback(() => {
    if (shouldBlock) {
      setDiscardRequested(true);
      return;
    }
    onDiscard();
  }, [onDiscard, shouldBlock]);

  const navigationBlocked = blocker.status === "blocked";
  const dialog = (
    <ConfirmDialog
      open={discardRequested || navigationBlocked}
      onOpenChange={(open) => {
        if (open) return;
        setDiscardRequested(false);
        blocker.reset?.();
      }}
      title="Discard unsaved changes?"
      description="Your changes have not been saved."
      confirmLabel="Discard changes"
      variant="destructive"
      onConfirm={() => {
        if (navigationBlocked) {
          leave(() => blocker.proceed?.());
          return;
        }
        setDiscardRequested(false);
        leave(onDiscard);
      }}
    />
  );

  return { requestDiscard, dialog };
}
