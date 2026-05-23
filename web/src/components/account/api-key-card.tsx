import { Copy, KeyRound, Loader2, RefreshCw, Trash2 } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { InputGroup, InputGroupAddon, InputGroupButton, InputGroupInput } from "@/components/ui/input-group";
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
  }

  async function handleRevoke() {
    await revoke.mutateAsync();
    setConfirmRevoke(false);
  }

  return (
    <Card className="gap-4 py-4">
      <CardHeader className="px-4">
        <CardTitle className="flex items-center gap-2">
          <KeyRound className="size-4" />
          API key
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
        {error ? (
          <Alert variant="destructive">
            <AlertTitle>Failed to load</AlertTitle>
            <AlertDescription>{error.message}</AlertDescription>
            <Button variant="outline" size="sm" onClick={() => void refetch()} className="mt-2 w-fit">
              Retry
            </Button>
          </Alert>
        ) : null}

        {isLoading ? (
          <Loader2 className="size-4 animate-spin" />
        ) : apiKey ? (
          <>
            <InputGroup>
              <InputGroupInput value={apiKey} readOnly className="font-mono text-xs" />
              <InputGroupAddon align="inline-end">
                <TooltipProvider>
                  <div className="flex items-center gap-1">
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <InputGroupButton size="icon-xs" onClick={() => void handleCopy()}>
                          <Copy />
                        </InputGroupButton>
                      </TooltipTrigger>
                      <TooltipContent>Copy</TooltipContent>
                    </Tooltip>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <InputGroupButton size="icon-xs" disabled={pending} onClick={() => setConfirmRotate(true)}>
                          <RefreshCw />
                        </InputGroupButton>
                      </TooltipTrigger>
                      <TooltipContent>Rotate</TooltipContent>
                    </Tooltip>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <InputGroupButton size="icon-xs" disabled={pending} onClick={() => setConfirmRevoke(true)}>
                          <Trash2 />
                        </InputGroupButton>
                      </TooltipTrigger>
                      <TooltipContent>Revoke</TooltipContent>
                    </Tooltip>
                  </div>
                </TooltipProvider>
              </InputGroupAddon>
            </InputGroup>
            {createdAt ? (
              <p className="text-muted-foreground text-xs" title={new Date(createdAt).toLocaleString()}>
                Created {formatRelative(createdAt)}
              </p>
            ) : null}
          </>
        ) : null}
      </CardContent>

      <AlertDialog open={confirmRotate} onOpenChange={setConfirmRotate}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Rotate API key?</AlertDialogTitle>
            <AlertDialogDescription>The current key stops working immediately.</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="ghost" size="sm" disabled={rotate.isPending}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              size="sm"
              disabled={rotate.isPending}
              onClick={(event) => {
                event.preventDefault();
                void handleRotate();
              }}
            >
              Rotate
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={confirmRevoke} onOpenChange={setConfirmRevoke}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Revoke API key?</AlertDialogTitle>
            <AlertDialogDescription>The current key stops working immediately.</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="ghost" size="sm" disabled={revoke.isPending}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              size="sm"
              disabled={revoke.isPending}
              onClick={(event) => {
                event.preventDefault();
                void handleRevoke();
              }}
            >
              Revoke
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Card>
  );
}
