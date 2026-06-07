import { useForm } from "@tanstack/react-form";
import { Link, useNavigate, useParams } from "@tanstack/react-router";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { Loader2 } from "lucide-react";
import { useCallback, useRef, useState } from "react";

import { SchemaSidebar } from "@/components/editor/schema-sidebar";
import { SQLEditor } from "@/components/editor/sql-editor";
import { MutableResourceTabs } from "@/components/layout/mutable-resource-tabs";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { LiveRunButton, SettingItem } from "@/components/queries/query-ui";
import { LabelScopeEditor } from "@/components/targeting/label-scope-editor";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Field, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { useCheck, useCreateCheck, useUpdateCheck, type CheckMutation } from "@/hooks/use-checks";
import { useSchemaSidebar } from "@/hooks/use-schema-sidebar";
import { firstErrorMessage, requiredString } from "@/lib/form-validation";
import { invalidSQLSyntaxMessage, validSQLSyntax } from "@/lib/sql-validation";
import { emptyLabelTargetSet, normalizeLabelTargetSet } from "@/lib/targeting";
import { cn } from "@/lib/utils";

const emptyCheck: CheckMutation = {
  name: "",
  description: "",
  query: "select 1;",
  targets: emptyLabelTargetSet(),
};

const checkQuerySchema = requiredString("Query").refine(validSQLSyntax, { message: invalidSQLSyntaxMessage });

