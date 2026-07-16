import { Upload, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import { MunkiIcon, type MunkiIconSize } from "@/components/munki/munki-icon";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useMunkiIcons } from "@/hooks/use-munki-icons";
import { cn } from "@/lib/utils";

export const MUNKI_ICON_ACCEPT = "image/png,image/jpeg,image/webp,image/x-icns,.icns";

interface EditableMunkiIconProps {
  title: string;
  iconUrl?: string;
  file: File | null;
  clearable: boolean;
  size?: MunkiIconSize;
  className?: string;
  onFileChange: (file: File | null) => void;
  onPickExisting: (object: { id: number; url: string }) => void;
  onClear: () => void;
}

export function EditableMunkiIcon({
  title,
  iconUrl,
  file,
  clearable,
  size = "lg",
  className,
  onFileChange,
  onPickExisting,
  onClear,
}: EditableMunkiIconProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [previewURL, setPreviewURL] = useState("");
  const [pickerOpen, setPickerOpen] = useState(false);
  const uploadLabel = iconUrl || file ? `Replace ${title}` : `Upload ${title}`;
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
    <div className={cn("group/munki-icon relative w-fit", className)}>
      <input
        ref={inputRef}
        type="file"
        accept={MUNKI_ICON_ACCEPT}
        className="sr-only"
        onChange={(event) => {
          const next = event.target.files?.[0] ?? null;
          resetInput();
          if (next) {
            onFileChange(next);
            setPickerOpen(false);
          }
        }}
      />
      <button
        type="button"
        className="relative block rounded-lg outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50"
        aria-label={uploadLabel}
        onClick={() => setPickerOpen(true)}
      >
        <MunkiIcon iconUrl={previewURL || iconUrl} size={size} loading="eager" />
        <span className="absolute inset-0 flex items-center justify-center rounded-lg bg-background/50 opacity-0 transition-opacity group-focus-within/munki-icon:opacity-100 group-hover/munki-icon:opacity-100">
          <Upload data-icon />
        </span>
      </button>
      {clearable ? (
        <Button
          type="button"
          variant="secondary"
          size="icon-xs"
          className="absolute -top-2 -right-2 rounded-full border opacity-0 shadow-sm transition-opacity group-focus-within/munki-icon:opacity-100 group-hover/munki-icon:opacity-100"
          aria-label={clearLabel}
          onClick={(event) => {
            event.stopPropagation();
            onClear();
            resetInput();
          }}
        >
          <X data-icon />
        </Button>
      ) : null}

      <IconPickerDialog
        open={pickerOpen}
        onOpenChange={setPickerOpen}
        onUpload={() => inputRef.current?.click()}
        onPick={(object) => {
          onPickExisting(object);
          setPickerOpen(false);
        }}
      />
    </div>
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
              <button
                key={object.id}
                type="button"
                title={object.filename}
                disabled={!object.content_url}
                className="flex items-center justify-center rounded-lg border p-2 outline-none hover:border-ring focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50"
                onClick={() =>
                  object.content_url
                    ? onPick({ id: object.id, url: object.content_url })
                    : undefined
                }
              >
                <MunkiIcon iconUrl={object.content_url} size="lg" />
              </button>
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
