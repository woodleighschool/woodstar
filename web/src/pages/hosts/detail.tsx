import { Link, useParams } from "@tanstack/react-router";
import { ChevronLeft } from "lucide-react";

import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/ui/page-header";

export function HostDetailPage() {
  const { hostId } = useParams({ from: "/_authed/hosts/$hostId" });

  return (
    <div className="flex flex-col">
      <PageHeader
        title={`Host ${hostId}`}
        actions={
          <Button asChild variant="outline" size="sm">
            <Link to="/hosts" className="gap-1">
              <ChevronLeft className="size-4" /> Back to hosts
            </Link>
          </Button>
        }
      />
    </div>
  );
}
