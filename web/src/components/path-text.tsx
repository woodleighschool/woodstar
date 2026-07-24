import { useLayoutEffect, useRef, useState } from "react";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

interface PathTextProps {
  className?: string;
  value?: string | null;
}

export function PathText({ value, className }: PathTextProps) {
  const triggerRef = useRef<HTMLSpanElement>(null);
  const measurementRef = useRef<HTMLSpanElement>(null);
  const [truncated, setTruncated] = useState(false);

  useLayoutEffect(() => {
    const trigger = triggerRef.current;
    const measurement = measurementRef.current;
    if (!trigger || !measurement) return undefined;

    const measure = () => {
      setTruncated(measurement.scrollWidth > trigger.clientWidth);
    };
    const observer = new ResizeObserver(measure);
    observer.observe(trigger);
    measure();
    return () => observer.disconnect();
  }, [value]);

  if (!value) return "-";

  const suffixLength = Math.min(8, Math.floor(value.length / 2));
  const splitAt = value.length - suffixLength;

  return (
    <Tooltip disabled={!truncated}>
      <TooltipTrigger
        render={
          <span
            key={value}
            ref={triggerRef}
            data-slot="path-text"
            aria-label={value}
            className={cn(
              "relative flex max-w-96 min-w-48 overflow-hidden whitespace-nowrap",
              className,
            )}
          >
            <span className="min-w-0 truncate">{value.slice(0, splitAt)}</span>
            <span className="shrink-0 whitespace-nowrap">{value.slice(splitAt)}</span>
            <span
              ref={measurementRef}
              aria-hidden
              className="pointer-events-none invisible absolute whitespace-nowrap"
            >
              {value}
            </span>
          </span>
        }
      />
      <TooltipContent className="max-w-[calc(100vw-2rem)] break-all whitespace-normal sm:max-w-lg">
        {value}
      </TooltipContent>
    </Tooltip>
  );
}
