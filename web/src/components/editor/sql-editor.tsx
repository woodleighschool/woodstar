import { autocompletion } from "@codemirror/autocomplete";
import { SQLite, sql } from "@codemirror/lang-sql";
import type { EditorState, Extension } from "@codemirror/state";
import { EditorView } from "@codemirror/view";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { forwardRef, useMemo } from "react";

import { useOsquerySchema, type OsqueryTable } from "@/hooks/use-osquery-schema";

import { completionsFromSchema } from "./autocomplete";
import { CodeEditor } from "./code-editor";

interface SQLEditorProps {
  value: string;
  onChange: (next: string) => void;
  placeholder?: string;
  readOnly?: boolean;
  className?: string;
  onTableMetaClick?: (tableName: string) => void;
}

export const SQLEditor = forwardRef<ReactCodeMirrorRef, SQLEditorProps>(function SQLEditor(
  { value, onChange, placeholder, readOnly, className, onTableMetaClick },
  ref,
) {
  const schema = useOsquerySchema();

  const extensions = useMemo<Extension[]>(() => {
    const next: Extension[] = [sql({ dialect: SQLite })];
    if (schema.data) {
      next.push(autocompletion({ override: [completionsFromSchema(schema.data)] }));
    }
    if (schema.data && onTableMetaClick) {
      next.push(tableMetaClick(schema.data, onTableMetaClick));
    }
    return next;
  }, [onTableMetaClick, schema.data]);

  return (
    <CodeEditor
      ref={ref}
      value={value}
      onChange={onChange}
      extensions={extensions}
      placeholder={placeholder}
      readOnly={readOnly}
      className={className}
    />
  );
});

function tableMetaClick(tables: OsqueryTable[], onTableMetaClick: (tableName: string) => void): Extension {
  const tableByName = new Map(tables.map((table) => [table.name.toLowerCase(), table.name]));
  return EditorView.domEventHandlers({
    mousedown(event, view) {
      if (!event.metaKey && !event.ctrlKey) return false;
      const pos = view.posAtCoords({ x: event.clientX, y: event.clientY });
      if (pos == null) return false;
      const token = identifierAt(view.state, pos);
      const tableName = token ? tableByName.get(token.toLowerCase()) : undefined;
      if (!tableName) return false;
      event.preventDefault();
      onTableMetaClick(tableName);
      return true;
    },
  });
}

function identifierAt(state: EditorState, pos: number): string | null {
  const line = state.doc.lineAt(pos);
  const offset = pos - line.from;
  const text = line.text;
  let start = offset;
  let end = offset;

  if (start > 0 && !isIdentifierChar(text[start]) && isIdentifierChar(text[start - 1])) {
    start -= 1;
    end -= 1;
  }
  while (start > 0 && isIdentifierChar(text[start - 1])) {
    start -= 1;
  }
  while (end < text.length && isIdentifierChar(text[end])) {
    end += 1;
  }
  return start === end ? null : text.slice(start, end);
}

function isIdentifierChar(char: string | undefined) {
  return char !== undefined && /[A-Za-z0-9_]/.test(char);
}
