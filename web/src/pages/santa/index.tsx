import { KeyRound } from "lucide-react";

import { PageHeader } from "@/components/layout/page-header";
import { SantaTokensDialog } from "@/components/secrets/santa-tokens-dialog";
import { Button } from "@/components/ui/button";

export function SantaHomePage() {
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
    </div>
  );
}
