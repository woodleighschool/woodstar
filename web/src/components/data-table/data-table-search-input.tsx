import * as React from "react";

import { Input } from "@components/ui/input";
import { useDebouncedCallback } from "@hooks/use-debounced-callback";
import { cn } from "@lib/utils";

interface DataTableSearchInputProps extends Omit<
  React.ComponentProps<typeof Input>,
  "value" | "onChange"
> {
  value: string;
  onValueChange: (value: string | undefined) => void;
  debounceMs?: number;
}

export function DataTableSearchInput({
  value,
  onValueChange,
  debounceMs = 300,
  placeholder = "Search",
  className,
  ...props
}: DataTableSearchInputProps) {
  const [draft, setDraft] = React.useState(value);
  const [previousValue, setPreviousValue] = React.useState(value);

  if (previousValue !== value) {
    setPreviousValue(value);
    setDraft(value);
  }

  const write = useDebouncedCallback((nextValue: string) => {
    const trimmed = nextValue.trim();
    onValueChange(trimmed === "" ? undefined : trimmed);
  }, debounceMs);

  return (
    <Input
      {...props}
      className={cn("w-full max-w-sm", className)}
      placeholder={placeholder}
      value={draft}
      onChange={(event) => {
        setDraft(event.target.value);
        write(event.target.value);
      }}
    />
  );
}
