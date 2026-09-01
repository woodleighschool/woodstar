import { Trash2 } from "lucide-react";
import { useMemo, useState } from "react";

import { BulkDeleteDialog } from "@components/bulk-delete-dialog";
import type { DataTableInstance } from "@components/data-table/types";
import {
  ActionBar,
  ActionBarGroup,
  ActionBarItem,
  ActionBarSelection,
  ActionBarSeparator,
} from "@components/ui/action-bar";
import { toast } from "@components/ui/toast";
import { countLabel } from "@lib/utils";

interface BulkDeleteMutation {
  mutate: (ids: number[], options?: { onSuccess?: () => void }) => void;
  reset: () => void;
  isPending: boolean;
}

export function BulkDeleteActionBar<TRow extends { id: number }>({
  table,
  bulkDelete,
  noun,
  pluralNoun,
  description,
}: {
  table: DataTableInstance<TRow>;
  bulkDelete: BulkDeleteMutation;
  noun: string;
  pluralNoun?: string;
  description?: string;
}) {
  const rows = table.getFilteredSelectedRowModel().rows;
  const ids = useMemo(() => rows.map((row) => row.original.id), [rows]);
  const [open, setOpen] = useState(false);
  const onConfirm = () => {
    const count = ids.length;
    bulkDelete.mutate(ids, {
      onSuccess: () => {
        toast.add({
          title: `Deleted ${countLabel(count, noun, pluralNoun)}`,
          type: "success",
        });
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
