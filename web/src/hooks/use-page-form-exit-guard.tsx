import type { AnyFormApi } from "@tanstack/react-form";
import { useBlocker } from "@tanstack/react-router";
import { useCallback, useRef, useState } from "react";

import { ConfirmDialog } from "@/components/confirm-dialog";

export function usePageFormExitGuard({
  form,
  onDiscard,
}: {
  form: AnyFormApi;
  onDiscard: () => void;
}) {
  const [discardRequested, setDiscardRequested] = useState(false);
  const allowExit = useRef(false);
  // Router blockers run outside render, so read the form store when the exit is attempted.
  const hasUnsavedChanges = useCallback(() => !form.state.isDefaultValue, [form]);
  const shouldBlockNavigation = useCallback(
    () => hasUnsavedChanges() && !allowExit.current,
    [hasUnsavedChanges],
  );

  const blocker = useBlocker({
    shouldBlockFn: shouldBlockNavigation,
    enableBeforeUnload: shouldBlockNavigation,
    withResolver: true,
  });

  const leave = useCallback((callback: () => void) => {
    allowExit.current = true;
    callback();
    queueMicrotask(() => {
      allowExit.current = false;
    });
  }, []);

  const requestDiscard = useCallback(() => {
    if (hasUnsavedChanges()) {
      setDiscardRequested(true);
      return;
    }
    onDiscard();
  }, [hasUnsavedChanges, onDiscard]);

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
