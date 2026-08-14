import { Info } from "lucide-react";

import { Badge } from "@components/ui/badge";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@components/ui/hover-card";

export function OsqueryResultError({ label, error }: { label: string; error: string }) {
  return (
    <HoverCard>
      <HoverCardTrigger
        render={
          <Badge
            variant="warning"
            className="h-auto py-1 leading-none"
            render={<button type="button" aria-label={label} />}
          >
            Error
            <Info data-icon="inline-end" />
          </Badge>
        }
      />
      <HoverCardContent
        align="start"
        className="w-96 max-w-[calc(100vw-2rem)] break-all whitespace-normal"
      >
        {error}
      </HoverCardContent>
    </HoverCard>
  );
}
