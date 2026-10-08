import { PageShell } from "@components/layout/page-layout";
import { QueryError } from "@components/query-error";
import { Skeleton } from "@components/ui/skeleton";

// Stands in for a page whose query has nothing to render: the error, or a
// placeholder while the data loads.
export function QueryGate({
  title,
  error,
  onRetry,
}: {
  title: string;
  error: { message?: string } | null | undefined;
  onRetry?: () => void;
}) {
  if (error) {
    return (
      <PageShell>
        <QueryError title={title} error={error} onRetry={onRetry} />
      </PageShell>
    );
  }
  return (
    <PageShell aria-busy>
      <Skeleton className="h-8 w-48" />
      <Skeleton className="h-64 w-full" />
    </PageShell>
  );
}
