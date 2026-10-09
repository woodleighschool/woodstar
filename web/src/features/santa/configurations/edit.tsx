import { getRouteApi } from "@tanstack/react-router";

import { QueryGate } from "@components/query-gate";

import { ConfigurationForm } from "./fields";
import { formFromConfiguration } from "./form-adapter";
import { useSantaConfiguration, useUpdateSantaConfiguration } from "./queries";

const routeApi = getRouteApi("/_authenticated/santa/configurations/$id/edit");

export function ConfigurationEditPage() {
  const navigate = routeApi.useNavigate();
  const search = routeApi.useSearch();
  const { id } = routeApi.useParams();
  const detail = useSantaConfiguration(id);
  const update = useUpdateSantaConfiguration();

  if (!detail.data) {
    return (
      <QueryGate
        title="Failed to Load Configuration"
        error={detail.error}
        onRetry={() => void detail.refetch()}
      />
    );
  }

  const configuration = detail.data;
  return (
    <ConfigurationForm
      key={configuration.id}
      initial={formFromConfiguration(configuration)}
      title="Edit Configuration"
      submitLabel="Save"
      activeTab={search.tab ?? "options"}
      onActiveTabChange={(value) =>
        void navigate({
          search: (previous) => ({
            ...previous,
            tab: value === "targets" ? "targets" : undefined,
          }),
        })
      }
      onCancel={() =>
        void navigate({
          to: "/santa/configurations/$id",
          params: { id: configuration.id },
        })
      }
      onSubmit={async (body) => (await update.mutateAsync({ id: configuration.id, body })).id}
      onSuccess={(savedID) => {
        if (savedID !== undefined) {
          void navigate({
            to: "/santa/configurations/$id",
            params: { id: savedID },
          });
        }
      }}
    />
  );
}
