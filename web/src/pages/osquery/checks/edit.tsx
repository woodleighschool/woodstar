import { Link, useNavigate, useParams } from "@tanstack/react-router";

import { LiveRunButton, SettingItem } from "@/components/queries/query-ui";
import { QueryGate } from "@/components/query-gate";
import { useCheck, useUpdateCheck } from "@/hooks/use-checks";
import { parseRouteID } from "@/lib/route-params";
import { CheckForm, checkFromDetail } from "@/pages/osquery/checks/fields";

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
      headerContext={
        <>
          <SettingItem label="Pass">
            <HostCount checkId={id} response="pass" value={check.passing_host_count} />
          </SettingItem>
          <SettingItem label="Fail">
            <HostCount checkId={id} response="fail" value={check.failing_host_count} />
          </SettingItem>
        </>
      }
      headerActions={<LiveRunButton to="/osquery/checks/$id/live" params={{ id: checkId }} />}
      onSubmit={async (value) => (await update.mutateAsync(value)).id}
      onSuccess={(savedID) => {
        if (savedID !== undefined) {
          void navigate({ to: "/osquery/checks/$id", params: { id: String(savedID) } });
        }
      }}
    />
  );
}

function HostCount({
  checkId,
  response,
  value,
}: {
  checkId: number;
  response: "pass" | "fail";
  value: number;
}) {
  return (
    <Link
      to="/osquery/checks/$id/results"
      params={{ id: String(checkId) }}
      search={{ response }}
      className="hover:underline"
    >
      {value}
    </Link>
  );
}
