import { Link } from "@tanstack/react-router";
import { Compass } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";

export function NotFoundPage() {
  return (
    <div className="flex min-h-dvh items-center justify-center p-8">
      <Empty>
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <Compass />
          </EmptyMedia>
          <EmptyTitle>Page Not Found</EmptyTitle>
          <EmptyDescription>That route doesn't exist in this Woodstar build.</EmptyDescription>
        </EmptyHeader>
        <EmptyContent>
          <Button asChild size="sm">
            <Link to="/hosts">Back to Hosts</Link>
          </Button>
        </EmptyContent>
      </Empty>
    </div>
  );
}
