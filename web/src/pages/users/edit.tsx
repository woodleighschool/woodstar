import { useNavigate, useParams } from "@tanstack/react-router";
import { Trash2 } from "lucide-react";
import { useState } from "react";

import { PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { UserDeleteDialog } from "@/components/users/user-delete-dialog";
import { useAuth } from "@/hooks/use-auth";
import { type User, useUpdateUser, useUser } from "@/hooks/use-users";
import { AccountPage } from "@/pages/account";
import { UserForm, userFromDetail } from "@/pages/users/fields";

export function UserEditPage() {
  const params = useParams({ strict: false });
  const userId = params.userId ?? "";
  const userID = Number(userId);
  const user = useUser(userID);
  const { user: currentUser } = useAuth();

  if (user.error) {
    return (
      <PageShell>
        <QueryError
          title="Failed to load user"
          error={user.error}
          onRetry={() => void user.refetch()}
        />
      </PageShell>
    );
  }

  if (!user.data) {
    return null;
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
      <UserForm
        initial={userFromDetail(user)}
        user={user}
        pending={update.isPending}
        error={update.error}
        onSubmit={async (body) => {
          const saved = await update.mutateAsync({ id: user.id, body });
          void navigate({
            to: "/directory/users/$userId/edit",
            params: { userId: String(saved.id) },
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
