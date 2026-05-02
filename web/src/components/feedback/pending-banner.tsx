import { Construction } from "lucide-react";

export function PendingBanner({ endpoint }: { endpoint: string }) {
  return (
    <div className="flex items-start gap-3 rounded-md border border-dashed bg-muted/40 px-4 py-3 text-sm">
      <Construction className="size-4 mt-0.5 text-muted-foreground" aria-hidden />
      <div className="space-y-1">
        <p className="font-medium">Backend endpoint pending</p>
        <p className="text-muted-foreground">
          This screen is wired to <code className="font-mono text-xs">{endpoint}</code>
          {" "}but the handler hasn&apos;t been implemented yet. The view will populate
          once the route lands.
        </p>
      </div>
    </div>
  );
}
