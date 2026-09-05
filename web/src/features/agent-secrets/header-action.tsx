import { KeyRound } from "lucide-react";
import { useState } from "react";

import { Button } from "@components/ui/button";
import { AgentSecretsDialog } from "@features/agent-secrets/secrets-dialog";
import { useCan } from "@features/authz/access";
import type { AgentSecret } from "@lib/api";

export function AgentSecretsHeaderAction({ agent }: { agent: AgentSecret["agent"] }) {
  const canEdit = useCan({ resource: "agents.secrets", access: "edit" });
  const [open, setOpen] = useState(false);

  if (!canEdit) return null;

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
