import { Link, useNavigate, useParams, useSearch } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { FileSliders, Loader2, Plus, Trash2 } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { BulkDeleteDialog } from "@/components/data-table/bulk-delete-dialog";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableSearch } from "@/components/data-table/data-table-search";
import { DraggableDataTable, DraggableDataTableRowDragHandle } from "@/components/data-table/draggable-data-table";
import { labelsFromIDs, type LabelChip } from "@/components/labels/label-chip-utils";
import { LabelChips } from "@/components/labels/label-chips";
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
import { ButtonGroup } from "@/components/ui/button-group";
import { Checkbox } from "@/components/ui/checkbox";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Field, FieldDescription, FieldGroup, FieldLabel, FieldLegend, FieldSet } from "@/components/ui/field";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import { useLabels } from "@/hooks/use-labels";
import {
  useBulkDeleteSantaConfigurations,
  useCreateSantaConfiguration,
  useReorderSantaConfigurations,
  useSantaConfiguration,
  useSantaConfigurations,
  useUpdateSantaConfiguration,
  type SantaConfiguration,
  type SantaConfigurationMutation,
} from "@/hooks/use-santa";
import type { ApiError } from "@/lib/api";
import { MAX_PAGE_SIZE } from "@/lib/pagination";
import { formatRelative } from "@/lib/utils";

type MediaAction = "none" | "allow" | "block" | "remount";

interface ConfigurationFormState {
  name: string;
  client_mode: SantaConfigurationMutation["client_mode"];
  label_ids: number[];
  enable_bundles: boolean;
  enable_transitive_rules: boolean;
  enable_all_event_upload: boolean;
  full_sync_interval_seconds: number;
  batch_size: number;
  allowed_path_regex: string;
  blocked_path_regex: string;
  removable_media_action: MediaAction;
  removable_media_remount_flags: string;
  encrypted_removable_media_action: MediaAction;
  encrypted_removable_media_remount_flags: string;
  event_detail_url: string;
  event_detail_text: string;
}

const CLIENT_MODE_OPTIONS: { value: NonNullable<SantaConfigurationMutation["client_mode"]>; label: string }[] = [
  { value: "monitor", label: "Monitor" },
  { value: "lockdown", label: "Lockdown" },
  { value: "standalone", label: "Standalone" },
];

const MEDIA_ACTION_OPTIONS: { value: MediaAction; label: string }[] = [
  { value: "none", label: "No policy" },
  { value: "allow", label: "Allow" },
  { value: "block", label: "Block" },
  { value: "remount", label: "Remount" },
];

// Santa client defaults sourced from upstream Santa. The form pre-fills these
// so the backend never substitutes hidden defaults.
const emptyConfigurationForm: ConfigurationFormState = {
  name: "",
  client_mode: "monitor",
  label_ids: [],
  enable_bundles: false,
  enable_transitive_rules: false,
  enable_all_event_upload: false,
  full_sync_interval_seconds: 600,
  batch_size: 50,
  allowed_path_regex: "",
  blocked_path_regex: "",
  removable_media_action: "none",
  removable_media_remount_flags: "",
  encrypted_removable_media_action: "none",
  encrypted_removable_media_remount_flags: "",
  event_detail_url: "",
  event_detail_text: "",
};

