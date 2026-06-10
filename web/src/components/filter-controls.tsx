import { XIcon } from "lucide-react";
import type { ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export function FilterChip({
  label,
  value,
  onRemove,
  className,
}: {
  label: string;
  value: ReactNode;
  onRemove: () => void;
  className?: string;
}) {
  return (
    <Button
      type="button"
      variant="outline"
      size="sm"
      className={cn("h-8 border-dashed font-normal", className)}
      onClick={onRemove}
    >
      <span className="text-muted-foreground">{label}:</span>
      <span className="max-w-48 truncate">{value}</span>
      <XIcon />
    </Button>
  );
}
