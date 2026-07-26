import { useNavigate } from "@tanstack/react-router";

import { UserCreateForm } from "@features/directory/users/fields";
import { useCreateUser } from "@features/directory/users/queries";

export function UserCreatePage() {
  const navigate = useNavigate();
  const create = useCreateUser();

  return (
    <UserCreateForm
      onSubmit={async (body) => {
        const user = await create.mutateAsync(body);
        void navigate({
          to: "/directory/users/$id",
          params: { id: String(user.id) },
        });
      }}
      onCancel={() => void navigate({ to: "/directory/users" })}
    />
  );
}
