import { revalidateLogic, useForm } from "@tanstack/react-form";
import { z } from "zod";

import type { MunkiClientResources, MunkiLink, MunkiMutation } from "@/lib/api";

const clientResourceLinkSchema = z
  .object({
    id: z.string(),
    label: z.string().trim().min(1, "Enter a label.").max(80),
    target: z.string().trim().max(2048),
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

export const clientResourcesSchema = z.object({
  banner: z.object({
    alignment: z.enum(["left", "center"]),
  }),
  links: z.array(clientResourceLinkSchema).max(12),
  footer: z.object({
    text: z.string().trim().max(500),
    links: z.array(clientResourceLinkSchema).max(12),
  }),
});

export type ClientResourcesDraft = z.infer<typeof clientResourcesSchema>;
export type ClientResourceLink = ClientResourcesDraft["links"][number];

export function clientResourceLinkErrors(link: ClientResourceLink) {
  const parsed = clientResourceLinkSchema.safeParse(link);
  if (parsed.success) return {};

  const errors: { label?: string; target?: string; openInBrowser?: string } = {};
  for (const issue of parsed.error.issues) {
    switch (issue.path[0]) {
      case "label":
        errors.label ??= issue.message;
        break;
      case "target":
        errors.target ??= issue.message;
        break;
      case "openInBrowser":
        errors.openInBrowser ??= issue.message;
        break;
    }
  }
  return errors;
}

export function emptyClientResourcesDraft(): ClientResourcesDraft {
  return {
    banner: { alignment: "left" },
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
    validationLogic: revalidateLogic(),
    validators: { onDynamic: clientResourcesSchema },
    onSubmit: ({ value }) => onSubmit(value),
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
