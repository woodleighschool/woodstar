import { Compass } from "lucide-react";

import { Link } from "@components/link";
import { Button } from "@components/ui/button";
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@components/ui/empty";

export function NotFoundPage() {
  return (
    <div className="flex min-h-full items-center justify-center p-8">
      <Empty>
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <Compass />
          </EmptyMedia>
          <EmptyTitle>Page Not Found</EmptyTitle>
          <EmptyDescription>
            Nothing exists at this address. It may have been deleted.
          </EmptyDescription>
        </EmptyHeader>
        <EmptyContent>
          <Button size="sm" render={<Link to="/" />} nativeButton={false}>
            Go Home
          </Button>
        </EmptyContent>
      </Empty>
    </div>
  );
}
