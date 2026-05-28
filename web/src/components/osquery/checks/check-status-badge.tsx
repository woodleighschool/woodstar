import { Badge } from "@/components/ui/badge";

export function CheckStatusBadge({ response }: { response: string | null | undefined }) {
  if (response === "pass") return <Badge variant="success">Passing</Badge>;
  if (response === "fail") return <Badge variant="destructive">Failing</Badge>;
  return <Badge variant="secondary">Not run</Badge>;
}
