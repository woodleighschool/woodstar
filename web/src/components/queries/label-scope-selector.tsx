import { Check, CircleSlash } from "lucide-react";
import { useMemo } from "react";

import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useLabels } from "@/hooks/use-labels";
import type { Schemas } from "@/lib/api";
import { cn } from "@/lib/utils";

type LabelScope = Schemas["LabelScopeBody"];
type LabelScopeMode = NonNullable<LabelScope["mode"]>;

const includeAny: LabelScopeMode = "include_any";
const includeAll: LabelScopeMode = "include_all";
const excludeAny: LabelScopeMode = "exclude_any";

export function LabelScopeSelector({
  value,
  onChange,
  entity = "report",
}: {
  value: LabelScope | undefined;
  onChange: (next: LabelScope) => void;
  entity?: "report" | "check";
}) {
  const labels = useLabels({ per_page: 500, order_key: "name", order_direction: "asc", kind: "regular" });
  const selectedTarget = value?.mode ? "Custom" : "All hosts";
  const selectedMode = value?.mode ?? includeAny;
  const selected = new Set(value?.label_ids ?? []);
  const labelRows = labels.data?.items ?? [];
  const modeOptions = useMemo(() => targetModeOptions(entity), [entity]);

  function setTarget(next: string) {
    if (next === "All hosts") {
      onChange({});
      return;
    }
    onChange({ mode: selectedMode, label_ids: value?.label_ids ?? [] });
  }

  function setMode(next: string) {
    onChange({ mode: next as LabelScopeMode, label_ids: value?.label_ids ?? [] });
  }

  function toggleLabel(id: number) {
    const next = new Set(selected);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    onChange({ mode: selectedMode, label_ids: [...next] });
  }

  return (
    <div className="grid max-w-3xl gap-3 rounded-md border p-4">
      <div className="grid gap-1">
        <Label>Target</Label>
        <p className="text-muted-foreground text-xs">
          {entity === "check" ? "Choose which hosts this check evaluates." : "Choose which hosts this report targets."}
        </p>
      </div>

      <div className="grid gap-2">
        <label className="flex items-center gap-2 text-sm">
          <input
            type="radio"
            name={`${entity}-target-type`}
            checked={selectedTarget === "All hosts"}
            onChange={() => setTarget("All hosts")}
          />
          All hosts
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="radio"
            name={`${entity}-target-type`}
            checked={selectedTarget === "Custom"}
            onChange={() => setTarget("Custom")}
          />
          Custom
        </label>
      </div>

      {selectedTarget === "Custom" ? (
        <div className="grid gap-3 pl-5">
          <Select value={selectedMode} onValueChange={setMode}>
            <SelectTrigger className="w-52">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {modeOptions.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <p className="text-muted-foreground text-xs">
            {modeOptions.find((option) => option.value === selectedMode)?.helpText}
          </p>

          <div className="grid max-h-56 gap-1 overflow-auto rounded-md border p-2">
            {labelRows.length === 0 ? (
              <div className="text-muted-foreground flex items-center gap-2 px-2 py-3 text-sm">
                <CircleSlash className="size-4" /> Add a custom label to target specific hosts.
              </div>
            ) : (
              labelRows.map((label) => {
                const checked = selected.has(label.id);
                return (
                  <button
                    key={label.id}
                    type="button"
                    className={cn(
                      "hover:bg-muted flex items-center gap-3 rounded-md px-2 py-2 text-left text-sm",
                      checked && "bg-muted",
                    )}
                    onClick={() => toggleLabel(label.id)}
                  >
                    <Checkbox checked={checked} tabIndex={-1} aria-hidden />
                    <span className="min-w-0 flex-1 truncate">{label.name}</span>
                    {checked ? <Check className="text-primary size-4" /> : null}
                  </button>
                );
              })
            )}
          </div>
        </div>
      ) : null}
    </div>
  );
}

function targetModeOptions(entity: "report" | "check") {
  const noun = entity === "check" ? "Check" : "Report";
  const options = [
    { value: includeAny, label: "Include any", helpText: `${noun} will target hosts that have any of these labels.` },
    { value: includeAll, label: "Include all", helpText: `${noun} will target hosts that have all of these labels.` },
  ];
  if (entity === "check") {
    options.push({
      value: excludeAny,
      label: "Exclude any",
      helpText: `${noun} will target hosts that don't have any of these labels.`,
    });
  }
  return options;
}
