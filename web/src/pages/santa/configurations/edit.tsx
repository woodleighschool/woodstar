import { useNavigate, useParams } from "@tanstack/react-router";

import { QueryGate } from "@/components/query-gate";
import {
  useSantaConfiguration,
  useUpdateSantaConfiguration,
} from "@/hooks/use-santa-configurations";
import { parseRouteID } from "@/lib/route-params";
import { ConfigurationForm, formFromConfiguration } from "@/pages/santa/configurations/fields";

export function ConfigurationEditPage() {
  const navigate = useNavigate();
  const params = useParams({ strict: false });
  const configurationId = params.configurationId ?? "";
  const id = parseRouteID(configurationId);
  const detail = useSantaConfiguration(id);
  const update = useUpdateSantaConfiguration();

  if (id === null) {
    return (
      <QueryGate
        title="Failed to load configuration"
        error={{ message: "Configuration route is invalid." }}
      />
    );
  }

  if (detail.error || !detail.data) {
    return (
      <QueryGate
        title="Failed to load configuration"
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
      onSubmit={async (body) => (await update.mutateAsync({ id: configuration.id, body })).id}
      onSuccess={(savedID) => {
        if (savedID !== undefined) {
          void navigate({
            to: "/santa/configurations/$configurationId",
            params: { configurationId: String(savedID) },
          });
        }
      }}
    />
  );
}
