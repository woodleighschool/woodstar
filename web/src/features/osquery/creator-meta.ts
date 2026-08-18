import type { UserSummary } from "@lib/api";
import { formatRelative, nonEmpty } from "@lib/utils";

export function creatorMeta(createdBy: UserSummary | undefined, updatedAt: string): string {
  const creator = nonEmpty(createdBy?.name) ?? nonEmpty(createdBy?.email);
  const edited = `Edited ${formatRelative(updatedAt)}`;
  return creator ? `Created by ${creator} · ${edited}` : edited;
}
