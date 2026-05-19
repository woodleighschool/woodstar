import type { SimpleIcon } from "simple-icons";
import { siAndroid, siApple, siGooglechrome, siLinux } from "simple-icons";

import { PLATFORM_LABELS, QUERYABLE_PLATFORMS, platformsFromValue, type QueryablePlatform } from "@/lib/targeting";

export type IconPath = Pick<SimpleIcon, "path" | "title">;

const windowsIcon: IconPath = {
  title: "Windows",
  path: "M3 3h8v8H3V3Zm10 0h8v8h-8V3ZM3 13h8v8H3v-8Zm10 0h8v8h-8v-8Z",
};

export const PLATFORM_ICONS: Record<QueryablePlatform, IconPath> = {
  darwin: siApple,
  windows: windowsIcon,
  linux: siLinux,
};

export const PLATFORM_ICON_TARGETS = QUERYABLE_PLATFORMS;

export const SOFTWARE_BRAND_ICONS = {
  android: siAndroid,
  apple: siApple,
  chrome: siGooglechrome,
  linux: siLinux,
  windows: windowsIcon,
} as const;

export function selectedPlatformIconTargets(value?: string | null) {
  const selected = platformsFromValue(value);
  return selected.length ? selected : PLATFORM_ICON_TARGETS;
}

export function platformIconTargetsLabel(platforms: readonly QueryablePlatform[]) {
  if (platforms.length === QUERYABLE_PLATFORMS.length) return "All Platforms";
  return platforms.map((platform) => PLATFORM_LABELS[platform]).join(", ");
}
