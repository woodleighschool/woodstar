import { Copy, KeyRound, RefreshCw, Trash2 } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

import { ConfirmDialog } from "@/components/confirm-dialog";
import { QueryError } from "@/components/query-error";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@/components/ui/input-group";
import { Skeleton } from "@/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { useAccount, useRevokeAPIKey, useRotateAPIKey } from "@/hooks/use-account";
import { formatRelative } from "@/lib/utils";
export function APIKeyCard() {
  const { data, isLoading, error, refetch } = useAccount();
  const rotate = useRotateAPIKey();
  const revoke = useRevokeAPIKey();
  const [confirmRotate, setConfirmRotate] = useState(false);
  const [confirmRevoke, setConfirmRevoke] = useState(false);
  const apiKey = data?.api_key ?? "";
  const createdAt = data?.api_key_created_at;
  const pending = rotate.isPending || revoke.isPending;
  async function handleCopy() {
    if (!apiKey) return;
    try {
      await navigator.clipboard.writeText(apiKey);
      toast.success("Copied");
    } catch {
      toast.error("Copy failed");
    }
  }
  async function handleRotate() {
    await rotate.mutateAsync();
    setConfirmRotate(false);
    toast.success("API key rotated");
  }
  async function handleRevoke() {
    await revoke.mutateAsync();
    setConfirmRevoke(false);
    toast.success("API key revoked");
  }
  return (
    <Card className="gap-4 py-4">
      <CardHeader className="px-4">
        <CardTitle className="flex items-center gap-2">
          <KeyRound className="size-4" />
          API Key
        </CardTitle>
        <CardDescription>For CLI and automation access.</CardDescription>
        {!isLoading && !apiKey ? (
          <CardAction>
            <Button type="button" size="sm" disabled={pending} onClick={() => void handleRotate()}>
              Generate
            </Button>
          </CardAction>
        ) : null}
      </CardHeader>
      <CardContent className="flex flex-col gap-3 px-4">
        <QueryError title="Failed to load API key" error={error} onRetry={() => void refetch()} />

        {isLoading ? (
          <Skeleton className="h-9 w-full" />
        ) : apiKey ? (
          <>
            <InputGroup>
              <InputGroupInput value={apiKey} readOnly className="font-mono text-xs" />
              <InputGroupAddon align="inline-end">
                <TooltipProvider>
                  <div className="flex items-center gap-1">
                    <Tooltip>
                      <TooltipTrigger
                        render={
                          <InputGroupButton
                            size="icon-xs"
                            aria-label="Copy"
                            onClick={() => void handleCopy()}
                          />
                        }
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
                            aria-label="Rotate"
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
                            aria-label="Revoke"
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
          </>
        ) : null}
      </CardContent>

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
    </Card>
  );
}
