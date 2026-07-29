import { useNavigate, useParams } from "@tanstack/react-router";
import { Pencil, Trash2 } from "lucide-react";
import { useState } from "react";

import { TableSurface } from "@components/data-table/table-surface";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { PathText } from "@components/path-text";
import { QueryGate } from "@components/query-gate";
import { TargetDetails } from "@components/targeting/target-details";
import { Button } from "@components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@components/ui/table";
import { useAuth } from "@features/auth/queries";
import { useLabelNameMap } from "@features/labels/components/label-ref-list";
import type { SantaRule } from "@lib/api";
import { parseRouteID } from "@lib/route-params";
import { formatRelative } from "@lib/utils";

import { RuleDeleteDialog } from "./delete-dialog";
import { POLICIES, ruleTypeLabel } from "./metadata";
import { useSantaRule } from "./queries";

export function RuleDetailPage() {
  const { id: ruleID } = useParams({
    from: "/_authenticated/santa/rules/$id",
  });
  const navigate = useNavigate();
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const id = parseRouteID(ruleID);
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
        <KeyValueRow label="Identifier" value={<PathText value={rule.identifier} />} />
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
          <TableSurface variant="embedded">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Label</TableHead>
                  <TableHead>Policy</TableHead>
                  <TableHead>CEL Expression</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rule.targets.include.map((target) => (
                  <TableRow key={`${target.label_id}:${target.policy}`}>
                    <TableCell>
                      <Link
                        to="/labels/$id"
                        params={{ id: String(target.label_id) }}
                        className="font-medium"
                      >
                        {labelsByID.get(target.label_id) ?? `Label ${target.label_id}`}
                      </Link>
                    </TableCell>
                    <TableCell>{POLICIES[target.policy].name}</TableCell>
                    <TableCell>{target.cel_expression || "-"}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableSurface>
        ) : (
          "-"
        )
      }
      excludeLabelIDs={rule.targets.exclude.map((target) => target.label_id)}
    />
  );
}
