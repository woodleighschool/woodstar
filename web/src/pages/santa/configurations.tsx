import { Link, useNavigate, useParams, useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { ArrowDown, ArrowUp, FileSliders, Loader2, MoreHorizontal, Plus } from "lucide-react";
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Field, FieldDescription, FieldGroup, FieldLabel, FieldLegend, FieldSet } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import {
  useCreateSantaConfiguration,
  useDeleteSantaConfiguration,
  useReorderSantaConfigurations,
  useSantaConfiguration,
  useSantaConfigurations,
  useUpdateSantaConfiguration,
  type SantaConfiguration,
  type SantaConfigurationMutation,
} from "@/hooks/use-santa";
import type { ApiError } from "@/lib/api";
import { formatRelative } from "@/lib/utils";

type BoolChoice = "omit" | "true" | "false";
type MediaActionChoice = "omit" | "allow" | "block" | "remount";

interface ConfigurationFormState {
  name: string;
  client_mode: SantaConfigurationMutation["client_mode"];
  label_ids: number[];
  enable_bundles: BoolChoice;
  enable_transitive_rules: BoolChoice;
  enable_all_event_upload: BoolChoice;
  full_sync_interval_seconds: string;
  batch_size: string;
  allowed_path_regex: string;
  blocked_path_regex: string;
  removable_media_action: MediaActionChoice;
  removable_media_remount_flags: string;
  encrypted_removable_media_action: MediaActionChoice;
  encrypted_removable_media_remount_flags: string;
  event_detail_url: string;
  event_detail_text: string;
}

const CLIENT_MODE_OPTIONS: { value: NonNullable<SantaConfigurationMutation["client_mode"]>; label: string }[] = [
  { value: "monitor", label: "Monitor" },
  { value: "lockdown", label: "Lockdown" },
  { value: "standalone", label: "Standalone" },
];

const BOOL_OPTIONS: { value: BoolChoice; label: string; description: string }[] = [
  { value: "omit", label: "Use default", description: "Do not override Santa's default behavior." },
  { value: "true", label: "Enabled", description: "Send an explicit enabled value to clients." },
  { value: "false", label: "Disabled", description: "Send an explicit disabled value to clients." },
];

