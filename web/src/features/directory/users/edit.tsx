import { useNavigate, useParams } from "@tanstack/react-router";
import { Trash2 } from "lucide-react";
import { useState } from "react";

import { QueryGate } from "@components/query-gate";
import { Button } from "@components/ui/button";
import { AccountPage } from "@features/account/page";
import { useAuth } from "@features/auth/queries";
import { UserDeleteDialog } from "@features/directory/users/delete-dialog";
import { UserForm, userFromDetail } from "@features/directory/users/fields";
import { useUpdateUser, useUser } from "@features/directory/users/queries";
import type { User } from "@lib/api";
import { parseRouteID } from "@lib/route-params";

export function UserEditPage() {
  const params = useParams({ strict: false });
  const userId = params.id ?? "";
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
    <>
      <UserForm
        initial={userFromDetail(user)}
        user={user}
        actions={
          <Button type="button" variant="outline" size="sm" onClick={() => setDeleteOpen(true)}>
            <Trash2 data-icon="inline-start" />
            Delete
          </Button>
        }
        onSubmit={async (body) => {
          await update.mutateAsync({ id: user.id, body });
        }}
        onCancel={() => void navigate({ to: "/directory/users" })}
      />

      <UserDeleteDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        user={user}
        onDeleted={() => void navigate({ to: "/directory/users" })}
      />
    </>
  );
}
