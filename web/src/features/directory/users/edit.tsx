import { useNavigate, useParams } from "@tanstack/react-router";

import { QueryGate } from "@components/query-gate";
import { AccountPage } from "@features/account/page";
import { useAuth } from "@features/authn/queries";
import { UserForm, userFromDetail } from "@features/directory/users/fields";
import { useUpdateUser, useUser } from "@features/directory/users/queries";
import type { User } from "@lib/api";

export function UserEditPage() {
  const { id } = useParams({ from: "/_authenticated/directory/users/$id" });
  const user = useUser(id);
  const { user: currentUser } = useAuth();

  if (!user.data) {
    return (
      <QueryGate
        title="Failed to Load User"
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

  return (
    <UserForm
      initial={userFromDetail(user)}
      user={user}
      onSubmit={async (body) => {
        await update.mutateAsync({ id: user.id, body });
      }}
      onSuccess={() => {
        void navigate({
          to: "/directory/users/$id",
          params: { id: user.id },
        });
      }}
      onCancel={() =>
        void navigate({
          to: "/directory/users/$id",
          params: { id: user.id },
        })
      }
    />
  );
}
