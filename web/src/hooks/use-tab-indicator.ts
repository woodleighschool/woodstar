import { useLayoutEffect, useRef, useState } from "react";

// Tracks the active tab trigger's box in list-local coordinates so a single
// indicator can glide between tabs with a CSS transition. Measuring relative to
// the list (offsetLeft/offsetWidth) keeps it correct inside a scrolled container,
// where motion's layout projection mis-positions.
export function useTabIndicator(activeValue?: string) {
  const listRef = useRef<HTMLDivElement>(null);
  const [box, setBox] = useState<{
    height: number;
    left: number;
    top: number;
    width: number;
  } | null>(null);

  useLayoutEffect(() => {
    const list = listRef.current;
    if (!list) return;
    const measure = () => {
      const active = list.querySelector<HTMLElement>('[data-state="active"]');
      if (active) {
        setBox({
          height: active.offsetHeight,
          left: active.offsetLeft,
          top: active.offsetTop,
          width: active.offsetWidth,
        });
      } else {
        setBox(null);
      }
    };
    measure();
    const resizeObserver = new ResizeObserver(measure);
    resizeObserver.observe(list);
    const mutationObserver = new MutationObserver(measure);
    mutationObserver.observe(list, {
      attributeFilter: ["data-state"],
      attributes: true,
      subtree: true,
    });
    return () => {
      resizeObserver.disconnect();
      mutationObserver.disconnect();
    };
  }, [activeValue]);

  return { listRef, box };
}
