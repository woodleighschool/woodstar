import type { ColumnDef } from "@tanstack/react-table";
import { Copy, KeyRound, Loader2, Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
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
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import {
  useCreateSantaSyncToken,
  useDeleteSantaSyncToken,
  useSantaSyncTokens,
  type SantaSyncToken,
} from "@/hooks/use-santa";
import { formatRelative } from "@/lib/utils";

export function SantaSyncTokensPage() {
  const query = useSantaSyncTokens();
  const create = useCreateSantaSyncToken();
  const remove = useDeleteSantaSyncToken();
  const [created, setCreated] = useState<SantaSyncToken | null>(null);
  const [deleting, setDeleting] = useState<SantaSyncToken | null>(null);

  async function createToken() {
    const token = await create.mutateAsync();
    setCreated(token);
  }

  return (
    <PageShell>
      <PageHeader
        title="Santa sync tokens"
        description="Bearer tokens used by Santa clients during sync."
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

      <SyncTokenTable query={query} deletingID={remove.variables ?? null} onDelete={setDeleting} />

      <Dialog open={created !== null} onOpenChange={(open) => !open && setCreated(null)}>
        <DialogContent className="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>New Santa sync token</DialogTitle>
          </DialogHeader>
          <div className="grid gap-3">
            <Input readOnly value={created?.value ?? ""} className="font-mono" />
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

      <SyncTokenDeleteDialog
        token={deleting}
        open={deleting !== null}
        pending={remove.isPending}
        error={remove.error?.message}
        onOpenChange={(open) => {
          if (!open) {
            remove.reset();
            setDeleting(null);
          }
        }}
        onConfirm={async () => {
          if (!deleting) return;
          await remove.mutateAsync(deleting.id);
          setDeleting(null);
        }}
      />
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
  onDelete: (token: SantaSyncToken) => void;
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

  const rows = query.data ?? [];
  const columns: ColumnDef<SantaSyncToken>[] = [
    {
      id: "value",
      accessorKey: "value",
      header: "Token",
      cell: ({ row }) => <span className="block max-w-[28rem] truncate font-mono text-xs">{row.original.value}</span>,
    },
    {
      id: "created_at",
      accessorKey: "created_at",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Created" />,
      cell: ({ row }) => (
        <span className="text-muted-foreground" title={new Date(row.original.created_at).toLocaleString()}>
          {formatRelative(row.original.created_at)}
        </span>
      ),
    },
    {
      id: "actions",
      header: () => null,
      enableSorting: false,
      cell: ({ row }) => (
        <Button
          type="button"
          size="icon"
          variant="ghost"
          disabled={deletingID === row.original.id}
          onClick={() => onDelete(row.original)}
        >
          {deletingID === row.original.id ? <Loader2 className="animate-spin" /> : <Trash2 />}
        </Button>
      ),
      meta: { headClassName: "w-12" },
    },
  ];

  return (
    <DataTable
      columns={columns}
      data={rows}
      totalCount={rows.length}
      page={1}
      perPage={rows.length || 50}
      sort={{}}
      onPageChange={() => undefined}
      onPerPageChange={() => undefined}
      onSortChange={() => undefined}
      isLoading={query.isLoading}
      clientSort
      empty={
        <Empty>
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <KeyRound />
            </EmptyMedia>
            <EmptyTitle>No sync tokens</EmptyTitle>
            <EmptyDescription>Create a token before enrolling Santa clients.</EmptyDescription>
          </EmptyHeader>
        </Empty>
      }
    />
  );
}

function SyncTokenDeleteDialog({
  token,
  open,
  pending,
  error,
  onOpenChange,
  onConfirm,
}: {
  token: SantaSyncToken | null;
  open: boolean;
  pending: boolean;
  error?: string;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => Promise<void>;
}) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete Santa sync token?</AlertDialogTitle>
          <AlertDialogDescription>
            {token
              ? `Token created ${formatRelative(token.created_at)} will stop authenticating Santa clients.`
              : "This token will stop authenticating Santa clients."}
          </AlertDialogDescription>
        </AlertDialogHeader>
        {error ? <p className="text-sm text-destructive">{error}</p> : null}
        <AlertDialogFooter>
          <AlertDialogCancel variant="ghost" size="sm" disabled={pending}>
            Cancel
          </AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            size="sm"
            disabled={pending}
            onClick={(event) => {
              event.preventDefault();
              void onConfirm();
            }}
          >
            Delete
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
