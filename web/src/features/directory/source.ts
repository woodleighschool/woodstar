import type { Group, User } from "@lib/api";
import { type EnumMetadataMap, enumOptions } from "@lib/enum-metadata";

export type DirectorySource = (User | Group)["source"];

export const DIRECTORY_SOURCE_VALUES = [
  "local",
  "entra",
] as const satisfies readonly DirectorySource[];

export const DIRECTORY_SOURCES = {
  local: {
    name: "Local",
    description: "Created and managed here.",
  },
  entra: {
    name: "Entra",
    description: "Synced from Microsoft Entra.",
  },
} satisfies EnumMetadataMap<DirectorySource>;

export const DIRECTORY_SOURCE_OPTIONS = enumOptions(DIRECTORY_SOURCES, DIRECTORY_SOURCE_VALUES);
