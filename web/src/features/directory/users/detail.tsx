import { useNavigate, useParams } from "@tanstack/react-router";
import { Pencil, Trash2 } from "lucide-react";
import { useState } from "react";

import { EnumBadge } from "@components/enum-badge";
import { KeyValueGrid, KeyValueItem } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Link } from "@components/link";
import { QueryGate } from "@components/query-gate";
import { Button } from "@components/ui/button";
import { Card, CardContent } from "@components/ui/card";
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
          context={<EnumBadge value={user.source} metadata={DIRECTORY_SOURCES} />}
          actions={
            isSelf || isAdmin ? (
              <>
                <Button size="sm" render={<Link {...editLink} />} nativeButton={false}>
                  <Pencil data-icon="inline-start" />
                  Edit
                </Button>
                {isAdmin && !isSelf ? (
                  <Button
                    type="button"
                    variant="destructive"
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

        <Card>
          <CardContent>
            <KeyValueGrid>
              <KeyValueItem label="Name" value={nonEmpty(user.name) ?? "-"} />
              <KeyValueItem label="Email" value={user.email} />
              <KeyValueItem
                label="Role"
                value={<EnumBadge value={userAccessRole(user.role)} metadata={USER_ACCESS_ROLES} />}
              />
              <KeyValueItem label="Can Login" value={user.can_login ? "Yes" : "No"} />
              <KeyValueItem label="Department" value={nonEmpty(user.department) ?? "-"} />
              <KeyValueItem label="Given Name" value={nonEmpty(user.given_name) ?? "-"} />
              <KeyValueItem label="Family Name" value={nonEmpty(user.family_name) ?? "-"} />
              <KeyValueItem
                label="User Principal Name"
                value={nonEmpty(user.user_principal_name) ?? "-"}
              />
              <KeyValueItem label="External ID" value={nonEmpty(user.external_id) ?? "-"} />
              <KeyValueItem label="Updated" value={formatRelative(user.updated_at)} />
              <KeyValueItem label="Created" value={formatRelative(user.created_at)} />
            </KeyValueGrid>
          </CardContent>
        </Card>
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
