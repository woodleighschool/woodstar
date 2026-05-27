import { xml } from "@codemirror/lang-xml";
import type { Extension } from "@codemirror/state";
import type { ColumnDef } from "@tanstack/react-table";
import {
  Check,
  Copy,
  ExternalLink,
  Eye,
  EyeOff,
  FileCode2,
  KeyRound,
  Loader2,
  Pencil,
  Plus,
  Terminal,
  Trash2,
} from "lucide-react";
import { useState, type ReactNode } from "react";
import { toast } from "sonner";
import { z } from "zod";

import { DataTable } from "@/components/data-table/data-table";
import { CodeEditor } from "@/components/editor/code-editor";
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
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
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
import {
  useAgentSecrets,
  useCreateAgentSecret,
  useDeleteAgentSecret,
  useUpdateAgentSecret,
  type AgentSecret,
} from "@/hooks/use-agent-secrets";
import { cn } from "@/lib/utils";

type Integration = AgentSecret["agent"];

const FLEETCTL_INSTALL_URL = "https://fleetdm.com/guides/fleetctl#installing-fleetctl";
const MIN_SECRET_LENGTH = 32;
const secretValueSchema = z.string().trim().min(MIN_SECRET_LENGTH, "Enrollment secret must be at least 32 characters.");
const xmlExtension = xml();
const xmlExtensions: Extension[] = [xmlExtension];

