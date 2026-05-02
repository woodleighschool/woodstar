import { KeyRound, Shield } from "lucide-react";

import { SantaTokensDialog } from "@/components/credentials/santa-tokens-dialog";
import { PendingBanner } from "@/components/feedback/pending-banner";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { EmptyState } from "@/components/ui/empty-state";
import { PageHeader } from "@/components/ui/page-header";
import { useResourceList } from "@/hooks/use-pending-list";
import { endpoints } from "@/lib/endpoints";
import { queryKeys } from "@/lib/query-keys";
import type { SantaProfile } from "@/lib/types";

export function SantaHomePage() {
  const profiles = useResourceList<SantaProfile>(
    endpoints.santaProfiles,
    queryKeys.santaProfiles,
  );

  return (
    <div className="flex flex-col">
      <PageHeader
        title="Santa"
        description="Application control sync server. Hosts must be enrolled via Orbit/osquery first."
        actions={
          <SantaTokensDialog
            trigger={
              <Button variant="outline" size="sm" className="gap-2">
                <KeyRound className="size-4" /> Manage sync tokens
              </Button>
            }
          />
        }
      />

      <div className="flex flex-col gap-4 p-6">
        <div className="grid gap-4 md:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Configuration profiles</CardTitle>
              <CardDescription>
                Scoped client-mode and rule-set profiles. Resolution order:
                explicit host override, manual labels, directory groups,
                default.
              </CardDescription>
            </CardHeader>
            <CardContent>
              {profiles.isPending ? (
                <PendingBanner endpoint={endpoints.santaProfiles.path} />
              ) : profiles.data.length === 0 ? (
                <EmptyState
                  icon={Shield}
                  title="No Santa profiles"
                  description="Add a default profile to control client mode and rule scope."
                />
              ) : (
                <p className="text-sm text-muted-foreground">
                  {profiles.data.length} profiles configured.
                </p>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Recent activity</CardTitle>
              <CardDescription>
                Block and allow events uploaded by Santa during sync.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <PendingBanner endpoint="/api/v1/santa/events" />
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
