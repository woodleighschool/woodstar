import type { ColumnDef } from "@tanstack/react-table";
import { MoreHorizontal, Plus, Search, Tags, X } from "lucide-react";
import { useState } from "react";

import { PageActions } from "@/components/layout/page-actions";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/ui/data-table";
import { DataTableColumnHeader } from "@/components/ui/data-table-column-header";
import { DataTableFacetedFilter } from "@/components/ui/data-table-faceted-filter";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { Label as FieldLabel } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { useDebouncedSearchParam } from "@/hooks/use-debounced-search-param";
import {
  useCreateLabel,
  useDeleteLabel,
  useLabels,
  useUpdateLabel,
  type Label,
  type LabelCreate,
  type LabelMutation,
} from "@/hooks/use-labels";
import { useTablePaginationParams } from "@/hooks/use-table-pagination-params";
import { cn, formatRelative } from "@/lib/utils";
import { useSearch } from "@tanstack/react-router";

const KIND_OPTIONS = [
  { value: "builtin", label: "Built-in" },
  { value: "regular", label: "Regular" },
];

const MEMBERSHIP_OPTIONS = [
  { value: "dynamic", label: "Dynamic" },
  { value: "manual", label: "Manual" },
  { value: "host_vitals", label: "Host vitals" },
];

const PLATFORM_OPTIONS = [{ value: "darwin", label: "Darwin" }];

type LabelMembershipType = NonNullable<LabelMutation["membership_type"]>;

export function LabelsPage() {
  const search = useSearch({ strict: false });
  const { state, setters } = useTablePaginationParams();
  const [draft, setDraft] = useDebouncedSearchParam("q");
  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<Label | null>(null);
  const [deleting, setDeleting] = useState<Label | null>(null);

  const query = useLabels({
    q: search.q,
    page: state.page,
    per_page: state.perPage,
    order_key: state.orderKey,
    order_direction: state.orderDirection,
    kind: search.kind,
    membership_type: search.membership_type,
    platform: search.platform,
  });

  const data = query.data?.items ?? [];
  const totalCount = query.data?.count ?? 0;
  const hasFilters = !!search.q || !!search.kind || !!search.membership_type || !!search.platform;

  const columns: ColumnDef<Label>[] = [
    {
      id: "name",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Name" />,
      cell: ({ row }) => row.original.name,
    },
    {
      id: "kind",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Kind" />,
      cell: ({ row }) => (
        <Badge variant={row.original.kind === "builtin" ? "secondary" : "outline"}>{row.original.kind}</Badge>
      ),
    },
    {
      id: "membership_type",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Membership" />,
      cell: ({ row }) => <span className="text-muted-foreground">{row.original.membership_type}</span>,
    },
    {
      id: "platform",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Platform" />,
      cell: ({ row }) => <span className="text-muted-foreground">{row.original.platform ?? "-"}</span>,
    },
    {
      id: "hosts_count",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Hosts" />,
      cell: ({ row }) => <span className="tabular-nums">{row.original.hosts_count}</span>,
    },
    {
      id: "updated_at",
      header: ({ column }) => <DataTableColumnHeader column={column} title="Updated" />,
      cell: ({ row }) => (
        <span
          className="text-muted-foreground"
          title={row.original.updated_at ? new Date(row.original.updated_at).toLocaleString() : ""}
        >
          {row.original.updated_at ? formatRelative(row.original.updated_at) : "-"}
        </span>
      ),
    },
    {
      id: "actions",
      header: () => null,
      enableSorting: false,
      cell: ({ row }) => <LabelRowActions label={row.original} onEdit={setEditing} onDelete={setDeleting} />,
      meta: { headClassName: "w-12" },
    },
  ];

  return (
    <>
      <PageActions>
        <Button size="sm" className="gap-2" onClick={() => setCreateOpen(true)}>
          <Plus className="size-4" /> Add label
        </Button>
      </PageActions>

      <div className="p-6">
        {query.error ? (
          <Alert variant="destructive">
            <AlertTitle>Failed to load labels</AlertTitle>
            <AlertDescription>{query.error.message}</AlertDescription>
            <Button variant="outline" size="sm" onClick={() => void query.refetch()} className="mt-2 w-fit">
              Retry
            </Button>
          </Alert>
        ) : (
          <DataTable
            columns={columns}
            data={data}
            totalCount={totalCount}
            page={state.page}
            perPage={state.perPage}
            sort={{ orderKey: state.orderKey, orderDirection: state.orderDirection }}
            onPageChange={setters.setPage}
            onPerPageChange={setters.setPerPage}
            onSortChange={(s) => setters.setSort(s.orderKey, s.orderDirection)}
            isLoading={query.isLoading}
            toolbar={
              <LabelsToolbar
                draft={draft}
                onDraftChange={setDraft}
                kind={search.kind}
                onKindChange={(v) => setters.setFilter("kind", v)}
                membership={search.membership_type}
                onMembershipChange={(v) => setters.setFilter("membership_type", v)}
                platform={search.platform}
                onPlatformChange={(v) => setters.setFilter("platform", v)}
                isFetching={query.isFetching}
                totalCount={totalCount}
              />
            }
            empty={
              <Empty>
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <Tags />
                  </EmptyMedia>
                  <EmptyTitle>{hasFilters ? "No matches" : "No labels yet"}</EmptyTitle>
                  <EmptyDescription>
                    {hasFilters ? "No labels matched the current filters." : "Built-in labels appear here."}
                  </EmptyDescription>
                </EmptyHeader>
              </Empty>
            }
          />
        )}
      </div>

      <LabelFormDialog mode="create" open={createOpen} onOpenChange={setCreateOpen} />

      {editing ? (
        <LabelFormDialog
          mode="edit"
          label={editing}
          open
          onOpenChange={(open) => {
            if (!open) setEditing(null);
          }}
        />
      ) : null}

      <LabelDeleteDialog
        label={deleting}
        open={deleting !== null}
        onOpenChange={(open) => {
          if (!open) setDeleting(null);
        }}
      />
    </>
  );
}

