import { useNavigate } from "@tanstack/react-router";

import { useCreateReport } from "@/hooks/use-reports";
import { emptyReport, ReportForm } from "@/pages/osquery/reports/fields";

export function ReportCreatePage() {
  const navigate = useNavigate();
  const create = useCreateReport();

  return (
    <ReportForm
      initial={emptyReport}
      title="New Report"
      submitLabel="Create"
      pending={create.isPending}
      error={create.error}
      onCancel={() => void navigate({ to: "/osquery/reports" })}
      onSubmit={async (value) => {
        const saved = await create.mutateAsync(value);
        void navigate({ to: "/osquery/reports/$reportId", params: { reportId: String(saved.id) } });
      }}
    />
  );
}
