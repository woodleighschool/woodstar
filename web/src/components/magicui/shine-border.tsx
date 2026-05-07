import { cn } from "@/lib/utils";

export interface ShineBorderProps extends React.HTMLAttributes<HTMLDivElement> {
  borderWidth?: number;
  duration?: number;
  shineColor?: string | string[];
}

/**
 * A subtle animated gradient stroke around an element. Single sweep on mount,
 * then loops slowly. Use sparingly (one-time events: setup, first run).
 */
export function ShineBorder({
  borderWidth = 1,
  duration = 14,
  shineColor = ["oklch(0.55 0.21 285)", "oklch(0.70 0.19 285)", "oklch(0.55 0.21 285)"],
  className,
  style,
  ...props
}: ShineBorderProps) {
  const colors = Array.isArray(shineColor) ? shineColor : [shineColor];
  return (
    <div
      aria-hidden
      style={
        {
          "--border-width": `${borderWidth}px`,
          "--duration": `${duration}s`,
          backgroundImage: `radial-gradient(transparent, transparent, ${colors.join(",")}, transparent, transparent)`,
          backgroundSize: "300% 300%",
          mask: `linear-gradient(#000 0 0) content-box, linear-gradient(#000 0 0)`,
          WebkitMask: `linear-gradient(#000 0 0) content-box, linear-gradient(#000 0 0)`,
          maskComposite: "exclude",
          WebkitMaskComposite: "xor",
          padding: `var(--border-width)`,
          ...style,
        } as React.CSSProperties
      }
      className={cn("animate-shine pointer-events-none absolute inset-0 size-full rounded-[inherit]", className)}
      {...props}
    />
  );
}
