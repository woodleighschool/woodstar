import { getRouteApi, useParams } from "@tanstack/react-router";
import { Pencil, Trash2 } from "lucide-react";
import { useMemo, useState } from "react";

import { ConfirmDialog } from "@components/confirm-dialog";
import { DataTable } from "@components/data-table/data-table";
import type { DataTableExportOptions } from "@components/data-table/data-table-export";
import { DataTableFacetedFilter } from "@components/data-table/data-table-faceted-filter";
import { DataTableSearchInput } from "@components/data-table/data-table-search-input";
import { DataTableSkeleton } from "@components/data-table/data-table-skeleton";
import type { DataTableInstance } from "@components/data-table/types";
import { useDataTable } from "@components/data-table/use-data-table";
import { useDataTableSearch } from "@components/data-table/use-data-table-search";
import { SQLEditor } from "@components/editor/sql-editor";
import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { PageHeader, PageShell } from "@components/layout/page-layout";
import { ScrollableTabs, ScrollableTabsList } from "@components/layout/scrollable-tabs";
import { Link } from "@components/link";
import { PanelEmptyState } from "@components/panel-empty-state";
import { QueryError } from "@components/query-error";
import { QueryGate } from "@components/query-gate";
import { LabelTargetDetails } from "@components/targeting/target-details";
import { Button } from "@components/ui/button";
import { Separator } from "@components/ui/separator";
import { Skeleton } from "@components/ui/skeleton";
import { TabsContent, TabsTrigger } from "@components/ui/tabs";
import { useAuth } from "@features/auth/queries";
import { LiveRunButton } from "@features/osquery/live/query-actions";
import { POLICY_RESULT_STATUS_OPTIONS } from "@features/osquery/policies/model";
import { parseRouteID } from "@lib/route-params";
import { formatRelative } from "@lib/utils";

import { PolicyDeleteDialog } from "./delete-dialog";
import {
  listAllPolicyResults,
  usePolicy,
  usePolicyResults,
  useResetPolicyResult,
  useRunPolicyRemediation,
} from "./queries";
import {
  policyResultFromStatus,
  type PolicyResultRow,
  createPolicyResultColumns,
} from "./query-results";
import { PolicyRemediationDialog } from "./remediation-dialog";
import { PolicyResultCountLink } from "./result-count-link";

const resultExportColumns: DataTableExportOptions<PolicyResultRow>["columns"] = [
  { header: "Host", value: (row) => row.host_name },
  { header: "Status", value: (row) => row.status },
  { header: "Error", value: (row) => row.error },
  { header: "Remediation", value: (row) => row.remediation?.status },
  { header: "Last Evaluated", value: (row) => row.updated_at },
];

const STATUS_FILTER_KEYS = [{ id: "status", multiple: true }] as const;
const routeApi = getRouteApi("/_authenticated/osquery/policies/$id/");

function PolicyResultsToolbar({ table }: { table: DataTableInstance<PolicyResultRow> }) {
  return (
    <DataTableFacetedFilter
      column={table.getColumn("status")}
      title="Status"
      options={POLICY_RESULT_STATUS_OPTIONS}
    />
  );
}

