import { revalidateLogic, useForm } from "@tanstack/react-form";
import { Copy, Eye, EyeOff, Pencil, Plus, Trash2 } from "lucide-react";
import { type ReactNode, useMemo, useState } from "react";
import { toast } from "sonner";
import { z } from "zod";

import { EmptyPanel } from "@/components/empty-panel";
import { FormField } from "@/components/form-field";
import { Pending } from "@/components/pending";
import { QueryError } from "@/components/query-error";
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
import { FieldGroup } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@/components/ui/input-group";
import { Skeleton } from "@/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  useAgentSecrets,
  useCreateAgentSecret,
  useDeleteAgentSecret,
  useUpdateAgentSecret,
} from "@/hooks/use-agent-secrets";
import { useFormExitGuard } from "@/hooks/use-form-exit-guard";
import type { AgentSecret } from "@/lib/api";
import {
  deleteDescription,
  type Integration,
  integrationLabel,
  secretUsageDescription,
} from "@/lib/enrollments";
const MIN_SECRET_LENGTH = 32;
const SECRET_MASK = "••••••••••••••••••••••••";
const secretValueSchema = z
  .string()
  .trim()
  .min(MIN_SECRET_LENGTH, "Enrollment secret must be at least 32 characters.");
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
            <DialogTitle>{`Manage ${integrationLabel(integration)} Enrollment Secrets`}</DialogTitle>
            <DialogDescription>{secretUsageDescription(integration)}</DialogDescription>
          </DialogHeader>

          <div className="flex justify-end border-t pt-5">
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
              Add Secret
            </Button>
          </div>

          <SecretList
            rows={rows}
            isLoading={query.isLoading}
            error={query.error}
            onRetry={() => void query.refetch()}
            deletingID={remove.variables ?? null}
            visibleSecrets={visibleSecrets}
            onToggleVisible={(secret) =>
              setVisibleSecrets((current) => ({ ...current, [secret.id]: !current[secret.id] }))
            }
            onCopy={(secret) => void copySecret(secret)}
            onEdit={setEditing}
            onDelete={setDeleting}
            emptyTitle={`No ${integrationLabel(integration)} secrets yet`}
          />

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {creatingValue ? (
        <SecretValueDialog
          key={`create-${integration}-${creatingValue}`}
          title={`New ${integrationLabel(integration)} Secret`}
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
          title={`Edit ${integrationLabel(editing.agent)} Secret`}
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
            const next = await update.mutateAsync({ id: editing.id, body: { value } });
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
  error,
  onRetry,
  deletingID,
  visibleSecrets,
  onToggleVisible,
  onCopy,
  onEdit,
  onDelete,
  emptyTitle,
}: {
  rows: AgentSecret[];
  isLoading: boolean;
  error: {
    message?: string;
  } | null;
  onRetry: () => void;
  deletingID: number | null;
  visibleSecrets: Record<number, boolean>;
  onToggleVisible: (secret: AgentSecret) => void;
  onCopy: (secret: AgentSecret) => void;
  onEdit: (secret: AgentSecret) => void;
  onDelete: (secret: AgentSecret) => void;
  emptyTitle: string;
}) {
  if (error) {
    return <QueryError title="Failed to load enrollment secrets" error={error} onRetry={onRetry} />;
  }
  if (isLoading) {
    return (
      <div className="grid gap-2">
        <Skeleton className="h-9 w-full" />
        <Skeleton className="h-9 w-full" />
        <Skeleton className="h-9 w-full" />
      </div>
    );
  }
  if (rows.length === 0) {
    return <EmptyPanel>{emptyTitle}</EmptyPanel>;
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
        <SecretAction label="Copy Enrollment Secret" disabled={disabled} onClick={onCopy}>
          <Copy />
        </SecretAction>
        <SecretAction
          label={visible ? "Hide Enrollment Secret" : "Show Enrollment Secret"}
          disabled={disabled}
          onClick={onToggleVisible}
        >
          {visible ? <EyeOff /> : <Eye />}
        </SecretAction>
        <SecretAction label="Edit Enrollment Secret" disabled={disabled} onClick={onEdit}>
          <Pencil />
        </SecretAction>
        <SecretAction label="Delete Enrollment Secret" disabled={disabled} onClick={onDelete}>
          <Trash2 />
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
      <TooltipTrigger
        render={
          <InputGroupButton
            size="icon-sm"
            aria-label={label}
            disabled={disabled}
            onClick={onClick}
          />
        }
      >
        {children}
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
  const form = useForm({
    defaultValues: { value: initialValue },
    validationLogic: revalidateLogic({ mode: "submit", modeAfterSubmission: "change" }),
    validators: { onDynamic: z.object({ value: secretValueSchema }) },
    onSubmit: async ({ value }) => {
      await onSave(value.value.trim());
    },
  });
  const exitGuard = useFormExitGuard({
    form,
    onDiscard: () => onOpenChange(false),
    blockNavigation: false,
  });
  function requestClose() {
    if (pending || form.state.isSubmitting) return;
    exitGuard.requestDiscard();
  }
  return (
    <>
      <Dialog
        open={open}
        onOpenChange={(nextOpen) => {
          if (!nextOpen) requestClose();
        }}
      >
        <DialogContent className="sm:max-w-2xl">
          <form
            noValidate
            className="grid gap-4"
            onSubmit={(event) => {
              event.preventDefault();
              event.stopPropagation();
              void form.handleSubmit();
            }}
          >
            <DialogHeader>
              <DialogTitle>{title}</DialogTitle>
              <DialogDescription>{description}</DialogDescription>
            </DialogHeader>

            <FieldGroup>
              <form.Field name="value">
                {(field) => (
                  <FormField
                    field={field}
                    label="Enrollment Secret"
                    htmlFor="agent-secret-value"
                    required
                    description={`Use a shared secret of at least ${MIN_SECRET_LENGTH} characters.`}
                  >
                    {(control) => (
                      <Input
                        {...control}
                        value={field.state.value}
                        required
                        minLength={MIN_SECRET_LENGTH}
                        className="font-mono"
                        autoComplete="off"
                        spellCheck={false}
                        onBlur={field.handleBlur}
                        onChange={(event) => field.handleChange(event.target.value)}
                      />
                    )}
                  </FormField>
                )}
              </form.Field>
            </FieldGroup>

            <DialogFooter>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                disabled={pending || form.state.isSubmitting}
                onClick={requestClose}
              >
                Cancel
              </Button>
              <form.Subscribe selector={(state) => state.isSubmitting}>
                {(isSubmitting) => (
                  <Pending
                    isPending={pending || isSubmitting}
                    render={<Button type="submit" size="sm" />}
                  >
                    {isSubmitting ? `${saveLabel}…` : saveLabel}
                  </Pending>
                )}
              </form.Subscribe>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
      {exitGuard.dialog}
    </>
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
          <AlertDialogTitle>Delete {integrationLabel(secret?.agent)} Secret?</AlertDialogTitle>
          <AlertDialogDescription>
            {secret
              ? deleteDescription(secret.agent)
              : "This secret will stop authenticating future enrollment requests."}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel
            render={<Button type="button" variant="ghost" size="sm" disabled={pending} />}
          >
            Cancel
          </AlertDialogCancel>
          <AlertDialogAction
            render={
              <Button
                type="button"
                variant="destructive"
                size="sm"
                disabled={pending}
                onClick={(event) => {
                  event.preventDefault();
                  void onConfirm();
                }}
              />
            }
          >
            Delete
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
