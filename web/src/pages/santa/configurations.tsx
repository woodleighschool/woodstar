import { useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { ArrowDown, ArrowUp, FileSliders, Loader2, MoreHorizontal, Plus } from "lucide-react";
import type { ReactNode } from "react";
import { useMemo, useState } from "react";
import { toast } from "sonner";

import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableSearch } from "@/components/data-table/data-table-search";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { LabelPicker } from "@/components/santa/label-picker";
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
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import {
  useCreateSantaConfiguration,
  useDeleteSantaConfiguration,
  useReorderSantaConfigurations,
  useSantaConfigurations,
  useUpdateSantaConfiguration,
  type SantaConfiguration,
  type SantaConfigurationMutation,
} from "@/hooks/use-santa";
import type { ApiError } from "@/lib/api";
import { formatRelative } from "@/lib/utils";

type BoolChoice = "omit" | "true" | "false";

interface ConfigurationFormState {
  name: string;
  client_mode: string;
  label_ids: number[];
  enable_bundles: BoolChoice;
  enable_transitive_rules: BoolChoice;
  enable_all_event_upload: BoolChoice;
  full_sync_interval_seconds: string;
  batch_size: string;
  allowed_path_regex: string;
  blocked_path_regex: string;
  removable_media_action: string;
  removable_media_remount_flags: string;
  encrypted_removable_media_action: string;
  encrypted_removable_media_remount_flags: string;
  event_detail_url: string;
  event_detail_text: string;
}

const emptyConfigurationForm: ConfigurationFormState = {
  name: "",
  client_mode: "monitor",
  label_ids: [],
  enable_bundles: "omit",
  enable_transitive_rules: "omit",
  enable_all_event_upload: "omit",
  full_sync_interval_seconds: "",
  batch_size: "",
  allowed_path_regex: "",
  blocked_path_regex: "",
  removable_media_action: "omit",
  removable_media_remount_flags: "",
  encrypted_removable_media_action: "omit",
  encrypted_removable_media_remount_flags: "",
  event_detail_url: "",
  event_detail_text: "",
};