export function SantaConfigurationsPage() {
  const search = useSearch({ strict: false });
  const [draft, setDraft] = useDebouncedSearchParam("q");
  const query = useSantaConfigurations({
    q: typeof search.q === "string" ? search.q : undefined,
    page_size: MAX_PAGE_SIZE,
  });
  const labels = useLabels({
    page_size: MAX_PAGE_SIZE,
    sort: "name.asc",
    label_type: "regular",
    platform: "darwin",
  });
  const bulkDelete = useBulkDeleteSantaConfigurations();
  const reorder = useReorderSantaConfigurations();
  const [selectedConfigurationIds, setSelectedConfigurationIds] = useState<string[]>([]);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [reorderWarningOpen, setReorderWarningOpen] = useState(false);
  const [reorderEnabled, setReorderEnabled] = useState(false);
  const serverRows = useMemo(
    () => [...(query.data?.items ?? [])].sort((a, b) => a.position - b.position),
    [query.data?.items],
  );
  const labelsByID = useMemo<ReadonlyMap<number, LabelChip>>(
    () => new Map((labels.data?.items ?? []).map((label) => [label.id, label])),
    [labels.data?.items],
  );
  const [orderedRows, setOrderedRows] = useState<SantaConfiguration[]>([]);
  const hasFilters = !!search.q;
  const canEnableReorder = !hasFilters && orderedRows.length > 0 && !query.isLoading;
  const selectedIDs = selectedConfigurationIds.map(Number);

  useEffect(() => {
    setOrderedRows(serverRows);
  }, [serverRows]);

  function enableReorder() {
    setSelectedConfigurationIds([]);
    setReorderEnabled(true);
    setReorderWarningOpen(false);
  }

  function deleteSelectedConfigurations() {
    bulkDelete.mutate(selectedIDs, {
      onSuccess: () => {
        setSelectedConfigurationIds([]);
        setDeleteOpen(false);
      },
    });
  }

  function moveOrder(next: SantaConfiguration[]) {
    const nextRows = next.map((row, position) => ({ ...row, position }));
    setOrderedRows(nextRows);
  }

  function saveOrder() {
    reorder.mutate(
      orderedRows.map((row) => row.id),
      {
        onSuccess: () => {
          setReorderEnabled(false);
        },
        onError: () => setOrderedRows(serverRows),
      },
    );
  }

  const columns: ColumnDef<SantaConfiguration>[] = [
    ...(reorderEnabled
      ? ([
          {
            id: "drag",
            header: () => null,
            enableSorting: false,
            enableHiding: false,
            cell: () => <DraggableDataTableRowDragHandle label="Reorder configuration" />,
            meta: { headClassName: "w-10", cellClassName: "w-10" },
          },
        ] satisfies ColumnDef<SantaConfiguration>[])
      : []),
    {
      id: "name",
      accessorKey: "name",
      header: "Name",
      enableSorting: false,
      cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
    },
    {
      id: "client_mode",
      accessorKey: "client_mode",
      header: "Client mode",
      enableSorting: false,
      cell: ({ row }) => <Badge variant="secondary">{clientModeLabel(row.original.client_mode)}</Badge>,
    },
    {
      id: "labels",
      header: "Targets",
      enableSorting: false,
      cell: ({ row }) => <TargetLabelsCell labelIDs={row.original.label_ids ?? []} labelsByID={labelsByID} />,
    },
    {
      id: "updated_at",
      accessorKey: "updated_at",
      header: "Updated",
      enableSorting: false,
      cell: ({ row }) => (
        <span className="text-muted-foreground" title={new Date(row.original.updated_at).toLocaleString()}>
          {formatRelative(row.original.updated_at)}
        </span>
      ),
    },
  ];

  return (
    <PageShell>
      <PageHeader
        title="Santa configurations"
        description="Configurations are evaluated in list order; each label can belong to one configuration."
        actions={
          <>
            <ButtonGroup>
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={reorderEnabled || !canEnableReorder}
                onClick={() => setReorderWarningOpen(true)}
              >
                Edit order
              </Button>
              {reorderEnabled ? (
                <>
                  <Button
                    type="button"
                    variant="destructive"
                    size="sm"
                    disabled={reorder.isPending}
                    onClick={saveOrder}
                  >
                    {reorder.isPending ? <Loader2 data-icon="inline-start" className="animate-spin" /> : null}
                    Save
                  </Button>
                  <Button type="button" variant="outline" size="sm" onClick={() => setReorderEnabled(false)}>
                    Cancel
                  </Button>
                </>
              ) : null}
            </ButtonGroup>
            {reorderEnabled ? null : (
              <Button asChild size="sm">
                <Link to="/santa/configurations/new">
                  <Plus data-icon="inline-start" />
                  Add configuration
                </Link>
              </Button>
            )}
          </>
        }
      />

      {query.error ? (
        <Alert variant="destructive">
          <AlertTitle>Failed to load configurations</AlertTitle>
          <AlertDescription>{query.error.message}</AlertDescription>
        </Alert>
      ) : reorderEnabled ? (
        <DraggableDataTable
          columns={columns}
          data={orderedRows}
          isLoading={query.isLoading}
          disabled={reorder.isPending || orderedRows.length <= 1}
          onRowReorder={moveOrder}
          empty={<ConfigurationsEmptyState hasFilters={hasFilters} />}
        />
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
          enableRowSelection
          selectedRowIds={selectedConfigurationIds}
          onSelectedRowIdsChange={setSelectedConfigurationIds}
          bulkActions={
            <Button variant="destructive" size="sm" onClick={() => setDeleteOpen(true)} disabled={bulkDelete.isPending}>
              <Trash2 data-icon="inline-start" />
              Delete
            </Button>
          }
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
          empty={<ConfigurationsEmptyState hasFilters={hasFilters} />}
        />
      )}

      <BulkDeleteDialog
        open={deleteOpen}
        onOpenChange={(open) => {
          if (!open) bulkDelete.reset();
          setDeleteOpen(open);
        }}
        count={selectedIDs.length}
        noun="configuration"
        description="Deleted configurations stop applying to matching hosts."
        error={bulkDelete.error?.message}
        pending={bulkDelete.isPending}
        onConfirm={deleteSelectedConfigurations}
      />
      <ReorderWarningDialog open={reorderWarningOpen} onOpenChange={setReorderWarningOpen} onConfirm={enableReorder} />
    </PageShell>
  );
}

