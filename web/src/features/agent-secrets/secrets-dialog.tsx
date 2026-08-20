import { revalidateLogic, useForm } from "@tanstack/react-form";
import { Check, Copy, Eye, EyeOff, Pencil, Plus, Trash2 } from "lucide-react";
import { useEffect, useId, useRef, useState } from "react";
import { z } from "zod";

import { AsyncButton } from "@components/async-button";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryError } from "@components/query-error";
import { Button } from "@components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@components/ui/dialog";
import { InputGroup, InputGroupInput } from "@components/ui/input-group";
import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemTitle,
} from "@components/ui/item";
import { Separator } from "@components/ui/separator";
import { Skeleton } from "@components/ui/skeleton";
import { ValidatedFormField } from "@components/validated-form-field";
import {
  useAgentSecrets,
  useCreateAgentSecret,
  useDeleteAgentSecret,
  useUpdateAgentSecret,
} from "@features/agent-secrets/queries";
import type { AgentSecret } from "@lib/api";

const MIN_SECRET_LENGTH = 32;
const SECRET_MASK = "••••••••••••••••••••••••••••••••";
const DOCS_BASE_URL = "https://woodleighschool.github.io/woodstar/docs/agent-protocols";

type SecretAgent = AgentSecret["agent"];

const dialogCopy: Record<SecretAgent, { title: string; description: string; docsURL: string }> = {
  orbit: {
    title: "Enroll Hosts",
    description: "Use these secrets to enroll hosts through Orbit or osquery.",
    docsURL: `${DOCS_BASE_URL}/orbit-and-osquery`,
  },
  munki: {
    title: "Munki Secrets",
    description: "Allow Munki clients to authenticate to Woodstar.",
    docsURL: `${DOCS_BASE_URL}/munki-repository`,
  },
  santa: {
    title: "Santa Secrets",
    description: "Allow Santa clients to authenticate to Woodstar.",
    docsURL: `${DOCS_BASE_URL}/santa-sync`,
  },
};

const secretEditorSchema = z.object({
  value: z.string().trim().min(MIN_SECRET_LENGTH, "Secret must be at least 32 characters."),
});

type ActiveRow =
  | { kind: "create"; value: string }
  | { kind: "edit"; id: number }
  | { kind: "delete"; id: number }
  | null;

export function AgentSecretsDialog({
  agent,
  open,
  onOpenChange,
}: {
  agent: SecretAgent;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const query = useAgentSecrets();
  const copy = dialogCopy[agent];
  const rows = (query.data ?? []).filter((secret) => secret.agent === agent);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{copy.title}</DialogTitle>
          <DialogDescription>
            {copy.description} See the{" "}
            <a
              href={copy.docsURL}
              target="_blank"
              rel="noreferrer"
              className="underline underline-offset-4 hover:text-foreground"
            >
              deployment guide
            </a>
            .
          </DialogDescription>
        </DialogHeader>

        <Separator />

        {query.error ? (
          <QueryError
            title="Failed to Load Secrets"
            error={query.error}
            onRetry={() => void query.refetch()}
          />
        ) : query.isLoading ? (
          <SecretListSkeleton />
        ) : (
          <SecretManager agent={agent} rows={rows} />
        )}
      </DialogContent>
    </Dialog>
  );
}