interface LabelsToolbarProps {
  draft: string;
  onDraftChange: (next: string) => void;
  kind: string | undefined;
  onKindChange: (next: string | undefined) => void;
  membership: string | undefined;
  onMembershipChange: (next: string | undefined) => void;
  platform: string | undefined;
  onPlatformChange: (next: string | undefined) => void;
  isFetching: boolean;
  totalCount: number;
}

function LabelsToolbar({
  draft,
  onDraftChange,
  kind,
  onKindChange,
  membership,
  onMembershipChange,
  platform,
  onPlatformChange,
  isFetching,
  totalCount,
}: LabelsToolbarProps) {
  return (
    <div className="flex items-center gap-2">
      <div className="relative max-w-md flex-1">
        <Search
          className="text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2"
          aria-hidden
        />
        <Input
          value={draft}
          onChange={(e) => onDraftChange(e.target.value)}
          placeholder="Search labels"
          className="pr-8 pl-8"
          aria-label="Search labels"
        />
        {draft ? (
          <button
            type="button"
            onClick={() => onDraftChange("")}
            className="text-muted-foreground hover:text-foreground absolute top-1/2 right-2 -translate-y-1/2 rounded p-0.5"
            aria-label="Clear search"
          >
            <X className="size-3.5" />
          </button>
        ) : null}
      </div>
      <DataTableFacetedFilter
        title="Kind"
        options={KIND_OPTIONS}
        selected={kind ? [kind] : []}
        onChange={(next) => onKindChange(next[0])}
        singleSelect
      />
      <DataTableFacetedFilter
        title="Membership"
        options={MEMBERSHIP_OPTIONS}
        selected={membership ? [membership] : []}
        onChange={(next) => onMembershipChange(next[0])}
        singleSelect
      />
      <DataTableFacetedFilter
        title="Platform"
        options={PLATFORM_OPTIONS}
        selected={platform ? [platform] : []}
        onChange={(next) => onPlatformChange(next[0])}
        singleSelect
      />
      <div className="text-muted-foreground ml-auto text-xs tabular-nums">
        {isFetching ? "Loading..." : `${totalCount} ${totalCount === 1 ? "label" : "labels"}`}
      </div>
    </div>
  );
}

