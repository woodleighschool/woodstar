import { revalidateLogic, useForm } from "@tanstack/react-form";
import { z } from "zod";

import type { MunkiClientResources, MunkiLink, MunkiMutation } from "@/lib/api";

export const clientResourceLinkSchema = z
  .object({
    id: z.string(),
    label: limitedString(80, "Label", "Enter a label."),
    target: limitedString(2048, "Target"),
    openInBrowser: z.boolean(),
  })
  .superRefine((link, context) => {
    const target = parseTarget(link.target);
    if (!target) {
      context.addIssue({
        code: "custom",
        path: ["target"],
        message: "Use an HTTP URL, email address, or Munki route.",
      });
      return;
    }
    if (target.username || target.password) {
      context.addIssue({
        code: "custom",
        path: ["target"],
        message: "URLs cannot contain credentials.",
      });
    }
    if (link.openInBrowser && target.protocol !== "http:" && target.protocol !== "https:") {
      context.addIssue({
        code: "custom",
        path: ["openInBrowser"],
        message: "Only HTTP links can open in the browser.",
      });
    }
  });

const clientResourceAssetSchema = z
  .object({
    name: z.string().trim().min(1),
    url: z.string().min(1),
    objectID: z.number().int().positive().nullable(),
    file: z.custom<File>((value) => value instanceof File).nullable(),
  })
  .superRefine((asset, context) => {
    if ((asset.objectID === null) === (asset.file === null)) {
      context.addIssue({
        code: "custom",
        message: "Choose either a stored banner or a replacement image.",
      });
    }
  });

export const clientResourcesSchema = z
  .object({
    banner: z.object({
      alignment: z.enum(["left", "center"]),
      asset: clientResourceAssetSchema.nullable(),
    }),
    links: z.array(clientResourceLinkSchema).max(12),
    footer: z.object({
      text: limitedString(500, "Footer text"),
      links: z.array(clientResourceLinkSchema).max(12),
    }),
  })
  .superRefine((draft, context) => {
    if (draft.banner.asset === null) {
      context.addIssue({
        code: "custom",
        path: ["banner", "asset"],
        message: "Choose a banner image.",
      });
    }
    addDuplicateLabelIssue(draft.links, ["links"], context);
    addDuplicateLabelIssue(draft.footer.links, ["footer", "links"], context);
  });

export type ClientResourcesDraft = z.infer<typeof clientResourcesSchema>;
export type ClientResourceLink = ClientResourcesDraft["links"][number];

export function emptyClientResourcesDraft(): ClientResourcesDraft {
  return {
    banner: { alignment: "left", asset: null },
    links: [],
    footer: { text: "", links: [] },
  };
}

export function emptyClientResourceLink(): ClientResourceLink {
  return {
    id: crypto.randomUUID(),
    label: "",
    target: "",
    openInBrowser: false,
  };
}

export function clientResourcesDraft(resource: MunkiClientResources | null): ClientResourcesDraft {
  if (!resource) return emptyClientResourcesDraft();
  return {
    banner: {
      alignment: resource.banner_alignment,
      asset: {
        name: resource.banner.filename,
        url: resource.banner.content_url,
        objectID: resource.banner.id,
        file: null,
      },
    },
    links: resource.links.map(clientResourceLink),
    footer: {
      text: resource.footer_text,
      links: resource.footer_links.map(clientResourceLink),
    },
  };
}

export function clientResourcesMutation(
  draft: ClientResourcesDraft,
): Omit<MunkiMutation, "banner_object_id"> {
  const parsed = clientResourcesSchema.parse(draft);
  return {
    banner_alignment: parsed.banner.alignment,
    links: parsed.links.map(apiLink),
    footer_text: parsed.footer.text,
    footer_links: parsed.footer.links.map(apiLink),
  };
}

export function useClientResourcesForm(
  initial: ClientResourcesDraft,
  onSubmit: (value: ClientResourcesDraft) => Promise<void>,
) {
  return useForm({
    defaultValues: initial,
    validationLogic: revalidateLogic({ mode: "submit", modeAfterSubmission: "change" }),
    validators: { onDynamic: clientResourcesSchema },
    onSubmit: async ({ value, formApi }) => {
      await onSubmit(value);
      formApi.reset(value);
    },
  });
}

export type ClientResourcesForm = ReturnType<typeof useClientResourcesForm>;

function clientResourceLink(link: MunkiLink): ClientResourceLink {
  return {
    id: crypto.randomUUID(),
    label: link.label,
    target: link.target,
    openInBrowser: link.open_in_browser,
  };
}

function apiLink(link: ClientResourceLink): MunkiLink {
  return {
    label: link.label,
    target: link.target,
    open_in_browser: link.openInBrowser,
  };
}

function parseTarget(value: string): URL | null {
  try {
    const target = new URL(value);
    return ["http:", "https:", "mailto:", "munki:"].includes(target.protocol) ? target : null;
  } catch {
    return null;
  }
}

function limitedString(maxLength: number, label: string, requiredMessage?: string) {
  const schema = requiredMessage ? z.string().trim().min(1, requiredMessage) : z.string().trim();
  return schema.refine(
    (value) => Array.from(value).length <= maxLength,
    `${label} cannot exceed ${maxLength} characters.`,
  );
}

function addDuplicateLabelIssue(
  links: ClientResourceLink[],
  path: (string | number)[],
  context: z.RefinementCtx,
) {
  const labels = new Set<string>();
  for (const link of links) {
    const label = link.label.trim().toLocaleLowerCase();
    if (!label || !labels.has(label)) {
      labels.add(label);
      continue;
    }
    context.addIssue({
      code: "custom",
      path,
      message: `Link labels must be unique. “${link.label.trim()}” is repeated.`,
    });
    return;
  }
}
