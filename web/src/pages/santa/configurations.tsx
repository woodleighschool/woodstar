import { useSearch } from "@tanstack/react-router";
import { FileSliders, Loader2, MoreHorizontal, Plus } from "lucide-react";
import type { ReactNode } from "react";
import { useMemo, useState } from "react";
import { toast } from "sonner";

import { DataTableSearch } from "@/components/data-table/data-table-search";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { LabelPicker } from "@/components/santa/label-picker";
import { SortableList } from "@/components/santa/sortable-list";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
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
  const orderedRows = useMemo(
    () => [...(query.data?.items ?? [])].sort((a, b) => a.position - b.position),
    [query.data?.items],
  );

  function saveOrder(next: SantaConfiguration[]) {
    reorder.mutate(
      next.map((row) => row.id),
      { onSuccess: () => toast.success("Configuration order saved") },
    );
  }

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

      <DataTableSearch
        value={draft}
        onChange={setDraft}
        placeholder="Search configurations"
        label="Search configurations"
      />

      {query.error ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to load configurations</AlertTitle>
          <AlertDescription>{query.error.message}</AlertDescription>
        </Alert>
      ) : query.isLoading ? (
        <div className="text-muted-foreground flex items-center gap-2 text-sm">
          <Loader2 className="size-4 animate-spin" /> Loading...
        </div>
      ) : orderedRows.length === 0 ? (
        <Empty>
          <EmptyHeader>
            <EmptyMedia variant="icon">
              <FileSliders />
            </EmptyMedia>
            <EmptyTitle>No configurations</EmptyTitle>
            <EmptyDescription>Add a Santa configuration to start sending client settings.</EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : (
        <SortableList
          items={orderedRows}
          onChange={saveOrder}
          renderItem={(row) => (
            <ConfigurationRow
              configuration={row}
              pending={reorder.isPending || remove.isPending}
              onEdit={() => setEditing(row)}
              onDelete={() => remove.mutate(row.id)}
            />
          )}
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
    </PageShell>
  );
}

function ConfigurationRow({
  configuration,
  pending,
  onEdit,
  onDelete,
}: {
  configuration: SantaConfiguration;
  pending: boolean;
  onEdit: () => void;
  onDelete: () => void;
}) {
  return (
    <div className="grid min-w-0 gap-2 sm:grid-cols-[1fr_auto] sm:items-center">
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-2">
          <span className="font-medium">{configuration.name}</span>
          <Badge variant="secondary">{configuration.client_mode}</Badge>
          <span className="text-muted-foreground text-xs tabular-nums">
            {configuration.label_ids?.length ?? 0} labels
          </span>
        </div>
        <div className="text-muted-foreground text-xs" title={new Date(configuration.updated_at).toLocaleString()}>
          Updated {formatRelative(configuration.updated_at)}
        </div>
      </div>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button type="button" variant="ghost" size="icon" disabled={pending}>
            <MoreHorizontal />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuGroup>
            <DropdownMenuItem onSelect={onEdit}>Edit</DropdownMenuItem>
            <DropdownMenuItem variant="destructive" onSelect={onDelete}>
              Delete
            </DropdownMenuItem>
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
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
    removable_media_action: configuration.removable_media_action ?? "omit",
    removable_media_remount_flags: (configuration.removable_media_remount_flags ?? []).join(" "),
    encrypted_removable_media_action: configuration.encrypted_removable_media_action ?? "omit",
    encrypted_removable_media_remount_flags: (configuration.encrypted_removable_media_remount_flags ?? []).join(" "),
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
    removable_media_action:
      optionalText(form.removable_media_action) === "omit" ? undefined : form.removable_media_action,
    removable_media_remount_flags: splitWords(form.removable_media_remount_flags),
    encrypted_removable_media_action:
      optionalText(form.encrypted_removable_media_action) === "omit"
        ? undefined
        : form.encrypted_removable_media_action,
    encrypted_removable_media_remount_flags: splitWords(form.encrypted_removable_media_remount_flags),
    event_detail_url: optionalText(form.event_detail_url),
    event_detail_text: optionalText(form.event_detail_text),
  };
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
