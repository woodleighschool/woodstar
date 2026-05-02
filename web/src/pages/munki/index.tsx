import { KeyRound, Layers } from "lucide-react";

import { MunkiTokensDialog } from "@/components/credentials/munki-tokens-dialog";
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
import type { MunkiManifestProfile } from "@/lib/types";

export function MunkiHomePage() {
  const profiles = useResourceList<MunkiManifestProfile>(
    endpoints.munkiManifestProfiles,
    queryKeys.munkiManifestProfiles,
  );

  return (
    <div className="flex flex-col">
      <PageHeader
        title="Munki"
        description="Managed software repo. Hosts must already exist in Woodstar inventory before Munki requests are accepted."
        actions={
          <MunkiTokensDialog
            trigger={
              <Button variant="outline" size="sm" className="gap-2">
                <KeyRound className="size-4" /> Manage repo tokens
              </Button>
            }
          />
        }
      />

      <div className="flex flex-col gap-4 p-6">
        <div className="grid gap-4 md:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>Manifest profiles</CardTitle>
              <CardDescription>
                Baseline, label, directory-group, and explicit-host assignments
                rendered into per-host Munki manifests.
              </CardDescription>
            </CardHeader>
            <CardContent>
              {profiles.isPending ? (
                <PendingBanner endpoint={endpoints.munkiManifestProfiles.path} />
              ) : profiles.data.length === 0 ? (
                <EmptyState
                  icon={Layers}
                  title="No manifest profiles"
                  description="Start with a baseline profile assigned to all managed Macs."
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
              <CardTitle>Packages</CardTitle>
              <CardDescription>
                Pkginfo records and the package payloads they reference.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <PendingBanner endpoint="/api/v1/munki/packages" />
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
