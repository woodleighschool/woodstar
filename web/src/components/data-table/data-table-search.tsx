import { SearchIcon, XIcon } from "lucide-react";

import { InputGroup, InputGroupAddon, InputGroupButton, InputGroupInput } from "@/components/ui/input-group";
import { cn } from "@/lib/utils";

interface DataTableSearchProps {
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  label: string;
  className?: string;
}

export function DataTableSearch({ value, onChange, placeholder, label, className }: DataTableSearchProps) {
  return (
    <InputGroup className={cn("max-w-md flex-1", className)}>
      <InputGroupInput
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
        aria-label={label}
      />
      <InputGroupAddon align="inline-start">
        <SearchIcon />
      </InputGroupAddon>
      {value ? (
        <InputGroupAddon align="inline-end">
          <InputGroupButton size="icon-xs" onClick={() => onChange("")}>
            <XIcon />
          </InputGroupButton>
        </InputGroupAddon>
      ) : null}
    </InputGroup>
  );
}