export function PolicyDetailPage() {
  const { id: policyId } = useParams({
    from: "/_authenticated/osquery/policies/$id",
  });
  const search = routeApi.useSearch();
  const navigate = routeApi.useNavigate();
  const activeTab = search.tab === "results" ? "results" : "overview";
  const tableSearch = useDataTableSearch({
    search,
    onSearchChange: (updater) => void navigate({ search: updater, replace: true }),
    filterKeys: STATUS_FILTER_KEYS,
  });
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [resetResult, setResetResult] = useState<PolicyResultRow | null>(null);
  const [viewRemediation, setViewRemediation] = useState<PolicyResultRow | null>(null);
  const id = parseRouteID(policyId);
  const policy = usePolicy(id);
  const reset = useResetPolicyResult();
  const runRemediation = useRunPolicyRemediation();
  const status = search.status;
  const results = usePolicyResults(activeTab === "results" ? id : null, {
    q: tableSearch.q,
    page: tableSearch.page,
    per_page: tableSearch.per_page,
    sort: tableSearch.sort,
    status,
  });
  const rows =
    results.data?.items.map((result) =>
      policyResultFromStatus(
        result,
        isAdmin
          ? {
              onReset: () => setResetResult(policyResultFromStatus(result)),
              onRunRemediation: policy.data?.remediation.configured
                ? () =>
                    runRemediation.mutate({
                      policyID: result.policy_id,
                      hostID: result.host_id,
                    })
                : undefined,
              onViewRemediation: result.remediation
                ? () => setViewRemediation(policyResultFromStatus(result))
                : undefined,
            }
          : {},
      ),
    ) ?? [];
  const totalCount = results.data?.count ?? 0;
  const pageCount = results.data ? Math.ceil(totalCount / tableSearch.per_page) : -1;
  const resultColumns = useMemo(
    () =>
      createPolicyResultColumns({
        timestampHeader: "Last Evaluated",
        includeError: true,
        includeRemediation: true,
        includeActions: isAdmin,
      }),
    [isAdmin],
  );
  const table = useDataTable({
    tableState: tableSearch,
    data: rows,
    columns: resultColumns,
    pageCount,
    rowCount: totalCount,
    getRowId: (row) => String(row.host_id),
  });

  if (id === null) {
    return (
      <QueryGate title="Failed to load policy" error={{ message: "Policy route is invalid." }} />
    );
  }

  if (policy.error) {
    return (
      <QueryGate
        title="Failed to load policy"
        error={policy.error}
        onRetry={() => void policy.refetch()}
      />
    );
  }

  if (!policy.data) {
    return (
      <PageShell>
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full" />
      </PageShell>
    );
  }

  const exportOptions: DataTableExportOptions<PolicyResultRow> = {
    filename: `osquery-policy-${id}-results`,
    columns: resultExportColumns,
    loadRows: () =>
      listAllPolicyResults(id, {
        q: tableSearch.q,
        sort: tableSearch.sort,
        status,
      }).then((result) => result.map((row) => policyResultFromStatus(row))),
  };
  return (
    <PageShell>
      <PageHeader
        title="Policy Details"
        meta={`Edited ${formatRelative(policy.data.updated_at)}`}
        actions={
          <>
            {isAdmin ? (
              <>
                <Button
                  size="sm"
                  render={<Link to="/osquery/policies/$id/edit" params={{ id: policyId }} />}
                  nativeButton={false}
                >
                  <Pencil data-icon="inline-start" />
                  Edit
                </Button>
              </>
            ) : null}
            <LiveRunButton kind="policy" id={id} sql={policy.data.query} />
            {isAdmin ? (
              <Button
                type="button"
                variant="destructive"
                size="sm"
                onClick={() => setDeleteOpen(true)}
              >
                <Trash2 data-icon="inline-start" />
                Delete
              </Button>
            ) : null}
          </>
        }
      />

      <ScrollableTabs value={activeTab}>
        <ScrollableTabsList>
          <TabsTrigger
            value="overview"
            render={
              <Link
                to="/osquery/policies/$id"
                params={{ id: policyId }}
                search={{ ...search, tab: undefined }}
              />
            }
            nativeButton={false}
          >
            Overview
          </TabsTrigger>
          <TabsTrigger
            value="results"
            render={
              <Link
                to="/osquery/policies/$id"
                params={{ id: policyId }}
                search={{ ...search, tab: "results" }}
              />
            }
            nativeButton={false}
          >
            Results
          </TabsTrigger>
        </ScrollableTabsList>

        <TabsContent value="overview" className="flex flex-col gap-5">
          <KeyValueSection title="Overview">
            <KeyValueRow label="Name" value={policy.data.name} />
            <KeyValueRow label="Description" value={policy.data.description} />
            <KeyValueRow label="Resolution" value={policy.data.resolution} />
            <KeyValueRow
              label="Remediation"
              value={
                !policy.data.remediation.configured
                  ? "Not configured"
                  : policy.data.remediation.automatic
                    ? "Configured · Automatic"
                    : "Configured · Manual only"
              }
            />
            <KeyValueRow
              label="Passing"
              value={
                <PolicyResultCountLink
                  policyId={policy.data.id}
                  count={policy.data.passing_host_count}
                  status="pass"
                />
              }
            />
            <KeyValueRow
              label="Failing"
              value={
                <PolicyResultCountLink
                  policyId={policy.data.id}
                  count={policy.data.failing_host_count}
                  status="fail"
                />
              }
            />
          </KeyValueSection>

          <section className="flex min-w-0 flex-col gap-3">
            <h2 className="text-base/snug font-medium text-foreground">Query</h2>
            <Separator />
            <SQLEditor value={policy.data.query} onChange={() => undefined} readOnly />
          </section>

          <LabelTargetDetails targets={policy.data.targets} />
        </TabsContent>

        <TabsContent value="results">
          {results.error ? (
            <QueryError
              title="Failed to load policy results"
              error={results.error}
              onRetry={() => void results.refetch()}
            />
          ) : results.isLoading ? (
            <DataTableSkeleton columnCount={isAdmin ? 6 : 5} filterCount={1} withExport />
          ) : (
            <DataTable
              table={table}
              pending={results.isPlaceholderData}
              exportOptions={exportOptions}
              empty={
                <PanelEmptyState>
                  {tableSearch.isFiltered ? "No matching policy results" : "No policy results yet"}
                </PanelEmptyState>
              }
            >
              <DataTableSearchInput
                loading={results.isPlaceholderData}
                value={tableSearch.q ?? ""}
                onValueChange={tableSearch.onQueryChange}
                placeholder="Search policy results"
              />
              <PolicyResultsToolbar table={table} />
            </DataTable>
          )}
        </TabsContent>
      </ScrollableTabs>

      {isAdmin ? (
        <>
          <PolicyDeleteDialog
            open={deleteOpen}
            onOpenChange={setDeleteOpen}
            policy={policy.data}
            onDeleted={() => void navigate({ to: "/osquery/policies" })}
          />
          <ConfirmDialog
            open={resetResult !== null}
            onOpenChange={(open) => {
              if (!open && !reset.isPending) setResetResult(null);
            }}
            title="Reset policy result?"
            description="The result returns to pending. Its next evaluation is treated as new and may trigger automatic remediation again."
            confirmLabel="Reset result"
            pending={reset.isPending}
            onConfirm={() => {
              if (!resetResult) return;
              reset.mutate(
                { policyID: id, hostID: resetResult.host_id },
                { onSuccess: () => setResetResult(null) },
              );
            }}
          />
          <PolicyRemediationDialog
            policyID={id}
            hostID={viewRemediation?.host_id ?? null}
            hostName={viewRemediation?.host_name ?? ""}
            open={viewRemediation !== null}
            onOpenChange={(open) => {
              if (!open) setViewRemediation(null);
            }}
          />
        </>
      ) : null}
    </PageShell>
  );
}
