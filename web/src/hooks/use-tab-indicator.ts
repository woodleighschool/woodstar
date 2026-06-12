import { useLayoutEffect, useRef, useState } from "react";

// Tracks the active tab trigger's box in list-local coordinates so a single
// indicator can glide between tabs with a CSS transition. Measuring relative to
// the list (offsetLeft/offsetWidth) keeps it correct inside a scrolled container,
// where motion's layout projection mis-positions.
export function useTabIndicator(activeValue: string | undefined) {
  const listRef = useRef<HTMLDivElement>(null);
  const [box, setBox] = useState<{ left: number; width: number } | null>(null);

  useLayoutEffect(() => {
    const list = listRef.current;
    if (!list) return;
    const measure = () => {
      const active = list.querySelector<HTMLElement>('[data-state="active"]');
      if (active) setBox({ left: active.offsetLeft, width: active.offsetWidth });
    };
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(list);
    return () => observer.disconnect();
  }, [activeValue]);

  return { listRef, box };
}
