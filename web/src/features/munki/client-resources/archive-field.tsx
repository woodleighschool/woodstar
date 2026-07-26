import { filesize } from "filesize";
import { FileArchive, Trash2 } from "lucide-react";
import { useRef } from "react";

import {
  Attachment,
  AttachmentAction,
  AttachmentActions,
  AttachmentContent,
  AttachmentDescription,
  AttachmentMedia,
  AttachmentTitle,
  AttachmentTrigger,
} from "@components/ui/attachment";
import { Input } from "@components/ui/input";
import { ValidatedFormField } from "@components/validated-form-field";
import type { MunkiObjectView } from "@lib/api";
import type { UploadProgress } from "@lib/upload";

import type { ClientResourcesForm } from "./fields";

export function ClientResourcesArchiveField({
  form,
  editable,
  metadata,
  uploading,
  progress,
  error,
}: {
  form: ClientResourcesForm;
  editable: boolean;
  metadata?: MunkiObjectView;
  uploading: boolean;
  progress: UploadProgress | null;
  error: Error | null;
}) {
  const inputRef = useRef<HTMLInputElement>(null);

  return (
    <section className="flex max-w-xl flex-col gap-4" aria-labelledby="client-resources-archive">
      <div className="flex flex-col gap-1">
        <h2 id="client-resources-archive" className="text-base font-medium">
          Custom ZIP
        </h2>
        <p className="text-sm text-muted-foreground">
          Serve an archive containing your Managed Software Center client resources.
        </p>
      </div>

      <form.Field name="archive_file">
        {(field) => (
          <ValidatedFormField
            field={field}
            label="Archive"
            htmlFor="munki-client-resources-archive"
            required={!metadata}
          >
            {(control) => {
              const file = field.state.value;
              const filename = file?.name ?? metadata?.filename ?? "Choose a ZIP archive";
              const description = error
                ? error.message
                : uploading
                  ? `Uploading${progress ? ` · ${progress.percent}%` : ""}`
                  : file
                    ? `${filesize(file.size)} selected`
                    : metadata?.size_bytes !== null && metadata?.size_bytes !== undefined
                      ? editable
                        ? `${filesize(metadata.size_bytes)} · select to replace`
                        : filesize(metadata.size_bytes)
                      : editable
                        ? "Select a ZIP archive."
                        : "No archive deployed.";
              const state =
                error || control["aria-invalid"]
                  ? "error"
                  : uploading
                    ? "uploading"
                    : file || metadata
                      ? "done"
                      : "idle";
              return (
                <div className="relative w-full">
                  {editable ? (
                    <Input
                      ref={inputRef}
                      key={file ? `${file.name}:${file.size}:${file.lastModified}` : "empty"}
                      id="munki-client-resources-archive-input"
                      type="file"
                      accept=".zip,application/zip"
                      hidden
                      onBlur={field.handleBlur}
                      onChange={(event) => field.handleChange(event.target.files?.[0] ?? null)}
                    />
                  ) : null}
                  <Attachment state={state} className="w-full">
                    <AttachmentMedia>
                      <FileArchive />
                    </AttachmentMedia>
                    <AttachmentContent>
                      <AttachmentTitle>{filename}</AttachmentTitle>
                      <AttachmentDescription>{description}</AttachmentDescription>
                    </AttachmentContent>
                    {editable && file ? (
                      <AttachmentActions>
                        <AttachmentAction
                          type="button"
                          aria-label="Remove archive"
                          onClick={() => field.handleChange(null)}
                        >
                          <Trash2 />
                        </AttachmentAction>
                      </AttachmentActions>
                    ) : null}
                    {editable ? (
                      <AttachmentTrigger
                        id="munki-client-resources-archive"
                        aria-invalid={control["aria-invalid"]}
                        onClick={() => inputRef.current?.click()}
                      />
                    ) : null}
                  </Attachment>
                </div>
              );
            }}
          </ValidatedFormField>
        )}
      </form.Field>
    </section>
  );
}
