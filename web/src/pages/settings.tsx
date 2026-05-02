import { PendingBanner } from "@/components/feedback/pending-banner";
import { PageHeader } from "@/components/ui/page-header";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useVersion } from "@/hooks/use-version";
import { endpoints } from "@/lib/endpoints";

function DefinitionList({
  rows,
}: {
  rows: Array<[string, React.ReactNode]>;
}) {
  return (
    <dl className="rounded-lg border bg-card divide-y">
      {rows.map(([label, value]) => (
        <div
          key={label}
          className="grid grid-cols-[10rem_1fr] gap-3 px-4 py-2 text-sm"
        >
          <dt className="text-muted-foreground">{label}</dt>
          <dd className="font-medium break-all">{value ?? "—"}</dd>
        </div>
      ))}
    </dl>
  );
}

export function SettingsPage() {
  const { data: version, isLoading } = useVersion();

  return (
    <div className="flex flex-col">
      <PageHeader
        title="Settings"
        description="Server, identity, and directory configuration."
      />

      <div className="p-6">
        <Tabs defaultValue="server">
          <TabsList>
            <TabsTrigger value="server">Server</TabsTrigger>
            <TabsTrigger value="team">Team</TabsTrigger>
            <TabsTrigger value="oidc">OIDC</TabsTrigger>
            <TabsTrigger value="directory">Directory</TabsTrigger>
          </TabsList>

          <TabsContent value="server">
            <DefinitionList
              rows={[
                ["Version", isLoading ? "loading…" : version?.version ?? "unknown"],
                [
                  "Started at",
                  isLoading
                    ? "loading…"
                    : version?.started_at
                      ? new Date(version.started_at).toLocaleString()
                      : "unknown",
                ],
                ["Database", "Postgres (configured via env)"],
                ["Frontend mode", import.meta.env.MODE],
              ]}
            />
          </TabsContent>

          <TabsContent value="team">
            <PendingBanner endpoint={endpoints.settingsTeam.path} />
          </TabsContent>

          <TabsContent value="oidc">
            <PendingBanner endpoint={endpoints.settingsOidc.path} />
          </TabsContent>

          <TabsContent value="directory">
            <PendingBanner endpoint={endpoints.settingsDirectory.path} />
          </TabsContent>
        </Tabs>
      </div>
    </div>
  );
}
