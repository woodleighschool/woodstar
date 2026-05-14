import * as React from "react";

import { APIKeyCard } from "@/components/account/api-key-card";
import { runtime } from "@/lib/runtime";

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
  return (
    <div className="flex flex-col gap-6 p-6">
      <APIKeyCard />
      <DefinitionList
        rows={[
          ["Version", runtime.version || "unknown"],
          ["Database", "Postgres (configured via env)"],
          ["Frontend mode", import.meta.env.MODE],
        ]}
      />
    </div>
  );
}
