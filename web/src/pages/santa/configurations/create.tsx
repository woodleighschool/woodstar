import { useNavigate } from "@tanstack/react-router";

import { useCreateSantaConfiguration } from "@/hooks/use-santa-configurations";
import { ConfigurationForm, emptyConfigurationForm } from "@/pages/santa/configurations/fields";

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
            to: "/santa/configurations/$configurationId",
            params: { configurationId: String(id) },
          });
        }
      }}
    />
  );
}
