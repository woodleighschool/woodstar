import { CircleAlert } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";

export function QueryError({
  title = "Failed to load",
  error,
  onRetry,
  className,
}: {
  title?: string;
  error: { message?: string } | null | undefined;
  onRetry?: () => void;
  className?: string;
}) {
  if (!error) return null;
  return (
    <Alert variant="destructive" className={className}>
      <CircleAlert />
      <AlertTitle>{title}</AlertTitle>
      {error.message ? <AlertDescription>{error.message}</AlertDescription> : null}
      {onRetry ? (
        <Button type="button" variant="outline" size="sm" onClick={onRetry} className="col-start-2 mt-2 w-fit">
          Retry
        </Button>
      ) : null}
    </Alert>
  );
}
