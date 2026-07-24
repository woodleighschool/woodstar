import { Link } from "@/components/link";
import { Button } from "@/components/ui/button";
import type { Label } from "@/lib/api";
import { cn } from "@/lib/utils";

interface LabelChip {
  id: number;
  name: string;
  builtin_key?: Label["builtin_key"];
}

export function LabelChips({ labels, className }: { labels: LabelChip[]; className?: string }) {
  return (
    <div className={cn("flex flex-wrap gap-1.5", className)}>
      {labels.map((label) => (
        <Button
          key={label.id}
          size="xs"
          variant="outline"
          className="font-normal"
          render={<Link to="/hosts" search={{ label_id: label.id }} />}
          nativeButton={false}
        >
          {label.name}
        </Button>
      ))}
    </div>
  );
}
