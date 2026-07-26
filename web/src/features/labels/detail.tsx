import { useNavigate, useParams } from "@tanstack/react-router";
import { Pencil, Trash2 } from "lucide-react";
import { useState } from "react";

import { KeyValueGrid, KeyValueItem } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { QueryGate } from "@components/query-gate";
import { Button } from "@components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@components/ui/card";
import { useAuth } from "@features/auth/queries";
import { labelDerivedAttributeSelectorLabel, labelMembershipLabel } from "@features/labels/model";
import type { Label } from "@lib/api";
import { parseRouteID } from "@lib/route-params";
import { formatRelative } from "@lib/utils";

import { LabelDeleteDialog } from "./delete-dialog";
import { useLabel } from "./queries";

export function LabelDetailPage() {
  const { id: labelID } = useParams({
    from: "/_authenticated/labels/$id",
  });
  const navigate = useNavigate();
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const id = parseRouteID(labelID);
  const query = useLabel(id);
  const [deleteOpen, setDeleteOpen] = useState(false);

  if (id === null) {
    return (
      <QueryGate title="Failed to load label" error={{ message: "Label route is invalid." }} />
    );
  }

  if (query.error || !query.data) {
    return (
      <QueryGate
        title="Failed to load label"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }

  const label = query.data;
  const mutable = label.label_type === "regular";

  return (
    <PageShell className="gap-6">
      <PageHeader
        title="Label Details"
        description={label.description || undefined}
        actions={
          isAdmin && mutable ? (
            <>
              <Button
                size="sm"
                render={<Link to="/labels/$id/edit" params={{ id: String(label.id) }} />}
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
            <KeyValueItem label="Name" value={label.name} />
            <KeyValueItem
              label="Membership"
              value={labelMembershipLabel(label.label_membership_type)}
            />
            <KeyValueItem label="Type" value={mutable ? "Regular" : "Built-In"} />
            <KeyValueItem
              label="Hosts"
              value={
                <Link to="/hosts" search={{ label_id: label.id }} className="font-medium">
                  {formatHostCount(label.hosts_count)}
                </Link>
              }
            />
            <KeyValueItem label="Created" value={formatRelative(label.created_at)} />
            <KeyValueItem label="Updated" value={formatRelative(label.updated_at)} />
          </KeyValueGrid>
        </CardContent>
      </Card>

      <MembershipCard label={label} />

      <LabelDeleteDialog
        label={label}
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        onDeleted={() => void navigate({ to: "/labels" })}
      />
    </PageShell>
  );
}

function MembershipCard({ label }: { label: Label }) {
  switch (label.label_membership_type) {
    case "dynamic":
      return (
        <Card>
          <CardHeader>
            <CardTitle>Query</CardTitle>
          </CardHeader>
          <CardContent>
            <CodeBlock value={label.query} />
          </CardContent>
        </Card>
      );
    case "manual":
      return (
        <Card>
          <CardHeader>
            <CardTitle>Hosts</CardTitle>
          </CardHeader>
          <CardContent>
            <HostLinks hostIDs={label.host_ids ?? []} />
          </CardContent>
        </Card>
      );
    case "derived":
      return (
        <Card>
          <CardHeader>
            <CardTitle>Criteria</CardTitle>
          </CardHeader>
          <CardContent>
            <KeyValueGrid>
              <KeyValueItem
                label="Attribute"
                value={
                  label.criteria
                    ? labelDerivedAttributeSelectorLabel(label.criteria.attribute)
                    : undefined
                }
              />
              <KeyValueItem
                label="Values"
                value={<ValueList values={label.criteria?.values ?? []} />}
                className="sm:col-span-2"
              />
            </KeyValueGrid>
          </CardContent>
        </Card>
      );
  }

  return null;
}

function CodeBlock({ value }: { value: string | undefined }) {
  if (!value) return <span className="text-muted-foreground">-</span>;
  return (
    <pre className="overflow-x-auto rounded-md border bg-muted/30 p-3 font-mono text-xs">
      <code>{value}</code>
    </pre>
  );
}

function HostLinks({ hostIDs }: { hostIDs: readonly number[] }) {
  if (hostIDs.length === 0) return <span className="text-muted-foreground">-</span>;
  return (
    <div className="flex flex-wrap gap-1.5">
      {hostIDs.map((hostID) => (
        <Button
          key={hostID}
          size="xs"
          variant="outline"
          className="font-normal"
          render={<Link to="/hosts/$id" params={{ id: String(hostID) }} />}
          nativeButton={false}
        >
          Host {hostID}
        </Button>
      ))}
    </div>
  );
}

function ValueList({ values }: { values: readonly string[] }) {
  if (values.length === 0) return <span className="text-muted-foreground">-</span>;
  return (
    <div className="flex flex-wrap gap-1.5">
      {values.map((value) => (
        <code key={value} className="rounded-sm bg-muted px-1.5 py-0.5 font-mono text-xs">
          {value}
        </code>
      ))}
    </div>
  );
}

function formatHostCount(count: number) {
  return `${count} ${count === 1 ? "host" : "hosts"}`;
}