function SecretManager({ agent, rows }: { agent: SecretAgent; rows: AgentSecret[] }) {
  const create = useCreateAgentSecret();
  const update = useUpdateAgentSecret();
  const remove = useDeleteAgentSecret();
  const [activeRow, setActiveRow] = useState<ActiveRow>(null);
  const [visibleSecrets, setVisibleSecrets] = useState<ReadonlySet<number>>(() => new Set());

  function setSecretVisible(id: number, visible: boolean) {
    setVisibleSecrets((current) => {
      const next = new Set(current);
      if (visible) next.add(id);
      else next.delete(id);
      return next;
    });
  }

  return (
    <div className="grid gap-3">
      {rows.length === 0 && activeRow?.kind !== "create" ? (
        <PanelEmptyState>No Secrets Yet</PanelEmptyState>
      ) : null}

      <ItemGroup>
        {rows.map((secret) => {
          if (activeRow?.kind === "edit" && activeRow.id === secret.id) {
            return (
              <SecretEditorItem
                key={secret.id}
                mode="edit"
                initialValue={secret.value}
                pending={update.isPending}
                onCancel={() => {
                  update.reset();
                  setActiveRow(null);
                }}
                onSave={async (value) => {
                  const saved = await update.mutateAsync({ id: secret.id, body: { value } });
                  setSecretVisible(saved.id, true);
                  setActiveRow(null);
                }}
              />
            );
          }

          if (activeRow?.kind === "delete" && activeRow.id === secret.id) {
            return (
              <SecretDeleteItem
                key={secret.id}
                pending={remove.isPending}
                onCancel={() => {
                  remove.reset();
                  setActiveRow(null);
                }}
                onDelete={async () => {
                  await remove.mutateAsync(secret.id);
                  setSecretVisible(secret.id, false);
                  setActiveRow(null);
                }}
              />
            );
          }

          return (
            <SecretDisplayItem
              key={secret.id}
              secret={secret}
              visible={visibleSecrets.has(secret.id)}
              mutationsDisabled={activeRow !== null}
              onToggleVisible={() => setSecretVisible(secret.id, !visibleSecrets.has(secret.id))}
              onEdit={() => setActiveRow({ kind: "edit", id: secret.id })}
              onDelete={() => setActiveRow({ kind: "delete", id: secret.id })}
            />
          );
        })}

        {activeRow?.kind === "create" ? (
          <SecretEditorItem
            key="create"
            mode="create"
            initialValue={activeRow.value}
            pending={create.isPending}
            onCancel={() => {
              create.reset();
              setActiveRow(null);
            }}
            onSave={async (value) => {
              const saved = await create.mutateAsync({ agent, value });
              setSecretVisible(saved.id, true);
              setActiveRow(null);
            }}
          />
        ) : null}
      </ItemGroup>

      <Button
        type="button"
        variant="outline"
        size="sm"
        className="w-fit"
        disabled={activeRow !== null}
        onClick={() => setActiveRow({ kind: "create", value: generateSecretValue() })}
      >
        <Plus data-icon="inline-start" />
        Add Secret
      </Button>
    </div>
  );
}

