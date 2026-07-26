import { toast } from "sonner";

import { ConfirmDialog } from "@components/confirm-dialog";
import type { OsqueryReport } from "@lib/api";

import { useDeleteReport } from "./queries";

export interface ReportDeleteDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  report: OsqueryReport | null;
  onDeleted?: () => void;
}

export function ReportDeleteDialog({
  open,
  onOpenChange,
  report,
  onDeleted,
}: ReportDeleteDialogProps) {
  const remove = useDeleteReport();

  async function handleConfirm() {
    if (!report) return;
    await remove.mutateAsync(report.id);
    onOpenChange(false);
    toast.success("Report deleted");
    onDeleted?.();
  }

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={(next) => {
        if (!next) remove.reset();
        onOpenChange(next);
      }}
      title="Delete Report"
      description={
        report
          ? `${report.name} will be permanently deleted.`
          : "This report will be permanently deleted."
      }
      confirmLabel="Delete"
      variant="destructive"
      pending={remove.isPending}
      onConfirm={() => void handleConfirm()}
    />
  );
}
