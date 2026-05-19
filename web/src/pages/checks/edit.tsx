import { useNavigate, useParams } from "@tanstack/react-router";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { Loader2 } from "lucide-react";
import { useRef, useState } from "react";

import { SchemaSidebar } from "@/components/editor/schema-sidebar";
import { SQLEditor } from "@/components/editor/sql-editor";
import { LabelScopeSelector } from "@/components/queries/label-scope-selector";
import { PlatformSelector } from "@/components/queries/platform-selector";
import { LiveRunButton, PageLead } from "@/components/queries/query-ui";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
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
  const detail = useCheck(checkId);

  if (mode === "edit") {
    if (detail.error) {
      return (
        <div className="p-6">
          <Alert variant="destructive">
            <AlertTitle>Failed to load check</AlertTitle>
            <AlertDescription>{detail.error.message}</AlertDescription>
          </Alert>
        </div>
      );
    }
    if (!detail.data) {
      return (
        <div className="text-muted-foreground flex items-center gap-2 p-6 text-sm">
          <Loader2 className="size-4 animate-spin" /> Loading check...
        </div>
      );
    }
  }

  const initial: CheckMutation =
    mode === "edit" && detail.data
      ? {
          name: detail.data.name,
          description: detail.data.description,
          query: detail.data.query,
          platform: detail.data.platform,
          label_scope: detail.data.label_scope ?? {},
        }
      : emptyCheck;

  return <CheckEditForm key={checkId || "new"} mode={mode} checkId={checkId} initial={initial} />;
}

function CheckEditForm({
  mode,
  checkId,
  initial,
}: {
  mode: "create" | "edit";
  checkId: string;
  initial: CheckMutation;
}) {
  const navigate = useNavigate();
  const createCheck = useCreateCheck();
  const updateCheck = useUpdateCheck(checkId);
  const [form, setForm] = useState<CheckMutation>(initial);
  const [schemaOpen, setSchemaOpen] = useSchemaSidebar();
  const editorRef = useRef<ReactCodeMirrorRef>(null);

  const error = createCheck.error ?? updateCheck.error;
  const pending = createCheck.isPending || updateCheck.isPending;

  async function submit() {
    const saved = mode === "create" ? await createCheck.mutateAsync(form) : await updateCheck.mutateAsync(form);
    void navigate({ to: "/checks/$checkId", params: { checkId: String(saved.id) } });
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
    <form
      className={cn(
        "flex h-full flex-col gap-5 p-6 transition-[padding] duration-200 ease-out",
        schemaOpen && "pr-[21rem]",
      )}
      onSubmit={(event) => {
        event.preventDefault();
        void submit();
      }}
    >
      <PageLead
        title={mode === "create" ? "New check" : "Edit check"}
        description="Checks pass when their SQL returns rows and fail when it returns none."
      />
      {error ? (
        <Alert variant="destructive">
          <AlertTitle>Unable to save check</AlertTitle>
          <AlertDescription>{error.message}</AlertDescription>
        </Alert>
      ) : null}
      <div className="grid max-w-3xl gap-4">
        <div className="grid gap-2">
          <Label htmlFor="check-name">Name</Label>
          <Input
            id="check-name"
            required
            value={form.name}
            onChange={(event) => setForm({ ...form, name: event.target.value })}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="check-description">Description</Label>
          <Textarea
            id="check-description"
            rows={3}
            placeholder="How does this check's failure put the organization at risk?"
            value={form.description ?? ""}
            onChange={(event) => setForm({ ...form, description: event.target.value })}
          />
        </div>
      </div>

      <div className="grid gap-4">
        <div className="grid max-w-3xl gap-4">
          <PlatformSelector value={form.platform} onChange={(platform) => setForm({ ...form, platform })} />
        </div>
        <LabelScopeSelector
          entity="check"
          value={form.label_scope}
          onChange={(label_scope) => setForm({ ...form, label_scope })}
        />
      </div>

      <div className="grid max-w-5xl gap-2">
        <Label>Query</Label>
        <SQLEditor
          ref={editorRef}
          value={form.query}
          onChange={(query) => setForm({ ...form, query })}
          placeholder="SELECT ..."
        />
      </div>
      <div className="flex max-w-3xl items-center gap-2 border-t pt-4">
        <Button type="submit" size="sm" disabled={pending}>
          {pending ? "Saving..." : "Save"}
        </Button>
        {mode === "edit" ? <LiveRunButton to="/checks/$checkId/live" params={{ checkId }} /> : null}
      </div>
      <SchemaSidebar open={schemaOpen} onOpenChange={setSchemaOpen} onInsertColumn={insertAtCursor} />
    </form>
  );
}
