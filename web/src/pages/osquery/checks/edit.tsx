import { Link, useNavigate, useParams } from "@tanstack/react-router";

import { PageShell } from "@/components/layout/page-layout";
import { LiveRunButton, SettingItem } from "@/components/queries/query-ui";
import { QueryError } from "@/components/query-error";
import { useCheck, useUpdateCheck } from "@/hooks/use-checks";
import { CheckForm, checkFromDetail } from "@/pages/osquery/checks/fields";

export function CheckEditPage() {
  const navigate = useNavigate();
  const params = useParams({ strict: false });
  const checkId = params.checkId ?? "";
  const id = Number(checkId);
  const detail = useCheck(id);
  const update = useUpdateCheck(id);

  if (detail.error) {
    return (
      <PageShell>
        <QueryError title="Failed to load check" error={detail.error} onRetry={() => void detail.refetch()} />
      </PageShell>
    );
  }
  if (!detail.data) return null;

  const check = detail.data;
  return (
    <CheckForm
      key={check.id}
      initial={checkFromDetail(check)}
      submitLabel="Save"
      pending={update.isPending}
      error={update.error}
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
      headerActions={<LiveRunButton to="/osquery/checks/$checkId/live" params={{ checkId }} />}
      onSubmit={async (value) => {
        const saved = await update.mutateAsync(value);
        void navigate({ to: "/osquery/checks/$checkId", params: { checkId: String(saved.id) } });
      }}
    />
  );
}

function HostCount({ checkId, response, value }: { checkId: number; response: "pass" | "fail"; value: number }) {
  return (
    <Link to="/hosts" search={{ check_id: checkId, check_response: response }} className="hover:underline">
      {value}
    </Link>
  );
}