function LabelRowActions({
  label,
  onEdit,
  onDelete,
}: {
  label: Label;
  onEdit: (label: Label) => void;
  onDelete: (label: Label) => void;
}) {
  if (label.kind !== "regular") return null;
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button type="button" size="icon" variant="ghost" aria-label={`Actions for ${label.name}`}>
          <MoreHorizontal className="size-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onSelect={() => onEdit(label)}>Edit</DropdownMenuItem>
        <DropdownMenuItem className="text-destructive focus:text-destructive" onSelect={() => onDelete(label)}>
          Delete
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

type LabelFormDialogProps =
  | {
      mode: "create";
      open: boolean;
      onOpenChange: (open: boolean) => void;
    }
  | {
      mode: "edit";
      label: Label;
      open: boolean;
      onOpenChange: (open: boolean) => void;
    };

function LabelFormDialog(props: LabelFormDialogProps) {
  const bodyKey = props.mode === "create" ? "create" : `edit-${props.label.id}`;

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        {props.open ? (
          <LabelFormBody
            key={bodyKey}
            mode={props.mode}
            editing={props.mode === "edit" ? props.label : null}
            onClose={() => props.onOpenChange(false)}
          />
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function LabelFormBody({
  mode,
  editing,
  onClose,
}: {
  mode: "create" | "edit";
  editing: Label | null;
  onClose: () => void;
}) {
  const create = useCreateLabel();
  const update = useUpdateLabel(editing?.id ?? null);
  const pending = create.isPending || update.isPending;
  const submitError = mode === "create" ? create.error : update.error;

  const [name, setName] = useState(editing?.name ?? "");
  const [description, setDescription] = useState(editing?.description ?? "");
  const [membershipType, setMembershipType] = useState<LabelMembershipType>(labelMembershipType(editing));
  const [platform, setPlatform] = useState(editing?.platform ?? "");
  const [query, setQuery] = useState(editing?.query ?? "select 1;");
  const queryRequired = membershipType === "dynamic";

  async function handleSubmit() {
    if (mode === "create") {
      const body: LabelCreate = {
        name,
        description,
        membership_type: membershipType,
        platform: platform.trim() === "" ? undefined : platform.trim(),
        query: queryRequired ? query : undefined,
      };
      await create.mutateAsync(body);
    } else {
      const body: LabelMutation = {
        name,
        description,
        membership_type: membershipType,
        platform: platform.trim() === "" ? undefined : platform.trim(),
        query: queryRequired ? query : undefined,
      };
      await update.mutateAsync(body);
    }
    onClose();
  }

  return (
    <>
      <DialogHeader>
        <DialogTitle>{mode === "create" ? "Add label" : "Edit label"}</DialogTitle>
        <DialogDescription>
          Dynamic labels are evaluated by osquery. Manual and host-vitals labels are populated by the server.
        </DialogDescription>
      </DialogHeader>

      <form
        className="grid gap-3"
        onSubmit={(event) => {
          event.preventDefault();
          void handleSubmit();
        }}
      >
        <div className="grid gap-1.5">
          <FieldLabel htmlFor="label-name">Name</FieldLabel>
          <Input id="label-name" required value={name} onChange={(event) => setName(event.target.value)} />
        </div>

        <div className="grid gap-1.5">
          <FieldLabel htmlFor="label-description">Description</FieldLabel>
          <Input id="label-description" value={description} onChange={(event) => setDescription(event.target.value)} />
        </div>

        <div className="grid gap-3 sm:grid-cols-2">
          <div className="grid gap-1.5">
            <FieldLabel htmlFor="label-membership">Membership</FieldLabel>
            <Select value={membershipType} onValueChange={(value) => setMembershipType(value as LabelMembershipType)}>
              <SelectTrigger id="label-membership">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {MEMBERSHIP_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="grid gap-1.5">
            <FieldLabel htmlFor="label-platform">Platform</FieldLabel>
            <Input
              id="label-platform"
              value={platform}
              onChange={(event) => setPlatform(event.target.value)}
              placeholder="darwin"
            />
          </div>
        </div>

        <div className={cn("grid gap-1.5", !queryRequired && "opacity-50")}>
          <FieldLabel htmlFor="label-query">Query</FieldLabel>
          <Textarea
            id="label-query"
            required={queryRequired}
            disabled={!queryRequired}
            rows={5}
            value={query}
            onChange={(event) => setQuery(event.target.value)}
          />
        </div>

        {submitError ? <p className="text-destructive text-sm">{submitError.message}</p> : null}

        <DialogFooter className="pt-2">
          <DialogClose asChild>
            <Button type="button" variant="ghost" size="sm" disabled={pending}>
              Cancel
            </Button>
          </DialogClose>
          <Button type="submit" size="sm" disabled={pending}>
            {mode === "create" ? "Create label" : "Save changes"}
          </Button>
        </DialogFooter>
      </form>
    </>
  );
}

function labelMembershipType(label: Label | null): LabelMembershipType {
  switch (label?.membership_type) {
    case "manual":
    case "host_vitals":
      return label.membership_type;
    default:
      return "dynamic";
  }
}

function LabelDeleteDialog({
  label,
  open,
  onOpenChange,
}: {
  label: Label | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const remove = useDeleteLabel();

  async function handleDelete() {
    if (!label) return;
    await remove.mutateAsync(label.id);
    onOpenChange(false);
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Delete label</DialogTitle>
          <DialogDescription>
            {label ? `${label.name} will be removed from hosts and filters.` : "This label will be removed."}
          </DialogDescription>
        </DialogHeader>
        {remove.error ? <p className="text-destructive text-sm">{remove.error.message}</p> : null}
        <DialogFooter>
          <DialogClose asChild>
            <Button type="button" variant="ghost" size="sm" disabled={remove.isPending}>
              Cancel
            </Button>
          </DialogClose>
          <Button
            type="button"
            variant="destructive"
            size="sm"
            disabled={remove.isPending}
            onClick={() => void handleDelete()}
          >
            Delete label
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
