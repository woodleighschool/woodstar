import { useNavigate, useParams } from "@tanstack/react-router";
import { Pencil, Trash2 } from "lucide-react";
import { useState } from "react";

import { SQLEditor } from "@components/editor/sql-editor";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link, TextLink } from "@components/link";
import { QueryGate } from "@components/query-gate";
import { TokenList } from "@components/token-list";
import { Badge } from "@components/ui/badge";
import { Button } from "@components/ui/button";
import { Separator } from "@components/ui/separator";
import { useAuth } from "@features/auth/queries";
import { useGroups } from "@features/directory/groups/queries";
import { useUsers } from "@features/directory/users/queries";
import { labelDerivedAttributeSelectorLabel, labelMembershipLabel } from "@features/labels/model";
import type { Label } from "@lib/api";
import { MAX_PAGE_SIZE } from "@lib/pagination";
import { parseRouteID } from "@lib/route-params";
import { countLabel, formatRelative } from "@lib/utils";

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
      <QueryGate title="Failed to Load Label" error={{ message: "Label route is invalid." }} />
    );
  }

  if (query.error || !query.data) {
    return (
      <QueryGate
        title="Failed to Load Label"
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

      <KeyValueSection title="Overview">
        <KeyValueRow label="Name" value={label.name} />
        <KeyValueRow label="Description" value={label.description} />
        <KeyValueRow label="Membership" value={labelMembershipLabel(label.label_membership_type)} />
        <KeyValueRow label="Type" value={mutable ? "Regular" : "Built-In"} />
        <KeyValueRow
          label="Hosts"
          value={
            <TextLink to="/hosts" search={{ label_id: label.id }} className="font-medium">
              {countLabel(label.hosts_count, "host")}
            </TextLink>
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
          <SQLEditor value={label.query ?? ""} onChange={() => undefined} readOnly />
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
    <div className="flex flex-wrap gap-1.5">
      {values.map((value) => {
        const user = users.get(value);
        return user ? (
          <Badge
            key={value}
            variant="outline"
            className="font-normal"
            render={<Link to="/directory/users/$id" params={{ id: value }} />}
          >
            {user.name}
          </Badge>
        ) : (
          <Badge key={value} variant="outline" className="font-normal">
            {value}
          </Badge>
        );
      })}
    </div>
  );
}

function PlainValues({ values }: { values: readonly string[] }) {
  return <TokenList values={values} />;
}
