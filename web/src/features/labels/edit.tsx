import { useNavigate, useParams } from "@tanstack/react-router";

import { PageShell } from "@components/layout/page-layout";
import { QueryGate } from "@components/query-gate";
import { Alert, AlertDescription, AlertTitle } from "@components/ui/alert";
import { LabelForm, labelFromDetail } from "@features/labels/fields";
import { useLabel, useUpdateLabel } from "@features/labels/queries";
import { parseRouteID } from "@lib/route-params";

export function LabelEditPage() {
  const navigate = useNavigate();
  const params = useParams({ strict: false });
  const labelId = params.id ?? "";
  const id = parseRouteID(labelId);
  const detail = useLabel(id);
  const update = useUpdateLabel(id);

  if (id === null) {
    return (
      <QueryGate title="Failed to Load Label" error={{ message: "Label route is invalid." }} />
    );
  }

  if (detail.error || !detail.data) {
    return (
      <QueryGate
        title="Failed to Load Label"
        error={detail.error}
        onRetry={() => void detail.refetch()}
      />
    );
  }

  const label = detail.data;
  if (label.label_type === "builtin") {
    return (
      <PageShell>
        <Alert>
          <AlertTitle>Built-In Label</AlertTitle>
          <AlertDescription>
            Built-in labels are managed by Woodstar and cannot be edited.
          </AlertDescription>
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
      onCancel={() =>
        void navigate({
          to: "/labels/$id",
          params: { id: String(label.id) },
        })
      }
      onSubmit={async (body) => (await update.mutateAsync(body)).id}
      onSuccess={() =>
        void navigate({
          to: "/labels/$id",
          params: { id: String(label.id) },
        })
      }
    />
  );
}
