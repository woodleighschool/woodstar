import { motion, useInView, useMotionValue, useSpring } from "motion/react";
import { useEffect, useRef } from "react";

import { cn } from "@/lib/utils";

export interface NumberTickerProps {
  value: number;
  decimalPlaces?: number;
  className?: string;
}

/** Smoothly animates a number from its previous value to the new one. */
export function NumberTicker({ value, decimalPlaces = 0, className }: NumberTickerProps) {
  const ref = useRef<HTMLSpanElement>(null);
  const motionValue = useMotionValue(value);
  const springValue = useSpring(motionValue, { damping: 30, stiffness: 80 });
  const inView = useInView(ref, { once: true, margin: "0px" });

  useEffect(() => {
    if (inView) motionValue.set(value);
  }, [inView, value, motionValue]);

  useEffect(() => {
    return springValue.on("change", (latest) => {
      if (!ref.current) return;
      ref.current.textContent = latest.toLocaleString(undefined, {
        minimumFractionDigits: decimalPlaces,
        maximumFractionDigits: decimalPlaces,
      });
    });
  }, [springValue, decimalPlaces]);

  return (
    <motion.span ref={ref} className={cn("inline-block tabular-nums", className)}>
      {(0).toLocaleString(undefined, {
        minimumFractionDigits: decimalPlaces,
        maximumFractionDigits: decimalPlaces,
      })}
    </motion.span>
  );
}
