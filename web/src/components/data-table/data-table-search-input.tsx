import { parseAsInteger, parseAsString, useQueryStates } from "nuqs";
import * as React from "react";

import { Input } from "@/components/ui/input";
import { useDebouncedCallback } from "@/hooks/use-debounced-callback";

interface DataTableSearchInputProps extends Omit<React.ComponentProps<typeof Input>, "value" | "onChange"> {
  debounceMs?: number;
}

// Standalone server search bound to the nuqs `q` key. A local draft keeps the
// input responsive while the URL (and therefore the fetch) updates debounced.
// Writing `q` resets `page` to its default, mirroring the facet-filter behaviour.
export function DataTableSearchInput({
  debounceMs = 300,
  placeholder = "Search",
  ...props
}: DataTableSearchInputProps) {
  const [{ q }, setSearch] = useQueryStates({
    q: parseAsString.withDefault(""),
    page: parseAsInteger.withDefault(1),
  });
  const [draft, setDraft] = React.useState(q);
  const [prev, setPrev] = React.useState(q);

  if (prev !== q) {
    setPrev(q);
    setDraft(q);
  }

  const write = useDebouncedCallback((value: string) => {
    void setSearch({ q: value.trim() === "" ? null : value, page: null });
  }, debounceMs);

  return (
    <Input
      {...props}
      placeholder={placeholder}
      value={draft}
      onChange={(event) => {
        setDraft(event.target.value);
        write(event.target.value);
      }}
    />
  );
}
