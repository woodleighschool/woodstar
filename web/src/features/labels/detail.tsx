import { useNavigate, useParams } from "@tanstack/react-router";
import { Pencil, Trash2 } from "lucide-react";
import { useState } from "react";

import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { QueryGate } from "@components/query-gate";
import { Button } from "@components/ui/button";
import { Separator } from "@components/ui/separator";
import { useAuth } from "@features/auth/queries";
import { useGroups } from "@features/directory/groups/queries";
import { useUsers } from "@features/directory/users/queries";
import { labelDerivedAttributeSelectorLabel, labelMembershipLabel } from "@features/labels/model";
import type { Label } from "@lib/api";
import { MAX_PAGE_SIZE } from "@lib/pagination";
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
        meta={`Edited ${formatRelative(label.updated_at)}`}
        actions={
          isAdmin && mutable ? (
            <>
              <Button
                variant="outline"
                size="sm"
                render={<Link to="/labels/$id/edit" params={{ id: String(label.id) }} />}
                nativeButton={false}
              >
                <Pencil data-icon="inline-start" />
                Edit
              </Button>
              <Button type="button" variant="outline" size="sm" onClick={() => setDeleteOpen(true)}>
                <Trash2 data-icon="inline-start" />
                Delete
              </Button>
            </>
          ) : null
        }
      />

      <KeyValueSection title="Overview">
        <KeyValueRow label="Name" value={label.name} />
        <KeyValueRow label="Description" value={label.description} />
        <KeyValueRow label="Membership" value={labelMembershipLabel(label.label_membership_type)} />
        <KeyValueRow label="Type" value={mutable ? "Regular" : "Built-In"} />
        <KeyValueRow
          label="Hosts"
          value={
            <Link to="/hosts" search={{ label_id: label.id }} className="font-medium">
              {formatHostCount(label.hosts_count)}
            </Link>
          }
        />
      </KeyValueSection>

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
        <section className="flex min-w-0 flex-col gap-3">
          <h2 className="text-base/snug font-medium">Query</h2>
          <Separator />
          <CodeBlock value={label.query} />
        </section>
      );
    case "manual":
      return null;
    case "derived":
      return (
        <KeyValueSection title="Criteria">
          <KeyValueRow
            label="Attribute"
            value={
              label.criteria
                ? labelDerivedAttributeSelectorLabel(label.criteria.attribute)
                : undefined
            }
          />
          <KeyValueRow
            label="Values"
            value={
              label.criteria ? (
                <CriteriaValues
                  attribute={label.criteria.attribute}
                  values={label.criteria.values}
                />
              ) : null
            }
          />
        </KeyValueSection>
      );
  }

  return null;
}

function CodeBlock({ value }: { value: string | undefined }) {
  if (!value) return <span className="text-muted-foreground">-</span>;
  return (
    <pre className="overflow-x-auto border-y bg-muted/20 px-3 py-2.5 font-mono text-xs">
      <code>{value}</code>
    </pre>
  );
}

function CriteriaValues({
  attribute,
  values,
}: {
  attribute: "user_department" | "directory_group" | "user";
  values: readonly string[];
}) {
  if (attribute === "directory_group") return <GroupValues values={values} />;
  if (attribute === "user") return <UserValues values={values} />;
  return <PlainValues values={values} />;
}

function GroupValues({ values }: { values: readonly string[] }) {
  const query = useGroups({ values: [...values], per_page: MAX_PAGE_SIZE });
  const names = new Map(
    (query.data?.items ?? []).map((group) => [group.external_id, group.display_name]),
  );
  return <PlainValues values={values.map((value) => names.get(value) ?? value)} />;
}

function UserValues({ values }: { values: readonly string[] }) {
  const query = useUsers({ values: [...values], per_page: MAX_PAGE_SIZE });
  const users = new Map((query.data?.items ?? []).map((user) => [String(user.id), user]));

  if (values.length === 0) return <span className="text-muted-foreground">-</span>;
  return (
    <div className="flex flex-wrap gap-x-3 gap-y-1">
      {values.map((value) => {
        const user = users.get(value);
        return user ? (
          <Link key={value} to="/directory/users/$id" params={{ id: value }}>
            {user.name}
          </Link>
        ) : (
          <span key={value}>{value}</span>
        );
      })}
    </div>
  );
}

function PlainValues({ values }: { values: readonly string[] }) {
  if (values.length === 0) return <span className="text-muted-foreground">-</span>;
  return (
    <div className="flex flex-wrap gap-x-3 gap-y-1">
      {values.map((value) => (
        <span key={value}>{value}</span>
      ))}
    </div>
  );
}

function formatHostCount(count: number) {
  return `${count} ${count === 1 ? "host" : "hosts"}`;
}
