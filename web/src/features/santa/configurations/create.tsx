import { getRouteApi } from "@tanstack/react-router";

import { ConfigurationForm } from "./fields";
import { emptyConfigurationForm } from "./form-adapter";
import { useCreateSantaConfiguration } from "./queries";

const routeApi = getRouteApi("/_authenticated/santa/configurations/new");

export function ConfigurationCreatePage() {
  const navigate = routeApi.useNavigate();
  const search = routeApi.useSearch();
  const create = useCreateSantaConfiguration();

  return (
    <ConfigurationForm
      initial={emptyConfigurationForm}
      title="Create Configuration"
      submitLabel="Create"
      activeTab={search.tab ?? "options"}
      onActiveTabChange={(value) =>
        void navigate({
          search: (previous) => ({
            ...previous,
            tab: value === "targets" ? "targets" : undefined,
          }),
        })
      }
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
