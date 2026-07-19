import { AlignCenter, AlignLeft, ImageIcon, ImageUp } from "lucide-react";

import { Button } from "@/components/ui/button";
import { FileUpload, FileUploadDropzone, FileUploadTrigger } from "@/components/ui/file-upload";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { cn } from "@/lib/utils";

import type { ClientResourcesDraft } from "./client-resources";
import {
  type ClientResourceAsset,
  clientResourceImageAccept,
  clientResourceImageMaxSize,
  validateClientResourceImage,
} from "./use-client-resource-asset";

const alignmentOptions = [
  { value: "left", label: "Left", icon: AlignLeft },
  { value: "center", label: "Centre", icon: AlignCenter },
] as const;

export function BannerEditor({
  asset,
  error,
  invalid,
  uploading,
  alignment,
  onAssetChange,
  onAssetReject,
  onAlignmentChange,
}: {
  asset: ClientResourceAsset | null;
  error: string | null;
  invalid: boolean;
  uploading: boolean;
  alignment: ClientResourcesDraft["banner"]["alignment"];
  onAssetChange: (file: File) => void;
  onAssetReject: (message: string) => void;
  onAlignmentChange: (alignment: ClientResourcesDraft["banner"]["alignment"]) => void;
}) {
  return (
    <FileUpload
      key={asset?.url ?? "empty"}
      className="relative"
      accept={clientResourceImageAccept}
      maxFiles={1}
      maxSize={clientResourceImageMaxSize}
      disabled={uploading}
      invalid={invalid || error !== null}
      label="Banner image"
      onFileAccept={onAssetChange}
      onFileReject={(_file, message) => onAssetReject(message)}
      onFileValidate={validateClientResourceImage}
    >
      <FileUploadDropzone className="group h-[200px] min-h-0 overflow-hidden rounded-none border-0 bg-muted p-0 hover:bg-muted data-dragging:bg-accent/40 data-invalid:ring-0">
        {asset ? (
          <img
            src={asset.url}
            alt=""
            draggable={false}
            className={cn(
              "pointer-events-none absolute top-0 h-[200px] w-auto max-w-none select-none",
              alignment === "center" ? "left-1/2 -translate-x-1/2" : "left-0",
            )}
          />
        ) : (
          <FileUploadTrigger
            render={<Button type="button" variant="outline" disabled={uploading} />}
          >
            <ImageIcon data-icon="inline-start" />
            Add banner
          </FileUploadTrigger>
        )}

        {!asset ? <p className="text-xs text-muted-foreground">JPG or PNG · 5 MB max</p> : null}

        <div
          aria-hidden="true"
          className="pointer-events-none absolute inset-0 border border-dashed border-primary/50 group-data-[dragging]:border-primary group-data-[invalid]:border-destructive"
        />
      </FileUploadDropzone>

      {asset ? (
        <div className="absolute top-2 right-2 flex items-center gap-2">
          <ToggleGroup
            value={[alignment]}
            variant="outline"
            size="sm"
            aria-label="Banner alignment"
            className="bg-background/90 shadow-sm backdrop-blur-sm"
            onValueChange={(value) => {
              const selected = alignmentOptions.find((option) => option.value === value[0]);
              if (selected) onAlignmentChange(selected.value);
            }}
          >
            {alignmentOptions.map((option) => {
              const Icon = option.icon;
              return (
                <ToggleGroupItem key={option.value} value={option.value} aria-label={option.label}>
                  <Icon />
                  {option.label}
                </ToggleGroupItem>
              );
            })}
          </ToggleGroup>

          <FileUploadTrigger
            render={<Button type="button" variant="secondary" size="sm" disabled={uploading} />}
          >
            <ImageUp data-icon="inline-start" />
            Replace
          </FileUploadTrigger>
        </div>
      ) : null}

      {error ? (
        <p
          role="alert"
          className="absolute bottom-2 left-1/2 -translate-x-1/2 rounded-md bg-background/90 px-2 py-1 text-xs text-destructive shadow-sm"
        >
          {error}
        </p>
      ) : null}
    </FileUpload>
  );
}
