import type { LucideIcon } from "lucide-react";

import { Card, CardAction, CardContent, CardHeader, CardTitle } from "@components/ui/card";
import { Skeleton } from "@components/ui/skeleton";

export function ResourceOverviewCard({
  title,
  count,
  loading,
  error,
  icon: Icon,
}: {
  title: string;
  count?: number;
  loading: boolean;
  error: unknown;
  icon: LucideIcon;
}) {
  return (
    <Card size="sm" className="h-full min-w-0 transition-colors hover:bg-muted/50">
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardAction>
          <Icon aria-hidden="true" className="text-muted-foreground" />
        </CardAction>
      </CardHeader>
      <CardContent>
        {loading ? (
          <Skeleton className="h-9 w-24" />
        ) : error ? (
          <p className="text-sm text-destructive">Count unavailable</p>
        ) : (
          <span className="text-3xl font-semibold tracking-tight tabular-nums">
            {(count ?? 0).toLocaleString()}
          </span>
        )}
      </CardContent>
    </Card>
  );
}
