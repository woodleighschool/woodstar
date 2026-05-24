import type { ColumnDef } from "@tanstack/react-table";
import { Copy, Eye, EyeOff, KeyRound, Loader2, Pencil, Plus, Trash2 } from "lucide-react";
import { useState, type ReactNode } from "react";
import { toast } from "sonner";
import { z } from "zod";

import { DataTable } from "@/components/data-table/data-table";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  useAgentSecrets,
  useCreateAgentSecret,
  useDeleteAgentSecret,
  useUpdateAgentSecret,
  type Agent,
  type AgentSecret,
} from "@/hooks/use-agent-secrets";

const AGENTS: Array<{ value: Agent; label: string }> = [
  { value: "orbit", label: "Orbit" },
  { value: "santa", label: "Santa" },
];

const MIN_SECRET_LENGTH = 32;
const secretValueSchema = z.string().trim().min(MIN_SECRET_LENGTH, "Secret must be at least 32 characters.");

export function SecretsPage() {
  const query = useAgentSecrets();
  const create = useCreateAgentSecret();
  const update = useUpdateAgentSecret();
  const remove = useDeleteAgentSecret();
  const [agent, setAgent] = useState<Agent>("orbit");
  const [creating, setCreating] = useState<{ agent: Agent; value: string } | null>(null);
  const [editing, setEditing] = useState<AgentSecret | null>(null);
  const [deleting, setDeleting] = useState<AgentSecret | null>(null);
  const [visibleSecrets, setVisibleSecrets] = useState<Record<number, boolean>>({});
  const secrets = query.data ?? [];
  const orbitSecrets = secrets.filter((secret) => secret.agent === "orbit");
  const santaSecrets = secrets.filter((secret) => secret.agent === "santa");

  async function copySecret(secret: AgentSecret) {
    await navigator.clipboard.writeText(secret.value);
    toast.success("Secret copied");
  }

  return (
    <PageShell className="gap-6">
      <PageHeader title="Secrets" />

      <Tabs value={agent} onValueChange={(value) => setAgent(value as Agent)}>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <TabsList>
            {AGENTS.map((option) => (
              <TabsTrigger key={option.value} value={option.value}>
                {option.label}
              </TabsTrigger>
            ))}
          </TabsList>
          <Button
            size="sm"
            onClick={() => {
              create.reset();
              setCreating({ agent, value: generateSecretValue() });
            }}
          >
            <Plus data-icon="inline-start" />
            New {agentLabel(agent)} secret
          </Button>
        </div>

        <TabsContent value="orbit" className="grid gap-4">
          <SecretTabDescription>
            Orbit secrets authenticate Orbit enrollment and osquery enrollment because Orbit is the expected wrapper for
            osquery in this deployment. Deleting an Orbit secret stops future enrollment with that shared secret, but
            already-enrolled hosts continue using their issued instance node keys.
          </SecretTabDescription>
          <SecretTable
            rows={orbitSecrets}
            isLoading={query.isLoading}
            error={query.error ?? null}
            onRetry={() => void query.refetch()}
            deletingID={remove.variables ?? null}
            visibleSecrets={visibleSecrets}
            onToggleVisible={(secret) =>
              setVisibleSecrets((current) => ({ ...current, [secret.id]: !current[secret.id] }))
            }
            onCopy={(secret) => void copySecret(secret)}
            onEdit={setEditing}
            onDelete={setDeleting}
            emptyTitle="No Orbit secrets"
            emptyDescription="Create an Orbit secret before enrolling managed hosts."
          />
        </TabsContent>

        <TabsContent value="santa" className="grid gap-4">
          <SecretTabDescription>
            Santa secrets authenticate every Santa sync request through the bearer authorization header. Deleting a
            Santa secret immediately stops Santa clients using that value from syncing until they are configured with
            another active secret.
          </SecretTabDescription>
          <SecretTable
            rows={santaSecrets}
            isLoading={query.isLoading}
            error={query.error ?? null}
            onRetry={() => void query.refetch()}
            deletingID={remove.variables ?? null}
            visibleSecrets={visibleSecrets}
            onToggleVisible={(secret) =>
              setVisibleSecrets((current) => ({ ...current, [secret.id]: !current[secret.id] }))
            }
            onCopy={(secret) => void copySecret(secret)}
            onEdit={setEditing}
            onDelete={setDeleting}
            emptyTitle="No Santa secrets"
            emptyDescription="Create a Santa secret before configuring Santa clients to sync."
          />
        </TabsContent>
      </Tabs>

      {creating ? (
        <SecretValueDialog
          key={`create-${creating.agent}-${creating.value}`}
          title={`New ${agentLabel(creating.agent)} secret`}
          description="Review or replace the generated secret before saving it."
          initialValue={creating.value}
          open
          pending={create.isPending}
          error={create.error?.message}
          saveLabel="Create"
          onOpenChange={(open) => {
            if (!open) {
              create.reset();
              setCreating(null);
            }
          }}
          onSave={async (value) => {
            const secret = await create.mutateAsync({ agent: creating.agent, value });
            setCreating(null);
            setVisibleSecrets((current) => ({ ...current, [secret.id]: true }));
          }}
        />
      ) : null}

      {editing ? (
        <SecretValueDialog
          key={editing.id}
          title={`Edit ${agentLabel(editing.agent)} secret`}
          description="Update the shared secret value accepted by this agent."
          initialValue={editing.value}
          open
          pending={update.isPending}
          error={update.error?.message}
          saveLabel="Save"
          onOpenChange={(open) => {
            if (!open) {
              update.reset();
              setEditing(null);
            }
          }}
          onSave={async (value) => {
            const next = await update.mutateAsync({ id: editing.id, value });
            setEditing(null);
            setVisibleSecrets((current) => ({ ...current, [next.id]: true }));
          }}
        />
      ) : null}

      <SecretDeleteDialog
        secret={deleting}
        open={deleting !== null}
        pending={remove.isPending}
        error={remove.error?.message}
        onOpenChange={(open) => {
          if (!open) {
            remove.reset();
            setDeleting(null);
          }
        }}
        onConfirm={async () => {
          if (!deleting) return;
          await remove.mutateAsync(deleting.id);
          setDeleting(null);
        }}
      />
    </PageShell>
  );
}