function ConfigurationsEmptyState({ hasFilters }: { hasFilters: boolean }) {
  return (
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
  );
}

function TargetLabelsCell({
  labelIDs,
  labelsByID,
}: {
  labelIDs: number[];
  labelsByID: ReadonlyMap<number, LabelChip>;
}) {
  const countText = `${labelIDs.length} label${labelIDs.length === 1 ? "" : "s"}`;

  if (labelIDs.length === 0) {
    return <span className="text-muted-foreground text-sm tabular-nums">{countText}</span>;
  }

  const labels = labelsFromIDs(labelIDs, labelsByID);

  return (
    <HoverCard openDelay={150} closeDelay={150}>
      <HoverCardTrigger asChild>
        <button
          type="button"
          className="text-muted-foreground rounded-sm text-sm tabular-nums underline-offset-4 hover:underline focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
        >
          {countText}
        </button>
      </HoverCardTrigger>
      <HoverCardContent align="start" side="top" className="w-auto max-w-80 p-2">
        <LabelChips labels={labels} />
      </HoverCardContent>
    </HoverCard>
  );
}

function ReorderWarningDialog({
  open,
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
}) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Reorder Santa configurations?</AlertDialogTitle>
          <AlertDialogDescription>
            Santa uses the first matching configuration for each host. Reordering can change client behavior
            immediately, so make sure you know what you&apos;re doing before continuing.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel variant="ghost" size="sm">
            Cancel
          </AlertDialogCancel>
          <AlertDialogAction
            size="sm"
            onClick={(event) => {
              event.preventDefault();
              onConfirm();
            }}
          >
            Continue
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
        />

        {error ? (
          <Alert variant="destructive">
            <AlertTitle>Unable to save configuration</AlertTitle>
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        ) : null}

        <Tabs defaultValue="settings" className="max-w-4xl">
          <TabsList>
            <TabsTrigger value="settings">Settings</TabsTrigger>
            <TabsTrigger value="targets">Targets</TabsTrigger>
          </TabsList>

          <TabsContent value="settings">
            <FieldGroup>
              <FieldSet>
                <FieldLegend>Basics</FieldLegend>
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
              </FieldSet>

              <FieldSet>
                <FieldLegend>Sync</FieldLegend>
                <BoolField
                  id="santa-enable-bundles"
                  label="Bundles"
                  description="Scan bundled applications."
                  value={form.enable_bundles}
                  onChange={(enable_bundles) => setForm({ ...form, enable_bundles })}
                />
                <BoolField
                  id="santa-enable-transitive-rules"
                  label="Transitive rules"
                  description="Allow compiled binaries to inherit allowlists."
                  value={form.enable_transitive_rules}
                  onChange={(enable_transitive_rules) => setForm({ ...form, enable_transitive_rules })}
                />
                <BoolField
                  id="santa-upload-all-events"
                  label="Upload all events"
                  description="Include explicitly allowed executions."
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
                    onChange={(event) => setForm({ ...form, full_sync_interval_seconds: Number(event.target.value) })}
                  />
                  <FieldDescription>Seconds between clean syncs. Santa default: 600.</FieldDescription>
                </Field>
                <Field>
                  <FieldLabel htmlFor="santa-batch-size">Batch size</FieldLabel>
                  <Input
                    id="santa-batch-size"
                    type="number"
                    min={5}
                    max={100}
                    inputMode="numeric"
                    value={form.batch_size}
                    onChange={(event) => setForm({ ...form, batch_size: Number(event.target.value) })}
                  />
                  <FieldDescription>Rule rows per download page. Santa default: 50.</FieldDescription>
                </Field>
              </FieldSet>

              <FieldSet>
                <FieldLegend>Execution</FieldLegend>
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
              </FieldSet>

              <FieldSet>
                <FieldLegend>Removable Media</FieldLegend>
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
              </FieldSet>
            </FieldGroup>
          </TabsContent>

          <TabsContent value="targets">
            <FieldGroup>
              <Field>
                <FieldLabel>Labels</FieldLabel>
                <LabelPicker value={form.label_ids} onChange={(label_ids) => setForm({ ...form, label_ids })} />
              </Field>
            </FieldGroup>
          </TabsContent>
        </Tabs>

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
  description,
  value,
  onChange,
}: {
  id: string;
  label: string;
  description: string;
  value: boolean;
  onChange: (value: boolean) => void;
}) {
  return (
    <Field>
      <div className="flex items-center gap-2">
        <Checkbox id={id} checked={value} onCheckedChange={(next) => onChange(next === true)} />
        <FieldLabel htmlFor={id}>{label}</FieldLabel>
      </div>
      <FieldDescription>{description}</FieldDescription>
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
  action: MediaAction;
  flags: string;
  onActionChange: (value: MediaAction) => void;
  onFlagsChange: (value: string) => void;
}) {
  return (
    <Field>
      <FieldLabel htmlFor={`${id}-action`}>{label}</FieldLabel>
      <Select value={action} onValueChange={(value) => onActionChange(value as MediaAction)}>
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
      {action === "remount" ? (
        <Input
          id={`${id}-flags`}
          placeholder="remount flags"
          value={flags}
          onChange={(event) => onFlagsChange(event.target.value)}
        />
      ) : null}
      <FieldDescription>Remount requires one or more mount flags.</FieldDescription>
    </Field>
  );
}

