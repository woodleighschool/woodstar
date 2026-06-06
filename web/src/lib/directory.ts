import type { Group, User } from "@/lib/api";
import { enumLabel, enumOptions, type EnumMetadataMap } from "@/lib/enum-metadata";

export type DirectorySource = (User | Group)["source"];

export const DIRECTORY_SOURCES = {
  local: {
    name: "Local",
    description: "Created and managed in Woodstar.",
    variant: "outline",
  },
  entra: {
    name: "Entra",
    description: "Synced from Microsoft Entra.",
    variant: "secondary",
  },
} satisfies EnumMetadataMap<DirectorySource>;

export const DIRECTORY_SOURCE_VALUES = ["local", "entra"] as const satisfies readonly DirectorySource[];

export const DIRECTORY_SOURCE_OPTIONS = enumOptions(DIRECTORY_SOURCES);

export function directorySourceLabel(source: string | null | undefined) {
  return enumLabel(DIRECTORY_SOURCES, source);
}
