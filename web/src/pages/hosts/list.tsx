import { KeyRound, Package } from "lucide-react";

import { OrbitEnrollSecretsDialog } from "@/components/secrets/orbit-enroll-secrets-dialog";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/ui/page-header";

export function HostsListPage() {
  return (
    <div className="flex flex-col">
      <PageHeader
        title="Hosts"
        description="Orbit/osquery-managed Macs in this Woodstar deployment."
        actions={
          <>
            <OrbitEnrollSecretsDialog
              trigger={
                <Button variant="outline" size="sm" className="gap-2">
                  <KeyRound className="size-4" /> Manage enroll secrets
                </Button>
              }
            />
            <Button size="sm" className="gap-2" disabled>
              <Package className="size-4" /> Generate enrollment package
            </Button>
          </>
        }
      />
    </div>
  );
}