export function EnrollmentsPage({ integration }: { integration: Integration }) {
  const query = useAgentSecrets();
  const create = useCreateAgentSecret();
  const update = useUpdateAgentSecret();
  const remove = useDeleteAgentSecret();
  const [creating, setCreating] = useState<{ integration: Integration; value: string } | null>(null);
  const [editing, setEditing] = useState<AgentSecret | null>(null);
  const [deleting, setDeleting] = useState<AgentSecret | null>(null);
  const [visibleSecrets, setVisibleSecrets] = useState<Record<number, boolean>>({});
  const secrets = query.data ?? [];
  const orbitSecrets = secrets.filter((secret) => secret.agent === "orbit");
  const santaSecrets = secrets.filter((secret) => secret.agent === "santa");
  const serverURL = defaultServerURL();

  async function copySecret(secret: AgentSecret) {
    await navigator.clipboard.writeText(secret.value);
    toast.success("Enrollment secret copied");
  }

  const activeSecrets = integration === "orbit" ? orbitSecrets : santaSecrets;

  return (
    <PageShell className="gap-6">
      <PageHeader
        title="Enrollments"
        description="Enrollment secrets and install snippets for Orbit and Santa."
        actions={
          <Button
            size="sm"
            onClick={() => {
              create.reset();
              setCreating({ integration, value: generateSecretValue() });
            }}
          >
            <Plus data-icon="inline-start" />
            Add secret
          </Button>
        }
      />

      <div className="grid gap-4">
        {integration === "orbit" ? (
          <EnrollmentInstructions
            title="Orbit package"
            description="Use fleetctl to build an Orbit package for osquery enrollment."
            icon={<Terminal />}
            command={orbitPackageCommand(serverURL)}
            copyLabel="Copy package command"
            link={{ href: FLEETCTL_INSTALL_URL, label: "Install fleetctl" }}
            notes={["The example builds a macOS pkg; change --type for another package format."]}
          />
        ) : (
          <>
            <EnrollmentInstructions
              title="Santa"
              description="Use this profile payload to point Santa at Woodstar."
              icon={<FileCode2 />}
              command={santaProfileTemplate(serverURL)}
              extensions={xmlExtensions}
              copyLabel="Copy profile template"
              multiline
              notes={["Santa must send protobuf over gzip."]}
            />
            <SecretTabDescription>
              Bearer credentials for Santa. Delete one to reject clients using that value.
            </SecretTabDescription>
          </>
        )}
        <SecretTable
          rows={activeSecrets}
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
          emptyTitle={integration === "orbit" ? "No Orbit secrets" : "No Santa secrets"}
          emptyDescription={
            integration === "orbit"
              ? "Create an Orbit secret before deploying hosts."
              : "Create a Santa secret before configuring clients."
          }
        />
      </div>

      {creating ? (
        <SecretValueDialog
          key={`create-${creating.integration}-${creating.value}`}
          title={`New ${integrationLabel(creating.integration)} secret`}
          description="Review the generated enrollment secret before saving it."
          initialValue={creating.value}
          open
          pending={create.isPending}
          saveLabel="Create"
          onOpenChange={(open) => {
            if (!open) {
              create.reset();
              setCreating(null);
            }
          }}
          onSave={async (value) => {
            const secret = await create.mutateAsync({ agent: creating.integration, value });
            setCreating(null);
            setVisibleSecrets((current) => ({ ...current, [secret.id]: true }));
          }}
        />
      ) : null}

      {editing ? (
        <SecretValueDialog
          key={editing.id}
          title={`Edit ${integrationLabel(editing.agent)} secret`}
          description="Update the shared enrollment secret accepted by this integration."
          initialValue={editing.value}
          open
          pending={update.isPending}
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

export function OrbitEnrollmentsPage() {
  return <EnrollmentsPage integration="orbit" />;
}

export function SantaEnrollmentsPage() {
  return <EnrollmentsPage integration="santa" />;
}

function SecretTabDescription({ children }: { children: ReactNode }) {
  return <p className="text-muted-foreground max-w-3xl text-sm leading-6">{children}</p>;
}

function EnrollmentInstructions({
  title,
  description,
  icon,
  command,
  extensions,
  copyLabel,
  link,
  multiline = false,
  notes,
}: {
  title: string;
  description: string;
  icon: ReactNode;
  command: string;
  extensions?: Extension[];
  copyLabel: string;
  link?: {
    href: string;
    label: string;
  };
  multiline?: boolean;
  notes: string[];
}) {
  return (
    <Card className="max-w-full overflow-hidden">
      <CardHeader>
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div className="flex min-w-0 items-start gap-3">
            <div className="bg-muted text-muted-foreground flex size-9 shrink-0 items-center justify-center rounded-md [&_svg]:size-4">
              {icon}
            </div>
            <div className="min-w-0">
              <CardTitle>{title}</CardTitle>
              <CardDescription className="mt-1">{description}</CardDescription>
            </div>
          </div>
          <div className="flex shrink-0 flex-wrap gap-2">
            {link ? (
              <Button type="button" variant="outline" size="sm" asChild>
                <a href={link.href} target="_blank" rel="noreferrer">
                  <ExternalLink data-icon="inline-start" />
                  {link.label}
                </a>
              </Button>
            ) : null}
          </div>
        </div>
      </CardHeader>
      <CardContent className="flex min-w-0 flex-col gap-4">
        <CopyableExample value={command} label={copyLabel} extensions={extensions} multiline={multiline} />
        <ul className="text-muted-foreground flex flex-col gap-1.5 text-sm">
          {notes.map((note) => (
            <li key={note}>{note}</li>
          ))}
        </ul>
      </CardContent>
    </Card>
  );
}

function CopyableExample({
  value,
  label,
  extensions,
  multiline = false,
}: {
  value: string;
  label: string;
  extensions?: Extension[];
  multiline?: boolean;
}) {
  const [copied, setCopied] = useState(false);

  function copy() {
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1200);
    void navigator.clipboard.writeText(value);
  }

  return (
    <div className="relative min-w-0" aria-label="Enrollment example">
      <CodeEditor
        value={value}
        onChange={() => null}
        extensions={extensions}
        readOnly
        lineNumbers={false}
        lineWrapping={multiline}
        highlightActiveLine={false}
        className={cn(
          "[&_.cm-line]:pr-12",
          multiline
            ? "max-h-80 min-h-56 overflow-auto [&_.cm-content]:py-1.5"
            : "min-h-9 [&_.cm-content]:py-2 [&_.cm-scroller]:overflow-x-auto [&_.cm-line]:whitespace-pre",
        )}
      />
      <Button
        type="button"
        variant="ghost"
        size="icon-sm"
        className="absolute right-1 top-1"
        aria-label={label}
        onClick={copy}
      >
        {copied ? <Check /> : <Copy />}
      </Button>
    </div>
  );
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
    <Button type="button" size="icon" variant="ghost" aria-label={label} disabled={disabled} onClick={onClick}>
      {children}
    </Button>
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
        <AlertTitle>Failed to load enrollments</AlertTitle>
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
      header: "Enrollment secret",
      cell: ({ row }) => {
        const visible = Boolean(visibleSecrets[row.original.id]);
        return (
          <span
            className="block max-w-[34rem] truncate font-mono text-xs"
            title={visible ? row.original.value : undefined}
          >
            {visible ? row.original.value : "••••••••••••••••••••••••"}
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
              label={visible ? "Mask enrollment secret" : "Show enrollment secret"}
              disabled={deletingID === row.original.id}
              onClick={() => onToggleVisible(row.original)}
            >
              {visible ? <EyeOff /> : <Eye />}
            </IconAction>
            <IconAction
              label="Copy enrollment secret"
              disabled={deletingID === row.original.id}
              onClick={() => onCopy(row.original)}
            >
              <Copy />
            </IconAction>
            <IconAction
              label="Edit enrollment secret"
              disabled={deletingID === row.original.id}
              onClick={() => onEdit(row.original)}
            >
              <Pencil />
            </IconAction>
            <IconAction
              label="Delete enrollment secret"
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
        <div className="grid gap-2">
          <Label htmlFor="agent-secret-value">Enrollment secret</Label>
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
            {secret ? deleteDescription(secret.agent) : "This secret will stop authenticating integration requests."}
          </AlertDialogDescription>
        </AlertDialogHeader>
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

function integrationLabel(integration?: Integration) {
  if (integration === "santa") return "Santa";
  if (integration === "orbit") return "Orbit";
  return "integration";
}

function deleteDescription(integration: Integration) {
  if (integration === "orbit") {
    return "Future Orbit and osquery enrollment stops for this secret. Existing hosts keep their issued osquery node keys.";
  }
  return "Santa clients using this bearer secret will be rejected until they have another active secret.";
}

function defaultServerURL() {
  if (window.location.hostname === "localhost" && window.location.port === "5173") {
    return "http://localhost:8080";
  }
  return window.location.origin;
}

function orbitPackageCommand(serverURL: string) {
  return ["fleetctl package", "--type=pkg", `--fleet-url=${serverURL}`, "--enroll-secret=REPLACE_WITH_SECRET"].join(
    " ",
  );
}

function santaProfileTemplate(serverURL: string) {
  return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>PayloadType</key>
  <string>com.northpolesec.santa</string>
  <key>PayloadVersion</key>
  <integer>1</integer>
  <key>PayloadIdentifier</key>
  <string>au.edu.vic.woodleigh.woodstar.santa</string>
  <key>PayloadUUID</key>
  <string>896c4448-0b5a-4e0f-9020-51035e9d112a</string>
  <key>PayloadDisplayName</key>
  <string>Woodstar - Santa</string>
  <key>ClientMode</key>
  <integer>1</integer>
  <key>SyncBaseURL</key>
  <string>${serverURL}/santa/sync</string>
  <key>SyncClientContentEncoding</key>
  <string>gzip</string>
  <key>SyncEnableProtoTransfer</key>
  <true/>
  <key>SyncExtraHeaders</key>
  <dict>
    <key>Authorization</key>
    <string>Bearer REPLACE_WITH_SECRET</string>
  </dict>
</dict>
</plist>`;
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
