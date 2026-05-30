import { useMemo } from "react";

import { LabelPicker } from "@/components/labels/label-picker";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { LabelScope } from "@/lib/api";

type LabelScopeMode = NonNullable<LabelScope["mode"]>;

const includeAny: LabelScopeMode = "include_any";
const includeAll: LabelScopeMode = "include_all";
const excludeAny: LabelScopeMode = "exclude_any";
const targetAll = "all";
const targetCustom = "custom";

export function LabelScopeSelector({
  value,
  onChange,
  entity = "report",
}: {
  value: LabelScope | undefined;
  onChange: (next: LabelScope) => void;
  entity?: "report" | "check";
}) {
  const selectedTarget = value?.mode ? targetCustom : targetAll;
  const selectedMode = value?.mode ?? includeAny;
  const selectedLabelIDs = value?.label_ids ?? [];
  const modeOptions = useMemo(() => targetModeOptions(entity), [entity]);

  function setTarget(next: string) {
    if (next === targetAll) {
      onChange({});
      return;
    }
    onChange({ mode: selectedMode, label_ids: value?.label_ids ?? [] });
  }

  function setMode(next: string) {
    onChange({ mode: next as LabelScopeMode, label_ids: value?.label_ids ?? [] });
  }

  return (
    <div className="grid gap-3 rounded-md border p-4">
      <div className="grid gap-1">
        <Label>Target</Label>
        <p className="text-muted-foreground text-xs">
          {entity === "check" ? "Choose which hosts this check evaluates." : "Choose which hosts this report targets."}
        </p>
      </div>

      <RadioGroup value={selectedTarget} onValueChange={setTarget} className="gap-2">
        <div className="flex items-center gap-2">
          <RadioGroupItem id={`${entity}-target-all`} value={targetAll} />
          <Label htmlFor={`${entity}-target-all`} className="font-normal">
            All Hosts
          </Label>
        </div>
        <div className="flex items-center gap-2">
          <RadioGroupItem id={`${entity}-target-custom`} value={targetCustom} />
          <Label htmlFor={`${entity}-target-custom`} className="font-normal">
            Custom
          </Label>
        </div>
      </RadioGroup>

      {selectedTarget === targetCustom ? (
        <div className="grid gap-3 pl-5">
          <Select value={selectedMode} onValueChange={setMode}>
            <SelectTrigger className="w-52">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {modeOptions.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
          <p className="text-muted-foreground text-xs">
            {modeOptions.find((option) => option.value === selectedMode)?.helpText}
          </p>

          <LabelPicker
            value={selectedLabelIDs}
            onChange={(label_ids) => onChange({ mode: selectedMode, label_ids })}
            emptyMessage="Create a label before using a custom target."
            emptyPlaceholder="No Labels Available"
            placeholder="Add Label"
          />
        </div>
      ) : null}
    </div>
  );
}

function targetModeOptions(entity: "report" | "check") {
  const noun = entity === "check" ? "Check" : "Report";
  const options = [
    { value: includeAny, label: "Include Any", helpText: `${noun} will target hosts that have any of these labels.` },
    { value: includeAll, label: "Include All", helpText: `${noun} will target hosts that have all of these labels.` },
  ];
  if (entity === "check") {
    options.push({
      value: excludeAny,
      label: "Exclude Any",
      helpText: `${noun} will target hosts that don't have any of these labels.`,
    });
  }
  return options;
}
