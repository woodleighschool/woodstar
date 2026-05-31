import { HighlightStyle, syntaxHighlighting } from "@codemirror/language";
import type { Extension } from "@codemirror/state";
import { EditorView } from "@codemirror/view";
import { tags as t } from "@lezer/highlight";
import CodeMirror, { type ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { useTheme } from "next-themes";
import { forwardRef, useMemo } from "react";

import { cn } from "@/lib/utils";

interface CodeEditorProps {
  value: string;
  onChange: (next: string) => void;
  extensions?: Extension[];
  placeholder?: string;
  readOnly?: boolean;
  className?: string;
  lineNumbers?: boolean;
  lineWrapping?: boolean;
  highlightActiveLine?: boolean;
}

const EMPTY_EXTENSIONS: Extension[] = [];

const surfaceTheme = EditorView.theme({
  "&": {
    fontSize: "0.85rem",
    backgroundColor: "var(--card)",
    color: "var(--foreground)",
  },
  "&.cm-focused": { outline: "none" },
  ".cm-content": {
    fontFamily: "var(--font-mono)",
    caretColor: "var(--foreground)",
  },
  ".cm-line": {
    paddingRight: "0.75rem",
  },
  ".cm-gutters": {
    backgroundColor: "var(--card)",
    color: "var(--muted-foreground)",
    border: "none",
    borderRight: "1px solid var(--border)",
  },
  ".cm-activeLine": { backgroundColor: "color-mix(in oklch, var(--muted) 50%, transparent)" },
  ".cm-activeLineGutter": { backgroundColor: "color-mix(in oklch, var(--muted) 50%, transparent)" },
  ".cm-selectionBackground, &.cm-focused .cm-selectionBackground, ::selection": {
    backgroundColor: "color-mix(in oklch, var(--primary) 25%, transparent)",
  },
  ".cm-cursor": { borderLeftColor: "var(--foreground)" },
  ".cm-tooltip": {
    backgroundColor: "var(--popover)",
    color: "var(--popover-foreground)",
    border: "1px solid var(--border)",
    borderRadius: "0.375rem",
    zIndex: "50",
  },
});

const lightHighlight = HighlightStyle.define([
  { tag: t.keyword, color: "oklch(0.5 0.18 285)", fontWeight: "600" },
  { tag: t.string, color: "oklch(0.45 0.15 150)" },
  { tag: t.number, color: "oklch(0.5 0.18 30)" },
  { tag: t.comment, color: "var(--muted-foreground)", fontStyle: "italic" },
  { tag: t.operator, color: "var(--foreground)" },
  { tag: [t.typeName, t.className], color: "oklch(0.5 0.15 240)" },
  { tag: t.variableName, color: "var(--foreground)" },
]);

const darkHighlight = HighlightStyle.define([
  { tag: t.keyword, color: "oklch(0.78 0.18 285)", fontWeight: "600" },
  { tag: t.string, color: "oklch(0.78 0.14 150)" },
  { tag: t.number, color: "oklch(0.78 0.16 30)" },
  { tag: t.comment, color: "var(--muted-foreground)", fontStyle: "italic" },
  { tag: t.operator, color: "var(--foreground)" },
  { tag: [t.typeName, t.className], color: "oklch(0.78 0.14 240)" },
  { tag: t.variableName, color: "var(--foreground)" },
]);

export const CodeEditor = forwardRef<ReactCodeMirrorRef, CodeEditorProps>(function CodeEditor(
  {
    value,
    onChange,
    extensions = EMPTY_EXTENSIONS,
    placeholder,
    readOnly,
    className,
    lineNumbers = true,
    lineWrapping = true,
    highlightActiveLine = true,
  },
  ref,
) {
  const { resolvedTheme } = useTheme();
  const isDark = resolvedTheme === "dark";
  const editorExtensions = useMemo(
    () => [
      ...extensions,
      ...(lineWrapping ? [EditorView.lineWrapping] : []),
      surfaceTheme,
      syntaxHighlighting(isDark ? darkHighlight : lightHighlight),
    ],
    [extensions, isDark, lineWrapping],
  );

  return (
    <div className={cn("border-input bg-card overflow-visible rounded-md border", className)}>
      <CodeMirror
        ref={ref}
        value={value}
        height="auto"
        theme={isDark ? "dark" : "light"}
        extensions={editorExtensions}
        onChange={onChange}
        placeholder={placeholder}
        readOnly={readOnly}
        basicSetup={{
          lineNumbers,
          highlightActiveLine,
          autocompletion: false,
          foldGutter: false,
        }}
      />
    </div>
  );
});
