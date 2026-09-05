import type { Access } from "@woodleighschool/authz";

import { Badge } from "@components/ui/badge";

const labels: Record<Access, string> = {
  none: "None",
  view: "View",
  edit: "Edit",
};

export function PermissionLevelBadge({ level }: { level: Access }) {
  return <Badge variant={level === "none" ? "outline" : "secondary"}>{labels[level]}</Badge>;
}

export function permissionLabel(value: string): string {
  return value
    .replaceAll(".", " ")
    .replaceAll("_", " ")
    .replace(/\b\w/g, (letter) => letter.toUpperCase());
}
