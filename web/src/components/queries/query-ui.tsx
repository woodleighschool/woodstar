import { Link } from "@tanstack/react-router";
import { Download, FileCode2, Play, Settings2 } from "lucide-react";
import type { ReactNode } from "react";

import { SQLEditor } from "@/components/editor/sql-editor";
import { selectedPlatformIconTargets } from "@/components/platform/platform-icon-data";
import { PlatformIconList } from "@/components/platform/platform-icons";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import type { Schemas } from "@/lib/api";
import { targetScopeLabel } from "@/lib/targeting";
import { cn, formatInterval } from "@/lib/utils";

type LabelScope = Schemas["LabelScope"];

export function PageLead({ title, description, actions }: { title: string; description: string; actions?: ReactNode }) {
  return (
    <div className="flex flex-wrap items-start justify-between gap-4">
      <div className="flex min-w-0 flex-col gap-1">
        <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
        <p className="text-muted-foreground max-w-3xl text-sm">{description}</p>
      </div>
      {actions ? <div className="flex shrink-0 items-center gap-2">{actions}</div> : null}
    </div>
  );
}

export function PlatformBadge({ platform }: { platform?: string | null }) {
  return <PlatformIconList platforms={selectedPlatformIconTargets(platform)} iconClassName="size-4" />;
}

export function TargetSummary({ scope, platform }: { scope?: LabelScope; platform?: string | null }) {
  return (
    <span className="inline-flex min-w-0 items-center gap-2">
      <span>{targetScopeLabel(scope)}</span>
      <PlatformIconList platforms={selectedPlatformIconTargets(platform)} iconClassName="size-3.5" />
    </span>
  );
}

export function IntervalIndicator({ interval }: { interval?: number | null }) {
  if (interval) return <span>Every {formatInterval(interval)}</span>;
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="text-muted-foreground">Off</span>
        </TooltipTrigger>
        <TooltipContent>Assign an interval to collect report data on a schedule.</TooltipContent>
      </Tooltip>
    </TooltipProvider>
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
          Show query
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>Query</DialogTitle>
          <DialogDescription>SQL used by this item.</DialogDescription>
        </DialogHeader>
        <SQLEditor
          value={sql}
          onChange={() => null}
          readOnly
          className="max-h-[60vh] overflow-auto"
        />
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
        Run live
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
      Export results
    </Button>
  );
}
