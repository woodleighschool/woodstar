import { useNavigate, useParams } from "@tanstack/react-router";

import { PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";
import {
  useSantaConfiguration,
  useUpdateSantaConfiguration,
} from "@/hooks/use-santa-configurations";
import { ConfigurationForm, formFromConfiguration } from "@/pages/santa/configurations/fields";

export function ConfigurationEditPage() {
  const navigate = useNavigate();
  const params = useParams({ strict: false });
  const configurationId = params.configurationId ?? "";
  const id = Number(configurationId);
  const detail = useSantaConfiguration(id);
  const update = useUpdateSantaConfiguration();

  if (detail.error) {
    return (
      <PageShell>
        <QueryError
          title="Failed to load configuration"
          error={detail.error}
          onRetry={() => void detail.refetch()}
        />
      </PageShell>
    );
  }
  if (!detail.data) return null;

  const configuration = detail.data;
  return (
    <ConfigurationForm
      key={configuration.id}
      initial={formFromConfiguration(configuration)}
      submitLabel="Save"
      onSubmit={async (body) => {
        const saved = await update.mutateAsync({ id: configuration.id, body });
        void navigate({
          to: "/santa/configurations/$configurationId",
          params: { configurationId: String(saved.id) },
        });
      }}
    />
  );
}
