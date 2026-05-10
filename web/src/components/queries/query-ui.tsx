import { Link } from "@tanstack/react-router";
import { Download, FileCode2, Play, Settings2 } from "lucide-react";
import type { ReactNode } from "react";

import { SQLEditor } from "@/components/editor/sql-editor";
import { Badge } from "@/components/ui/badge";
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
import { platformLabel, targetSummary } from "@/lib/targeting";
import { cn, formatInterval } from "@/lib/utils";

type LabelScope = Schemas["LabelScopeBody"];

export function PageLead({ title, description, actions }: { title: string; description: string; actions?: ReactNode }) {
  return (
    <div className="flex flex-wrap items-start justify-between gap-4">
      <div className="min-w-0 space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
        <p className="text-muted-foreground max-w-3xl text-sm">{description}</p>
      </div>
      {actions ? <div className="flex shrink-0 items-center gap-2">{actions}</div> : null}
    </div>
  );
}

export function PlatformBadge({ platform }: { platform?: string | null }) {
  return (
    <Badge variant="outline" className="capitalize">
      {platformLabel(platform)}
    </Badge>
  );
}

export function TargetSummary({ scope, platform }: { scope?: LabelScope; platform?: string | null }) {
  return <span>{targetSummary(scope, platform)}</span>;
}

export function IntervalIndicator({ interval }: { interval?: number | null }) {
  if (!interval) {
    return (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <Badge variant="muted">Off</Badge>
          </TooltipTrigger>
          <TooltipContent>Assign an interval to collect report data on a schedule.</TooltipContent>
        </Tooltip>
      </TooltipProvider>
    );
  }

  return (
    <div className="flex flex-wrap items-center gap-2">
      <Badge variant="default">On</Badge>
      <span className="text-muted-foreground text-xs">Every {formatInterval(interval)}</span>
    </div>
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
          <FileCode2 className="size-4" />
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
          minHeight="12rem"
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
        <Play className="size-4" />
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
        <Settings2 className="size-4" />
        {children}
      </Link>
    </Button>
  );
}

export function ExportButton({ disabled, onClick }: { disabled?: boolean; onClick: () => void }) {
  return (
    <Button variant="outline" size="sm" disabled={disabled} onClick={onClick}>
      <Download className="size-4" />
      Export results
    </Button>
  );
}
