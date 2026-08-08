import { useNavigate, useParams } from "@tanstack/react-router";
import { Pencil, Trash2 } from "lucide-react";
import { useState } from "react";

import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { QueryGate } from "@components/query-gate";
import { LabelTargetDetails } from "@components/targeting/target-details";
import { Button } from "@components/ui/button";
import { useAuth } from "@features/auth/queries";
import { parseRouteID } from "@lib/route-params";
import { formatRelative } from "@lib/utils";

import { RuleDeleteDialog } from "./delete-dialog";
import { POLICIES, ruleTypeLabel } from "./metadata";
import { useSantaRule } from "./queries";

export function RuleDetailPage() {
  const { id: configurationID, ruleId } = useParams({
    from: "/_authenticated/santa/configurations/$id/rules/$ruleId",
  });
  const navigate = useNavigate();
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const id = parseRouteID(ruleId);
  const query = useSantaRule(id);
  const [deleteOpen, setDeleteOpen] = useState(false);

  if (id === null) {
    return <QueryGate title="Failed to load rule" error={{ message: "Rule route is invalid." }} />;
  }

  if (query.error || !query.data) {
    return (
      <QueryGate
        title="Failed to load rule"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }

  const rule = query.data;

  return (
    <PageShell className="gap-6">
      <PageHeader
        title="Rule Details"
        meta={`Edited ${formatRelative(rule.updated_at)}`}
        actions={
          isAdmin ? (
            <>
              <Button
                size="sm"
                render={
                  <Link
                    to="/santa/configurations/$id/rules/$ruleId/edit"
                    params={{ id: configurationID, ruleId: String(rule.id) }}
                  />
                }
                nativeButton={false}
              >
                <Pencil data-icon="inline-start" />
                Edit
              </Button>
              <Button
                type="button"
                variant="destructive"
                size="sm"
                onClick={() => setDeleteOpen(true)}
              >
                <Trash2 data-icon="inline-start" />
                Delete
              </Button>
            </>
          ) : null
        }
      />

      <KeyValueSection title="Overview">
        <KeyValueRow label="Name" value={rule.name} />
        <KeyValueRow label="Description" value={rule.description} />
        <KeyValueRow label="Rule Type" value={ruleTypeLabel(rule.rule_type)} />
        <KeyValueRow label="Identifier" value={rule.identifier || "-"} />
        <KeyValueRow label="Policy" value={POLICIES[rule.policy].name} />
        {rule.policy === "cel" ? (
          <KeyValueRow label="CEL Expression" value={rule.cel_expression} />
        ) : null}
        <KeyValueRow label="Custom URL" value={rule.custom_url} />
        <KeyValueRow label="Custom Message" value={rule.custom_message} />
      </KeyValueSection>

      <LabelTargetDetails targets={rule.targets} />

      <RuleDeleteDialog
        rule={rule}
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        onDeleted={() =>
          void navigate({
            to: "/santa/configurations/$id/rules",
            params: { id: configurationID },
          })
        }
      />
    </PageShell>
  );
}
