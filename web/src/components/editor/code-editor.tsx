import { acceptCompletion, startCompletion } from "@codemirror/autocomplete";
import { HighlightStyle, syntaxHighlighting } from "@codemirror/language";
import { type Extension, Prec } from "@codemirror/state";
import { EditorView, keymap } from "@codemirror/view";
import { tags as t } from "@lezer/highlight";
import CodeMirror, { type ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { forwardRef, useMemo } from "react";

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

const syntaxHighlight = HighlightStyle.define([
  { tag: t.keyword, color: "var(--code-keyword)", fontWeight: "600" },
  { tag: t.string, color: "var(--code-string)" },
  { tag: t.number, color: "var(--code-number)" },
  { tag: t.comment, color: "var(--muted-foreground)", fontStyle: "italic" },
  { tag: t.operator, color: "var(--foreground)" },
  { tag: [t.typeName, t.className], color: "var(--code-type)" },
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
  const editorExtensions = useMemo(
    () => [
      editorCompletionKeymap,
      ...extensions,
      ...(lineWrapping ? [EditorView.lineWrapping] : []),
      surfaceTheme,
      syntaxHighlighting(syntaxHighlight),
    ],
    [extensions, lineWrapping],
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
