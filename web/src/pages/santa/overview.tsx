import { Link } from "@tanstack/react-router";
import { Activity, FileSliders, KeyRound, ListChecks } from "lucide-react";
import type { ReactNode } from "react";

import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useSantaConfigurations, useSantaEvents, useSantaRules, useSantaSyncTokens } from "@/hooks/use-santa";

export function SantaOverviewPage() {
  const configurations = useSantaConfigurations({ per_page: 1 });
  const rules = useSantaRules({ per_page: 1 });
  const tokens = useSantaSyncTokens();
  const events = useSantaEvents({ limit: 5 });

  return (
    <PageShell className="gap-6">
      <PageHeader
        title="Santa"
        description="Manage sync credentials, policy, configuration, and recent execution events."
      />

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <MetricCard title="Configurations" value={configurations.data?.count} icon={<FileSliders />} />
        <MetricCard title="Rules" value={rules.data?.count} icon={<ListChecks />} />
        <MetricCard title="Sync tokens" value={tokens.data?.length} icon={<KeyRound />} to="/santa/sync-tokens" />
        <MetricCard title="Recent events" value={events.data?.items?.length} icon={<Activity />} to="/santa/events" />
      </div>

      <div className="grid gap-4 lg:grid-cols-[1.1fr_0.9fr]">
        <Card>
          <CardHeader>
            <CardTitle>Recent execution events</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-3">
            {(events.data?.items ?? []).length > 0 ? (
              events.data?.items?.map((event) => (
                <div
                  key={event.id}
                  className="flex min-w-0 items-center justify-between gap-3 rounded-md border px-3 py-2"
                >
                  <div className="min-w-0">
                    <div className="truncate text-sm font-medium">
                      {event.executable.file_name || event.file_path || event.executable.sha256}
                    </div>
                    <div className="text-muted-foreground truncate text-xs">
                      {event.file_path || event.executable.sha256}
                    </div>
                  </div>
                  <DecisionBadge decision={event.decision} />
                </div>
              ))
            ) : (
              <div className="text-muted-foreground text-sm">No Santa execution events have been uploaded yet.</div>
            )}
            <Button asChild variant="outline" size="sm" className="w-fit">
              <Link to="/santa/events">View events</Link>
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Operational surface</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-3 text-sm">
            <SurfaceLink icon={<KeyRound />} label="Create or revoke Santa sync tokens" to="/santa/sync-tokens" />
            <SurfaceLink icon={<Activity />} label="Browse recent execution events" to="/santa/events" />
          </CardContent>
        </Card>
      </div>
    </PageShell>
  );
}

function MetricCard({
  title,
  value,
  icon,
  to,
}: {
  title: string;
  value: number | undefined;
  icon: ReactNode;
  to?: string;
}) {
  return (
    <Card>
      <CardContent className="flex items-center justify-between gap-3 p-4">
        <div>
          <div className="text-muted-foreground text-sm">{title}</div>
          <div className="text-2xl font-semibold tabular-nums">{value ?? "-"}</div>
        </div>
        {to ? (
          <Button asChild variant="outline" size="icon">
            <Link to={to}>
              <span className="sr-only">{title}</span>
              {icon}
            </Link>
          </Button>
        ) : (
          <div className="text-muted-foreground flex size-9 items-center justify-center rounded-md border">{icon}</div>
        )}
      </CardContent>
    </Card>
  );
}

function SurfaceLink({ icon, label, to }: { icon: ReactNode; label: string; to: string }) {
  return (
    <Button asChild variant="ghost" className="justify-start">
      <Link to={to}>
        {icon}
        {label}
      </Link>
    </Button>
  );
}

function DecisionBadge({ decision }: { decision: string }) {
  const blocked = decision.startsWith("block_");
  return <Badge variant={blocked ? "destructive" : "secondary"}>{decision.replaceAll("_", " ")}</Badge>;
}
