import { cn } from "@/lib/utils";

type LogoSize = "sm" | "md";

interface LogoProps {
  className?: string;
  size?: LogoSize;
}

const sizeClass: Record<LogoSize, string> = {
  sm: "size-9",
  md: "size-12",
};

export function Logo({ className, size = "sm" }: LogoProps) {
  return (
    <div
      className={cn(
        "relative flex shrink-0 items-center justify-center",
        sizeClass[size],
        className,
      )}
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        fill="currentColor"
        viewBox="0 0 1024 1024"
        className="size-full!"
      >
        <defs>
          <path
            id="petal"
            d="
              M512 110
              C586 240, 580 365, 512 470
              C444 365, 438 240, 512 110
              Z
            "
          />
        </defs>

        <use href="#petal" />
        <use href="#petal" transform="rotate(60 512 512)" />
        <use href="#petal" transform="rotate(120 512 512)" />
        <use href="#petal" transform="rotate(180 512 512)" />
        <use href="#petal" transform="rotate(240 512 512)" />
        <use href="#petal" transform="rotate(300 512 512)" />
      </svg>
    </div>
  );
}
