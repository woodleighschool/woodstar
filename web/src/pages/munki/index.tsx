import { KeyRound } from "lucide-react";

import { MunkiTokensDialog } from "@/components/secrets/munki-tokens-dialog";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/ui/page-header";

export function MunkiHomePage() {
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
    </div>
  );
}
