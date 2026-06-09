import { Upload, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import { MunkiIcon, type MunkiIconSize } from "@/components/munki/munki-icon";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export const MUNKI_ICON_ACCEPT = "image/png,image/jpeg,image/webp,image/icns,.icns";

interface EditableMunkiIconProps {
  title: string;
  iconUrl?: string;
  file: File | null;
  clearable: boolean;
  size?: MunkiIconSize;
  className?: string;
  onFileChange: (file: File | null) => void;
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
  onClear,
}: EditableMunkiIconProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [previewURL, setPreviewURL] = useState("");
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
          onFileChange(event.target.files?.[0] ?? null);
          resetInput();
        }}
      />
      <button
        type="button"
        className="focus-visible:border-ring focus-visible:ring-ring/50 relative block rounded-lg outline-none focus-visible:ring-[3px]"
        aria-label={uploadLabel}
        onClick={() => inputRef.current?.click()}
      >
        <MunkiIcon iconUrl={previewURL || iconUrl} size={size} loading="eager" />
        <span className="absolute inset-0 flex items-center justify-center rounded-lg bg-background/50 opacity-0 transition-opacity group-hover/munki-icon:opacity-100 group-focus-within/munki-icon:opacity-100">
          <Upload data-icon />
        </span>
      </button>
      {clearable ? (
        <Button
          type="button"
          variant="secondary"
          size="icon-xs"
          className="absolute -top-2 -right-2 rounded-full border shadow-sm opacity-0 transition-opacity group-hover/munki-icon:opacity-100 group-focus-within/munki-icon:opacity-100"
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
    </div>
  );
}
