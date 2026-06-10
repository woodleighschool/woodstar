import { useNavigate, useParams } from "@tanstack/react-router";

import { PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { useLabel, useUpdateLabel } from "@/hooks/use-labels";
import { LabelForm, labelFromDetail } from "@/pages/labels/fields";

export function LabelEditPage() {
  const navigate = useNavigate();
  const params = useParams({ strict: false });
  const labelId = params.labelId ?? "";
  const id = Number(labelId);
  const detail = useLabel(id);
  const update = useUpdateLabel(id);

  if (detail.error) {
    return (
      <PageShell>
        <QueryError title="Failed to load label" error={detail.error} onRetry={() => void detail.refetch()} />
      </PageShell>
    );
  }
  if (!detail.data) return null;

  const label = detail.data;
  if (label.label_type === "builtin") {
    return (
      <PageShell>
        <Alert>
          <AlertTitle>Built-In Label</AlertTitle>
          <AlertDescription>Built-in labels are managed by Woodstar and cannot be edited.</AlertDescription>
        </Alert>
      </PageShell>
    );
  }

  return (
    <LabelForm
      key={label.id}
      initial={labelFromDetail(label)}
      title="Edit Label"
      submitLabel="Save"
      pending={update.isPending}
      error={update.error}
      onCancel={() => void navigate({ to: "/labels" })}
      onSubmit={async (body) => {
        await update.mutateAsync(body);
        void navigate({ to: "/labels" });
      }}
    />
  );
}
