import { Badge } from "@/components/ui/badge";

export function CheckStatusBadge({ passes }: { passes: boolean | null | undefined }) {
  if (passes === true) return <Badge variant="default">passing</Badge>;
  if (passes === false) return <Badge variant="destructive">failing</Badge>;
  return <Badge variant="secondary">no response</Badge>;
}
