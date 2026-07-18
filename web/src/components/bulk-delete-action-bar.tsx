import type { Table as TanStackTable } from "@tanstack/react-table";
import { Trash2 } from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";

import { BulkDeleteDialog } from "@/components/bulk-delete-dialog";
import {
  ActionBar,
  ActionBarGroup,
  ActionBarItem,
  ActionBarSelection,
  ActionBarSeparator,
} from "@/components/ui/action-bar";

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
    <>
      <ActionBar
        open={ids.length > 0}
        onOpenChange={(next) => {
          if (!next) table.toggleAllRowsSelected(false);
        }}
      >
        <ActionBarSelection>{ids.length} selected</ActionBarSelection>
        <ActionBarSeparator />
        <ActionBarGroup>
          <ActionBarItem
            variant="destructive"
            onSelect={(event) => {
              event.preventDefault();
              setOpen(true);
            }}
            disabled={bulkDelete.isPending}
          >
            <Trash2 data-icon="inline-start" />
            Delete
          </ActionBarItem>
        </ActionBarGroup>
      </ActionBar>
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
    </>
  );
}
