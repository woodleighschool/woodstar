import { Copy, Eye, EyeOff, KeyRound, Loader2, Pencil, Plus, Trash2 } from "lucide-react";
import { useMemo, useState, type ReactNode } from "react";
import { toast } from "sonner";
import { z } from "zod";

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
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { InputGroup, InputGroupAddon, InputGroupButton, InputGroupInput } from "@/components/ui/input-group";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  useAgentSecrets,
  useCreateAgentSecret,
  useDeleteAgentSecret,
  useUpdateAgentSecret,
  type AgentSecret,
} from "@/hooks/use-agent-secrets";

import { deleteDescription, integrationLabel, secretUsageDescription, type Integration } from "./types";

const MIN_SECRET_LENGTH = 32;
const SECRET_MASK = "••••••••••••••••••••••••";
const secretValueSchema = z.string().trim().min(MIN_SECRET_LENGTH, "Enrollment secret must be at least 32 characters.");

export function AgentSecretsDialog({
  integration,
  open,
  onOpenChange,
}: {
  integration: Integration;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const query = useAgentSecrets();
  const create = useCreateAgentSecret();
  const update = useUpdateAgentSecret();
  const remove = useDeleteAgentSecret();
  const rows = useMemo(
    () => (query.data ?? []).filter((secret) => secret.agent === integration),
    [integration, query.data],
  );

  const [creatingValue, setCreatingValue] = useState<string | null>(null);
  const [editing, setEditing] = useState<AgentSecret | null>(null);
  const [deleting, setDeleting] = useState<AgentSecret | null>(null);
  const [visibleSecrets, setVisibleSecrets] = useState<Record<number, boolean>>({});

  async function copySecret(secret: AgentSecret) {
    try {
      await navigator.clipboard.writeText(secret.value);
      toast.success("Enrollment secret copied.");
    } catch {
      toast.error("Could not copy to clipboard.");
    }
  }

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="sm:max-w-4xl">
          <DialogHeader>
            <DialogTitle>{`Manage ${integrationLabel(integration)} enrollment secrets`}</DialogTitle>
            <DialogDescription>{secretUsageDescription(integration)}</DialogDescription>
          </DialogHeader>

          <div className="flex flex-wrap items-center justify-between gap-3 border-t pt-5">
            <p className="text-muted-foreground text-sm">
              Add, reveal, copy, rotate, or remove the shared secrets accepted by this enrollment endpoint.
            </p>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => {
                create.reset();
                setCreatingValue(generateSecretValue());
              }}
            >
              <Plus data-icon="inline-start" />
              Add secret
            </Button>
          </div>

          <SecretList
            rows={rows}
            isLoading={query.isLoading}
            errorMessage={query.error?.message ?? null}
            onRetry={() => void query.refetch()}
            deletingID={remove.variables ?? null}
            visibleSecrets={visibleSecrets}
            onToggleVisible={(secret) =>
              setVisibleSecrets((current) => ({ ...current, [secret.id]: !current[secret.id] }))
            }
            onCopy={(secret) => void copySecret(secret)}
            onEdit={setEditing}
            onDelete={setDeleting}
            emptyTitle={`No ${integrationLabel(integration)} secrets`}
            emptyDescription={`Add a ${integrationLabel(integration)} enrollment secret before deploying new clients.`}
          />

          <DialogFooter>
            <Button type="button" onClick={() => onOpenChange(false)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {creatingValue ? (
        <SecretValueDialog
          key={`create-${integration}-${creatingValue}`}
          title={`New ${integrationLabel(integration)} secret`}
          description="Use this value in the matching deployment profile. Hosts that have already enrolled keep their issued node keys."
          initialValue={creatingValue}
          open
          pending={create.isPending}
          saveLabel="Create"
          onOpenChange={(nextOpen) => {
            if (!nextOpen) {
              create.reset();
              setCreatingValue(null);
            }
          }}
          onSave={async (value) => {
            const secret = await create.mutateAsync({ agent: integration, value });
            setCreatingValue(null);
            setVisibleSecrets((current) => ({ ...current, [secret.id]: true }));
            toast.success(`${integrationLabel(integration)} enrollment secret created.`);
          }}
        />
      ) : null}

      {editing ? (
        <SecretValueDialog
          key={editing.id}
          title={`Edit ${integrationLabel(editing.agent)} secret`}
          description="Changing this value affects future enrollments that use this secret. Existing enrolled hosts keep their issued node keys."
          initialValue={editing.value}
          open
          pending={update.isPending}
          saveLabel="Save"
          onOpenChange={(nextOpen) => {
            if (!nextOpen) {
              update.reset();
              setEditing(null);
            }
          }}
          onSave={async (value) => {
            const next = await update.mutateAsync({ id: editing.id, value });
            setEditing(null);
            setVisibleSecrets((current) => ({ ...current, [next.id]: true }));
            toast.success(`${integrationLabel(editing.agent)} enrollment secret updated.`);
          }}
        />
      ) : null}

      <SecretDeleteDialog
        secret={deleting}
        open={deleting !== null}
        pending={remove.isPending}
        onOpenChange={(nextOpen) => {
          if (!nextOpen) {
            remove.reset();
            setDeleting(null);
          }
        }}
        onConfirm={async () => {
          if (!deleting) return;
          await remove.mutateAsync(deleting.id);
          setDeleting(null);
          toast.success(`${integrationLabel(deleting.agent)} enrollment secret deleted.`);
        }}
      />
    </>
  );
}

