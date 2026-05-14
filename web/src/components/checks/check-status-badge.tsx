import { Badge } from "@/components/ui/badge";

export function CheckStatusBadge({ response }: { response: string | null | undefined }) {
  if (response === "pass") return <Badge variant="default">passing</Badge>;
  if (response === "fail") return <Badge variant="destructive">failing</Badge>;
  return <Badge variant="secondary">not run</Badge>;
}
