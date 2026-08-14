import { CircleX } from "lucide-react";

import { Button } from "@components/ui/button";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@components/ui/hover-card";

export function OsqueryResultError({ label, error }: { label: string; error: string }) {
  return (
    <HoverCard>
      <HoverCardTrigger
        render={
          <Button
            type="button"
            size="icon-xs"
            variant="ghost"
            className="text-destructive hover:text-destructive"
            aria-label={label}
          >
            <CircleX />
          </Button>
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
