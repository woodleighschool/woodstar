import { Tooltip, TooltipContent, TooltipTrigger } from "@components/ui/tooltip";
import type { Host } from "@lib/api";

export function HostOnlineDot({ status }: { status: Host["status"] }) {
  if (status !== "online") return null;

  return (
    <Tooltip>
      <TooltipTrigger
        type="button"
        aria-label="Osquery online"
        className="inline-flex size-3 shrink-0 items-center justify-center rounded-full"
      >
        <span className="size-2 rounded-full bg-success" />
      </TooltipTrigger>
      <TooltipContent>Osquery online</TooltipContent>
    </Tooltip>
  );
}
