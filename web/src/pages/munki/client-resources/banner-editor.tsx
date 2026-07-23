import { AlignCenter, AlignLeft, ImageIcon, ImageUp, Maximize2, Minimize2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { ButtonGroup } from "@/components/ui/button-group";
import { FileUpload, FileUploadDropzone, FileUploadTrigger } from "@/components/ui/file-upload";
import { cn } from "@/lib/utils";

import type { ClientResourceAsset, ClientResourcesFormInput } from "./form-schema";
import {
  clientResourceImageAccept,
  clientResourceImageMaxSize,
  validateClientResourceImage,
} from "./use-client-resource-asset";

const fitOptions = [
  { value: "height", label: "Fit", icon: Minimize2 },
  { value: "cover", label: "Fill", icon: Maximize2 },
] as const;
const positionOptions = [
  { value: 0, label: "Left", icon: AlignLeft },
  { value: 50, label: "Centre", icon: AlignCenter },
] as const;

export function BannerEditor({
  asset,
  error,
  invalid,
  uploading,
  fit,
  focalX,
  onAssetChange,
  onAssetReject,
  onFitChange,
  onFocalXChange,
}: {
  asset: ClientResourceAsset | null;
  error: string | null;
  invalid: boolean;
  uploading: boolean;
  fit: ClientResourcesFormInput["banner"]["fit"];
  focalX: ClientResourcesFormInput["banner"]["focalX"];
  onAssetChange: (file: File) => void;
  onAssetReject: (message: string) => void;
  onFitChange: (fit: ClientResourcesFormInput["banner"]["fit"]) => void;
  onFocalXChange: (focalX: ClientResourcesFormInput["banner"]["focalX"]) => void;
}) {
  return (
    <FileUpload
      key={asset?.url ?? "empty"}
      className="relative overflow-hidden rounded-tr-2xl"
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
      {asset ? (
        <div className="relative h-[200px] overflow-hidden bg-muted">
          <img
            src={asset.url}
            alt=""
            draggable={false}
            className={cn(
              "pointer-events-none absolute top-0 select-none",
              fit === "cover" ? "left-0 size-full object-cover" : "h-[200px] w-auto max-w-none",
            )}
            style={
              fit === "cover"
                ? { objectPosition: `${focalX}% center` }
                : { left: `${focalX}%`, transform: `translateX(-${focalX}%)` }
            }
          />

          <div
            aria-hidden="true"
            className="pointer-events-none absolute inset-0 rounded-tr-2xl border border-dashed border-primary/50"
          />

          <div className="absolute top-2 right-2 flex items-center gap-2">
            <ButtonGroup aria-label="Banner sizing">
              {fitOptions.map((option) => {
                const Icon = option.icon;
                return (
                  <Button
                    key={option.value}
                    type="button"
                    size="sm"
                    variant={fit === option.value ? "default" : "secondary"}
                    aria-pressed={fit === option.value}
                    onClick={() => onFitChange(option.value)}
                  >
                    <Icon data-icon="inline-start" />
                    {option.label}
                  </Button>
                );
              })}
            </ButtonGroup>
            <ButtonGroup aria-label="Banner position">
              {positionOptions.map((option) => {
                const Icon = option.icon;
                return (
                  <Button
                    key={option.value}
                    type="button"
                    size="sm"
                    variant={focalX === option.value ? "default" : "secondary"}
                    aria-pressed={focalX === option.value}
                    onClick={() => onFocalXChange(option.value)}
                  >
                    <Icon data-icon="inline-start" />
                    {option.label}
                  </Button>
                );
              })}
            </ButtonGroup>
            <FileUploadTrigger
              render={<Button type="button" variant="secondary" size="sm" disabled={uploading} />}
            >
              <ImageUp data-icon="inline-start" />
              Replace
            </FileUploadTrigger>
          </div>
        </div>
      ) : (
        <FileUploadDropzone className="group h-[200px] min-h-0 rounded-tr-2xl border-0 bg-muted/50 p-4 data-invalid:ring-0">
          <div className="flex size-10 items-center justify-center rounded-full border bg-background">
            <ImageIcon className="size-5 text-muted-foreground" />
          </div>
          <div className="space-y-1 text-center">
            <p className="text-sm font-medium">Add banner image</p>
            <p className="text-xs text-muted-foreground">Drag and drop a JPG or PNG here</p>
          </div>
          <FileUploadTrigger
            render={<Button type="button" variant="outline" size="sm" disabled={uploading} />}
          >
            Choose image
          </FileUploadTrigger>
          <p className="text-xs text-muted-foreground">5 MB max</p>
          <div
            aria-hidden="true"
            className="pointer-events-none absolute inset-0 rounded-tr-2xl border border-dashed border-primary/50 group-data-dragging:border-primary group-data-invalid:border-destructive"
          />
        </FileUploadDropzone>
      )}

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
