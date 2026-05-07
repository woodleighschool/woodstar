import { MoreHorizontal, Plus, Search, Tags, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import { EmptyState } from "@/components/feedback/empty-state";
import { ErrorState } from "@/components/feedback/error-state";
import { Spinner } from "@/components/feedback/spinner";
import { PageHeader } from "@/components/layout/page-header";
import { FilterPopover, type FilterGroup } from "@/components/lists/filter-popover";
import type { SortState } from "@/components/lists/sort-state";
import { SortableTableHead } from "@/components/lists/sortable-table-head";
import { TablePagination } from "@/components/lists/table-pagination";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
import { Input } from "@/components/ui/input";
import { Label as FieldLabel } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";
import {
  useCreateLabel,
  useDeleteLabel,
  useLabels,
  useUpdateLabel,
  type Label,
  type LabelMutation,
} from "@/hooks/use-labels";
import { cn, formatRelative } from "@/lib/utils";

const DEFAULT_PAGE_SIZE = 50;
const SEARCH_DEBOUNCE_MS = 200;

const KIND_OPTIONS = [
  { value: "builtin", label: "Built-in" },
  { value: "custom", label: "Custom" },
];

const MEMBERSHIP_OPTIONS = [
  { value: "dynamic", label: "Dynamic" },
  { value: "static", label: "Static" },
  { value: "identity", label: "Identity" },
];

const PLATFORM_OPTIONS = [{ value: "darwin", label: "Darwin" }];

type LabelMembershipType = NonNullable<LabelMutation["membership_type"]>;

export interface LabelsSearch {
  q?: string;
  page?: number;
  per_page?: number;
  order_key?: string;
  order_direction?: "asc" | "desc";
  kind?: string;
  membership_type?: string;
  platform?: string;
}

export function LabelsPage({
  search,
  setSearch,
}: {
  search: LabelsSearch;
  setSearch: (updater: (prev: LabelsSearch) => LabelsSearch) => void;
}) {
  const activeQ = search.q ?? "";
  const activePage = search.page ?? 1;
  const activePerPage = search.per_page ?? DEFAULT_PAGE_SIZE;
  const activeSort: SortState = { orderKey: search.order_key, orderDirection: search.order_direction };
  const [searchInput, setSearchInput] = useState(activeQ);
  const [lastActiveQ, setLastActiveQ] = useState(activeQ);
  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<Label | null>(null);
  const [deleting, setDeleting] = useState<Label | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  if (lastActiveQ !== activeQ) {
    setLastActiveQ(activeQ);
    setSearchInput(activeQ);
  }

  useEffect(
    () => () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    },
    [],
  );

  const query = useLabels({
    q: activeQ,
    page: activePage - 1,
    per_page: activePerPage,
    order_key: activeSort.orderKey,
    order_direction: activeSort.orderDirection,
    kind: search.kind,
    membership_type: search.membership_type,
    platform: search.platform,
  });

  const writeQ = (next: string) => {
    const trimmed = next.trim();
    setSearch((prev) => ({ ...prev, q: trimmed === "" ? undefined : trimmed, page: undefined }));
  };

  const filterGroups: FilterGroup[] = [
    {
      id: "kind",
      label: "Kind",
      options: KIND_OPTIONS,
      selected: search.kind ? [search.kind] : [],
      onChange: (values) => {
        setSearch((prev) => ({ ...prev, kind: values[0] || undefined, page: undefined }));
      },
    },
    {
      id: "membership_type",
      label: "Membership",
      options: MEMBERSHIP_OPTIONS,
      selected: search.membership_type ? [search.membership_type] : [],
      onChange: (values) => {
        setSearch((prev) => ({ ...prev, membership_type: values[0] || undefined, page: undefined }));
      },
    },
    {
      id: "platform",
      label: "Platform",
      options: PLATFORM_OPTIONS,
      selected: search.platform ? [search.platform] : [],
      onChange: (values) => {
        setSearch((prev) => ({ ...prev, platform: values[0] || undefined, page: undefined }));
      },
    },
  ];

  const data = query.data?.items ?? [];
  const hasFilters = activeQ !== "" || Boolean(search.kind || search.membership_type || search.platform);

  return (
    <div className="flex flex-col">
      <PageHeader
        title="Labels"
        description="Host groups used to scope reports, checks, and module rules."
        actions={
          <Button size="sm" className="gap-2" onClick={() => setCreateOpen(true)}>
            <Plus className="size-4" /> Add label
          </Button>
        }
      />

      <div className="p-6">
        <div className="flex flex-col gap-4">
          <div className="flex items-center gap-2">
            <div className="relative flex-1 max-w-md">
              <Search
                className="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
                aria-hidden
              />
              <Input
                value={searchInput}
                onChange={(event) => {
                  setSearchInput(event.target.value);
                  if (debounceRef.current) clearTimeout(debounceRef.current);
                  debounceRef.current = setTimeout(() => writeQ(event.target.value), SEARCH_DEBOUNCE_MS);
                }}
                onKeyDown={(event) => {
                  if (event.key === "Enter") {
                    if (debounceRef.current) clearTimeout(debounceRef.current);
                    writeQ(searchInput);
                  }
                }}
                placeholder="Search labels"
                className="pl-8 pr-8"
                aria-label="Search labels"
              />
              {searchInput ? (
                <button
                  type="button"
                  onClick={() => {
                    if (debounceRef.current) clearTimeout(debounceRef.current);
                    setSearchInput("");
                    writeQ("");
                  }}
                  className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-0.5 text-muted-foreground hover:text-foreground"
                  aria-label="Clear search"
                >
                  <X className="size-3.5" />
                </button>
              ) : null}
            </div>
            <div className="text-xs text-muted-foreground tabular-nums">
              {query.isFetching ? (
                <span className="inline-flex items-center gap-1">
                  <Spinner className="size-3" /> Loading…
                </span>
              ) : (
                <>
                  {query.data?.count ?? 0} label{query.data?.count === 1 ? "" : "s"}
                </>
              )}
            </div>
            <FilterPopover groups={filterGroups} />
          </div>

          {query.error ? (
            <ErrorState message={query.error.message} onRetry={() => query.refetch()} />
          ) : query.isLoading ? (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Spinner /> Loading…
            </div>
          ) : data.length === 0 ? (
            <EmptyState
              icon={Tags}
              title={hasFilters ? "No matches" : "No labels yet"}
              description={hasFilters ? "No labels matched the current filters." : "Built-in labels appear here."}
            />
          ) : (
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <SortableTableHead orderKey="name" active={activeSort} onSort={(next) => setSort(setSearch, next)}>
                      Name
                    </SortableTableHead>
                    <SortableTableHead orderKey="kind" active={activeSort} onSort={(next) => setSort(setSearch, next)}>
                      Kind
                    </SortableTableHead>
                    <SortableTableHead
                      orderKey="membership_type"
                      active={activeSort}
                      onSort={(next) => setSort(setSearch, next)}
                    >
                      Membership
                    </SortableTableHead>
                    <SortableTableHead
                      orderKey="platform"
                      active={activeSort}
                      onSort={(next) => setSort(setSearch, next)}
                    >
                      Platform
                    </SortableTableHead>
                    <SortableTableHead
                      orderKey="hosts_count"
                      active={activeSort}
                      onSort={(next) => setSort(setSearch, next)}
                    >
                      Hosts
                    </SortableTableHead>
                    <SortableTableHead
                      orderKey="updated_at"
                      active={activeSort}
                      onSort={(next) => setSort(setSearch, next)}
                    >
                      Updated
                    </SortableTableHead>
                    <TableHead className="w-12" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.map((label) => (
                    <LabelRow key={label.id} label={label} onEdit={setEditing} onDelete={setDeleting} />
                  ))}
                </TableBody>
              </Table>
              <TablePagination
                page={activePage}
                perPage={activePerPage}
                totalCount={query.data?.count ?? data.length}
                visibleCount={data.length}
                onPageChange={(page) => {
                  setSearch((prev) => ({ ...prev, page: page <= 1 ? undefined : page }));
                }}
                onPerPageChange={(perPage) => {
                  setSearch((prev) => ({
                    ...prev,
                    per_page: perPage === DEFAULT_PAGE_SIZE ? undefined : perPage,
                    page: undefined,
                  }));
                }}
              />
            </div>
          )}
        </div>
      </div>

      <LabelFormDialog mode="create" open={createOpen} onOpenChange={setCreateOpen} />

      {editing ? (
        <LabelFormDialog
          mode="edit"
          label={editing}
          open={editing !== null}
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
    </div>
  );
}

function setSort(setSearch: (updater: (prev: LabelsSearch) => LabelsSearch) => void, next: SortState) {
  setSearch((prev) => ({
    ...prev,
    order_key: next.orderKey,
    order_direction: next.orderDirection,
    page: undefined,
  }));
}

function LabelRow({
  label,
  onEdit,
  onDelete,
}: {
  label: Label;
  onEdit: (label: Label) => void;
  onDelete: (label: Label) => void;
}) {
  const canMutate = label.kind === "custom";

  return (
    <TableRow>
      <TableCell className="font-medium">{label.name}</TableCell>
      <TableCell>
        <Badge variant={label.kind === "builtin" ? "secondary" : "outline"}>{label.kind}</Badge>
      </TableCell>
      <TableCell className="text-muted-foreground">{label.membership_type}</TableCell>
      <TableCell className="text-muted-foreground">{label.platform || "-"}</TableCell>
      <TableCell className="tabular-nums">{label.hosts_count}</TableCell>
      <TableCell className="text-muted-foreground" title={new Date(label.updated_at).toLocaleString()}>
        {formatRelative(label.updated_at)}
      </TableCell>
      <TableCell className="text-right">
        {canMutate ? (
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
        ) : null}
      </TableCell>
    </TableRow>
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
  const update = useUpdateLabel(editing?.id ?? "");
  const pending = create.isPending || update.isPending;
  const submitError = mode === "create" ? create.error : update.error;

  const [name, setName] = useState(editing?.name ?? "");
  const [description, setDescription] = useState(editing?.description ?? "");
  const [membershipType, setMembershipType] = useState<LabelMembershipType>(labelMembershipType(editing));
  const [platform, setPlatform] = useState(editing?.platform ?? "");
  const [query, setQuery] = useState(editing?.query ?? "select 1;");
  const queryRequired = membershipType === "dynamic";

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const body: LabelMutation = {
      name,
      description,
      membership_type: membershipType,
      platform: platform.trim() === "" ? undefined : platform.trim(),
      query: queryRequired ? query : undefined,
    };

    if (mode === "create") {
      await create.mutateAsync(body);
    } else {
      await update.mutateAsync(body);
    }
    onClose();
  }

  return (
    <>
      <DialogHeader>
        <DialogTitle>{mode === "create" ? "Add label" : "Edit label"}</DialogTitle>
        <DialogDescription>
          Dynamic labels are evaluated by osquery. Static and identity labels can be populated by later modules.
        </DialogDescription>
      </DialogHeader>

      <form className="grid gap-3" onSubmit={handleSubmit}>
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
            className="font-mono text-xs"
          />
        </div>

        {submitError ? <p className="text-sm text-destructive">{submitError.message}</p> : null}

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
    case "static":
    case "identity":
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
        {remove.error ? <p className="text-sm text-destructive">{remove.error.message}</p> : null}
        <DialogFooter>
          <DialogClose asChild>
            <Button type="button" variant="ghost" size="sm" disabled={remove.isPending}>
              Cancel
            </Button>
          </DialogClose>
          <Button type="button" variant="destructive" size="sm" disabled={remove.isPending} onClick={handleDelete}>
            Delete label
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
