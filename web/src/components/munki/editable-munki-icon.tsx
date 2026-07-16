import { Upload, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import { MunkiIcon } from "@/components/munki/munki-icon";
import {
  Attachment,
  AttachmentAction,
  AttachmentActions,
  AttachmentContent,
  AttachmentDescription,
  AttachmentMedia,
  AttachmentTitle,
  AttachmentTrigger,
} from "@/components/ui/attachment";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Field, FieldLabel } from "@/components/ui/field";
import { useMunkiIcons } from "@/hooks/use-munki-icons";
import { formatBytes } from "@/lib/utils";

export const MUNKI_ICON_ACCEPT = "image/png,image/jpeg,image/webp,image/x-icns,.icns";

interface EditableMunkiIconProps {
  title: string;
  iconUrl?: string;
  file: File | null;
  clearable: boolean;
  onFileChange: (file: File | null) => void;
  onPickExisting: (object: { id: number; url: string }) => void;
  onClear: () => void;
}

export function EditableMunkiIcon({
  title,
  iconUrl,
  file,
  clearable,
  onFileChange,
  onPickExisting,
  onClear,
}: EditableMunkiIconProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [previewURL, setPreviewURL] = useState("");
  const [pickerOpen, setPickerOpen] = useState(false);
  const displayURL = previewURL || iconUrl;
  const hasIcon = !!file || !!displayURL;
  const uploadLabel = hasIcon ? `Replace ${title}` : `Choose ${title}`;
  const clearLabel = `Clear ${title}`;

  useEffect(() => {
    if (!file) {
      setPreviewURL("");
      return;
    }
    const url = URL.createObjectURL(file);
    setPreviewURL(url);
    return () => URL.revokeObjectURL(url);
  }, [file]);

  function resetInput() {
    if (inputRef.current) {
      inputRef.current.value = "";
    }
  }

  return (
    <Field>
      <FieldLabel>Icon</FieldLabel>
      <div className="relative w-full">
        <input
          ref={inputRef}
          type="file"
          accept={MUNKI_ICON_ACCEPT}
          hidden
          onChange={(event) => {
            const next = event.target.files?.[0] ?? null;
            resetInput();
            if (next) {
              onFileChange(next);
              setPickerOpen(false);
            }
          }}
        />
        <Attachment state={hasIcon ? "done" : "idle"} className="w-full">
          <AttachmentMedia variant={displayURL ? "image" : "icon"}>
            {displayURL ? <MunkiIcon iconUrl={displayURL} size="md" loading="eager" /> : <Upload />}
          </AttachmentMedia>
          <AttachmentContent>
            <AttachmentTitle>
              {file?.name ?? (displayURL ? "Current icon" : "Choose an icon")}
            </AttachmentTitle>
            <AttachmentDescription>
              {file
                ? `${formatBytes(file.size)} selected`
                : displayURL
                  ? "Select to replace or choose another uploaded icon."
                  : "Upload a new image or choose an uploaded icon."}
            </AttachmentDescription>
          </AttachmentContent>
          {clearable ? (
            <AttachmentActions>
              <AttachmentAction
                type="button"
                aria-label={clearLabel}
                onClick={() => {
                  onClear();
                  resetInput();
                }}
              >
                <X />
              </AttachmentAction>
            </AttachmentActions>
          ) : null}
          <AttachmentTrigger aria-label={uploadLabel} onClick={() => setPickerOpen(true)} />
        </Attachment>
      </div>

      <IconPickerDialog
        open={pickerOpen}
        onOpenChange={setPickerOpen}
        onUpload={() => inputRef.current?.click()}
        onPick={(object) => {
          onPickExisting(object);
          setPickerOpen(false);
        }}
      />
    </Field>
  );
}

function IconPickerDialog({
  open,
  onOpenChange,
  onUpload,
  onPick,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onUpload: () => void;
  onPick: (object: { id: number; url: string }) => void;
}) {
  const icons = useMunkiIcons(open);
  const items = icons.data?.items ?? [];

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Choose icon</DialogTitle>
        </DialogHeader>
        {items.length > 0 ? (
          <div className="grid max-h-72 grid-cols-4 gap-2 overflow-y-auto p-1">
            {items.map((object) => (
              <Attachment key={object.id} orientation="vertical" className="w-full">
                <AttachmentMedia variant="image">
                  <MunkiIcon iconUrl={object.content_url} size="lg" />
                </AttachmentMedia>
                <AttachmentContent>
                  <AttachmentTitle>{object.filename}</AttachmentTitle>
                </AttachmentContent>
                <AttachmentTrigger
                  aria-label={`Choose ${object.filename}`}
                  disabled={!object.content_url}
                  onClick={() =>
                    object.content_url
                      ? onPick({ id: object.id, url: object.content_url })
                      : undefined
                  }
                />
              </Attachment>
            ))}
          </div>
        ) : (
          <p className="py-6 text-center text-sm text-muted-foreground">
            {icons.isLoading ? "Loading icons..." : "No icons uploaded yet."}
          </p>
        )}
        <DialogFooter>
          <DialogClose asChild>
            <Button type="button" variant="outline">
              Cancel
            </Button>
          </DialogClose>
          <Button type="button" onClick={onUpload}>
            <Upload data-icon="inline-start" />
            Upload New
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
