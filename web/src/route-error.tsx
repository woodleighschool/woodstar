import type { ErrorComponentProps } from "@tanstack/react-router";
import { useRouter } from "@tanstack/react-router";
import { CircleAlert } from "lucide-react";

import { Button } from "@components/ui/button";
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@components/ui/empty";

export function RouteErrorPage({ error }: ErrorComponentProps) {
  const router = useRouter();
  return (
    <div className="flex min-h-dvh items-center justify-center p-8">
      <Empty>
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <CircleAlert />
          </EmptyMedia>
          <EmptyTitle>Something Went Wrong</EmptyTitle>
          {error.message ? <EmptyDescription>{error.message}</EmptyDescription> : null}
        </EmptyHeader>
        <EmptyContent>
          <Button size="sm" onClick={() => void router.invalidate()}>
            Try Again
          </Button>
        </EmptyContent>
      </Empty>
    </div>
  );
}
