import type { Table as TanStackTable } from "@tanstack/react-table";
import { Trash2 } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";

import { BulkDeleteDialog } from "@/components/bulk-delete-dialog";
import { Button } from "@/components/ui/button";

interface BulkDeleteMutation {
  mutate: (ids: number[], options?: { onSuccess?: () => void }) => void;
  reset: () => void;
  isPending: boolean;
}

export function BulkDeleteActionBar<TRow extends { id: number }>({
  table,
  useBulkDelete,
  noun,
  pluralNoun,
  description,
}: {
  table: TanStackTable<TRow>;
  useBulkDelete: () => BulkDeleteMutation;
  noun: string;
  pluralNoun?: string;
  description?: string;
}) {
  const rows = table.getFilteredSelectedRowModel().rows;
  const ids = useMemo(() => rows.map((row) => Number(row.original.id)), [rows]);
  const [open, setOpen] = useState(false);
  const bulkDelete = useBulkDelete();
  const plural = pluralNoun ?? `${noun}s`;

  const onConfirm = () => {
    const count = ids.length;
    bulkDelete.mutate(ids, {
      onSuccess: () => {
        toast.success(`Deleted ${count} ${count === 1 ? noun : plural}`);
        table.toggleAllRowsSelected(false);
        setOpen(false);
      },
    });
  };

  return (
    <div className="flex items-center gap-3 rounded-md border bg-background p-1 pl-3 shadow-sm">
      <span className="text-sm text-muted-foreground">{ids.length} selected</span>
      <Button
        variant="destructive"
        size="sm"
        onClick={() => setOpen(true)}
        disabled={bulkDelete.isPending}
      >
        <Trash2 />
        Delete
      </Button>
      <BulkDeleteDialog
        open={open}
        onOpenChange={(next) => {
          if (!next) bulkDelete.reset();
          setOpen(next);
        }}
        count={ids.length}
        noun={noun}
        pluralNoun={pluralNoun}
        description={description}
        pending={bulkDelete.isPending}
        onConfirm={onConfirm}
      />
    </div>
  );
}