export function SantaConfigurationsPage() {
  const search = useSearch({ strict: false });
  const [draft, setDraft] = useDebouncedSearchParam("q");
  const query = useSantaConfigurations({ q: typeof search.q === "string" ? search.q : undefined, per_page: 500 });
  const create = useCreateSantaConfiguration();
  const update = useUpdateSantaConfiguration();
  const remove = useDeleteSantaConfiguration();
  const reorder = useReorderSantaConfigurations();
  const [editing, setEditing] = useState<SantaConfiguration | "new" | null>(null);
  const [deleting, setDeleting] = useState<SantaConfiguration | null>(null);
  const orderedRows = useMemo(
    () => [...(query.data?.items ?? [])].sort((a, b) => a.position - b.position),
    [query.data?.items],
  );
  const hasFilters = !!search.q;

  function saveOrder(next: SantaConfiguration[]) {
    reorder.mutate(
      next.map((row) => row.id),
      { onSuccess: () => toast.success("Configuration order saved") },
    );
  }

  function moveConfiguration(configuration: SantaConfiguration, direction: "up" | "down") {
    const index = orderedRows.findIndex((row) => row.id === configuration.id);
    const targetIndex = direction === "up" ? index - 1 : index + 1;
    if (index < 0 || targetIndex < 0 || targetIndex >= orderedRows.length) return;

    const next = [...orderedRows];
    [next[index], next[targetIndex]] = [next[targetIndex], next[index]];
    saveOrder(next);
  }

  const columns: ColumnDef<SantaConfiguration>[] = [
    {
      id: "position",
      accessorKey: "position",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Order" />,
      cell: ({ row }) => <span className="text-muted-foreground tabular-nums">{row.original.position}</span>,
      meta: { headClassName: "w-24" },
    },
    {
      id: "name",
      accessorKey: "name",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
      cell: ({ row }) => (
        <div className="flex flex-wrap items-center gap-2">
          <span className="font-medium">{row.original.name}</span>
          <Badge variant="secondary">{row.original.client_mode}</Badge>
        </div>
      ),
    },
    {
      id: "labels",
      header: "Labels",
      enableSorting: false,
      cell: ({ row }) => (
        <span className="text-muted-foreground text-sm tabular-nums">{row.original.label_ids?.length ?? 0}</span>
      ),
    },
    {
      id: "updated_at",
      accessorKey: "updated_at",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Updated" />,
      cell: ({ row }) => (
        <span className="text-muted-foreground" title={new Date(row.original.updated_at).toLocaleString()}>
          {formatRelative(row.original.updated_at)}
        </span>
      ),
    },
    {
      id: "actions",
      header: () => null,
      enableSorting: false,
      cell: ({ row }) => {
        const index = orderedRows.findIndex((configuration) => configuration.id === row.original.id);
        return (
          <ConfigurationRowActions
            pending={reorder.isPending || remove.isPending}
            canMoveUp={index > 0}
            canMoveDown={index >= 0 && index < orderedRows.length - 1}
            onEdit={() => setEditing(row.original)}
            onMoveUp={() => moveConfiguration(row.original, "up")}
            onMoveDown={() => moveConfiguration(row.original, "down")}
            onDelete={() => setDeleting(row.original)}
          />
        );
      },
      meta: { headClassName: "w-12" },
    },
  ];

  return (
    <PageShell>
      <PageHeader
        title="Santa configurations"
        description="Resolve host settings top-down by label membership."
        actions={
          <Button size="sm" onClick={() => setEditing("new")}>
            <Plus data-icon="inline-start" />
            Add configuration
          </Button>
        }
      />

      {query.error ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to load configurations</AlertTitle>
          <AlertDescription>{query.error.message}</AlertDescription>
        </Alert>
      ) : (
        <DataTable
          columns={columns}
          data={orderedRows}
          totalCount={orderedRows.length}
          page={1}
          perPage={orderedRows.length || 50}
          sort={{}}
          onPageChange={() => undefined}
          onPerPageChange={() => undefined}
          onSortChange={() => undefined}
          isLoading={query.isLoading}
          clientSort
          onRowClick={setEditing}
          toolbar={
            <div className="flex items-center gap-2">
              <DataTableSearch
                value={draft}
                onChange={setDraft}
                placeholder="Search configurations"
                label="Search configurations"
              />
            </div>
          }
          empty={
            <Empty>
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <FileSliders />
                </EmptyMedia>
                <EmptyTitle>{hasFilters ? "No matches" : "No configurations yet"}</EmptyTitle>
                <EmptyDescription>
                  {hasFilters
                    ? "No Santa configurations matched the current filters."
                    : "Add a Santa configuration to start sending client settings."}
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
          }
        />
      )}

      <ConfigurationDialog
        key={editing === "new" ? "new" : (editing?.id ?? "closed")}
        open={editing !== null}
        configuration={editing === "new" ? null : editing}
        pending={create.isPending || update.isPending}
        error={configurationError(create.error ?? update.error)}
        onOpenChange={(open) => {
          if (!open) {
            create.reset();
            update.reset();
            setEditing(null);
          }
        }}
        onSubmit={async (body) => {
          if (editing === "new") await create.mutateAsync(body);
          else if (editing) await update.mutateAsync({ id: editing.id, body });
          setEditing(null);
        }}
      />
      <ConfigurationDeleteDialog
        configuration={deleting}
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

function ConfigurationRowActions({
  pending,
  canMoveUp,
  canMoveDown,
  onEdit,
  onMoveUp,
  onMoveDown,
  onDelete,
}: {
  pending: boolean;
  canMoveUp: boolean;
  canMoveDown: boolean;
  onEdit: () => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
  onDelete: () => void;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button type="button" variant="ghost" size="icon" disabled={pending}>
          <MoreHorizontal />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuGroup>
          <DropdownMenuItem onSelect={onEdit}>Edit</DropdownMenuItem>
          <DropdownMenuItem disabled={!canMoveUp || pending} onSelect={onMoveUp}>
            <ArrowUp />
            Move up
          </DropdownMenuItem>
          <DropdownMenuItem disabled={!canMoveDown || pending} onSelect={onMoveDown}>
            <ArrowDown />
            Move down
          </DropdownMenuItem>
          <DropdownMenuItem variant="destructive" onSelect={onDelete}>
            Delete
          </DropdownMenuItem>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function ConfigurationDeleteDialog({
  configuration,
  open,
  pending,
  error,
  onOpenChange,
  onConfirm,
}: {
  configuration: SantaConfiguration | null;
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
          <AlertDialogTitle>Delete Santa configuration?</AlertDialogTitle>
          <AlertDialogDescription>
            {configuration
              ? `${configuration.name} will stop applying to matching hosts.`
              : "This Santa configuration will stop applying to matching hosts."}
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

function ConfigurationDialog({
  open,
  configuration,
  pending,
  error,
  onOpenChange,
  onSubmit,
}: {
  open: boolean;
  configuration: SantaConfiguration | null;
  pending: boolean;
  error?: string;
  onOpenChange: (open: boolean) => void;
  onSubmit: (body: SantaConfigurationMutation) => Promise<void>;
}) {
  const [form, setForm] = useState(() => formFromConfiguration(configuration));

  async function submit() {
    await onSubmit(configurationBody(form));
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>{configuration ? "Edit Santa configuration" : "New Santa configuration"}</DialogTitle>
        </DialogHeader>
        {error ? (
          <Alert variant="destructive">
            <AlertTitle>Unable to save configuration</AlertTitle>
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        ) : null}
        <div className="grid gap-4 md:grid-cols-2">
          <Field label="Name">
            <Input value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} />
          </Field>
          <Field label="Client mode">
            <Select value={form.client_mode} onValueChange={(client_mode) => setForm({ ...form, client_mode })}>
              <SelectTrigger className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="monitor">monitor</SelectItem>
                <SelectItem value="lockdown">lockdown</SelectItem>
                <SelectItem value="standalone">standalone</SelectItem>
              </SelectContent>
            </Select>
          </Field>
          <BoolField
            label="Enable bundles"
            value={form.enable_bundles}
            onChange={(enable_bundles) => setForm({ ...form, enable_bundles })}
          />
          <BoolField
            label="Enable transitive rules"
            value={form.enable_transitive_rules}
            onChange={(enable_transitive_rules) => setForm({ ...form, enable_transitive_rules })}
          />
          <BoolField
            label="Upload all events"
            value={form.enable_all_event_upload}
            onChange={(enable_all_event_upload) => setForm({ ...form, enable_all_event_upload })}
          />
          <Field label="Full sync interval seconds">
            <Input
              type="number"
              min={60}
              value={form.full_sync_interval_seconds}
              onChange={(event) => setForm({ ...form, full_sync_interval_seconds: event.target.value })}
            />
          </Field>
          <Field label="Batch size">
            <Input
              type="number"
              min={1}
              value={form.batch_size}
              onChange={(event) => setForm({ ...form, batch_size: event.target.value })}
            />
          </Field>
          <Field label="Allowed path regex">
            <Input
              value={form.allowed_path_regex}
              onChange={(event) => setForm({ ...form, allowed_path_regex: event.target.value })}
            />
          </Field>
          <Field label="Blocked path regex">
            <Input
              value={form.blocked_path_regex}
              onChange={(event) => setForm({ ...form, blocked_path_regex: event.target.value })}
            />
          </Field>
          <ActionField
            label="Removable media"
            action={form.removable_media_action}
            flags={form.removable_media_remount_flags}
            onActionChange={(removable_media_action) => setForm({ ...form, removable_media_action })}
            onFlagsChange={(removable_media_remount_flags) => setForm({ ...form, removable_media_remount_flags })}
          />
          <ActionField
            label="Encrypted removable media"
            action={form.encrypted_removable_media_action}
            flags={form.encrypted_removable_media_remount_flags}
            onActionChange={(encrypted_removable_media_action) =>
              setForm({ ...form, encrypted_removable_media_action })
            }
            onFlagsChange={(encrypted_removable_media_remount_flags) =>
              setForm({ ...form, encrypted_removable_media_remount_flags })
            }
          />
          <Field label="Event detail URL">
            <Input
              value={form.event_detail_url}
              onChange={(event) => setForm({ ...form, event_detail_url: event.target.value })}
            />
          </Field>
          <Field label="Event detail text">
            <Input
              value={form.event_detail_text}
              onChange={(event) => setForm({ ...form, event_detail_text: event.target.value })}
            />
          </Field>
          <div className="grid gap-2 md:col-span-2">
            <Label>Target labels</Label>
            <LabelPicker value={form.label_ids} onChange={(label_ids) => setForm({ ...form, label_ids })} />
          </div>
        </div>
        <DialogFooter>
          <Button type="button" variant="ghost" size="sm" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="button" size="sm" disabled={pending || form.name.trim() === ""} onClick={() => void submit()}>
            {pending ? <Loader2 data-icon="inline-start" className="animate-spin" /> : null}
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="grid gap-2">
      <Label>{label}</Label>
      {children}
    </div>
  );
}

function BoolField({
  label,
  value,
  onChange,
}: {
  label: string;
  value: BoolChoice;
  onChange: (value: BoolChoice) => void;
}) {
  return (
    <Field label={label}>
      <Select value={value} onValueChange={(next) => onChange(next as BoolChoice)}>
        <SelectTrigger className="w-full">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="omit">omit</SelectItem>
          <SelectItem value="true">true</SelectItem>
          <SelectItem value="false">false</SelectItem>
        </SelectContent>
      </Select>
    </Field>
  );
}

function ActionField({
  label,
  action,
  flags,
  onActionChange,
  onFlagsChange,
}: {
  label: string;
  action: string;
  flags: string;
  onActionChange: (value: string) => void;
  onFlagsChange: (value: string) => void;
}) {
  return (
    <div className="grid gap-2">
      <Label>{label}</Label>
      <div className="grid gap-2 sm:grid-cols-2">
        <Select value={action} onValueChange={onActionChange}>
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="omit">omit</SelectItem>
            <SelectItem value="allow">allow</SelectItem>
            <SelectItem value="block">block</SelectItem>
            <SelectItem value="remount">remount</SelectItem>
          </SelectContent>
        </Select>
        <Input placeholder="remount flags" value={flags} onChange={(event) => onFlagsChange(event.target.value)} />
      </div>
    </div>
  );
}

function formFromConfiguration(configuration: SantaConfiguration | null): ConfigurationFormState {
  if (!configuration) return { ...emptyConfigurationForm };
  return {
    name: configuration.name,
    client_mode: configuration.client_mode,
    label_ids: configuration.label_ids ?? [],
    enable_bundles: boolChoice(configuration.enable_bundles),
    enable_transitive_rules: boolChoice(configuration.enable_transitive_rules),
    enable_all_event_upload: boolChoice(configuration.enable_all_event_upload),
    full_sync_interval_seconds: configuration.full_sync_interval_seconds?.toString() ?? "",
    batch_size: configuration.batch_size?.toString() ?? "",
    allowed_path_regex: configuration.allowed_path_regex ?? "",
    blocked_path_regex: configuration.blocked_path_regex ?? "",
    removable_media_action: configuration.removable_media_policy?.action ?? "omit",
    removable_media_remount_flags: (configuration.removable_media_policy?.remount_flags ?? []).join(" "),
    encrypted_removable_media_action: configuration.encrypted_removable_media_policy?.action ?? "omit",
    encrypted_removable_media_remount_flags: (configuration.encrypted_removable_media_policy?.remount_flags ?? []).join(
      " ",
    ),
    event_detail_url: configuration.event_detail_url ?? "",
    event_detail_text: configuration.event_detail_text ?? "",
  };
}

function configurationBody(form: ConfigurationFormState): SantaConfigurationMutation {
  return {
    name: form.name.trim(),
    client_mode: form.client_mode,
    label_ids: form.label_ids,
    enable_bundles: optionalBool(form.enable_bundles),
    enable_transitive_rules: optionalBool(form.enable_transitive_rules),
    enable_all_event_upload: optionalBool(form.enable_all_event_upload),
    full_sync_interval_seconds: optionalNumber(form.full_sync_interval_seconds),
    batch_size: optionalNumber(form.batch_size),
    allowed_path_regex: optionalText(form.allowed_path_regex),
    blocked_path_regex: optionalText(form.blocked_path_regex),
    removable_media_policy: removableMediaPolicyBody(form.removable_media_action, form.removable_media_remount_flags),
    encrypted_removable_media_policy: removableMediaPolicyBody(
      form.encrypted_removable_media_action,
      form.encrypted_removable_media_remount_flags,
    ),
    event_detail_url: optionalText(form.event_detail_url),
    event_detail_text: optionalText(form.event_detail_text),
  };
}

function removableMediaPolicyBody(action: string, flags: string) {
  const cleanedAction = optionalText(action);
  if (!cleanedAction || cleanedAction === "omit") return undefined;
  return { action: cleanedAction, remount_flags: splitWords(flags) };
}

function boolChoice(value: boolean | undefined): BoolChoice {
  if (value === true) return "true";
  if (value === false) return "false";
  return "omit";
}

function optionalBool(value: BoolChoice) {
  if (value === "true") return true;
  if (value === "false") return false;
  return undefined;
}

function optionalText(value: string) {
  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
}

function optionalNumber(value: string) {
  const trimmed = value.trim();
  return trimmed === "" ? undefined : Number(trimmed);
}

function splitWords(value: string) {
  return value
    .split(/[\s,]+/)
    .map((part) => part.trim())
    .filter(Boolean);
}

function configurationError(error: ApiError | null) {
  if (!error) return undefined;
  if (error.body && typeof error.body === "object" && "configuration_name" in error.body) {
    const body = error.body as { configuration_name?: string };
    if (body.configuration_name) return `Label already belongs to ${body.configuration_name}.`;
  }
  return error.message;
}
