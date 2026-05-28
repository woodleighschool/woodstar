import { Badge } from "@/components/ui/badge";
import type { CheckHostStatus } from "@/hooks/use-checks";

export function CheckStatusBadge({ response }: { response: CheckHostStatus["response"] | undefined }) {
  if (response === "pass") {
    return (
      <Badge variant="outline" className="gap-1.5">
        <span className="bg-status-online size-1.5 rounded-full" />
        Passing
      </Badge>
    );
  }
  if (response === "fail") return <Badge variant="destructive">Failing</Badge>;
  return <Badge variant="secondary">Not Run</Badge>;
}
