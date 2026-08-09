import { useNavigate, useParams } from "@tanstack/react-router";
import { Pencil, Trash2 } from "lucide-react";
import { useState } from "react";

import { BooleanIndicator } from "@components/boolean-indicator";
import { EnumBadge } from "@components/enum-badge";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { QueryGate } from "@components/query-gate";
import { Button } from "@components/ui/button";
import { useAuth } from "@features/auth/queries";
import { DIRECTORY_SOURCES } from "@features/directory/source";
import { UserDeleteDialog } from "@features/directory/users/delete-dialog";
import { USER_ACCESS_ROLES, userAccessRole } from "@features/directory/users/metadata";
import { useUser } from "@features/directory/users/queries";
import { parseRouteID } from "@lib/route-params";
import { formatRelative, nonEmpty } from "@lib/utils";

export function UserDetailPage() {
  const { id: userID } = useParams({
    from: "/_authenticated/directory/users/$id",
  });
  const navigate = useNavigate();
  const { user: currentUser } = useAuth();
  const id = parseRouteID(userID);
  const query = useUser(id);
  const [deleteOpen, setDeleteOpen] = useState(false);

  if (id === null) {
    return <QueryGate title="Failed to load user" error={{ message: "User route is invalid." }} />;
  }
  if (query.error || !query.data) {
    return (
      <QueryGate
        title="Failed to load user"
        error={query.error}
        onRetry={() => void query.refetch()}
      />
    );
  }

  const user = query.data;
  const isSelf = currentUser?.id === user.id;
  const isAdmin = currentUser?.role === "admin";
  const editLink = isSelf
    ? ({ to: "/account" } as const)
    : ({
        to: "/directory/users/$id/edit",
        params: { id: String(user.id) },
      } as const);

  return (
    <>
      <PageShell>
        <PageHeader
          title="User Details"
          meta={`Edited ${formatRelative(user.updated_at)}`}
          actions={
            isSelf || isAdmin ? (
              <>
                <Button
                  variant={isAdmin && !isSelf ? "outline" : "default"}
                  size="sm"
                  render={<Link {...editLink} />}
                  nativeButton={false}
                >
                  <Pencil data-icon="inline-start" />
                  Edit
                </Button>
                {isAdmin && !isSelf ? (
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => setDeleteOpen(true)}
                  >
                    <Trash2 data-icon="inline-start" />
                    Delete
                  </Button>
                ) : null}
              </>
            ) : null
          }
        />

        <KeyValueSection title="Overview">
          <KeyValueRow label="Name" value={nonEmpty(user.name) ?? "-"} />
          <KeyValueRow label="Email" value={user.email} />
          <KeyValueRow
            label="Source"
            value={<EnumBadge value={user.source} metadata={DIRECTORY_SOURCES} />}
          />
          <KeyValueRow
            label="Role"
            value={<EnumBadge value={userAccessRole(user.role)} metadata={USER_ACCESS_ROLES} />}
          />
          <KeyValueRow label="Can Login" value={<BooleanIndicator value={user.can_login} />} />
          <KeyValueRow label="Department" value={nonEmpty(user.department) ?? "-"} />
          <KeyValueRow label="Given Name" value={nonEmpty(user.given_name) ?? "-"} />
          <KeyValueRow label="Family Name" value={nonEmpty(user.family_name) ?? "-"} />
          <KeyValueRow
            label="User Principal Name"
            value={nonEmpty(user.user_principal_name) ?? "-"}
          />
          <KeyValueRow label="External ID" value={nonEmpty(user.external_id) ?? "-"} />
        </KeyValueSection>
      </PageShell>

      <UserDeleteDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        user={user}
        onDeleted={() => void navigate({ to: "/directory/users" })}
      />
    </>
  );
}
