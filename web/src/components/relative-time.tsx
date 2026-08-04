import { formatDateTime, formatRelative } from "@lib/utils";

export function RelativeTime({ value, className }: { value: string; className?: string }) {
  const relative = formatRelative(value);
  const exact = formatDateTime(value);

  return (
    <time dateTime={value} title={exact} aria-label={`${relative}; ${exact}`} className={className}>
      {relative}
    </time>
  );
}
