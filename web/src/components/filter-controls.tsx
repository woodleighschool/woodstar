import { XIcon } from "lucide-react";
import type { ComponentType, ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { cn } from "@/lib/utils";

export interface FilterOption {
  value: string;
  label: string;
  icon?: ComponentType<{ className?: string }>;
}

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

export function FilterSelect({
  label,
  value,
  options,
  onChange,
  className,
}: {
  label: string;
  value: string;
  options: FilterOption[];
  onChange: (next: string) => void;
  className?: string;
}) {
  return (
    <Select value={value} onValueChange={onChange}>
      <SelectTrigger size="sm" className={cn("h-8 border-dashed font-normal", className)}>
        <span className="text-muted-foreground">{label}:</span>
        <SelectValue />
      </SelectTrigger>
      <SelectContent align="start">
        {options.map((option) => (
          <SelectItem key={option.value} value={option.value}>
            {option.icon ? <option.icon className="text-muted-foreground" /> : null}
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
