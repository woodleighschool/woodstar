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
      onCancel={() => void navigate({ to: "/osquery/reports" })}
      onSubmit={async (value) => (await create.mutateAsync(value)).id}
      onSuccess={(id) => {
        if (id !== undefined) {
          void navigate({ to: "/osquery/reports/$reportId", params: { reportId: String(id) } });
        }
      }}
    />
  );
}
