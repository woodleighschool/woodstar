import { useNavigate } from "@tanstack/react-router";

import { ConfigurationForm } from "./fields";
import { emptyConfigurationForm } from "./form-adapter";
import { useCreateSantaConfiguration } from "./queries";

export function ConfigurationCreatePage() {
  const navigate = useNavigate();
  const create = useCreateSantaConfiguration();

  return (
    <ConfigurationForm
      initial={emptyConfigurationForm}
      title="Create Configuration"
      submitLabel="Create"
      onCancel={() => void navigate({ to: "/santa/configurations" })}
      onSubmit={async (body) => (await create.mutateAsync(body)).id}
      onSuccess={(id) => {
        if (id !== undefined) {
          void navigate({
            to: "/santa/configurations/$id",
            params: { id: String(id) },
          });
        }
      }}
    />
  );
}
