import { useNavigate, useParams } from "@tanstack/react-router";

import { QueryGate } from "@components/query-gate";
import { parseRouteID } from "@lib/route-params";

import { CheckForm, checkFromDetail } from "./fields";
import { useCheck, useUpdateCheck } from "./queries";

export function CheckEditPage() {
  const navigate = useNavigate();
  const params = useParams({ strict: false });
  const checkId = params.id ?? "";
  const id = parseRouteID(checkId);
  const detail = useCheck(id);
  const update = useUpdateCheck(id);

  if (id === null) {
    return (
      <QueryGate title="Failed to load check" error={{ message: "Check route is invalid." }} />
    );
  }

  if (detail.error || !detail.data) {
    return (
      <QueryGate
        title="Failed to load check"
        error={detail.error}
        onRetry={() => void detail.refetch()}
      />
    );
  }

  const check = detail.data;
  return (
    <CheckForm
      key={check.id}
      initial={checkFromDetail(check)}
      title="Edit Check"
      submitLabel="Save"
      onCancel={() =>
        void navigate({
          to: "/osquery/checks/$id",
          params: { id: String(check.id) },
        })
      }
      onSubmit={async (value) => (await update.mutateAsync(value)).id}
      onSuccess={(savedID) => {
        if (savedID !== undefined) {
          void navigate({
            to: "/osquery/checks/$id",
            params: { id: String(savedID) },
          });
        }
      }}
    />
  );
}
