import { AlertTriangle } from "lucide-react";

import { Button } from "@/components/ui/button";

export interface ErrorStateProps {
  title?: string;
  message?: string;
  onRetry?: () => void;
}

export function ErrorState({
  title = "Couldn't load this view",
  message,
  onRetry,
}: ErrorStateProps) {
  return (
    <div className="flex flex-col items-start gap-3 rounded-md border border-destructive/40 bg-destructive/5 px-4 py-3 text-sm">
      <div className="flex items-start gap-2">
        <AlertTriangle className="size-4 mt-0.5 text-destructive" aria-hidden />
        <div className="space-y-1">
          <p className="font-medium">{title}</p>
          {message ? <p className="text-muted-foreground">{message}</p> : null}
        </div>
      </div>
      {onRetry ? (
        <Button variant="outline" size="sm" onClick={onRetry}>
          Retry
        </Button>
      ) : null}
    </div>
  );
}
