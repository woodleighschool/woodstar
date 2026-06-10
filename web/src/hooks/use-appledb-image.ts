import { useQuery } from "@tanstack/react-query";

import { queryKeys } from "@/lib/query-keys";

interface ManifestEntry {
  key: string;
  count: number;
  index: { id: string; dark?: boolean }[];
}

interface DeviceFile {
  key?: string;
  imageKey?: string;
}

const MANIFEST_URL = "https://img.appledb.dev/main.json";
const DEVICE_URL = (model: string) => `https://api.appledb.dev/device/${encodeURIComponent(model)}.json`;
const IMAGE_BASE = "https://img.appledb.dev/device@main";

const DAY_MS = 24 * 60 * 60 * 1000;

/**
 * Resolves an Apple device hero image URL from a hardware model identifier via appledb.
 * Returns null while loading, when the model is unknown, or when appledb
 * has no image for the resolved key. Both fetches cached for 24h.
 */
export function useAppleDbImage(hardwareModel: string | null | undefined): string | null {
  const { data: device } = useQuery<DeviceFile | null>({
    queryKey: queryKeys.appledbDevice(hardwareModel),
    queryFn: async ({ signal }) => {
      const r = await fetch(DEVICE_URL(hardwareModel!), { signal });
      if (r.status === 404) return null;
      if (!r.ok) throw new Error(`appledb device ${r.status}`);
      return (await r.json()) as DeviceFile;
    },
    enabled: Boolean(hardwareModel),
    staleTime: DAY_MS,
    gcTime: 7 * DAY_MS,
    retry: false,
  });

  const { data: manifest } = useQuery<ManifestEntry[]>({
    queryKey: queryKeys.appledbManifest,
    queryFn: async ({ signal }) => {
      const r = await fetch(MANIFEST_URL, { signal });
      if (!r.ok) throw new Error(`appledb manifest ${r.status}`);
      return (await r.json()) as ManifestEntry[];
    },
    staleTime: DAY_MS,
    gcTime: 7 * DAY_MS,
    retry: false,
  });

  if (!hardwareModel || !device || !manifest) return null;
  const imageKey = device.imageKey ?? device.key ?? hardwareModel;
  const entry = manifest.find((e) => e.key === imageKey);
  const first = entry?.index[0];
  if (!entry || !first) return null;
  return `${IMAGE_BASE}/${encodeURIComponent(entry.key)}/${encodeURIComponent(first.id)}.png`;
}
