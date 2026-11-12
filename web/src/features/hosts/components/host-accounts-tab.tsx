import { BooleanIndicator } from "@components/boolean-indicator";
import { DataTableRowExpander } from "@components/data-table/data-table-row-expander";
import { DataTableStatic } from "@components/data-table/data-table-static";
import type { DataTableColumnDef } from "@components/data-table/types";
import { KeyValueRow, KeyValueRows } from "@components/key-value";
import { PanelEmptyState } from "@components/panel-empty-state";
import { StatusIndicator } from "@components/ui/status";
import { Tooltip, TooltipContent, TooltipTrigger } from "@components/ui/tooltip";
import type { HostAccount, HostDetail } from "@lib/api";

const columns: DataTableColumnDef<HostAccount>[] = [
  {
    id: "expand",
    header: () => <span className="sr-only">Details</span>,
    cell: ({ row }) => <DataTableRowExpander row={row} label={row.original.username} />,
    size: 44,
    minSize: 44,
    maxSize: 44,
    enableResizing: false,
  },
  {
    accessorKey: "username",
    header: () => "Username",
    size: 180,
    cell: ({ row }) => (
      <div className="flex min-w-0 items-center gap-2">
        <span className="font-medium">{row.original.username}</span>
        {row.original.logged_in ? (
          <Tooltip>
            <TooltipTrigger render={<span className="inline-flex" />}>
              <StatusIndicator aria-hidden className="bg-success before:animate-none" />
              <span className="sr-only">Logged in at last inventory</span>
            </TooltipTrigger>
            <TooltipContent>Logged in at last inventory</TooltipContent>
          </Tooltip>
        ) : null}
      </div>
    ),
  },
  {
    accessorKey: "display_name",
    header: () => "Name",
    size: 186,
    cell: ({ row }) => row.original.display_name || "-",
  },
  {
    accessorKey: "is_admin",
    header: () => "Admin Group",
    size: 120,
    cell: ({ row }) => <BooleanIndicator value={row.original.is_admin} />,
  },
  {
    accessorKey: "secure_token",
    header: () => "Secure Token",
    size: 130,
    cell: ({ row }) => securityValue(row.original.secure_token),
  },
  {
    accessorKey: "filevault_unlock",
    header: () => "FileVault Unlock",
    size: 145,
    cell: ({ row }) => securityValue(row.original.filevault_unlock),
  },
];

export function HostAccountsTab({ host }: { host: HostDetail }) {
  const groups = [
    { heading: "Login Accounts", accounts: host.accounts.filter(isLoginAccount) },
    {
      heading: "Other Accounts",
      accounts: host.accounts.filter((account) => !isLoginAccount(account)),
    },
  ];

  return (
    <div className="flex min-w-0 flex-col gap-6">
      {groups.map(({ heading, accounts }) => (
        <DataTableStatic
          key={heading}
          heading={heading}
          columns={columns}
          data={accounts}
          empty={<PanelEmptyState>No {heading}</PanelEmptyState>}
          getRowId={(account) => `${account.uid}:${account.username}`}
          getRowCanExpand={() => true}
          renderSubRow={({ original: account }) => (
            <KeyValueRows className="p-3">
              <KeyValueRow label="UID" value={account.uid} />
              <KeyValueRow label="UUID" value={account.uuid} />
              <KeyValueRow label="Home Directory" value={account.home_directory} />
              <KeyValueRow label="Shell" value={account.shell} />
              <KeyValueRow label="Hidden" value={<BooleanIndicator value={account.is_hidden} />} />
              <KeyValueRow label="Volume Owner" value={securityValue(account.volume_owner)} />
            </KeyValueRows>
          )}
        />
      ))}
    </div>
  );
}

function securityValue(value: boolean | null) {
  return value === null ? "-" : <BooleanIndicator value={value} />;
}

function isLoginAccount(account: HostAccount) {
  return (
    account.logged_in ||
    account.is_admin ||
    account.filevault_unlock === true ||
    (account.shell !== "" && !/\/(?:false|nologin|shutdown|halt|true)$/.test(account.shell))
  );
}
