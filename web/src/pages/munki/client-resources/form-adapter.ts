import type {
  MunkiBuilder,
  MunkiClientResources,
  MunkiClientResourcesBuilder,
  MunkiLink,
} from "@/lib/api";

import {
  type ClientResourceLink,
  type ClientResourcesFormInput,
  type ClientResourcesFormOutput,
  clientResourcesBuilderSchema,
} from "./form-schema";

export function emptyClientResourcesForm(): ClientResourcesFormInput {
  return {
    custom: false,
    archive_object_id: null,
    archive_file: null,
    banner: { fit: "height", focalX: 0, asset: null },
    links: [],
    footer: { text: "", links: [] },
  };
}

export function clientResourcesFormFromResource(
  resource: MunkiClientResources | null,
): ClientResourcesFormInput {
  const form = emptyClientResourcesForm();
  if (!resource) return form;

  form.custom = resource.custom;
  form.archive_object_id = resource.custom ? resource.archive.id : null;
  if (resource.builder) applyBuilder(form, resource.builder);
  return form;
}

export function clientResourcesBuilderMutation(
  form: ClientResourcesFormOutput,
): Omit<MunkiBuilder, "banner_object_id"> {
  const builder = clientResourcesBuilderSchema.parse(form);
  return {
    banner_fit: builder.banner.fit,
    banner_focal_x: builder.banner.focalX,
    links: builder.links.map(apiLink),
    footer_text: builder.footer.text,
    footer_links: builder.footer.links.map(apiLink),
  };
}

function applyBuilder(form: ClientResourcesFormInput, builder: MunkiClientResourcesBuilder) {
  form.banner = {
    fit: builder.banner_fit,
    focalX: builder.banner_focal_x,
    asset: {
      name: builder.banner.filename,
      url: builder.banner.content_url,
      objectID: builder.banner.id,
      file: null,
    },
  };
  form.links = builder.links.map(clientResourceLink);
  form.footer = {
    text: builder.footer_text,
    links: builder.footer_links.map(clientResourceLink),
  };
}

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
