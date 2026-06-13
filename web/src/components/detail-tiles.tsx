import type { ReactNode } from "react";

export interface DetailTile {
  label: string;
  value: ReactNode;
}

// Label/value grid shared by detail-page sections.
export function DetailTiles({ tiles }: { tiles: DetailTile[] }) {
  return (
    <dl className="grid grid-cols-[repeat(auto-fit,minmax(170px,1fr))] gap-x-8 gap-y-5">
      {tiles.map((tile) => (
        <div key={tile.label} className="flex min-w-0 flex-col gap-1">
          <dt className="text-xs font-semibold text-muted-foreground">{tile.label}</dt>
          <dd className="truncate text-sm text-foreground">{tile.value}</dd>
        </div>
      ))}
    </dl>
  );
}
