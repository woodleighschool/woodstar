import { Link } from "@tanstack/react-router";
import { FileCode2, Play } from "lucide-react";
import { lazy, Suspense, type ReactNode } from "react";

import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { useAuth } from "@/hooks/use-auth";
import { cn } from "@/lib/utils";

const LazySQLEditor = lazy(() =>
  import("@/components/editor/sql-editor").then((module) => ({ default: module.SQLEditor })),
);

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
          <LazySQLEditor
            value={sql}
            onChange={() => null}
            readOnly
            className="[&_.cm-scroller]:max-h-[60vh] [&_.cm-scroller]:overflow-auto"
          />
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
  const { user } = useAuth();
  const isAdmin = user?.role === "admin";

  if (!isAdmin) return null;

  return (
    <Button asChild variant="outline" size="sm">
      <Link to={to} params={params} search={search}>
        <Play data-icon="inline-start" />
        Run Live
      </Link>
    </Button>
  );
}
