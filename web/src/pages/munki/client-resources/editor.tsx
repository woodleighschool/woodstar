import {
  AppWindow,
  Brush,
  Folder,
  MoreHorizontal,
  PackageOpen,
  Pencil,
  Plus,
  Search,
  Trash2,
  TriangleAlert,
} from "lucide-react";
import { type ReactNode, useState } from "react";
import { toast } from "sonner";

import { FormField } from "@/components/form-field";
import { SoftwareArtwork } from "@/components/software/software-icon";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Editable, EditableArea, EditableInput, EditablePreview } from "@/components/ui/editable";
import {
  Sortable,
  SortableContent,
  SortableItem,
  SortableItemHandle,
} from "@/components/ui/sortable";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

import { BannerEditor } from "./banner-editor";
import type { ClientResourcesForm } from "./fields";
import { type ClientResourceLink, type ClientResourcesFormInput } from "./form-schema";
import { LinkDialog } from "./link-dialog";
import { createClientResourceAsset } from "./use-client-resource-asset";
const navigationItems = [
  { label: "Software", icon: AppWindow, active: true },
  { label: "Categories", icon: Folder, active: false },
  { label: "My Items", icon: Brush, active: false },
  { label: "Updates", icon: PackageOpen, active: false },
];
const defaultCategories = [
  "Business",
  "Education",
  "Productivity",
  "Security",
  "Utilities",
] as const;
const sampleSoftware = [
  { name: "1Password", detail: "Security - AgileBits Inc." },
  { name: "ActivInspire", detail: "Education - Promethean" },
  { name: "Adobe Creative Cloud", detail: "Creativity - Adobe" },
  { name: "Audacity", detail: "Creativity - Audacity" },
] as const;
export function ClientResourcesEditor({
  form,
  draft,
  bannerUploading,
}: {
  form: ClientResourcesForm;
  draft: ClientResourcesFormInput;
  bannerUploading: boolean;
}) {
  return (
    <section className="flex w-full max-w-7xl min-w-0 flex-col gap-5">
      <Alert className="border-warning/30 bg-warning/10 text-warning">
        <TriangleAlert />
        <AlertDescription className="block text-foreground/80">
          <strong>Please Note:</strong> This web preview provides a close approximation of Managed
          Software Center.
        </AlertDescription>
      </Alert>

      <div className="min-w-0 overflow-x-auto pb-2">
        <div className="w-full min-w-4xl overflow-hidden rounded-2xl border bg-background shadow-sm">
          <div className="flex">
            <aside className="w-60 shrink-0 border-r bg-muted/45 px-4 py-5">
              <div className="flex h-7 items-center gap-2 px-2" aria-hidden="true">
                <span className="size-4 rounded-full border border-muted-foreground/50" />
                <span className="size-4 rounded-full border border-muted-foreground/50" />
                <span className="size-4 rounded-full border border-muted-foreground/50" />
              </div>

              <div className="mt-3 flex h-9 items-center gap-2 rounded-lg border bg-background/55 px-3 text-sm text-muted-foreground shadow-inner">
                <Search className="size-4" />
                <span>Search</span>
              </div>

              <nav className="mt-8 flex flex-col gap-1">
                {navigationItems.map((item) => {
                  const Icon = item.icon;
                  return (
                    <div
                      key={item.label}
                      className={cn(
                        `
                          flex items-center gap-3 rounded-lg px-3 py-2 text-sm
                          font-medium
                        `,
                        item.active
                          ? "bg-accent text-accent-foreground"
                          : `
                          text-muted-foreground
                        `,
                      )}
                    >
                      <Icon className="size-5" />
                      {item.label}
                    </div>
                  );
                })}
              </nav>
            </aside>

            <main className="min-w-0 flex-1 bg-background">
              <form.Field name="banner.asset">
                {(field) => (
                  <FormField field={field}>
                    {(control) => (
                      <BannerEditor
                        asset={field.state.value}
                        error={null}
                        invalid={Boolean(control["aria-invalid"])}
                        uploading={bannerUploading}
                        fit={draft.banner.fit}
                        focalX={draft.banner.focalX}
                        onAssetChange={(file) => {
                          field.handleChange(createClientResourceAsset(file));
                        }}
                        onAssetReject={(message) => toast.error(message)}
                        onFitChange={(fit) => form.setFieldValue("banner.fit", fit)}
                        onFocalXChange={(focalX) => form.setFieldValue("banner.focalX", focalX)}
                      />
                    )}
                  </FormField>
                )}
              </form.Field>

              <form.Field name="links" mode="array">
                {(field) => (
                  <FormField field={field}>
                    {(control) => (
                      <div
                        {...control}
                        tabIndex={-1}
                        className="relative flex min-h-12 items-center justify-center border border-dashed border-primary/50 bg-muted/60 px-12 py-2 text-xs text-muted-foreground"
                      >
                        <EditableLinks
                          items={field.state.value}
                          emptyState={<MunkiCategories />}
                          addLabel="Add a link (replaces the category list)"
                          onAdd={field.pushValue}
                          onReplace={field.replaceValue}
                          onRemove={field.removeValue}
                          onReorder={field.handleChange}
                        />
                      </div>
                    )}
                  </FormField>
                )}
              </form.Field>

              <div className="px-8 py-6">
                <h2 className="border-b pb-3 text-2xl font-semibold">All items</h2>
                <div className="grid grid-cols-2 gap-x-9">
                  {sampleSoftware.map((item) => (
                    <SoftwareItem key={item.name} item={item} />
                  ))}
                </div>
              </div>

              <form.Field name="footer.text">
                {(textField) => (
                  <form.Field name="footer.links" mode="array">
                    {(linksField) => (
                      <FormField field={linksField}>
                        {(control) => (
                          <footer
                            {...control}
                            tabIndex={-1}
                            className="relative flex min-h-12 flex-wrap items-center justify-center gap-y-1 rounded-br-2xl border border-dashed border-primary/50 px-10 py-2 text-[11px] text-muted-foreground"
                          >
                            <Editable
                              value={textField.state.value}
                              onValueChange={textField.handleChange}
                              placeholder="Add footer text"
                              className="w-56 gap-0"
                            >
                              <EditableArea className="block w-full">
                                <EditablePreview className="h-7 px-1.5 py-0.5 text-center text-[11px]" />
                                <EditableInput className="h-7 border-transparent bg-transparent px-1.5 py-0.5 text-center text-[11px] shadow-none" />
                              </EditableArea>
                            </Editable>

                            <EditableLinks
                              items={linksField.state.value}
                              leadingSeparator={textField.state.value.length > 0}
                              addLabel="Add footer link"
                              onAdd={linksField.pushValue}
                              onReplace={linksField.replaceValue}
                              onRemove={linksField.removeValue}
                              onReorder={linksField.handleChange}
                            />
                          </footer>
                        )}
                      </FormField>
                    )}
                  </form.Field>
                )}
              </form.Field>
            </main>
          </div>
        </div>
      </div>
    </section>
  );
}
function EditableLinks({
  items,
  emptyState,
  leadingSeparator = false,
  addLabel = "Add link",
  onAdd,
  onReplace,
  onRemove,
  onReorder,
}: {
  items: ClientResourceLink[];
  emptyState?: ReactNode;
  leadingSeparator?: boolean;
  addLabel?: string;
  onAdd: (item: ClientResourceLink) => void;
  onReplace: (index: number, item: ClientResourceLink) => void;
  onRemove: (index: number) => void;
  onReorder: (items: ClientResourceLink[]) => void;
}) {
  const [dialogIndex, setDialogIndex] = useState<number | "new" | null>(null);
  const editedLink = typeof dialogIndex === "number" ? (items[dialogIndex] ?? null) : null;
  return (
    <>
      {items.length > 0 ? (
        <Sortable value={items} getItemValue={(item) => item.id} onValueChange={onReorder}>
          <SortableContent className="flex flex-wrap items-center justify-center gap-y-1">
            {items.map((link, index) => (
              <SortableItem key={link.id} value={link.id} className="flex items-center">
                {index > 0 || leadingSeparator ? <span className="px-2 text-border">|</span> : null}
                <SortableItemHandle
                  render={
                    <button
                      type="button"
                      className="cursor-grab rounded-sm px-0.5 outline-none hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring active:cursor-grabbing"
                    >
                      {link.label || "Untitled link"}
                    </button>
                  }
                />
                <DropdownMenu>
                  <DropdownMenuTrigger
                    render={
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon-xs"
                        className="ml-0.5 size-5"
                      />
                    }
                  >
                    <MoreHorizontal />
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="start">
                    <DropdownMenuGroup>
                      <DropdownMenuItem onClick={() => setDialogIndex(index)}>
                        <Pencil />
                        Edit
                      </DropdownMenuItem>
                      <DropdownMenuItem variant="destructive" onClick={() => onRemove(index)}>
                        <Trash2 />
                        Remove
                      </DropdownMenuItem>
                    </DropdownMenuGroup>
                  </DropdownMenuContent>
                </DropdownMenu>
              </SortableItem>
            ))}
          </SortableContent>
        </Sortable>
      ) : (
        emptyState
      )}

      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              className="absolute right-2"
              disabled={items.length >= 12}
              onClick={() => setDialogIndex("new")}
            />
          }
        >
          <Plus />
        </TooltipTrigger>
        <TooltipContent>{addLabel}</TooltipContent>
      </Tooltip>

      <LinkDialog
        open={dialogIndex !== null}
        onOpenChange={(open) => {
          if (!open) setDialogIndex(null);
        }}
        link={editedLink}
        onSave={(link) => {
          if (typeof dialogIndex === "number") {
            onReplace(dialogIndex, link);
          } else {
            onAdd(link);
          }
        }}
      />
    </>
  );
}
function MunkiCategories() {
  return (
    <div className="flex flex-wrap items-center justify-center gap-y-1">
      {defaultCategories.map((category, index) => (
        <span key={category} className="flex items-center">
          {index > 0 ? (
            <span aria-hidden="true" className="px-2 text-border">
              |
            </span>
          ) : null}
          <span>{category}</span>
        </span>
      ))}
    </div>
  );
}
function SoftwareItem({ item }: { item: (typeof sampleSoftware)[number] }) {
  return (
    <div className="flex min-w-0 gap-4 border-b py-4">
      <SoftwareArtwork size="lg" className="size-16" />
      <div className="flex min-w-0 flex-1 flex-col items-start">
        <p className="truncate text-sm font-medium">{item.name}</p>
        <p className="truncate text-xs text-muted-foreground">{item.detail}</p>
        <Badge
          variant="ghost"
          className="mt-3 h-6 bg-msc-small-button-background px-2.5 text-[10px] font-bold text-msc-small-button-foreground"
        >
          INSTALL
        </Badge>
      </div>
    </div>
  );
}
