import { useNavigate } from "@tanstack/react-router";

import { emptyReport, ReportForm } from "./fields";
import { useCreateReport } from "./queries";

export function ReportCreatePage() {
  const navigate = useNavigate();
  const create = useCreateReport();

  return (
    <ReportForm
      initial={emptyReport}
      title="Create Report"
      submitLabel="Create"
      onCancel={() => void navigate({ to: "/osquery/reports" })}
      onSubmit={async (value) => (await create.mutateAsync(value)).id}
      onSuccess={(id) => {
        if (id !== undefined) {
          void navigate({
            to: "/osquery/reports/$id",
            params: { id: String(id) },
          });
        }
      }}
    />
  );
}