export function CheckMutationPage({ mode }: { mode: "create" | "edit" }) {
  const params = useParams({ strict: false });
  const checkId = params.checkId ?? "";
  const checkID = mode === "edit" ? Number(checkId) : null;
  const detail = useCheck(checkID);

  if (mode === "edit") {
    if (detail.error) {
      return (
        <PageShell>
          <Alert variant="destructive">
            <AlertTitle>Failed to Load Check</AlertTitle>
            <AlertDescription>{detail.error.message}</AlertDescription>
          </Alert>
        </PageShell>
      );
    }
    if (!detail.data) {
      return (
        <PageShell className="text-muted-foreground flex-row items-center gap-2 text-sm">
          <Loader2 className="animate-spin" /> Loading Check...
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
          targets: normalizeLabelTargetSet(detail.data.targets),
        }
      : emptyCheck;

  return (
    <CheckForm
      key={checkId || "new"}
      mode={mode}
      checkId={checkID}
      checkParam={checkId}
      passingHostCount={detail.data?.passing_host_count ?? 0}
      failingHostCount={detail.data?.failing_host_count ?? 0}
      initial={initial}
    />
  );
}

export function CheckNewPage() {
  return <CheckMutationPage mode="create" />;
}

export function CheckResourcePage() {
  return <CheckMutationPage mode="edit" />;
}

function CheckForm({
  mode,
  checkId,
  checkParam,
  passingHostCount,
  failingHostCount,
  initial,
}: {
  mode: "create" | "edit";
  checkId: number | null;
  checkParam: string;
  passingHostCount: number;
  failingHostCount: number;
  initial: CheckMutation;
}) {
  const navigate = useNavigate();
  const createCheck = useCreateCheck();
  const updateCheck = useUpdateCheck(checkId);
  const [schemaOpen, setSchemaOpen] = useSchemaSidebar();
  const [selectedSchemaTable, setSelectedSchemaTable] = useState<string | null>(null);
  const editorRef = useRef<ReactCodeMirrorRef>(null);
  const form = useForm({
    defaultValues: initial,
    onSubmit: async ({ value }) => {
      const payload = trimCheck(value);
      const saved = mode === "create" ? await createCheck.mutateAsync(payload) : await updateCheck.mutateAsync(payload);
      void navigate({ to: "/osquery/checks/$checkId", params: { checkId: String(saved.id) } });
    },
  });
  const pending = createCheck.isPending || updateCheck.isPending;

  function insertAtCursor(snippet: string) {
    const view = editorRef.current?.view;
    if (!view) {
      form.setFieldValue("query", (current) => `${current} ${snippet}`);
      return;
    }
    view.dispatch({ changes: { from: view.state.selection.main.from, insert: snippet } });
  }

  const selectSchemaTable = useCallback(
    (tableName: string) => {
      setSelectedSchemaTable(tableName);
      setSchemaOpen(true);
    },
    [setSchemaOpen],
  );

  return (
    <PageShell asChild className={cn("h-full transition-[padding] duration-200 ease-out", schemaOpen && "pr-[21rem]")}>
      <form
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          void form.handleSubmit();
        }}
      >
        <form.Subscribe selector={(state) => state.values.name}>
          {(name) => (
            <PageHeader
              title={mode === "create" ? "New Check" : name || "Check"}
              context={
                mode === "edit" ? (
                  <>
                    <SettingItem label="Pass">
                      <HostCount checkId={checkId} response="pass" value={passingHostCount} />
                    </SettingItem>
                    <SettingItem label="Fail">
                      <HostCount checkId={checkId} response="fail" value={failingHostCount} />
                    </SettingItem>
                  </>
                ) : null
              }
              actions={
                mode === "edit" ? (
                  <LiveRunButton to="/osquery/checks/$checkId/live" params={{ checkId: checkParam }} />
                ) : null
              }
            />
          )}
        </form.Subscribe>

        <MutableResourceTabs
          tabs={[
            {
              value: "options",
              label: "Options",
              content: (
                <div className="flex max-w-5xl flex-col gap-6">
                  <FieldGroup className="max-w-3xl">
                    <form.Field
                      name="name"
                      validators={{ onSubmit: requiredString("Name") }}
                      children={(field) => {
                        const error = firstErrorMessage(field.state.meta.errors);
                        return (
                          <Field data-invalid={error ? true : undefined}>
                            <FieldLabel htmlFor="check-name" required>
                              Name
                            </FieldLabel>
                            <Input
                              id="check-name"
                              name={field.name}
                              required
                              aria-invalid={error ? true : undefined}
                              value={field.state.value}
                              onBlur={field.handleBlur}
                              onChange={(event) => field.handleChange(event.target.value)}
                            />
                            {error ? <FieldError>{error}</FieldError> : null}
                          </Field>
                        );
                      }}
                    />

                    <form.Field
                      name="description"
                      children={(field) => (
                        <Field>
                          <FieldLabel htmlFor="check-description">Description</FieldLabel>
                          <Textarea
                            id="check-description"
                            name={field.name}
                            rows={3}
                            value={field.state.value ?? ""}
                            onBlur={field.handleBlur}
                            onChange={(event) => field.handleChange(event.target.value)}
                          />
                        </Field>
                      )}
                    />
                  </FieldGroup>

                  <form.Field
                    name="query"
                    validators={{ onSubmit: checkQuerySchema }}
                    children={(field) => {
                      const error = firstErrorMessage(field.state.meta.errors);
                      return (
                        <Field data-invalid={error ? true : undefined}>
                          <FieldLabel required>Query</FieldLabel>
                          <SQLEditor
                            ref={editorRef}
                            value={field.state.value}
                            onChange={field.handleChange}
                            onTableMetaClick={selectSchemaTable}
                            placeholder="SELECT ..."
                            invalid={error ? true : undefined}
                          />
                          {error ? <FieldError>{error}</FieldError> : null}
                        </Field>
                      );
                    }}
                  />
                </div>
              ),
            },
            {
              value: "scope",
              label: "Scope",
              content: (
                <form.Field
                  name="targets"
                  children={(field) => (
                    <LabelScopeEditor
                      value={normalizeLabelTargetSet(field.state.value)}
                      onChange={field.handleChange}
                    />
                  )}
                />
              ),
            },
          ]}
        />
        <div className="flex items-center gap-2 border-t pt-4">
          <Button type="submit" size="sm" disabled={pending}>
            {pending ? "Saving..." : "Save"}
          </Button>
          {mode === "create" ? (
            <Button type="button" variant="outline" size="sm" onClick={() => void navigate({ to: "/osquery/checks" })}>
              Cancel
            </Button>
          ) : null}
        </div>
        <SchemaSidebar
          open={schemaOpen}
          onOpenChange={setSchemaOpen}
          onInsertColumn={insertAtCursor}
          selectedTable={selectedSchemaTable}
          onSelectedTableChange={setSelectedSchemaTable}
        />
      </form>
    </PageShell>
  );
}

function HostCount({ checkId, response, value }: { checkId: number | null; response: "pass" | "fail"; value: number }) {
  if (checkId === null) return value;
  return (
    <Link to="/hosts" search={{ check_id: checkId, check_response: response }} className="hover:underline">
      {value}
    </Link>
  );
}

function trimCheck(form: CheckMutation): CheckMutation {
  return {
    ...form,
    name: form.name.trim(),
    description: form.description?.trim() ?? "",
    query: form.query.trim(),
    targets: normalizeLabelTargetSet(form.targets),
  };
}
