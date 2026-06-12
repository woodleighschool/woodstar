import { cn } from "@/lib/utils";

type WoodstarMarkSize = "sm" | "md";

interface WoodstarMarkProps {
  className?: string;
  size?: WoodstarMarkSize;
}

const sizeClass: Record<WoodstarMarkSize, string> = {
  sm: "size-8",
  md: "size-10",
};

const iconClass: Record<WoodstarMarkSize, string> = {
  sm: "size-6",
  md: "size-8",
};

export function WoodstarMark({ className, size = "sm" }: WoodstarMarkProps) {
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
        className={cn("text-primary dark:text-white", iconClass[size])}
      >
        <path d="M512,572c-109-174-88-379,0-572,88,193,109,398,0,572Z" />
        <path d="M566,586c79-166,238-226,396-267-87,130-213,250-396,267Z" />
        <path d="M458,586c-79-166-238-227-395-266,88,131,213,249,395,266Z" />
        <path d="M558,644c180-49,342,37,466,163-181,24-346-13-466-163Z" />
        <path d="M466,645c-180-49-342,36-466,162,181,25,346-12,466-162Z" />
        <path d="M512,664c70,124,56,237,0,360-56-123-70-236,0-360Z" />
      </svg>
    </div>
  );
}
