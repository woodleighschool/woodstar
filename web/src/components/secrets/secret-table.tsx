import { Eye, EyeOff } from "lucide-react";
import { useState } from "react";

import { ErrorState } from "@/components/feedback/error-state";
import { Spinner } from "@/components/feedback/spinner";
import { Button } from "@/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import type { ApiError } from "@/lib/api";
import { formatRelative } from "@/lib/utils";

export interface SecretRow {
  id: string;
  value: string;
  created_at: string;
}

export interface SecretTableProps {
  data: SecretRow[];
  isLoading: boolean;
  error: ApiError | null;
  onRetry?: () => void;
  emptyTitle: string;
  emptyDescription: string;
}

export function SecretTable({ data, isLoading, error, onRetry, emptyTitle, emptyDescription }: SecretTableProps) {
  const [visible, setVisible] = useState<Record<string, boolean>>({});

  if (error) {
    return <ErrorState message={error.message} onRetry={onRetry} />;
  }

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Spinner /> Loading…
      </div>
    );
  }

  if (data.length === 0) {
    return (
      <div className="rounded-md border border-dashed bg-muted/30 px-4 py-6 text-sm">
        <p className="font-medium">{emptyTitle}</p>
        <p className="text-muted-foreground">{emptyDescription}</p>
      </div>
    );
  }

  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Secret</TableHead>
            <TableHead>Created</TableHead>
            <TableHead className="w-12" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.map((row) => {
            const shown = visible[row.id] ?? false;
            return (
              <TableRow key={row.id}>
                <TableCell className="font-mono text-xs">{shown ? row.value : "••••••••••••••••"}</TableCell>
                <TableCell>{formatRelative(row.created_at)}</TableCell>
                <TableCell className="text-right">
                  <Button
                    type="button"
                    size="icon"
                    variant="ghost"
                    onClick={() =>
                      setVisible((current) => ({
                        ...current,
                        [row.id]: !shown,
                      }))
                    }
                    aria-label={shown ? "Hide secret" : "Show secret"}
                  >
                    {shown ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                  </Button>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}
