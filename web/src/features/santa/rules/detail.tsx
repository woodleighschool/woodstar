import { useNavigate, useParams } from "@tanstack/react-router";
import { Pencil, Trash2 } from "lucide-react";
import { useState } from "react";

import { EnumBadge } from "@components/enum-badge";
import { KeyValueGrid, KeyValueItem } from "@components/key-value";
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
import { POLICIES, RULE_TYPES } from "./metadata";
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
        context={<EnumBadge value={rule.rule_type} metadata={RULE_TYPES} />}
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

      <Card>
        <CardContent>
          <KeyValueGrid>
            <KeyValueItem label="Name" value={rule.name} />
            <KeyValueItem
              label="Identifier"
              value={<PathText value={rule.identifier} />}
              className="sm:col-span-2"
            />
            <KeyValueItem label="Custom URL" value={rule.custom_url} className="sm:col-span-2" />
            <KeyValueItem
              label="Custom Message"
              value={rule.custom_message}
              className="sm:col-span-2"
            />
            <KeyValueItem label="Created" value={formatRelative(rule.created_at)} />
            <KeyValueItem label="Updated" value={formatRelative(rule.updated_at)} />
          </KeyValueGrid>
        </CardContent>
      </Card>

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
      <CardContent className="space-y-5">
        <div className="space-y-2">
          <h3 className="text-xs font-semibold text-muted-foreground">Include</h3>
          {rule.targets.include.length ? (
            <div className="overflow-x-auto rounded-md border">
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
                      <TableCell>
                        <EnumBadge value={target.policy} metadata={POLICIES} />
                      </TableCell>
                      <TableCell>
                        {target.cel_expression ? (
                          <code className="font-mono text-xs">{target.cel_expression}</code>
                        ) : (
                          <span className="text-muted-foreground">-</span>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          ) : (
            <PanelEmptyState>No included labels.</PanelEmptyState>
          )}
        </div>

        <KeyValueGrid>
          <KeyValueItem
            label="Exclude"
            value={
              <LabelRefList labelIDs={rule.targets.exclude.map((target) => target.label_id)} />
            }
          />
        </KeyValueGrid>
      </CardContent>
    </Card>
  );
}
