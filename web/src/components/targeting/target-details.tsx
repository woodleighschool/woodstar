import type { ReactNode } from "react";

import { KeyValueRow, KeyValueSection } from "@components/key-value";
import { Link } from "@components/link";
import { Badge } from "@components/ui/badge";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@components/ui/hover-card";
import { LabelRefList } from "@features/labels/components/label-ref-list";
import type { LabelRef } from "@lib/api";

export function TargetBadge({
  labelID,
  label,
  details,
}: {
  labelID: number;
  label: string;
  details: readonly { label: string; value: ReactNode }[];
}) {
  return (
    <HoverCard>
      <HoverCardTrigger
        render={
          <Badge
            variant="outline"
            className="font-normal"
            render={<Link to="/labels/$id" params={{ id: String(labelID) }} />}
          />
        }
      >
        {label}
      </HoverCardTrigger>
      <HoverCardContent align="start" className="w-80">
        <dl className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-2">
          {details.map((detail) => (
            <div key={detail.label} className="contents">
              <dt className="text-muted-foreground">{detail.label}</dt>
              <dd className="min-w-0 text-right">{detail.value}</dd>
            </div>
          ))}
        </dl>
      </HoverCardContent>
    </HoverCard>
  );
}

export function TargetDetails({
  include,
  excludeLabelIDs,
}: {
  include: ReactNode;
  excludeLabelIDs: readonly number[];
}) {
  return (
    <KeyValueSection title="Targets">
      <KeyValueRow label="Include" value={include} />
      <KeyValueRow label="Exclude" value={<LabelRefList labelIDs={excludeLabelIDs} />} />
    </KeyValueSection>
  );
}

export function LabelTargetDetails({
  targets,
}: {
  targets: {
    include: readonly LabelRef[];
    exclude: readonly LabelRef[];
  };
}) {
  return (
    <TargetDetails
      include={<LabelRefList labelIDs={targets.include.map((target) => target.label_id)} />}
      excludeLabelIDs={targets.exclude.map((target) => target.label_id)}
    />
  );
}