function SecretList({
  rows,
  isLoading,
  errorMessage,
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
  errorMessage: string | null;
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
  if (errorMessage) {
    return (
      <Alert variant="destructive">
        <AlertTitle>Failed to load enrollment secrets</AlertTitle>
        <AlertDescription>{errorMessage}</AlertDescription>
        <Button variant="outline" size="sm" onClick={onRetry} className="mt-2 w-fit">
          Retry
        </Button>
      </Alert>
    );
  }

  if (isLoading) {
    return (
      <div className="text-muted-foreground flex items-center gap-2 rounded-md border px-3 py-8 text-sm">
        <Loader2 className="animate-spin" />
        Loading enrollment secrets...
      </div>
    );
  }

  if (rows.length === 0) {
    return (
      <div className="rounded-md border">
        <Empty>
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <KeyRound />
            </EmptyMedia>
            <EmptyTitle>{emptyTitle}</EmptyTitle>
            <EmptyDescription>{emptyDescription}</EmptyDescription>
          </EmptyHeader>
        </Empty>
      </div>
    );
  }

  return (
    <div className="grid gap-2">
      {rows.map((secret) => (
        <SecretRow
          key={secret.id}
          secret={secret}
          disabled={deletingID === secret.id}
          visible={Boolean(visibleSecrets[secret.id])}
          onToggleVisible={() => onToggleVisible(secret)}
          onCopy={() => onCopy(secret)}
          onEdit={() => onEdit(secret)}
          onDelete={() => onDelete(secret)}
        />
      ))}
    </div>
  );
}

function SecretRow({
  secret,
  disabled,
  visible,
  onToggleVisible,
  onCopy,
  onEdit,
  onDelete,
}: {
  secret: AgentSecret;
  disabled: boolean;
  visible: boolean;
  onToggleVisible: () => void;
  onCopy: () => void;
  onEdit: () => void;
  onDelete: () => void;
}) {
  return (
    <InputGroup>
      <InputGroupInput
        readOnly
        className="font-mono"
        value={visible ? secret.value : SECRET_MASK}
        title={visible ? secret.value : undefined}
      />
      <InputGroupAddon align="inline-end">
        <SecretAction label="Copy enrollment secret" disabled={disabled} onClick={onCopy}>
          <Copy />
        </SecretAction>
        <SecretAction
          label={visible ? "Hide enrollment secret" : "Show enrollment secret"}
          disabled={disabled}
          onClick={onToggleVisible}
        >
          {visible ? <EyeOff /> : <Eye />}
        </SecretAction>
        <SecretAction label="Edit enrollment secret" disabled={disabled} onClick={onEdit}>
          <Pencil />
        </SecretAction>
        <SecretAction label="Delete enrollment secret" disabled={disabled} onClick={onDelete}>
          {disabled ? <Loader2 className="animate-spin" /> : <Trash2 />}
        </SecretAction>
      </InputGroupAddon>
    </InputGroup>
  );
}

function SecretAction({
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
        <InputGroupButton size="icon-sm" aria-label={label} disabled={disabled} onClick={onClick}>
          {children}
        </InputGroupButton>
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}

function SecretValueDialog({
  title,
  description,
  initialValue,
  open,
  pending,
  saveLabel,
  onOpenChange,
  onSave,
}: {
  title: string;
  description: string;
  initialValue: string;
  open: boolean;
  pending: boolean;
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

        <FieldGroup>
          <Field data-invalid={Boolean(message)}>
            <FieldLabel htmlFor="agent-secret-value">Enrollment secret</FieldLabel>
            <Input
              id="agent-secret-value"
              value={value}
              className="font-mono"
              autoComplete="off"
              spellCheck={false}
              aria-invalid={Boolean(message)}
              onChange={(event) => setValue(event.target.value)}
            />
            <FieldDescription>Use a shared secret of at least {MIN_SECRET_LENGTH} characters.</FieldDescription>
            {message ? <FieldError>{message}</FieldError> : null}
          </Field>
        </FieldGroup>

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
  onOpenChange,
  onConfirm,
}: {
  secret: AgentSecret | null;
  open: boolean;
  pending: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => Promise<void>;
}) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete {integrationLabel(secret?.agent)} secret?</AlertDialogTitle>
          <AlertDialogDescription>
            {secret
              ? deleteDescription(secret.agent)
              : "This secret will stop authenticating future enrollment requests."}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel asChild>
            <Button type="button" variant="ghost" size="sm" disabled={pending}>
              Cancel
            </Button>
          </AlertDialogCancel>
          <AlertDialogAction asChild>
            <Button
              type="button"
              variant="destructive"
              size="sm"
              disabled={pending}
              onClick={(event) => {
                event.preventDefault();
                void onConfirm();
              }}
            >
              {pending ? <Loader2 data-icon="inline-start" className="animate-spin" /> : null}
              Delete
            </Button>
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
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
