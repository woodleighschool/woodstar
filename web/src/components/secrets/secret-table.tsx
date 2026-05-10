import { Eye, EyeOff, KeyRound, Loader2 } from "lucide-react";
import { useState } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import type { ApiError } from "@/lib/api";
import { formatRelative } from "@/lib/utils";

export interface SecretRow {
  id: number;
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
  const [visible, setVisible] = useState<Record<number, boolean>>({});

  if (error) {
    return (
      <Alert variant="destructive">
        <AlertTitle>Failed to load secrets</AlertTitle>
        <AlertDescription>{error.message}</AlertDescription>
        {onRetry ? (
          <Button variant="outline" size="sm" onClick={onRetry} className="mt-2 w-fit">
            Retry
          </Button>
        ) : null}
      </Alert>
    );
  }

  if (isLoading) {
    return (
      <div className="text-muted-foreground flex items-center gap-2 text-sm">
        <Loader2 className="size-4 animate-spin" /> Loading...
      </div>
    );
  }

  if (data.length === 0) {
    return (
      <Empty>
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <KeyRound />
          </EmptyMedia>
          <EmptyTitle>{emptyTitle}</EmptyTitle>
          <EmptyDescription>{emptyDescription}</EmptyDescription>
        </EmptyHeader>
      </Empty>
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
                <TableCell>{shown ? row.value : "••••••••••••••••"}</TableCell>
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
