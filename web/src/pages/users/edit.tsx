import { useNavigate, useParams } from "@tanstack/react-router";
import { Trash2 } from "lucide-react";
import { useState } from "react";

import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { QueryGate } from "@/components/query-gate";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { UserDeleteDialog } from "@/components/users/user-delete-dialog";
import { useAuth } from "@/hooks/use-auth";
import { useUpdateUser, useUser } from "@/hooks/use-users";
import type { User } from "@/lib/api";
import { parseRouteID } from "@/lib/route-params";
import { AccountPage } from "@/pages/account";
import { UserForm, userFromDetail } from "@/pages/users/fields";

export function UserEditPage() {
  const params = useParams({ strict: false });
  const userId = params.userId ?? "";
  const userID = parseRouteID(userId);
  const user = useUser(userID);
  const { user: currentUser } = useAuth();

  if (userID === null) {
    return <QueryGate title="Failed to load user" error={{ message: "User route is invalid." }} />;
  }

  if (user.error || !user.data) {
    return (
      <QueryGate
        title="Failed to load user"
        error={user.error}
        onRetry={() => void user.refetch()}
      />
    );
  }

  if (currentUser?.id === user.data.id) {
    return <AccountPage />;
  }

  return <UserEdit key={user.data.updated_at} user={user.data} />;
}

function UserEdit({ user }: { user: User }) {
  const navigate = useNavigate();
  const update = useUpdateUser();
  const [deleteOpen, setDeleteOpen] = useState(false);

  return (
    <PageShell className="max-w-3xl gap-4">
      <PageHeader title="Edit User" />
      <UserForm
        initial={userFromDetail(user)}
        user={user}
        onSubmit={async (body) => (await update.mutateAsync({ id: user.id, body })).id}
        onSuccess={(id) => {
          if (id === undefined) return;
          void navigate({
            to: "/directory/users/$userId/edit",
            params: { userId: String(id) },
          });
        }}
      />

      <Card className="gap-4 py-4">
        <CardHeader className="px-4">
          <CardTitle>Delete User</CardTitle>
        </CardHeader>
        <CardContent className="px-4">
          <Button type="button" variant="destructive" size="sm" onClick={() => setDeleteOpen(true)}>
            <Trash2 data-icon="inline-start" />
            Delete
          </Button>
        </CardContent>
      </Card>

      <UserDeleteDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        user={user}
        onDeleted={() => void navigate({ to: "/directory/users" })}
      />
    </PageShell>
  );
}
