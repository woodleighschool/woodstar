import { useNavigate, useParams } from "@tanstack/react-router";
import { Pencil, Trash2 } from "lucide-react";
import { useState } from "react";

import { TableSurface } from "@components/data-table/table-surface";
import { KeyValueRow, KeyValueRows, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { PathText } from "@components/path-text";
import { QueryGate } from "@components/query-gate";
import { Button } from "@components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@components/ui/table";
import { useAuth } from "@features/auth/queries";
import { LabelRefList, useLabelNameMap } from "@features/labels/components/label-ref-list";
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
        description={rule.description || undefined}
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
        <KeyValueRow label="Rule Type" value={ruleTypeLabel(rule.rule_type)} />
        <KeyValueRow label="Identifier" value={<PathText value={rule.identifier} />} />
        <KeyValueRow label="Custom URL" value={rule.custom_url} />
        <KeyValueRow label="Custom Message" value={rule.custom_message} />
      </KeyValueSection>

      <RuleTargetsCard rule={rule} />

      <RuleDeleteDialog
        rule={rule}
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        onDeleted={() => void navigate({ to: "/santa/rules" })}
      />
    </PageShell>
  );
}

function RuleTargetsCard({ rule }: { rule: SantaRule }) {
  const labelsByID = useLabelNameMap();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Targets</CardTitle>
      </CardHeader>
      <CardContent className="space-y-5 px-0">
        <div className="space-y-2">
          <h3 className="px-4 text-xs font-semibold text-muted-foreground">Include</h3>
          {rule.targets.include.length ? (
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
                      <TableCell>
                        {target.cel_expression ? (
                          <span className="wrap-break-word">{target.cel_expression}</span>
                        ) : (
                          <span className="text-muted-foreground">-</span>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableSurface>
          ) : (
            <PanelEmptyState>No included labels.</PanelEmptyState>
          )}
        </div>

        <KeyValueRows className="px-4">
          <KeyValueRow
            label="Exclude"
            value={
              <LabelRefList labelIDs={rule.targets.exclude.map((target) => target.label_id)} />
            }
          />
        </KeyValueRows>
      </CardContent>
    </Card>
  );
}
