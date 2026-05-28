import { KeyRound } from "lucide-react";
import { useState } from "react";

import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Button } from "@/components/ui/button";
import { runtime } from "@/lib/runtime";

import { DeploymentInstructions } from "./instructions";
import { AgentSecretsDialog } from "./secrets-dialog";
import { enrollmentDescription, enrollmentTitle, type Integration } from "./types";

export function EnrollmentsPage({ integration }: { integration: Integration }) {
  const [secretsOpen, setSecretsOpen] = useState(false);

  return (
    <PageShell className="gap-6">
      <PageHeader
        title={enrollmentTitle(integration)}
        description={enrollmentDescription(integration)}
        actions={
          <Button size="sm" onClick={() => setSecretsOpen(true)}>
            <KeyRound data-icon="inline-start" />
            Manage Secrets
          </Button>
        }
      />

      <DeploymentInstructions integration={integration} publicURL={runtime.publicURL} />
      <AgentSecretsDialog integration={integration} open={secretsOpen} onOpenChange={setSecretsOpen} />
    </PageShell>
  );
}
