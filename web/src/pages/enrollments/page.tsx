import { KeyRound } from "lucide-react";
import { useState } from "react";

import { DeploymentInstructions } from "@/components/enrollments/instructions";
import { AgentSecretsDialog } from "@/components/enrollments/secrets-dialog";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Button } from "@/components/ui/button";
import { enrollmentDescription, enrollmentTitle, type Integration } from "@/lib/enrollments";
import { runtime } from "@/lib/runtime";

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