const MEDIA_ACTION_OPTIONS: { value: MediaActionChoice; label: string }[] = [
  { value: "omit", label: "No policy" },
  { value: "allow", label: "Allow" },
  { value: "block", label: "Block" },
  { value: "remount", label: "Remount" },
];

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
  const query = useSantaConfigurations({ q: typeof search.q === "string" ? search.q : undefined, page_size: 500 });
  const remove = useDeleteSantaConfiguration();
  const reorder = useReorderSantaConfigurations();
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
          <Badge variant="secondary">{clientModeLabel(row.original.client_mode)}</Badge>
        </div>
      ),
    },
    {
      id: "labels",
      header: "Targets",
      enableSorting: false,
      cell: ({ row }) => (
        <span className="text-muted-foreground text-sm tabular-nums">
          {row.original.label_ids?.length ?? 0} label{row.original.label_ids?.length === 1 ? "" : "s"}
        </span>
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
            configuration={row.original}
            pending={reorder.isPending || remove.isPending}
            canMoveUp={index > 0}
            canMoveDown={index >= 0 && index < orderedRows.length - 1}
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
          <Button asChild size="sm">
            <Link to="/santa/configurations/new">
              <Plus data-icon="inline-start" />
              Add configuration
            </Link>
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
          pagination={{ pageIndex: 0, pageSize: orderedRows.length || 50 }}
          sorting={[]}
          onPaginationChange={() => undefined}
          onSortingChange={() => undefined}
          isLoading={query.isLoading}
          clientSort
          rowHref={(row) => ({
            to: "/santa/configurations/$configurationId/edit",
            params: { configurationId: String(row.id) },
          })}
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
  configuration,
  pending,
  canMoveUp,
  canMoveDown,
  onMoveUp,
  onMoveDown,
  onDelete,
}: {
  configuration: SantaConfiguration;
  pending: boolean;
  canMoveUp: boolean;
  canMoveDown: boolean;
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
          <DropdownMenuItem asChild>
            <Link
              to="/santa/configurations/$configurationId/edit"
              params={{ configurationId: String(configuration.id) }}
            >
              Edit
            </Link>
          </DropdownMenuItem>
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

export function SantaConfigurationEditPage({ mode }: { mode: "create" | "edit" }) {
  const params = useParams({ strict: false });
  const configurationId = params.configurationId ?? "";
  const detail = useSantaConfiguration(configurationId);

  if (mode === "edit") {
    if (detail.error) {
      return (
        <PageShell>
          <Alert variant="destructive">
            <AlertTitle>Failed to load configuration</AlertTitle>
            <AlertDescription>{detail.error.message}</AlertDescription>
          </Alert>
        </PageShell>
      );
    }
    if (!detail.data) {
      return (
        <PageShell className="text-muted-foreground flex-row items-center gap-2 text-sm">
          <Loader2 className="animate-spin" /> Loading configuration...
        </PageShell>
      );
    }
  }

  const initial = mode === "edit" && detail.data ? formFromConfiguration(detail.data) : emptyConfigurationForm;

  return (
    <ConfigurationForm key={configurationId || "new"} mode={mode} configurationId={configurationId} initial={initial} />
  );
}

function ConfigurationForm({
  mode,
  configurationId,
  initial,
}: {
  mode: "create" | "edit";
  configurationId: string;
  initial: ConfigurationFormState;
}) {
  const navigate = useNavigate();
  const create = useCreateSantaConfiguration();
  const update = useUpdateSantaConfiguration();
  const [form, setForm] = useState<ConfigurationFormState>(initial);
  const error = configurationError(create.error ?? update.error);
  const pending = create.isPending || update.isPending;

  async function submit() {
    const body = configurationBody(form);
    if (mode === "create") await create.mutateAsync(body);
    else await update.mutateAsync({ id: Number(configurationId), body });
    void navigate({ to: "/santa/configurations" });
  }

  return (
    <PageShell asChild>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          void submit();
        }}
      >
        <PageHeader
          title={mode === "create" ? "New Santa configuration" : "Edit Santa configuration"}
          description="Set Santa client behavior and assign the configuration to labels."
          actions={
            <>
              <Button asChild type="button" variant="outline" size="sm">
                <Link to="/santa/configurations">Cancel</Link>
              </Button>
              <Button type="submit" size="sm" disabled={pending || form.name.trim() === ""}>
                {pending ? <Loader2 data-icon="inline-start" className="animate-spin" /> : null}
                Save
              </Button>
            </>
          }
        />

        {error ? (
          <Alert variant="destructive">
            <AlertTitle>Unable to save configuration</AlertTitle>
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        ) : null}

        <FieldGroup className="max-w-4xl">
          <FieldSet>
            <FieldLegend>Basics</FieldLegend>
            <div className="grid gap-4 md:grid-cols-2">
              <Field>
                <FieldLabel htmlFor="santa-configuration-name">Name</FieldLabel>
                <Input
                  id="santa-configuration-name"
                  required
                  value={form.name}
                  onChange={(event) => setForm({ ...form, name: event.target.value })}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="santa-client-mode">Client mode</FieldLabel>
                <Select
                  value={form.client_mode}
                  onValueChange={(client_mode) =>
                    setForm({
                      ...form,
                      client_mode: client_mode as SantaConfigurationMutation["client_mode"],
                    })
                  }
                >
                  <SelectTrigger id="santa-client-mode" className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {CLIENT_MODE_OPTIONS.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
                <FieldDescription>Monitor observes; lockdown enforces Santa rules.</FieldDescription>
              </Field>
            </div>
          </FieldSet>

          <FieldSet>
            <FieldLegend>Sync</FieldLegend>
            <div className="grid gap-4 md:grid-cols-3">
              <BoolField
                id="santa-enable-bundles"
                label="Bundles"
                value={form.enable_bundles}
                onChange={(enable_bundles) => setForm({ ...form, enable_bundles })}
              />
              <BoolField
                id="santa-enable-transitive-rules"
                label="Transitive rules"
                value={form.enable_transitive_rules}
                onChange={(enable_transitive_rules) => setForm({ ...form, enable_transitive_rules })}
              />
              <BoolField
                id="santa-upload-all-events"
                label="Upload all events"
                value={form.enable_all_event_upload}
                onChange={(enable_all_event_upload) => setForm({ ...form, enable_all_event_upload })}
              />
              <Field>
                <FieldLabel htmlFor="santa-full-sync-interval">Full sync interval</FieldLabel>
                <Input
                  id="santa-full-sync-interval"
                  type="number"
                  min={60}
                  inputMode="numeric"
                  value={form.full_sync_interval_seconds}
                  onChange={(event) => setForm({ ...form, full_sync_interval_seconds: event.target.value })}
                />
                <FieldDescription>Seconds between clean syncs.</FieldDescription>
              </Field>
              <Field>
                <FieldLabel htmlFor="santa-batch-size">Batch size</FieldLabel>
                <Input
                  id="santa-batch-size"
                  type="number"
                  min={1}
                  inputMode="numeric"
                  value={form.batch_size}
                  onChange={(event) => setForm({ ...form, batch_size: event.target.value })}
                />
                <FieldDescription>Maximum rule rows per download page.</FieldDescription>
              </Field>
            </div>
          </FieldSet>

          <FieldSet>
            <FieldLegend>Execution</FieldLegend>
            <div className="grid gap-4 md:grid-cols-2">
              <Field>
                <FieldLabel htmlFor="santa-allowed-path-regex">Allowed path regex</FieldLabel>
                <Input
                  id="santa-allowed-path-regex"
                  value={form.allowed_path_regex}
                  onChange={(event) => setForm({ ...form, allowed_path_regex: event.target.value })}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="santa-blocked-path-regex">Blocked path regex</FieldLabel>
                <Input
                  id="santa-blocked-path-regex"
                  value={form.blocked_path_regex}
                  onChange={(event) => setForm({ ...form, blocked_path_regex: event.target.value })}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="santa-event-detail-url">Event detail URL</FieldLabel>
                <Input
                  id="santa-event-detail-url"
                  value={form.event_detail_url}
                  onChange={(event) => setForm({ ...form, event_detail_url: event.target.value })}
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="santa-event-detail-text">Event detail text</FieldLabel>
                <Input
                  id="santa-event-detail-text"
                  value={form.event_detail_text}
                  onChange={(event) => setForm({ ...form, event_detail_text: event.target.value })}
                />
              </Field>
            </div>
          </FieldSet>

          <FieldSet>
            <FieldLegend>Removable Media</FieldLegend>
            <div className="grid gap-4 md:grid-cols-2">
              <MediaActionField
                id="santa-removable-media"
                label="Removable media"
                action={form.removable_media_action}
                flags={form.removable_media_remount_flags}
                onActionChange={(removable_media_action) => setForm({ ...form, removable_media_action })}
                onFlagsChange={(removable_media_remount_flags) => setForm({ ...form, removable_media_remount_flags })}
              />
              <MediaActionField
                id="santa-encrypted-removable-media"
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
            </div>
          </FieldSet>

          <FieldSet>
            <FieldLegend>Targets</FieldLegend>
            <Field>
              <FieldLabel>Labels</FieldLabel>
              <LabelPicker value={form.label_ids} onChange={(label_ids) => setForm({ ...form, label_ids })} />
              <FieldDescription>
                Configurations are evaluated in list order; each label can belong to one configuration.
              </FieldDescription>
            </Field>
          </FieldSet>
        </FieldGroup>

        <div className="flex max-w-4xl items-center gap-2 border-t pt-4">
          <Button type="submit" size="sm" disabled={pending || form.name.trim() === ""}>
            {pending ? <Loader2 data-icon="inline-start" className="animate-spin" /> : null}
            Save
          </Button>
          <Button asChild type="button" variant="ghost" size="sm">
            <Link to="/santa/configurations">Cancel</Link>
          </Button>
        </div>
      </form>
    </PageShell>
  );
}

function BoolField({
  id,
  label,
  value,
  onChange,
}: {
  id: string;
  label: string;
  value: BoolChoice;
  onChange: (value: BoolChoice) => void;
}) {
  const option = BOOL_OPTIONS.find((item) => item.value === value);
  return (
    <Field>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <Select value={value} onValueChange={(next) => onChange(next as BoolChoice)}>
        <SelectTrigger id={id} className="w-full">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            {BOOL_OPTIONS.map((item) => (
              <SelectItem key={item.value} value={item.value}>
                {item.label}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
      {option ? <FieldDescription>{option.description}</FieldDescription> : null}
    </Field>
  );
}

function MediaActionField({
  id,
  label,
  action,
  flags,
  onActionChange,
  onFlagsChange,
}: {
  id: string;
  label: string;
  action: MediaActionChoice;
  flags: string;
  onActionChange: (value: MediaActionChoice) => void;
  onFlagsChange: (value: string) => void;
}) {
  return (
    <Field>
      <FieldLabel htmlFor={`${id}-action`}>{label}</FieldLabel>
      <div className="grid gap-2 sm:grid-cols-2">
        <Select value={action} onValueChange={(value) => onActionChange(value as MediaActionChoice)}>
          <SelectTrigger id={`${id}-action`} className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              {MEDIA_ACTION_OPTIONS.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
        <Input
          id={`${id}-flags`}
          placeholder="remount flags"
          disabled={action !== "remount"}
          value={flags}
          onChange={(event) => onFlagsChange(event.target.value)}
        />
      </div>
      <FieldDescription>Remount requires one or more mount flags.</FieldDescription>
    </Field>
  );
}

function formFromConfiguration(configuration: SantaConfiguration): ConfigurationFormState {
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

function removableMediaPolicyBody(action: MediaActionChoice, flags: string) {
  if (action === "omit") return undefined;
  return { action, remount_flags: splitWords(flags) };
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

function clientModeLabel(mode: SantaConfiguration["client_mode"]) {
  return CLIENT_MODE_OPTIONS.find((option) => option.value === mode)?.label ?? mode;
}