function SecretDisplayItem({
  secret,
  visible,
  mutationsDisabled,
  onToggleVisible,
  onEdit,
  onDelete,
}: {
  secret: AgentSecret;
  visible: boolean;
  mutationsDisabled: boolean;
  onToggleVisible: () => void;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const [copyState, setCopyState] = useState<"idle" | "copied" | "error">("idle");
  const copyResetTimer = useRef<number | null>(null);

  useEffect(
    () => () => {
      if (copyResetTimer.current !== null) window.clearTimeout(copyResetTimer.current);
    },
    [],
  );

  async function copySecret() {
    if (copyResetTimer.current !== null) window.clearTimeout(copyResetTimer.current);
    try {
      await navigator.clipboard.writeText(secret.value);
      setCopyState("copied");
    } catch {
      setCopyState("error");
    }
    copyResetTimer.current = window.setTimeout(() => setCopyState("idle"), 2000);
  }

  return (
    <Item variant="outline" size="sm" className="flex-nowrap">
      <ItemContent className="min-w-0">
        <ItemTitle className="line-clamp-none max-w-full font-mono break-all">
          {visible ? secret.value : SECRET_MASK}
        </ItemTitle>
        {copyState === "error" ? (
          <ItemDescription>Could not copy this secret.</ItemDescription>
        ) : null}
      </ItemContent>
      <ItemActions className="shrink-0">
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          aria-label={copyState === "copied" ? "Secret copied" : "Copy secret"}
          onClick={() => void copySecret()}
        >
          {copyState === "copied" ? <Check /> : <Copy />}
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          aria-label={visible ? "Hide secret" : "Show secret"}
          onClick={onToggleVisible}
        >
          {visible ? <EyeOff /> : <Eye />}
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          aria-label="Edit secret"
          disabled={mutationsDisabled}
          onClick={onEdit}
        >
          <Pencil />
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          aria-label="Delete secret"
          disabled={mutationsDisabled}
          onClick={onDelete}
        >
          <Trash2 />
        </Button>
      </ItemActions>
    </Item>
  );
}

function SecretEditorItem({
  mode,
  initialValue,
  pending,
  onCancel,
  onSave,
}: {
  mode: "create" | "edit";
  initialValue: string;
  pending: boolean;
  onCancel: () => void;
  onSave: (value: string) => Promise<void>;
}) {
  const formID = useId();
  const inputID = useId();
  const [submitError, setSubmitError] = useState<string | null>(null);
  const form = useForm({
    defaultValues: { value: initialValue },
    validationLogic: revalidateLogic({ mode: "submit", modeAfterSubmission: "change" }),
    validators: { onDynamic: secretEditorSchema },
    onSubmit: async ({ value }) => {
      setSubmitError(null);
      try {
        await onSave(value.value.trim());
      } catch (error) {
        setSubmitError(errorMessage(error));
      }
    },
  });

  return (
    <Item variant="outline" size="sm" className="items-start">
      <ItemContent className="min-w-0">
        <form
          id={formID}
          noValidate
          onSubmit={(event) => {
            event.preventDefault();
            event.stopPropagation();
            void form.handleSubmit();
          }}
        >
          <form.Field name="value">
            {(field) => (
              <ValidatedFormField field={field} htmlFor={inputID}>
                {(control) => (
                  <InputGroup>
                    <InputGroupInput
                      {...control}
                      name={field.name}
                      type="text"
                      value={field.state.value}
                      required
                      minLength={MIN_SECRET_LENGTH}
                      aria-label="Secret"
                      className="font-mono"
                      autoComplete="off"
                      spellCheck={false}
                      onBlur={field.handleBlur}
                      onChange={(event) => field.handleChange(event.target.value)}
                    />
                  </InputGroup>
                )}
              </ValidatedFormField>
            )}
          </form.Field>
        </form>
        {submitError ? (
          <ItemDescription className="text-destructive">{submitError}</ItemDescription>
        ) : null}
      </ItemContent>
      <ItemActions className="self-end">
        <Button type="button" variant="ghost" size="sm" disabled={pending} onClick={onCancel}>
          Cancel
        </Button>
        <form.Subscribe
          selector={(state) => ({
            canSubmit: state.canSubmit,
            isDirty: state.isDirty,
            isSubmitting: state.isSubmitting,
          })}
        >
          {({ canSubmit, isDirty, isSubmitting }) => (
            <AsyncButton
              type="submit"
              form={formID}
              size="sm"
              isPending={pending || isSubmitting}
              disabled={!canSubmit || (mode === "edit" && !isDirty)}
            >
              {mode === "create" ? "Create" : "Save"}
            </AsyncButton>
          )}
        </form.Subscribe>
      </ItemActions>
    </Item>
  );
}

function SecretDeleteItem({
  pending,
  onCancel,
  onDelete,
}: {
  pending: boolean;
  onCancel: () => void;
  onDelete: () => Promise<void>;
}) {
  const [deleteError, setDeleteError] = useState<string | null>(null);

  async function deleteSecret() {
    setDeleteError(null);
    try {
      await onDelete();
    } catch (error) {
      setDeleteError(errorMessage(error));
    }
  }

  return (
    <Item variant="outline" size="sm">
      <ItemContent className="min-w-0">
        <ItemTitle className="font-mono">{SECRET_MASK}</ItemTitle>
        <ItemDescription>Delete this secret?</ItemDescription>
        {deleteError ? (
          <ItemDescription className="text-destructive">{deleteError}</ItemDescription>
        ) : null}
      </ItemContent>
      <ItemActions>
        <Button type="button" variant="ghost" size="sm" disabled={pending} onClick={onCancel}>
          Cancel
        </Button>
        <AsyncButton
          type="button"
          variant="destructive"
          size="sm"
          isPending={pending}
          onClick={() => void deleteSecret()}
        >
          Delete
        </AsyncButton>
      </ItemActions>
    </Item>
  );
}

function SecretListSkeleton() {
  return (
    <div className="grid gap-2">
      <Skeleton className="h-12 w-full" />
      <Skeleton className="h-12 w-full" />
      <Skeleton className="h-12 w-full" />
    </div>
  );
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : "An unexpected error occurred.";
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
