import { useParams } from "@tanstack/react-router";

import type { MunkiArtifact, MunkiArtifactMutation, MunkiArtifactUploadMutation } from "@/hooks/munki/artifacts";

export function useSoftwareIDParam() {
  const params = useParams({ strict: false });
  const id = Number(params.softwareId);
  return Number.isFinite(id) && id > 0 ? id : null;
}

export function usePackageIDParam() {
  const params = useParams({ strict: false });
  const id = Number(params.packageId);
  return Number.isFinite(id) && id > 0 ? id : null;
}

export function optionalText(value: string) {
  const trimmed = value.trim();
  return trimmed === "" ? undefined : trimmed;
}

export function uniqueOptions(values: string[]) {
  return Array.from(new Set(values.map((value) => value.trim()).filter(Boolean))).sort((a, b) => a.localeCompare(b));
}

export async function uploadSelectedArtifact(
  file: File,
  kind: "package" | "icon",
  createUpload: (body: MunkiArtifactUploadMutation) => Promise<{
    upload_url: string;
    headers?: Record<string, string>;
    artifact: MunkiArtifactMutation;
  }>,
  createArtifact: (body: MunkiArtifactMutation) => Promise<MunkiArtifact>,
) {
  const sha256 = await fileSHA256(file);
  const upload = await createUpload({
    kind,
    filename: file.name,
    content_type: file.type || undefined,
    size_bytes: file.size,
    sha256,
  });
  const headers = new Headers(upload.headers);
  if (file.type && !headers.has("Content-Type")) {
    headers.set("Content-Type", file.type);
  }
  const response = await fetch(upload.upload_url, {
    method: "PUT",
    headers,
    body: file,
  });
  if (!response.ok) {
    throw new Error(`Upload failed with HTTP ${response.status}`);
  }
  return createArtifact(upload.artifact);
}

async function fileSHA256(file: File) {
  const digest = await crypto.subtle.digest("SHA-256", await file.arrayBuffer());
  return Array.from(new Uint8Array(digest))
    .map((byte) => byte.toString(16).padStart(2, "0"))
    .join("");
}
