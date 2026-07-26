import { KeyRound } from "lucide-react";
import { useState } from "react";

import { PageHeader, PageShell } from "@components/layout/page-layout";
import { Button } from "@components/ui/button";
import { useAuth } from "@features/auth/queries";
import { DeploymentInstructions } from "@features/enrollments/instructions";
import {
  enrollmentDescription,
  enrollmentTitle,
  type Integration,
} from "@features/enrollments/metadata";
import { AgentSecretsDialog } from "@features/enrollments/secrets-dialog";
import { runtime } from "@lib/runtime";

export function EnrollmentsPage({ integration }: { integration: Integration }) {
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const [secretsOpen, setSecretsOpen] = useState(false);

  return (
    <PageShell className="gap-6">
      <PageHeader
        title={enrollmentTitle(integration)}
        description={enrollmentDescription(integration)}
        actions={
          isAdmin ? (
            <Button size="sm" onClick={() => setSecretsOpen(true)}>
              <KeyRound data-icon="inline-start" />
              Manage Secrets
            </Button>
          ) : null
        }
      />

      <DeploymentInstructions integration={integration} serverURL={runtime.serverURL} />
      {isAdmin ? (
        <AgentSecretsDialog
          integration={integration}
          open={secretsOpen}
          onOpenChange={setSecretsOpen}
        />
      ) : null}
    </PageShell>
  );
}
