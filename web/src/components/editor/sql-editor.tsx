import { autocompletion } from "@codemirror/autocomplete";
import { sql } from "@codemirror/lang-sql";
import { HighlightStyle, syntaxHighlighting } from "@codemirror/language";
import { EditorView } from "@codemirror/view";
import { tags as t } from "@lezer/highlight";
import CodeMirror, { type ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { useTheme } from "next-themes";
import { forwardRef, useMemo } from "react";

import { useOsquerySchema } from "@/hooks/use-osquery-schema";
import { cn } from "@/lib/utils";

import { completionsFromSchema } from "./autocomplete";

interface SQLEditorProps {
  value: string;
  onChange: (next: string) => void;
  placeholder?: string;
  readOnly?: boolean;
  className?: string;
}

const MIN_EDITOR_LINES = 1;
const MAX_EDITOR_LINES = 20;
const EDITOR_LINE_HEIGHT_REM = 1.275;
const EDITOR_VERTICAL_CHROME_REM = 1;

function editorHeightFor(value: string) {
  const lineCount = Math.max(MIN_EDITOR_LINES, value.split(/\r\n|\r|\n/).length);
  const visibleLines = Math.min(lineCount, MAX_EDITOR_LINES);
  return `${visibleLines * EDITOR_LINE_HEIGHT_REM + EDITOR_VERTICAL_CHROME_REM}rem`;
}

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
  },
  ".cm-tooltip-autocomplete > ul > li[aria-selected]": {
    backgroundColor: "var(--accent)",
    color: "var(--accent-foreground)",
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

export const SQLEditor = forwardRef<ReactCodeMirrorRef, SQLEditorProps>(function SQLEditor(
  { value, onChange, placeholder, readOnly, className },
  ref,
) {
  const schema = useOsquerySchema();
  const { resolvedTheme } = useTheme();
  const isDark = resolvedTheme === "dark";
  const maxHeight = editorHeightFor(value);

  const extensions = useMemo(() => {
    const base = [
      sql(),
      EditorView.lineWrapping,
      surfaceTheme,
      syntaxHighlighting(isDark ? darkHighlight : lightHighlight),
    ];
    if (schema.data) {
      base.push(autocompletion({ override: [completionsFromSchema(schema.data)] }));
    }
    return base;
  }, [schema.data, isDark]);

  return (
    <div className={cn("border-input bg-card overflow-hidden rounded-md border", className)}>
      <CodeMirror
        ref={ref}
        value={value}
        height="auto"
        minHeight={editorHeightFor("")}
        maxHeight={maxHeight}
        theme={isDark ? "dark" : "light"}
        extensions={extensions}
        onChange={onChange}
        placeholder={placeholder}
        readOnly={readOnly}
        basicSetup={{
          lineNumbers: true,
          highlightActiveLine: true,
          autocompletion: false,
          foldGutter: false,
        }}
      />
    </div>
  );
});
