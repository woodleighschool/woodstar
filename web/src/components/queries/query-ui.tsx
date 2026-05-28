import { Link } from "@tanstack/react-router";
import { Download, FileCode2, Play, Settings2 } from "lucide-react";
import { lazy, Suspense, type ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { Schemas } from "@/lib/api";
import { targetScopeLabel } from "@/lib/targeting";
import { cn, formatInterval } from "@/lib/utils";

type LabelScope = Schemas["LabelScope"];

const LazySQLEditor = lazy(() =>
  import("@/components/editor/sql-editor").then((module) => ({ default: module.SQLEditor })),
);

export function TargetSummary({ scope }: { scope?: LabelScope }) {
  return (
    <span className="inline-flex min-w-0 items-center gap-2">
      <span>{targetScopeLabel(scope)}</span>
    </span>
  );
}

export function IntervalIndicator({ interval }: { interval?: number | null }) {
  if (interval) return <span>Every {formatInterval(interval)}</span>;
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="text-muted-foreground">Off</span>
      </TooltipTrigger>
      <TooltipContent>Runs only on demand.</TooltipContent>
    </Tooltip>
  );
}

export function DetailSettings({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div className={cn("border-y bg-muted/20 px-4 py-3", className)}>
      <div className="flex flex-wrap items-center gap-x-6 gap-y-2 text-sm">{children}</div>
    </div>
  );
}

export function SettingItem({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-muted-foreground">{label}:</span>
      <div className="font-medium">{children}</div>
    </div>
  );
}

export function ShowQueryButton({ sql }: { sql: string }) {
  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm">
          <FileCode2 data-icon="inline-start" />
          Show Query
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>Query</DialogTitle>
        </DialogHeader>
        <Suspense fallback={<div className="bg-muted h-40 rounded-md" />}>
          <LazySQLEditor value={sql} onChange={() => null} readOnly className="max-h-[60vh] overflow-auto" />
        </Suspense>
      </DialogContent>
    </Dialog>
  );
}

export function LiveRunButton({
  to,
  params,
  search,
}: {
  to: string;
  params?: Record<string, string>;
  search?: Record<string, string>;
}) {
  return (
    <Button asChild variant="outline" size="sm">
      <Link to={to} params={params} search={search}>
        <Play data-icon="inline-start" />
        Run Live
      </Link>
    </Button>
  );
}

export function EditButton({
  to,
  params,
  children,
}: {
  to: string;
  params?: Record<string, string>;
  children: string;
}) {
  return (
    <Button asChild size="sm">
      <Link to={to} params={params}>
        <Settings2 data-icon="inline-start" />
        {children}
      </Link>
    </Button>
  );
}

export function ExportButton({ disabled, onClick }: { disabled?: boolean; onClick: () => void }) {
  return (
    <Button variant="outline" size="sm" disabled={disabled} onClick={onClick}>
      <Download data-icon="inline-start" />
      Export Results
    </Button>
  );
}
