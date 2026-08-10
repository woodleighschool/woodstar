import { KeyRound } from "lucide-react";
import { useState } from "react";

import { Button } from "@components/ui/button";
import { AgentSecretsDialog } from "@features/agent-secrets/secrets-dialog";
import { useAuth } from "@features/auth/queries";
import type { AgentSecret } from "@lib/api";

export function AgentSecretsHeaderAction({ agent }: { agent: AgentSecret["agent"] }) {
  const { user } = useAuth();
  const [open, setOpen] = useState(false);

  if (user?.role !== "admin") return null;

  return (
    <>
      <Button size="sm" onClick={() => setOpen(true)}>
        <KeyRound data-icon="inline-start" />
        {agent === "orbit" ? "Enroll Hosts" : "Secrets"}
      </Button>
      {open ? <AgentSecretsDialog agent={agent} open onOpenChange={setOpen} /> : null}
    </>
  );
}
