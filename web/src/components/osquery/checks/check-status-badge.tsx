import { Badge } from "@/components/ui/badge";
import type { OsqueryCheckHostStatus } from "@/lib/api";

export function CheckStatusBadge({
  response,
}: {
  response: OsqueryCheckHostStatus["response"] | undefined;
}) {
  if (response === "pass") {
    return (
      <Badge variant="outline" className="gap-1.5">
        <span className="size-1.5 rounded-full bg-status-online" />
        Passing
      </Badge>
    );
  }
  if (response === "fail") return <Badge variant="destructive">Failing</Badge>;
  return <Badge variant="secondary">Not Run</Badge>;
}