function formFromConfiguration(configuration: SantaConfiguration): ConfigurationFormState {
  return {
    name: configuration.name,
    client_mode: configuration.client_mode,
    label_ids: configuration.label_ids ?? [],
    enable_bundles: configuration.enable_bundles,
    enable_transitive_rules: configuration.enable_transitive_rules,
    enable_all_event_upload: configuration.enable_all_event_upload,
    full_sync_interval_seconds: configuration.full_sync_interval_seconds,
    batch_size: configuration.batch_size,
    allowed_path_regex: configuration.allowed_path_regex ?? "",
    blocked_path_regex: configuration.blocked_path_regex ?? "",
    removable_media_action: configuration.removable_media_policy?.action ?? "none",
    removable_media_remount_flags: (configuration.removable_media_policy?.remount_flags ?? []).join(" "),
    encrypted_removable_media_action: configuration.encrypted_removable_media_policy?.action ?? "none",
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
    enable_bundles: form.enable_bundles,
    enable_transitive_rules: form.enable_transitive_rules,
    enable_all_event_upload: form.enable_all_event_upload,
    full_sync_interval_seconds: form.full_sync_interval_seconds,
    batch_size: form.batch_size,
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

function removableMediaPolicyBody(action: MediaAction, flags: string) {
  if (action === "none") return undefined;
  return { action, remount_flags: splitWords(flags) };
}

function optionalText(value: string) {
  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
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
