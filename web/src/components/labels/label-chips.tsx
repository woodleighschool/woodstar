import { Link } from "@tanstack/react-router";

import type { LabelChip } from "@/components/labels/label-chip-utils";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export function LabelChips({ labels, className }: { labels: LabelChip[]; className?: string }) {
  return (
    <div className={cn("flex flex-wrap gap-1.5", className)}>
      {labels.map((label) => (
        <Button key={label.id} asChild size="xs" variant="outline" className="font-normal">
          <Link to="/hosts" search={{ label_id: String(label.id) }}>
            {label.name}
          </Link>
        </Button>
      ))}
    </div>
  );
}
