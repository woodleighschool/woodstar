import * as React from "react";

import { useVersion } from "@/hooks/use-version";

function DefinitionList({ rows }: { rows: Array<[string, React.ReactNode]> }) {
  return (
    <dl className="bg-card divide-y rounded-lg border">
      {rows.map(([label, value]) => (
        <div key={label} className="grid grid-cols-[10rem_1fr] gap-3 px-4 py-2 text-sm">
          <dt className="text-muted-foreground">{label}</dt>
          <dd className="font-medium break-all">{value ?? "-"}</dd>
        </div>
      ))}
    </dl>
  );
}

export function SettingsPage() {
  const { data: version, isLoading } = useVersion();

  return (
    <div className="p-6">
      <DefinitionList
        rows={[
          ["Version", isLoading ? "loading..." : (version?.version ?? "unknown")],
          [
            "Started at",
            isLoading ? "loading..." : version?.started_at ? new Date(version.started_at).toLocaleString() : "unknown",
          ],
          ["Database", "Postgres (configured via env)"],
          ["Frontend mode", import.meta.env.MODE],
        ]}
      />
    </div>
  );
}
