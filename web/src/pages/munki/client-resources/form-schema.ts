import { z } from "zod";

const clientResourceLinkInputSchema = z.object({
  id: z.string(),
  label: z.string(),
  target: z.string(),
  openInBrowser: z.boolean(),
});

export const clientResourceLinkSchema = clientResourceLinkInputSchema
  .extend({
    label: limitedString(80, "Label", "Enter a label."),
    target: limitedString(2048, "Target"),
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

const clientResourceAssetInputSchema = z.object({
  name: z.string(),
  url: z.string(),
  objectID: z.number().int().positive().nullable(),
  file: z.custom<File | null>((value) => value === null || value instanceof File),
});

const clientResourceAssetSchema = clientResourceAssetInputSchema.superRefine((asset, context) => {
  if ((asset.objectID === null) === (asset.file === null)) {
    context.addIssue({
      code: "custom",
      message: "Choose either a stored banner or a replacement image.",
    });
  }
});

export const clientResourcesBuilderSchema = z
  .object({
    banner: z.object({
      fit: z.enum(["height", "cover"]),
      focalX: z.number().int().min(0).max(100),
      asset: clientResourceAssetSchema.nullable(),
    }),
    links: z.array(clientResourceLinkSchema).max(12),
    footer: z.object({
      text: limitedString(500, "Footer text"),
      links: z.array(clientResourceLinkSchema).max(12),
    }),
  })
  .superRefine((builder, context) => {
    if (builder.banner.asset === null) {
      context.addIssue({
        code: "custom",
        path: ["banner", "asset"],
        message: "Choose a banner image.",
      });
    }
    addDuplicateLabelIssue(builder.links, ["links"], context);
    addDuplicateLabelIssue(builder.footer.links, ["footer", "links"], context);
  });

const clientResourcesFormShape = z.object({
  custom: z.boolean(),
  archive_object_id: z.number().int().positive().nullable(),
  archive_file: z.custom<File | null>((value) => value === null || value instanceof File),
  banner: z.object({
    fit: z.enum(["height", "cover"]),
    focalX: z.number(),
    asset: clientResourceAssetInputSchema.nullable(),
  }),
  links: z.array(clientResourceLinkInputSchema),
  footer: z.object({
    text: z.string(),
    links: z.array(clientResourceLinkInputSchema),
  }),
});

export const clientResourcesFormSchema = clientResourcesFormShape.superRefine((form, context) => {
  if (form.custom) {
    if (form.archive_object_id === null && form.archive_file === null) {
      context.addIssue({
        code: "custom",
        path: ["archive_file"],
        message: "Choose a client resources archive.",
      });
    }
    return;
  }

  const builder = clientResourcesBuilderSchema.safeParse(form);
  if (builder.success) return;
  for (const issue of builder.error.issues) {
    context.addIssue({ code: "custom", path: issue.path, message: issue.message });
  }
});

export type ClientResourcesFormInput = z.input<typeof clientResourcesFormSchema>;
export type ClientResourcesFormOutput = z.output<typeof clientResourcesFormSchema>;
export type ClientResourceLink = ClientResourcesFormInput["links"][number];
export type ClientResourceAsset = NonNullable<ClientResourcesFormInput["banner"]["asset"]>;

export function emptyClientResourceLink(): ClientResourceLink {
  return {
    id: crypto.randomUUID(),
    label: "",
    target: "",
    openInBrowser: false,
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
