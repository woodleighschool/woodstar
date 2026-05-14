import { Copy, KeyRound, Loader2, RefreshCw, Trash2 } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
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
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <KeyRound className="size-4" />
          API key
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
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
            <code className="bg-muted block break-all rounded-md px-3 py-2 font-mono text-sm">{apiKey}</code>
            {createdAt ? (
              <p className="text-muted-foreground text-xs" title={new Date(createdAt).toLocaleString()}>
                Created {formatRelative(createdAt)}
              </p>
            ) : null}
            <div className="flex flex-wrap gap-2">
              <Button type="button" size="sm" variant="outline" onClick={() => void handleCopy()}>
                <Copy data-icon="inline-start" /> Copy
              </Button>
              <Button type="button" size="sm" variant="outline" disabled={pending} onClick={() => setConfirmRotate(true)}>
                <RefreshCw data-icon="inline-start" /> Rotate
              </Button>
              <Button type="button" size="sm" variant="ghost" disabled={pending} onClick={() => setConfirmRevoke(true)}>
                <Trash2 data-icon="inline-start" /> Revoke
              </Button>
            </div>
          </>
        ) : (
          <Button type="button" size="sm" disabled={pending} onClick={() => void handleRotate()}>
            Generate
          </Button>
        )}
      </CardContent>

      <Dialog open={confirmRotate} onOpenChange={setConfirmRotate}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Rotate API key?</DialogTitle>
            <DialogDescription>The current key stops working immediately.</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="ghost" size="sm" disabled={rotate.isPending}>
                Cancel
              </Button>
            </DialogClose>
            <Button type="button" size="sm" disabled={rotate.isPending} onClick={() => void handleRotate()}>
              Rotate
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={confirmRevoke} onOpenChange={setConfirmRevoke}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Revoke API key?</DialogTitle>
            <DialogDescription>The current key stops working immediately.</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <DialogClose asChild>
              <Button type="button" variant="ghost" size="sm" disabled={revoke.isPending}>
                Cancel
              </Button>
            </DialogClose>
            <Button type="button" variant="destructive" size="sm" disabled={revoke.isPending} onClick={() => void handleRevoke()}>
              Revoke
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  );
}
