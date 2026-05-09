import ReactMarkdown, { type Components } from "react-markdown";

import { cn } from "@/lib/utils";

interface MarkdownProps {
  children: string;
  className?: string;
}

const components: Components = {
  p: ({ children }) => <p className="mb-2 last:mb-0">{children}</p>,
  a: ({ children, href }) => (
    <a href={href} target="_blank" rel="noreferrer" className="text-primary hover:underline">
      {children}
    </a>
  ),
  ul: ({ children }) => <ul className="mb-2 list-disc space-y-1 pl-4 last:mb-0">{children}</ul>,
  ol: ({ children }) => <ol className="mb-2 list-decimal space-y-1 pl-4 last:mb-0">{children}</ol>,
  li: ({ children }) => <li>{children}</li>,
  code: ({ children }) => <span>{children}</span>,
  pre: ({ children }) => (
    <div className="text-foreground mb-2 overflow-x-auto text-sm leading-relaxed whitespace-pre-wrap last:mb-0">
      {children}
    </div>
  ),
  h1: ({ children }) => <h4 className="mt-2 mb-1 text-sm font-semibold">{children}</h4>,
  h2: ({ children }) => <h4 className="mt-2 mb-1 text-sm font-semibold">{children}</h4>,
  h3: ({ children }) => <h4 className="mt-2 mb-1 text-sm font-semibold">{children}</h4>,
};

export function Markdown({ children, className }: MarkdownProps) {
  return (
    <div className={cn("text-sm leading-relaxed break-words", className)}>
      <ReactMarkdown components={components}>{children}</ReactMarkdown>
    </div>
  );
}
