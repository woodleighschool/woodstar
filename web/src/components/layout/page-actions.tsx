import { createContext, useContext, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";

interface PageActionsContextValue {
  slotEl: HTMLElement | null;
  setSlotEl: (el: HTMLElement | null) => void;
}

const PageActionsContext = createContext<PageActionsContextValue | null>(null);

/**
 * Wraps the app shell so pages can portal CTAs into the topbar's
 * right-side slot via <PageActions>.
 */
export function PageActionsProvider({ children }: { children: ReactNode }) {
  const [slotEl, setSlotEl] = useState<HTMLElement | null>(null);
  return <PageActionsContext.Provider value={{ slotEl, setSlotEl }}>{children}</PageActionsContext.Provider>;
}

/** Renders inside the topbar; pages portal their actions into this element. */
export function PageActionsSlot({ className }: { className?: string }) {
  const ctx = useContext(PageActionsContext);
  if (!ctx) throw new Error("PageActionsSlot must be inside PageActionsProvider");
  const { setSlotEl } = ctx;
  return <div ref={setSlotEl} className={className} />;
}

/** Renders children into the topbar's page-actions slot. */
export function PageActions({ children }: { children: ReactNode }) {
  const ctx = useContext(PageActionsContext);
  if (!ctx?.slotEl) return null;
  return createPortal(children, ctx.slotEl);
}
