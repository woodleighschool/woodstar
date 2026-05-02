import { ErrorState } from "@/components/feedback/error-state";
import { PendingBanner } from "@/components/feedback/pending-banner";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/data-table";
import { Spinner } from "@/components/ui/spinner";
import type { ApiError } from "@/lib/api";
import type { EndpointSpec } from "@/lib/endpoints";
import { formatRelative } from "@/lib/utils";

export interface CredentialRow {
  id: string;
  label?: string;
  preview: string;
  created_at: string;
  last_used_at?: string;
  trailing?: React.ReactNode;
}

export interface CredentialTableProps {
  endpoint: EndpointSpec;
  data: CredentialRow[];
  isPending: boolean;
  isLoading: boolean;
  error: ApiError | null;
  onRetry?: () => void;
  emptyTitle: string;
  emptyDescription: string;
  trailingHeader?: string;
}

function relativeOrNever(value: string | undefined): string {
  if (!value) return "never";
  return formatRelative(value);
}

export function CredentialTable({
  endpoint,
  data,
  isPending,
  isLoading,
  error,
  onRetry,
  emptyTitle,
  emptyDescription,
  trailingHeader,
}: CredentialTableProps) {
  if (isPending) {
    return <PendingBanner endpoint={endpoint.path} />;
  }

  if (error) {
    return <ErrorState message={error.message} onRetry={onRetry} />;
  }

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Spinner /> Loading…
      </div>
    );
  }

  if (data.length === 0) {
    return (
      <div className="rounded-md border border-dashed bg-muted/30 px-4 py-6 text-sm">
        <p className="font-medium">{emptyTitle}</p>
        <p className="text-muted-foreground">{emptyDescription}</p>
      </div>
    );
  }

  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Label</TableHead>
            <TableHead>Preview</TableHead>
            <TableHead>Created</TableHead>
            <TableHead>Last used</TableHead>
            {trailingHeader ? (
              <TableHead className="text-right">{trailingHeader}</TableHead>
            ) : null}
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.map((row) => (
            <TableRow key={row.id}>
              <TableCell className="font-medium">
                {row.label ?? <span className="text-muted-foreground">—</span>}
              </TableCell>
              <TableCell className="font-mono text-xs">{row.preview}</TableCell>
              <TableCell>{relativeOrNever(row.created_at)}</TableCell>
              <TableCell>{relativeOrNever(row.last_used_at)}</TableCell>
              {trailingHeader ? (
                <TableCell className="text-right">{row.trailing}</TableCell>
              ) : null}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}
