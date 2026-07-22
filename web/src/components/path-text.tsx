import { useCallback, useRef, useState } from "react";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

interface PathTextProps {
  className?: string;
  value?: string | null;
}

export function PathText({ value, className }: PathTextProps) {
  const textRef = useRef<HTMLSpanElement>(null);
  const [truncated, setTruncated] = useState(false);

  const measure = () => {
    const text = textRef.current;
    setTruncated(text !== null && text.scrollWidth > text.clientWidth);
  };

  const setTextRef = useCallback((text: HTMLSpanElement | null) => {
    textRef.current = text;
    setTruncated(text !== null && text.scrollWidth > text.clientWidth);
  }, []);

  if (!value) return "-";

  return (
    <Tooltip disabled={!truncated}>
      <TooltipTrigger
        render={
          <span
            key={value}
            ref={setTextRef}
            className={cn("block max-w-80 truncate", className)}
            onFocus={measure}
            onMouseEnter={measure}
          />
        }
      >
        {value}
      </TooltipTrigger>
      <TooltipContent className="max-w-[calc(100vw-2rem)] break-all whitespace-normal sm:max-w-lg">
        {value}
      </TooltipContent>
    </Tooltip>
  );
}
