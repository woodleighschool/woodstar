import { Copy, KeyRound, Loader2, Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import {
  useCreateSantaSyncToken,
  useDeleteSantaSyncToken,
  useSantaSyncTokens,
  type CreatedSantaSyncToken,
} from "@/hooks/use-santa";
import { formatRelative } from "@/lib/utils";

export function SantaSyncTokensPage() {
  const query = useSantaSyncTokens();
  const create = useCreateSantaSyncToken();
  const remove = useDeleteSantaSyncToken();
  const [created, setCreated] = useState<CreatedSantaSyncToken | null>(null);

  async function createToken() {
    const token = await create.mutateAsync();
    setCreated(token);
  }

  return (
    <PageShell>
      <PageHeader
        title="Santa sync tokens"
        description="Bearer tokens used by Santa clients during sync. Plaintext is only shown when a token is created."
        actions={
          <Button size="sm" disabled={create.isPending} onClick={() => void createToken()}>
            {create.isPending ? (
              <Loader2 data-icon="inline-start" className="animate-spin" />
            ) : (
              <Plus data-icon="inline-start" />
            )}
            New token
          </Button>
        }
      />

      {create.error ? (
        <Alert variant="destructive">
          <AlertTitle>Unable to create token</AlertTitle>
          <AlertDescription>{create.error.message}</AlertDescription>
        </Alert>
      ) : null}

      <SyncTokenTable query={query} deletingID={remove.variables ?? null} onDelete={(id) => remove.mutate(id)} />

      <Dialog open={created !== null} onOpenChange={(open) => !open && setCreated(null)}>
        <DialogContent className="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>New Santa sync token</DialogTitle>
          </DialogHeader>
          <div className="grid gap-3">
            <Input readOnly value={created?.value ?? ""} className="font-mono" />
            <p className="text-muted-foreground text-sm">Store this token now. Woodstar will not show it again.</p>
          </div>
          <DialogFooter>
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={() => {
                void navigator.clipboard.writeText(created?.value ?? "");
                toast.success("Token copied");
              }}
            >
              <Copy data-icon="inline-start" />
              Copy
            </Button>
            <Button type="button" size="sm" onClick={() => setCreated(null)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </PageShell>
  );
}

function SyncTokenTable({
  query,
  deletingID,
  onDelete,
}: {
  query: ReturnType<typeof useSantaSyncTokens>;
  deletingID: number | null;
  onDelete: (id: number) => void;
}) {
  if (query.error) {
    return (
      <Alert variant="destructive">
        <AlertTitle>Failed to load sync tokens</AlertTitle>
        <AlertDescription>{query.error.message}</AlertDescription>
        <Button variant="outline" size="sm" onClick={() => void query.refetch()} className="mt-2 w-fit">
          Retry
        </Button>
      </Alert>
    );
  }

  if (query.isLoading) {
    return (
      <div className="text-muted-foreground flex items-center gap-2 text-sm">
        <Loader2 className="size-4 animate-spin" /> Loading...
      </div>
    );
  }

  const rows = query.data ?? [];
  if (rows.length === 0) {
    return (
      <Empty>
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <KeyRound />
          </EmptyMedia>
          <EmptyTitle>No sync tokens</EmptyTitle>
          <EmptyDescription>Create a token before enrolling Santa clients.</EmptyDescription>
        </EmptyHeader>
      </Empty>
    );
  }

  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Hash</TableHead>
            <TableHead>Created</TableHead>
            <TableHead>Last used</TableHead>
            <TableHead className="w-12" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row) => (
            <TableRow key={row.id}>
              <TableCell className="max-w-[24rem] truncate font-mono text-xs">{row.value_hash}</TableCell>
              <TableCell className="text-muted-foreground" title={new Date(row.created_at).toLocaleString()}>
                {formatRelative(row.created_at)}
              </TableCell>
              <TableCell
                className="text-muted-foreground"
                title={row.last_used_at ? new Date(row.last_used_at).toLocaleString() : undefined}
              >
                {formatRelative(row.last_used_at)}
              </TableCell>
              <TableCell className="text-right">
                <Button
                  type="button"
                  size="icon"
                  variant="ghost"
                  disabled={deletingID === row.id}
                  onClick={() => onDelete(row.id)}
                >
                  {deletingID === row.id ? <Loader2 className="animate-spin" /> : <Trash2 />}
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
