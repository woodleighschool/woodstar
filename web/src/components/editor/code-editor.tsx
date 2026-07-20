import { acceptCompletion, startCompletion } from "@codemirror/autocomplete";
import { HighlightStyle, syntaxHighlighting } from "@codemirror/language";
import { type Extension, Prec } from "@codemirror/state";
import { EditorView, keymap } from "@codemirror/view";
import { tags as t } from "@lezer/highlight";
import CodeMirror, { type ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { forwardRef, useMemo, useSyncExternalStore } from "react";

import { cn } from "@/lib/utils";

interface CodeEditorProps {
  value: string;
  onChange: (next: string) => void;
  extensions?: Extension[];
  placeholder?: string;
  readOnly?: boolean;
  className?: string;
  invalid?: boolean;
  lineNumbers?: boolean;
  lineWrapping?: boolean;
  highlightActiveLine?: boolean;
}

const EMPTY_EXTENSIONS: Extension[] = [];
const DARK_MODE_QUERY = "(prefers-color-scheme: dark)";

function subscribeToColorScheme(onChange: () => void) {
  const media = window.matchMedia(DARK_MODE_QUERY);
  media.addEventListener("change", onChange);
  return () => media.removeEventListener("change", onChange);
}

function prefersDarkMode() {
  return window.matchMedia(DARK_MODE_QUERY).matches;
}

const editorCompletionKeymap = Prec.highest(
  keymap.of([
    {
      key: "Tab",
      run(view) {
        if (view.state.readOnly) return false;
        if (acceptCompletion(view)) return true;
        startCompletion(view);
        return true;
      },
    },
  ]),
);

const surfaceTheme = EditorView.theme({
  "&": {
    fontSize: "0.85rem",
    backgroundColor: "var(--code-editor-background, var(--card))",
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
    backgroundColor: "var(--code-editor-background, var(--card))",
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
  ".cm-placeholder": { color: "var(--muted-foreground)" },
  ".cm-tooltip": {
    backgroundColor: "var(--popover)",
    color: "var(--popover-foreground)",
    border: "1px solid var(--border)",
    borderRadius: "0.375rem",
    zIndex: "50",
  },
  ".cm-tooltip-autocomplete > ul > li[aria-selected]": {
    backgroundColor: "var(--accent)",
    color: "var(--accent-foreground)",
  },
  ".cm-lintRange-error": {
    backgroundImage: "none",
    textDecoration: "underline wavy var(--destructive)",
    textDecorationSkipInk: "none",
    textDecorationThickness: "1.5px",
  },
  ".cm-lintRange-warning": {
    backgroundImage: "none",
    textDecoration: "underline wavy var(--warning)",
    textDecorationSkipInk: "none",
    textDecorationThickness: "1.5px",
  },
  ".cm-lintRange-info": {
    backgroundImage: "none",
    textDecoration: "underline wavy var(--info)",
    textDecorationSkipInk: "none",
    textDecorationThickness: "1.5px",
  },
  ".cm-tooltip-lint": {
    backgroundColor: "var(--popover)",
    color: "var(--popover-foreground)",
    borderColor: "var(--border)",
  },
  ".cm-diagnostic": {
    borderLeft: "3px solid var(--border)",
    paddingLeft: "0.625rem",
  },
  ".cm-diagnostic-error": { borderLeftColor: "var(--destructive)" },
  ".cm-diagnostic-warning": { borderLeftColor: "var(--warning)" },
  ".cm-diagnostic-info": { borderLeftColor: "var(--info)" },
});

const lightHighlight = HighlightStyle.define([
  { tag: t.keyword, color: "var(--primary)", fontWeight: "600" },
  { tag: t.string, color: "oklch(0.43 0.08 130)" },
  { tag: t.number, color: "oklch(0.45 0.11 55)" },
  { tag: t.comment, color: "var(--muted-foreground)", fontStyle: "italic" },
  { tag: t.operator, color: "var(--foreground)" },
  { tag: [t.typeName, t.className], color: "oklch(0.42 0.09 235)" },
  { tag: t.variableName, color: "var(--foreground)" },
]);

const darkHighlight = HighlightStyle.define([
  { tag: t.keyword, color: "oklch(0.8 0.08 150)", fontWeight: "600" },
  { tag: t.string, color: "oklch(0.78 0.09 130)" },
  { tag: t.number, color: "oklch(0.78 0.11 70)" },
  { tag: t.comment, color: "var(--muted-foreground)", fontStyle: "italic" },
  { tag: t.operator, color: "var(--foreground)" },
  { tag: [t.typeName, t.className], color: "oklch(0.78 0.08 235)" },
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
    invalid,
    lineNumbers = true,
    lineWrapping = true,
    highlightActiveLine = true,
  },
  ref,
) {
  const isDark = useSyncExternalStore(subscribeToColorScheme, prefersDarkMode);
  const editorExtensions = useMemo(
    () => [
      editorCompletionKeymap,
      ...extensions,
      ...(lineWrapping ? [EditorView.lineWrapping] : []),
      surfaceTheme,
      syntaxHighlighting(isDark ? darkHighlight : lightHighlight),
    ],
    [extensions, isDark, lineWrapping],
  );

  return (
    <div
      aria-invalid={invalid ? true : undefined}
      className={cn(
        `
          overflow-visible rounded-md border border-input
          bg-(--code-editor-background)
        `,
        readOnly
          ? "[--code-editor-background:var(--background)]"
          : "[--code-editor-background:var(--card)]",
        `
          aria-invalid:border-destructive aria-invalid:ring-[3px]
          aria-invalid:ring-destructive/20
        `,
        className,
      )}
    >
      <CodeMirror
        ref={ref}
        value={value}
        height="auto"
        theme="none"
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
