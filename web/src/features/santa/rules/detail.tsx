import { useNavigate, useParams } from "@tanstack/react-router";
import { Pencil, Trash2 } from "lucide-react";
import { useState } from "react";

import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { QueryGate } from "@components/query-gate";
import { TargetBadge, TargetDetails } from "@components/targeting/target-details";
import { Button } from "@components/ui/button";
import { useCan } from "@features/authz/access";
import { useLabelNameMap } from "@features/labels/components/label-ref-list";
import type { SantaRule } from "@lib/api";
import { parseRouteID } from "@lib/route-params";

import { RuleDeleteDialog } from "./delete-dialog";
import { POLICIES, ruleTypeLabel } from "./metadata";
import { useSantaRule } from "./queries";

export function RuleDetailPage() {
  const { id: ruleID } = useParams({
    from: "/_authenticated/santa/rules/$id",
  });
  const navigate = useNavigate();
  const canEdit = useCan({ resource: "santa.rules", access: "edit" });
  const id = parseRouteID(ruleID);
  const query = useSantaRule(id);
  const [deleteOpen, setDeleteOpen] = useState(false);

  if (id === null) {
    return <QueryGate title="Failed to Load Rule" error={{ message: "Rule route is invalid." }} />;
  }

  if (query.error || !query.data) {
    return (
      <QueryGate
        title="Failed to Load Rule"
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
        actions={
          canEdit ? (
            <>
              <Button
                size="sm"
                render={<Link to="/santa/rules/$id/edit" params={{ id: String(rule.id) }} />}
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
        <KeyValueRow label="Custom URL" value={rule.custom_url} />
        <KeyValueRow label="Custom Message" value={rule.custom_message} />
      </KeyValueSection>

      <RuleTargets rule={rule} />

      <RuleDeleteDialog
        rule={rule}
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        onDeleted={() => void navigate({ to: "/santa/rules" })}
      />
    </PageShell>
  );
}

function RuleTargets({ rule }: { rule: SantaRule }) {
  const labelsByID = useLabelNameMap();

  return (
    <TargetDetails
      include={
        rule.targets.include.length > 0 ? (
          <div className="flex flex-wrap gap-1.5">
            {rule.targets.include.map((target) => {
              const label = labelsByID.get(target.label_id) ?? `Label ${target.label_id}`;
              return (
                <TargetBadge
                  key={`${target.label_id}:${target.policy}`}
                  labelID={target.label_id}
                  label={label}
                  details={[
                    { label: "Policy", value: POLICIES[target.policy].name },
                    {
                      label: "CEL Expression",
                      value: (
                        <span className="font-mono text-xs break-all">
                          {target.cel_expression || "-"}
                        </span>
                      ),
                    },
                  ]}
                />
              );
            })}
          </div>
        ) : (
          "-"
        )
      }
      excludeLabelIDs={rule.targets.exclude.map((target) => target.label_id)}
    />
  );
}
