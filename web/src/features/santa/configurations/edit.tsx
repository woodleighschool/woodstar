import { getRouteApi, useParams } from "@tanstack/react-router";

import { QueryGate } from "@components/query-gate";
import { parseRouteID } from "@lib/route-params";

import { ConfigurationForm } from "./fields";
import { formFromConfiguration } from "./form-adapter";
import { useSantaConfiguration, useUpdateSantaConfiguration } from "./queries";

const routeApi = getRouteApi("/_authenticated/santa/configurations/$id/edit");

export function ConfigurationEditPage() {
  const navigate = routeApi.useNavigate();
  const search = routeApi.useSearch();
  const params = useParams({ strict: false });
  const configurationId = params.id ?? "";
  const id = parseRouteID(configurationId);
  const detail = useSantaConfiguration(id);
  const update = useUpdateSantaConfiguration();

  if (id === null) {
    return (
      <QueryGate
        title="Failed to Load Configuration"
        error={{ message: "Configuration route is invalid." }}
      />
    );
  }

  if (detail.error || !detail.data) {
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
          params: { id: String(configuration.id) },
        })
      }
      onSubmit={async (body) => (await update.mutateAsync({ id: configuration.id, body })).id}
      onSuccess={(savedID) => {
        if (savedID !== undefined) {
          void navigate({
            to: "/santa/configurations/$id",
            params: { id: String(savedID) },
          });
        }
      }}
    />
  );
}
