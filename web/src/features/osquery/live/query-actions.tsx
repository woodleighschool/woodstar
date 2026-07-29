import { FileCode2, Play } from "lucide-react";
import { lazy, Suspense } from "react";

import { Button } from "@components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@components/ui/dialog";
import { useAuth } from "@features/auth/queries";

import { useOpenOsqueryLive } from "./history";

const LazySQLEditor = lazy(() =>
  import("@components/editor/sql-editor").then((module) => ({
    default: module.SQLEditor,
  })),
);

export function ShowQueryButton({ sql }: { sql: string }) {
  return (
    <Dialog>
      <DialogTrigger render={<Button variant="outline" size="sm" />}>
        <FileCode2 data-icon="inline-start" />
        Show Query
      </DialogTrigger>
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>Query</DialogTitle>
        </DialogHeader>
        <Suspense fallback={<div className="h-40 rounded-md bg-muted" />}>
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
  kind,
  id,
  sql,
}: {
  kind: "report" | "check";
  id: number;
  sql: string;
}) {
  const { user } = useAuth();
  const openLive = useOpenOsqueryLive();
  const isAdmin = user?.role === "admin";
  if (!isAdmin) return null;
  return (
    <Button variant="outline" size="sm" onClick={() => void openLive({ kind, id, sql })}>
      <Play data-icon="inline-start" />
      Run Live
    </Button>
  );
}
