import { Link, useNavigate, useParams } from "@tanstack/react-router";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { Loader2 } from "lucide-react";
import { useRef, useState } from "react";

import { SchemaSidebar } from "@/components/editor/schema-sidebar";
import { SQLEditor } from "@/components/editor/sql-editor";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { PlatformSelector } from "@/components/queries/platform-selector";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Textarea } from "@/components/ui/textarea";
import { useCreateLabel, useLabel, useUpdateLabel, type LabelCreate, type LabelMutation } from "@/hooks/use-labels";
import { useSchemaSidebar } from "@/hooks/use-schema-sidebar";
import { DEFAULT_TARGET_PLATFORMS } from "@/lib/targeting";
import { cn } from "@/lib/utils";

type MembershipType = "dynamic" | "manual" | "derived";

const MEMBERSHIP_OPTIONS: { value: MembershipType; label: string; helpText: string }[] = [
  {
    value: "dynamic",
    label: "Dynamic",
    helpText: "osquery evaluates a SQL query on each host. Hosts where the query returns rows match the label.",
  },
  {
    value: "manual",
    label: "Manual",
    helpText: "Membership is managed by the server. Useful for ad-hoc grouping and host targeting.",
  },
  {
    value: "derived",
    label: "Derived",
    helpText: "Membership is computed from non-osquery host attributes such as directory department or group.",
  },
];

interface FormState {
  name: string;
  description: string;
  query: string;
  label_membership_type: MembershipType;
  platforms: string[];
}

const empty: FormState = {
  name: "",
  description: "",
  query: "select 1 from os_version where major >= 13;",
  label_membership_type: "dynamic",
  platforms: [...DEFAULT_TARGET_PLATFORMS],
};

export function LabelEditPage({ mode }: { mode: "create" | "edit" }) {
  const params = useParams({ strict: false });
  const labelId = params.labelId ?? "";
  const detail = useLabel(labelId);

  if (mode === "edit") {
    if (detail.error) {
      return (
        <PageShell>
          <Alert variant="destructive">
            <AlertTitle>Failed to load label</AlertTitle>
            <AlertDescription>{detail.error.message}</AlertDescription>
          </Alert>
        </PageShell>
      );
    }
    if (!detail.data) {
      return (
        <PageShell className="text-muted-foreground flex-row items-center gap-2 text-sm">
          <Loader2 className="size-4 animate-spin" /> Loading label...
        </PageShell>
      );
    }
    if (detail.data.label_type === "builtin") {
      return (
        <PageShell>
          <Alert>
            <AlertTitle>Built-in label</AlertTitle>
            <AlertDescription>Built-in labels are managed by Woodstar and cannot be edited.</AlertDescription>
          </Alert>
        </PageShell>
      );
    }
  }

  const initial: FormState =
    mode === "edit" && detail.data
      ? {
          name: detail.data.name,
          description: detail.data.description,
          query: detail.data.query ?? empty.query,
          label_membership_type: membershipFromString(detail.data.label_membership_type),
          platforms: [...detail.data.platforms],
        }
      : empty;

  return <LabelEditForm key={labelId || "new"} mode={mode} labelId={labelId} initial={initial} />;
}

function LabelEditForm({ mode, labelId, initial }: { mode: "create" | "edit"; labelId: string; initial: FormState }) {
  const navigate = useNavigate();
  const createLabel = useCreateLabel();
  const updateLabel = useUpdateLabel(labelId ? Number(labelId) : null);
  const [form, setForm] = useState<FormState>(initial);
  const [schemaOpen, setSchemaOpen] = useSchemaSidebar();
  const editorRef = useRef<ReactCodeMirrorRef>(null);

  const error = createLabel.error ?? updateLabel.error;
  const pending = createLabel.isPending || updateLabel.isPending;
  const isDynamic = form.label_membership_type === "dynamic";
  const memberOption = MEMBERSHIP_OPTIONS.find((o) => o.value === form.label_membership_type);

  async function submit() {
    if (mode === "create") {
      const body: LabelCreate = {
        name: form.name,
        description: form.description,
        label_membership_type: form.label_membership_type,
        platforms: form.platforms,
        query: isDynamic ? form.query : undefined,
      };
      await createLabel.mutateAsync(body);
    } else {
      const body: LabelMutation = {
        name: form.name,
        description: form.description,
        label_membership_type: form.label_membership_type,
        platforms: form.platforms,
        query: isDynamic ? form.query : undefined,
      };
      await updateLabel.mutateAsync(body);
    }
    void navigate({ to: "/labels" });
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
    <PageShell
      asChild
      className={cn("h-full transition-[padding] duration-200 ease-out", isDynamic && schemaOpen && "pr-[21rem]")}
    >
      <form
        onSubmit={(event) => {
          event.preventDefault();
          void submit();
        }}
      >
        <PageHeader
          title={mode === "create" ? "New label" : "Edit label"}
          description="Labels group hosts for filtering, reports, checks, and future Santa/Munki targeting."
          actions={
            <>
              {mode === "edit" ? (
                <Button asChild type="button" variant="outline" size="sm">
                  <Link to="/labels">Cancel</Link>
                </Button>
              ) : null}
              <Button type="submit" size="sm" disabled={pending}>
                {pending ? "Saving..." : mode === "create" ? "Save label" : "Save"}
              </Button>
            </>
          }
        />
        {error ? (
          <Alert variant="destructive">
            <AlertTitle>Unable to save label</AlertTitle>
            <AlertDescription>{error.message}</AlertDescription>
          </Alert>
        ) : null}

        <FieldGroup className="max-w-3xl">
          <Field>
            <FieldLabel htmlFor="label-name">Name</FieldLabel>
            <Input
              id="label-name"
              required
              value={form.name}
              onChange={(event) => setForm({ ...form, name: event.target.value })}
            />
          </Field>

          <Field>
            <FieldLabel htmlFor="label-description">Description</FieldLabel>
            <Textarea
              id="label-description"
              rows={3}
              placeholder="Optional. Tell admins why this label exists."
              value={form.description}
              onChange={(event) => setForm({ ...form, description: event.target.value })}
            />
          </Field>

          <Field>
            <FieldLabel>Type</FieldLabel>
            <RadioGroup
              value={form.label_membership_type}
              onValueChange={(value) => setForm({ ...form, label_membership_type: value as MembershipType })}
              className="gap-2"
            >
              {MEMBERSHIP_OPTIONS.map((option) => (
                <div key={option.value} className="flex items-center gap-2">
                  <RadioGroupItem id={`membership-${option.value}`} value={option.value} />
                  <Label htmlFor={`membership-${option.value}`} className="font-normal">
                    {option.label}
                  </Label>
                </div>
              ))}
            </RadioGroup>
            {memberOption ? <FieldDescription>{memberOption.helpText}</FieldDescription> : null}
          </Field>

          <PlatformSelector value={form.platforms} onChange={(platforms) => setForm({ ...form, platforms })} />
        </FieldGroup>

        {isDynamic ? (
          <div className="flex flex-1">
            <SQLEditor
              ref={editorRef}
              value={form.query}
              onChange={(query) => setForm({ ...form, query })}
              placeholder="SELECT ..."
              className="flex-1"
            />
          </div>
        ) : null}

        {isDynamic ? (
          <SchemaSidebar open={schemaOpen} onOpenChange={setSchemaOpen} onInsertColumn={insertAtCursor} />
        ) : null}
      </form>
    </PageShell>
  );
}

function membershipFromString(value: string | undefined): MembershipType {
  switch (value) {
    case "manual":
    case "derived":
      return value;
    default:
      return "dynamic";
  }
}
