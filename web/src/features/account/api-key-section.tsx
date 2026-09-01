import { Copy, KeyRound, RefreshCw, Trash2 } from "lucide-react";
import { useState } from "react";

import { AsyncButton } from "@components/async-button";
import { ConfirmDialog } from "@components/confirm-dialog";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@components/ui/input-group";
import { toast } from "@components/ui/toast";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@components/ui/tooltip";
import { useRevokeAPIKey, useRotateAPIKey } from "@features/account/queries";
import type { Account } from "@lib/api";
import { formatRelative } from "@lib/utils";

export function APIKeySection({ account }: { account: Account }) {
  const rotate = useRotateAPIKey();
  const revoke = useRevokeAPIKey();
  const [confirmRotate, setConfirmRotate] = useState(false);
  const [confirmRevoke, setConfirmRevoke] = useState(false);
  const apiKey = account.api_key ?? "";
  const createdAt = account.api_key_created_at;
  const pending = rotate.isPending || revoke.isPending;

  async function handleCopy() {
    if (!apiKey) return;
    try {
      await navigator.clipboard.writeText(apiKey);
      toast.add({ title: "Copied", type: "success" });
    } catch {
      toast.add({ title: "Copy Failed", type: "error" });
    }
  }

  async function handleRotate() {
    await rotate.mutateAsync();
    setConfirmRotate(false);
    toast.add({ title: "API Key Rotated", type: "success" });
  }

  async function handleRevoke() {
    await revoke.mutateAsync();
    setConfirmRevoke(false);
    toast.add({ title: "API Key Revoked", type: "success" });
  }

  return (
    <section className="flex max-w-3xl flex-col gap-4 border-t pt-6">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="flex min-w-0 flex-col gap-1">
          <h2 className="flex items-center gap-2 text-base font-semibold">
            <KeyRound className="size-4" />
            API Key
          </h2>
          <p className="text-sm text-muted-foreground">For CLI and automation access.</p>
        </div>
        {!apiKey ? (
          <AsyncButton
            type="button"
            size="sm"
            isPending={rotate.isPending}
            onClick={() => void handleRotate()}
          >
            Generate
          </AsyncButton>
        ) : null}
      </div>

      {apiKey ? (
        <div className="flex flex-col gap-2">
          <InputGroup>
            <InputGroupInput
              aria-label="API key"
              value={apiKey}
              readOnly
              className="font-mono text-xs"
            />
            <InputGroupAddon align="inline-end">
              <TooltipProvider>
                <div className="flex items-center gap-1">
                  <Tooltip>
                    <TooltipTrigger
                      render={<InputGroupButton size="icon-xs" onClick={() => void handleCopy()} />}
                    >
                      <Copy />
                    </TooltipTrigger>
                    <TooltipContent>Copy</TooltipContent>
                  </Tooltip>
                  <Tooltip>
                    <TooltipTrigger
                      render={
                        <InputGroupButton
                          size="icon-xs"
                          disabled={pending}
                          onClick={() => setConfirmRotate(true)}
                        />
                      }
                    >
                      <RefreshCw />
                    </TooltipTrigger>
                    <TooltipContent>Rotate</TooltipContent>
                  </Tooltip>
                  <Tooltip>
                    <TooltipTrigger
                      render={
                        <InputGroupButton
                          size="icon-xs"
                          disabled={pending}
                          onClick={() => setConfirmRevoke(true)}
                        />
                      }
                    >
                      <Trash2 />
                    </TooltipTrigger>
                    <TooltipContent>Revoke</TooltipContent>
                  </Tooltip>
                </div>
              </TooltipProvider>
            </InputGroupAddon>
          </InputGroup>
          {createdAt ? (
            <p
              className="text-xs text-muted-foreground"
              title={new Date(createdAt).toLocaleString()}
            >
              Created {formatRelative(createdAt)}
            </p>
          ) : null}
        </div>
      ) : null}

      <ConfirmDialog
        open={confirmRotate}
        onOpenChange={setConfirmRotate}
        title="Rotate API Key?"
        description="The current key stops working immediately."
        confirmLabel="Rotate"
        pending={rotate.isPending}
        onConfirm={() => void handleRotate()}
      />

      <ConfirmDialog
        open={confirmRevoke}
        onOpenChange={setConfirmRevoke}
        title="Revoke API Key?"
        description="The current key stops working immediately."
        confirmLabel="Revoke"
        variant="destructive"
        pending={revoke.isPending}
        onConfirm={() => void handleRevoke()}
      />
    </section>
  );
}
