import { Badge } from "@components/ui/badge";

import { MUNKI_SOFTWARE_ACTIONS, type MunkiSoftwareAction } from "./actions";

export function MunkiActionBadges({ actions }: { actions: readonly MunkiSoftwareAction[] }) {
  return (
    <div className="flex flex-wrap gap-1">
      {actions.map((action) => (
        <Badge key={action} variant="secondary" className="font-normal">
          {MUNKI_SOFTWARE_ACTIONS[action].name}
        </Badge>
      ))}
    </div>
  );
}
