import { Link, useNavigate, useParams } from "@tanstack/react-router";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { Loader2 } from "lucide-react";
import { useRef, useState } from "react";

import { SchemaSidebar } from "@/components/editor/schema-sidebar";
import { SQLEditor } from "@/components/editor/sql-editor";
import { PlatformSelector } from "@/components/queries/platform-selector";
import { PageLead } from "@/components/queries/query-ui";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Textarea } from "@/components/ui/textarea";
import { useCreateLabel, useLabel, useUpdateLabel, type LabelCreate, type LabelMutation } from "@/hooks/use-labels";
import { useSchemaSidebar } from "@/hooks/use-schema-sidebar";
import { cn } from "@/lib/utils";

type MembershipType = "dynamic" | "manual" | "host_vitals";

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
    value: "host_vitals",
    label: "Host vitals",
    helpText: "Membership is derived from host fields. Currently reserved for future host-vitals criteria.",
  },
];

interface FormState {
  name: string;
  description: string;
  query: string;
  label_membership_type: MembershipType;
  platform?: string;
}

const empty: FormState = {
  name: "",
  description: "",
  query: "select 1 from os_version where major >= 13;",
  label_membership_type: "dynamic",
  platform: undefined,
};

export function LabelEditPage({ mode }: { mode: "create" | "edit" }) {
  const params = useParams({ strict: false });
  const labelId = params.labelId ?? "";
  const detail = useLabel(labelId);

  if (mode === "edit") {
    if (detail.error) {
      return (
        <div className="p-6">
          <Alert variant="destructive">
            <AlertTitle>Failed to load label</AlertTitle>
            <AlertDescription>{detail.error.message}</AlertDescription>
          </Alert>
        </div>
      );
    }
    if (!detail.data) {
      return (
        <div className="text-muted-foreground flex items-center gap-2 p-6 text-sm">
          <Loader2 className="size-4 animate-spin" /> Loading label...
        </div>
      );
    }
    if (detail.data.label_type === "builtin") {
      return (
        <div className="p-6">
          <Alert>
            <AlertTitle>Built-in label</AlertTitle>
            <AlertDescription>Built-in labels are managed by Woodstar and cannot be edited.</AlertDescription>
          </Alert>
        </div>
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
          platform: detail.data.platform ?? undefined,
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
    const platform = form.platform?.trim() === "" ? undefined : form.platform?.trim();
    if (mode === "create") {
      const body: LabelCreate = {
        name: form.name,
        description: form.description,
        label_membership_type: form.label_membership_type,
        platform: isDynamic ? platform : undefined,
        query: isDynamic ? form.query : undefined,
      };
      await createLabel.mutateAsync(body);
    } else {
      const body: LabelMutation = {
        name: form.name,
        description: form.description,
        label_membership_type: form.label_membership_type,
        platform: isDynamic ? platform : undefined,
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
    <form
      className={cn(
        "flex h-full flex-col gap-5 p-6 transition-[padding] duration-200 ease-out",
        isDynamic && schemaOpen && "pr-[21rem]",
      )}
      onSubmit={(event) => {
        event.preventDefault();
        void submit();
      }}
    >
      <PageLead
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

      <div className="grid max-w-3xl gap-4">
        <div className="grid gap-2">
          <Label htmlFor="label-name">Name</Label>
          <Input
            id="label-name"
            required
            value={form.name}
            onChange={(event) => setForm({ ...form, name: event.target.value })}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="label-description">Description</Label>
          <Textarea
            id="label-description"
            rows={3}
            placeholder="Optional. Tell admins why this label exists."
            value={form.description}
            onChange={(event) => setForm({ ...form, description: event.target.value })}
          />
        </div>

        <div className="grid gap-2">
          <Label>Type</Label>
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
          {memberOption ? <p className="text-muted-foreground text-xs">{memberOption.helpText}</p> : null}
        </div>

        {isDynamic ? (
          <PlatformSelector value={form.platform} onChange={(platform) => setForm({ ...form, platform })} />
        ) : null}
      </div>

      {isDynamic ? (
        <div className="flex flex-1">
          <SQLEditor
            ref={editorRef}
            value={form.query}
            onChange={(query) => setForm({ ...form, query })}
            placeholder="SELECT ..."
            minHeight="20rem"
            className="flex-1"
          />
        </div>
      ) : null}

      {isDynamic ? (
        <SchemaSidebar open={schemaOpen} onOpenChange={setSchemaOpen} onInsertColumn={insertAtCursor} />
      ) : null}
    </form>
  );
}

function membershipFromString(value: string | undefined): MembershipType {
  switch (value) {
    case "manual":
    case "host_vitals":
      return value;
    default:
      return "dynamic";
  }
}
