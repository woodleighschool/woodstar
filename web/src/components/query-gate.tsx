import { PageShell } from "@/components/layout/page-layout";
import { QueryError } from "@/components/query-error";

export function QueryGate({
  title,
  error,
  onRetry,
}: {
  title: string;
  error: { message?: string } | null | undefined;
  onRetry?: () => void;
}) {
  if (!error) return null;
  return (
    <PageShell>
      <QueryError title={title} error={error} onRetry={onRetry} />
    </PageShell>
  );
}
