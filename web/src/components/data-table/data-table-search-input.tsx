import * as React from "react";

import { InputGroup, InputGroupAddon, InputGroupInput } from "@components/ui/input-group";
import { Spinner } from "@components/ui/spinner";
import { useDebouncedCallback } from "@hooks/use-debounced-callback";
import { cn } from "@lib/utils";

interface DataTableSearchInputProps extends Omit<
  React.ComponentProps<typeof InputGroupInput>,
  "value" | "onChange"
> {
  value: string;
  onValueChange: (value: string | undefined) => void;
  debounceMs?: number;
  loading?: boolean;
}

export function DataTableSearchInput({
  value,
  onValueChange,
  debounceMs = 300,
  loading = false,
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
  const searchPending = loading || draft.trim() !== value;

  return (
    <InputGroup
      className={cn("max-w-sm min-w-48 flex-1 basis-48", className)}
      aria-busy={searchPending}
    >
      <InputGroupInput
        {...props}
        placeholder={placeholder}
        value={draft}
        onChange={(event) => {
          setDraft(event.target.value);
          write(event.target.value);
        }}
      />
      <InputGroupAddon
        align="inline-end"
        aria-hidden={!searchPending}
        className={cn(
          "transition-opacity duration-150",
          searchPending ? "opacity-100 delay-150" : "opacity-0 delay-0",
        )}
      >
        <Spinner className="size-3.5" />
      </InputGroupAddon>
    </InputGroup>
  );
}
