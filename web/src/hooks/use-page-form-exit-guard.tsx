import type { AnyFormApi } from "@tanstack/react-form";
import { type ShouldBlockFn, useBlocker } from "@tanstack/react-router";
import { useCallback, useRef, useState } from "react";

import { ConfirmDialog } from "@components/confirm-dialog";

export function usePageFormExitGuard({
  form,
  onDiscard,
  enabled = true,
}: {
  form: AnyFormApi;
  onDiscard: () => unknown;
  enabled?: boolean;
}) {
  const [discardRequested, setDiscardRequested] = useState(false);
  const allowExit = useRef(false);
  // Router blockers run outside render, so read the form store when the exit is attempted.
  const hasUnsavedChanges = useCallback(
    () => enabled && !form.state.isDefaultValue,
    [enabled, form],
  );
  const shouldBlockNavigation = useCallback<ShouldBlockFn>(
    ({ current, next }) =>
      current.pathname !== next.pathname && hasUnsavedChanges() && !allowExit.current,
    [hasUnsavedChanges],
  );
  const shouldBlockBeforeUnload = useCallback(
    () => hasUnsavedChanges() && !allowExit.current,
    [hasUnsavedChanges],
  );

  const blocker = useBlocker({
    shouldBlockFn: shouldBlockNavigation,
    enableBeforeUnload: shouldBlockBeforeUnload,
    withResolver: true,
  });

  const runWithoutPrompt = useCallback(async (callback: () => unknown) => {
    allowExit.current = true;
    try {
      await callback();
    } finally {
      allowExit.current = false;
    }
  }, []);

  const requestDiscard = useCallback(() => {
    if (hasUnsavedChanges()) {
      setDiscardRequested(true);
      return;
    }
    void runWithoutPrompt(onDiscard);
  }, [hasUnsavedChanges, onDiscard, runWithoutPrompt]);

  const navigationBlocked = blocker.status === "blocked";
  const dialog = (
    <ConfirmDialog
      open={discardRequested || navigationBlocked}
      onOpenChange={(open) => {
        if (open) return;
        setDiscardRequested(false);
        blocker.reset?.();
      }}
      title="Discard Unsaved Changes?"
      description="Your changes have not been saved."
      confirmLabel="Discard Changes"
      variant="destructive"
      onConfirm={() => {
        if (navigationBlocked) {
          void runWithoutPrompt(() => blocker.proceed?.());
          return;
        }
        setDiscardRequested(false);
        void runWithoutPrompt(onDiscard);
      }}
    />
  );

  return { requestDiscard, runWithoutPrompt, dialog };
}
