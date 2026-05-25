import { Link, useNavigate, useParams } from "@tanstack/react-router";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { Loader2 } from "lucide-react";
import { useMemo, useRef, useState } from "react";
import { z } from "zod";

import { SchemaSidebar } from "@/components/editor/schema-sidebar";
import { SQLEditor } from "@/components/editor/sql-editor";
import { PageHeader, PageShell } from "@/components/layout/page-layout";
import { PlatformSelector } from "@/components/queries/platform-selector";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Field, FieldDescription, FieldError, FieldGroup, FieldLabel } from "@/components/ui/field";
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

const labelFormSchema = z
  .object({
    name: z.string().trim().min(1, "Name is required."),
    description: z.string().trim(),
    query: z.string().trim(),
    label_membership_type: z.enum(["dynamic", "manual", "derived"]),
    platforms: z.array(z.enum(["darwin", "windows", "linux"])).min(1, "Pick at least one platform."),
  })
  .refine((value) => value.label_membership_type !== "dynamic" || value.query !== "", {
    message: "Dynamic labels need a query.",
    path: ["query"],
  });

type LabelFormParse = ReturnType<typeof labelFormSchema.safeParse>;

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
  const [showErrors, setShowErrors] = useState(false);
  const [schemaOpen, setSchemaOpen] = useSchemaSidebar();
  const editorRef = useRef<ReactCodeMirrorRef>(null);

  const pending = createLabel.isPending || updateLabel.isPending;
  const isDynamic = form.label_membership_type === "dynamic";
  const memberOption = MEMBERSHIP_OPTIONS.find((o) => o.value === form.label_membership_type);
  const parsed = useMemo(() => labelFormSchema.safeParse(form), [form]);
  const errors = useMemo(() => fieldErrors(parsed), [parsed]);

  async function submit() {
    if (!parsed.success) {
      setShowErrors(true);
      return;
    }
    const cleaned = parsed.data;
    const body: LabelCreate | LabelMutation = {
      name: cleaned.name,
      description: cleaned.description,
      label_membership_type: cleaned.label_membership_type,
      platforms: cleaned.platforms,
      query: cleaned.label_membership_type === "dynamic" ? cleaned.query : undefined,
    };
    if (mode === "create") {
      await createLabel.mutateAsync(body);
    } else {
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
        />

        <FieldGroup className="max-w-3xl">
          <Field data-invalid={showErrors && errors.name ? true : undefined}>
            <FieldLabel htmlFor="label-name">Name</FieldLabel>
            <Input
              id="label-name"
              required
              value={form.name}
              onChange={(event) => setForm({ ...form, name: event.target.value })}
            />
            {showErrors && errors.name ? <FieldError>{errors.name}</FieldError> : null}
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

          <Field data-invalid={showErrors && errors.platforms ? true : undefined}>
            <PlatformSelector value={form.platforms} onChange={(platforms) => setForm({ ...form, platforms })} />
            {showErrors && errors.platforms ? <FieldError>{errors.platforms}</FieldError> : null}
          </Field>
        </FieldGroup>

        {isDynamic ? (
          <div className="max-w-3xl">
            <SQLEditor
              ref={editorRef}
              value={form.query}
              onChange={(query) => setForm({ ...form, query })}
              placeholder="SELECT ..."
            />
            {showErrors && errors.query ? <FieldError className="mt-2">{errors.query}</FieldError> : null}
          </div>
        ) : null}

        {isDynamic ? (
          <SchemaSidebar open={schemaOpen} onOpenChange={setSchemaOpen} onInsertColumn={insertAtCursor} />
        ) : null}

        <div className="flex max-w-3xl items-center gap-2 border-t pt-4">
          <Button type="submit" size="sm" disabled={pending}>
            {pending ? "Saving..." : "Save"}
          </Button>
          {mode === "edit" ? (
            <Button asChild type="button" variant="ghost" size="sm">
              <Link to="/labels">Cancel</Link>
            </Button>
          ) : null}
        </div>
      </form>
    </PageShell>
  );
}

function fieldErrors(result: LabelFormParse): Record<string, string> {
  if (result.success) return {};
  const out: Record<string, string> = {};
  for (const issue of result.error.issues) {
    const key = issue.path[0];
    if (typeof key === "string" && !out[key]) out[key] = issue.message;
  }
  return out;
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