function SecretTabDescription({ children }: { children: ReactNode }) {
  return <p className="text-muted-foreground max-w-3xl text-sm leading-6">{children}</p>;
}

function IconAction({
  label,
  disabled,
  onClick,
  children,
}: {
  label: string;
  disabled?: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button type="button" size="icon" variant="ghost" aria-label={label} disabled={disabled} onClick={onClick}>
          {children}
        </Button>
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}

function SecretTable({
  rows,
  isLoading,
  error,
  onRetry,
  deletingID,
  visibleSecrets,
  onToggleVisible,
  onCopy,
  onEdit,
  onDelete,
  emptyTitle,
  emptyDescription,
}: {
  rows: AgentSecret[];
  isLoading: boolean;
  error: Error | null;
  onRetry: () => void;
  deletingID: number | null;
  visibleSecrets: Record<number, boolean>;
  onToggleVisible: (secret: AgentSecret) => void;
  onCopy: (secret: AgentSecret) => void;
  onEdit: (secret: AgentSecret) => void;
  onDelete: (secret: AgentSecret) => void;
  emptyTitle: string;
  emptyDescription: string;
}) {
  if (error) {
    return (
      <Alert variant="destructive">
        <AlertTitle>Failed to load secrets</AlertTitle>
        <AlertDescription>{error.message}</AlertDescription>
        <Button variant="outline" size="sm" onClick={onRetry} className="mt-2 w-fit">
          Retry
        </Button>
      </Alert>
    );
  }

  const columns: ColumnDef<AgentSecret>[] = [
    {
      id: "value",
      accessorKey: "value",
      header: "Secret",
      cell: ({ row }) => {
        const visible = Boolean(visibleSecrets[row.original.id]);
        return (
          <span
            className="block max-w-[34rem] truncate font-mono text-xs"
            title={visible ? row.original.value : undefined}
          >
            {visible ? row.original.value : "************************"}
          </span>
        );
      },
    },
    {
      id: "actions",
      header: () => null,
      enableSorting: false,
      cell: ({ row }) => {
        const visible = Boolean(visibleSecrets[row.original.id]);
        return (
          <div className="flex justify-end gap-1">
            <IconAction
              label={visible ? "Mask secret" : "Show secret"}
              disabled={deletingID === row.original.id}
              onClick={() => onToggleVisible(row.original)}
            >
              {visible ? <EyeOff /> : <Eye />}
            </IconAction>
            <IconAction
              label="Copy secret"
              disabled={deletingID === row.original.id}
              onClick={() => onCopy(row.original)}
            >
              <Copy />
            </IconAction>
            <IconAction
              label="Edit secret"
              disabled={deletingID === row.original.id}
              onClick={() => onEdit(row.original)}
            >
              <Pencil />
            </IconAction>
            <IconAction
              label="Delete secret"
              disabled={deletingID === row.original.id}
              onClick={() => onDelete(row.original)}
            >
              {deletingID === row.original.id ? <Loader2 className="animate-spin" /> : <Trash2 />}
            </IconAction>
          </div>
        );
      },
      meta: { headClassName: "w-40" },
    },
  ];

  return (
    <DataTable
      columns={columns}
      data={rows}
      totalCount={rows.length}
      pagination={{ pageIndex: 0, pageSize: rows.length || 50 }}
      sorting={[]}
      onPaginationChange={() => undefined}
      onSortingChange={() => undefined}
      isLoading={isLoading}
      clientSort
      empty={
        <Empty>
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <KeyRound />
            </EmptyMedia>
            <EmptyTitle>{emptyTitle}</EmptyTitle>
            <EmptyDescription>{emptyDescription}</EmptyDescription>
          </EmptyHeader>
        </Empty>
      }
    />
  );
}

function SecretValueDialog({
  title,
  description,
  initialValue,
  open,
  pending,
  error,
  saveLabel,
  onOpenChange,
  onSave,
}: {
  title: string;
  description: string;
  initialValue: string;
  open: boolean;
  pending: boolean;
  error?: string;
  saveLabel: string;
  onOpenChange: (open: boolean) => void;
  onSave: (value: string) => Promise<void>;
}) {
  const [value, setValue] = useState(initialValue);
  const secretValue = secretValueSchema.safeParse(value);
  const message = secretValue.success ? null : secretValue.error.issues[0]?.message;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <div className="grid gap-2">
          <Label htmlFor="agent-secret-value">Secret</Label>
          <Input
            id="agent-secret-value"
            value={value}
            className="font-mono"
            autoComplete="off"
            spellCheck={false}
            onChange={(event) => setValue(event.target.value)}
          />
          {message ? <p className="text-sm text-destructive">{message}</p> : null}
        </div>
        {error ? <p className="text-sm text-destructive">{error}</p> : null}
        <DialogFooter>
          <Button type="button" variant="ghost" size="sm" disabled={pending} onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            type="button"
            size="sm"
            disabled={pending || !secretValue.success}
            onClick={() => {
              if (!secretValue.success) return;
              void onSave(secretValue.data);
            }}
          >
            {pending ? <Loader2 data-icon="inline-start" className="animate-spin" /> : null}
            {saveLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function SecretDeleteDialog({
  secret,
  open,
  pending,
  error,
  onOpenChange,
  onConfirm,
}: {
  secret: AgentSecret | null;
  open: boolean;
  pending: boolean;
  error?: string;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => Promise<void>;
}) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete {agentLabel(secret?.agent)} secret?</AlertDialogTitle>
          <AlertDialogDescription>
            {secret ? deleteDescription(secret.agent) : "This secret will stop authenticating agent requests."}
          </AlertDialogDescription>
        </AlertDialogHeader>
        {error ? <p className="text-sm text-destructive">{error}</p> : null}
        <AlertDialogFooter>
          <AlertDialogCancel variant="ghost" size="sm" disabled={pending}>
            Cancel
          </AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            size="sm"
            disabled={pending}
            onClick={(event) => {
              event.preventDefault();
              void onConfirm();
            }}
          >
            Delete
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function agentLabel(agent?: Agent) {
  if (agent === "santa") return "Santa";
  if (agent === "orbit") return "Orbit";
  return "agent";
}

function deleteDescription(agent: Agent) {
  if (agent === "orbit") {
    return "Future Orbit and osquery enrollment with this shared secret will stop. Existing hosts keep using their issued node keys.";
  }
  return "Santa clients using this bearer secret will stop syncing until they are configured with another active secret.";
}

function generateSecretValue() {
  const bytes = new Uint8Array(32);
  crypto.getRandomValues(bytes);
  let value = "";
  for (const byte of bytes) {
    value += String.fromCharCode(byte);
  }
  return btoa(value).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}
