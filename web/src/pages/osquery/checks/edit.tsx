import { useNavigate, useParams } from "@tanstack/react-router";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { Loader2 } from "lucide-react";
import { useRef, useState } from "react";

import { SchemaSidebar } from "@/components/editor/schema-sidebar";
import { SQLEditor } from "@/components/editor/sql-editor";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { LabelScopeSelector } from "@/components/queries/label-scope-selector";
import { LiveRunButton } from "@/components/queries/query-ui";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useCheck, useCreateCheck, useUpdateCheck, type CheckMutation } from "@/hooks/use-checks";
import { useSchemaSidebar } from "@/hooks/use-schema-sidebar";
import { cn } from "@/lib/utils";

const emptyCheck: CheckMutation = {
  name: "",
  description: "",
  query: "select 1;",
  label_scope: {},
};

export function CheckEditPage({ mode }: { mode: "create" | "edit" }) {
  const params = useParams({ strict: false });
  const checkId = params.checkId ?? "";
  const checkID = mode === "edit" ? Number(checkId) : null;
  const detail = useCheck(checkID);

  if (mode === "edit") {
    if (detail.error) {
      return (
        <PageShell>
          <Alert variant="destructive">
            <AlertTitle>Failed to load check</AlertTitle>
            <AlertDescription>{detail.error.message}</AlertDescription>
          </Alert>
        </PageShell>
      );
    }
    if (!detail.data) {
      return (
        <PageShell className="text-muted-foreground flex-row items-center gap-2 text-sm">
          <Loader2 className="size-4 animate-spin" /> Loading check...
        </PageShell>
      );
    }
  }

  const initial: CheckMutation =
    mode === "edit" && detail.data
      ? {
          name: detail.data.name,
          description: detail.data.description,
          query: detail.data.query,
          label_scope: detail.data.label_scope ?? {},
        }
      : emptyCheck;

  return <CheckEditForm key={checkId || "new"} mode={mode} checkId={checkID} checkParam={checkId} initial={initial} />;
}

export function CheckCreatePage() {
  return <CheckEditPage mode="create" />;
}

export function CheckUpdatePage() {
  return <CheckEditPage mode="edit" />;
}

function CheckEditForm({
  mode,
  checkId,
  checkParam,
  initial,
}: {
  mode: "create" | "edit";
  checkId: number | null;
  checkParam: string;
  initial: CheckMutation;
}) {
  const navigate = useNavigate();
  const createCheck = useCreateCheck();
  const updateCheck = useUpdateCheck(checkId);
  const [form, setForm] = useState<CheckMutation>(initial);
  const [schemaOpen, setSchemaOpen] = useSchemaSidebar();
  const editorRef = useRef<ReactCodeMirrorRef>(null);

  const pending = createCheck.isPending || updateCheck.isPending;

  async function submit() {
    const payload = trimCheck(form);
    const saved = mode === "create" ? await createCheck.mutateAsync(payload) : await updateCheck.mutateAsync(payload);
    void navigate({ to: "/osquery/checks/$checkId", params: { checkId: String(saved.id) } });
  }

  function insertAtCursor(snippet: string) {
    const view = editorRef.current?.view;
    if (!view) {
      setForm((prev) => ({ ...prev, query: prev.query + " " + snippet }));
      return;
    }
    view.dispatch({ changes: { from: view.state.selection.main.from, insert: snippet } });
  }

  return (
    <PageShell asChild className={cn("h-full transition-[padding] duration-200 ease-out", schemaOpen && "pr-[21rem]")}>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          void submit();
        }}
      >
        <PageHeader title={mode === "create" ? "New check" : "Edit check"} />
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor="check-name">Name</FieldLabel>
            <Input
              id="check-name"
              required
              value={form.name}
              onChange={(event) => setForm({ ...form, name: event.target.value })}
            />
          </Field>

          <Field>
            <FieldLabel htmlFor="check-description">Description</FieldLabel>
            <Textarea
              id="check-description"
              rows={3}
              placeholder="What failure means"
              value={form.description ?? ""}
              onChange={(event) => setForm({ ...form, description: event.target.value })}
            />
          </Field>
        </FieldGroup>

        <LabelScopeSelector
          entity="check"
          value={form.label_scope}
          onChange={(label_scope) => setForm({ ...form, label_scope })}
        />

        <div className="grid gap-2">
          <Label>Query</Label>
          <SQLEditor
            ref={editorRef}
            value={form.query}
            onChange={(query) => setForm({ ...form, query })}
            placeholder="SELECT ..."
          />
        </div>
        <div className="flex items-center gap-2 border-t pt-4">
          <Button type="submit" size="sm" disabled={pending}>
            {pending ? "Saving..." : "Save"}
          </Button>
          {mode === "edit" ? (
            <LiveRunButton to="/osquery/checks/$checkId/live" params={{ checkId: checkParam }} />
          ) : null}
        </div>
        <SchemaSidebar open={schemaOpen} onOpenChange={setSchemaOpen} onInsertColumn={insertAtCursor} />
      </form>
    </PageShell>
  );
}

function trimCheck(form: CheckMutation): CheckMutation {
  return {
    ...form,
    name: form.name.trim(),
    description: form.description?.trim() ?? "",
    query: form.query.trim(),
  };
}
