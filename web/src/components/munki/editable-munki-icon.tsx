import { Upload, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import { SoftwareArtwork } from "@/components/software/software-icon";
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
import { formatBytes } from "@/components/ui/file-upload";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useMunkiIcons } from "@/hooks/use-munki-icons";
export const MUNKI_ICON_ACCEPT = "image/png,image/jpeg,image/webp,image/x-icns,.icns";

export type MunkiSoftwareIconValue =
  | { kind: "none" }
  | { kind: "stored"; objectID: number; filename: string; url: string }
  | { kind: "upload"; file: File };

interface EditableMunkiIconProps {
  value: MunkiSoftwareIconValue;
  onChange: (value: MunkiSoftwareIconValue) => void;
}

export function EditableMunkiIcon({ value, onChange }: EditableMunkiIconProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [previewURL, setPreviewURL] = useState("");
  const [pickerOpen, setPickerOpen] = useState(false);
  const displayURL = value.kind === "stored" ? value.url : previewURL;
  const hasIcon = value.kind !== "none";

  useEffect(() => {
    if (value.kind !== "upload") {
      setPreviewURL("");
      return undefined;
    }
    const url = URL.createObjectURL(value.file);
    setPreviewURL(url);
    return () => URL.revokeObjectURL(url);
  }, [value]);

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
              onChange({ kind: "upload", file: next });
              setPickerOpen(false);
            }
          }}
        />
        <Attachment state={hasIcon ? "done" : "idle"} className="w-full">
          <AttachmentMedia className="overflow-visible rounded-none bg-transparent">
            <SoftwareArtwork src={displayURL} size="md" loading="eager" />
          </AttachmentMedia>
          <AttachmentContent>
            <AttachmentTitle>
              {value.kind === "upload"
                ? value.file.name
                : value.kind === "stored"
                  ? value.filename
                  : "Choose an icon"}
            </AttachmentTitle>
            <AttachmentDescription>
              {value.kind === "upload"
                ? `${formatBytes(value.file.size)} selected`
                : value.kind === "stored"
                  ? "Select to replace or choose another uploaded icon."
                  : "Upload a new image or choose an uploaded icon."}
            </AttachmentDescription>
          </AttachmentContent>
          {hasIcon ? (
            <AttachmentActions>
              <AttachmentAction
                type="button"
                onClick={() => {
                  onChange({ kind: "none" });
                  resetInput();
                }}
              >
                <X />
              </AttachmentAction>
            </AttachmentActions>
          ) : null}
          <AttachmentTrigger onClick={() => setPickerOpen(true)} />
        </Attachment>
      </div>

      <IconPickerDialog
        open={pickerOpen}
        onOpenChange={setPickerOpen}
        onUpload={() => inputRef.current?.click()}
        onPick={(object) => {
          onChange({ kind: "stored", ...object });
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
  onPick: (object: { objectID: number; filename: string; url: string }) => void;
}) {
  const icons = useMunkiIcons(open);
  const items = icons.data?.items ?? [];
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Choose icon</DialogTitle>
        </DialogHeader>
        {items.length > 0 ? (
          <div className="grid max-h-80 grid-cols-[repeat(auto-fill,3.5rem)] overflow-y-auto p-1">
            {items.map((object) => (
              <Tooltip key={object.id}>
                <TooltipTrigger
                  render={
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon-lg"
                      className="size-14"
                      disabled={!object.content_url}
                      onClick={() =>
                        object.content_url
                          ? onPick({
                              objectID: object.id,
                              filename: object.filename,
                              url: object.content_url,
                            })
                          : undefined
                      }
                    />
                  }
                >
                  <SoftwareArtwork src={object.content_url} size="md" className="size-12" />
                </TooltipTrigger>
                <TooltipContent>{object.filename}</TooltipContent>
              </Tooltip>
            ))}
          </div>
        ) : (
          <p className="py-6 text-center text-sm text-muted-foreground">
            {icons.isLoading ? "Loading icons..." : "No icons uploaded yet."}
          </p>
        )}
        <DialogFooter>
          <DialogClose render={<Button type="button" variant="outline" />}>Cancel</DialogClose>
          <Button type="button" onClick={onUpload}>
            <Upload data-icon="inline-start" />
            Upload New
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
