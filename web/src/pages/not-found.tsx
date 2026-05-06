import { Link } from "@tanstack/react-router";
import { Compass } from "lucide-react";

import { EmptyState } from "@/components/feedback/empty-state";
import { Button } from "@/components/ui/button";

export function NotFoundPage() {
  return (
    <div className="flex min-h-dvh items-center justify-center p-8">
      <EmptyState
        icon={Compass}
        title="Page not found"
        description="That route doesn't exist in this Woodstar build."
        action={
          <Button asChild size="sm">
            <Link to="/hosts">Back to hosts</Link>
          </Button>
        }
      />
    </div>
  );
}
