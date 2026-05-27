import { autocompletion } from "@codemirror/autocomplete";
import { sql } from "@codemirror/lang-sql";
import type { Extension } from "@codemirror/state";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { forwardRef, useMemo } from "react";

import { useOsquerySchema } from "@/hooks/use-osquery-schema";

import { completionsFromSchema } from "./autocomplete";
import { CodeEditor } from "./code-editor";

interface SQLEditorProps {
  value: string;
  onChange: (next: string) => void;
  placeholder?: string;
  readOnly?: boolean;
  className?: string;
}

export const SQLEditor = forwardRef<ReactCodeMirrorRef, SQLEditorProps>(function SQLEditor(
  { value, onChange, placeholder, readOnly, className },
  ref,
) {
  const schema = useOsquerySchema();

  const extensions = useMemo<Extension[]>(() => {
    const next: Extension[] = [sql()];
    if (schema.data) {
      next.push(autocompletion({ override: [completionsFromSchema(schema.data)] }));
    }
    return next;
  }, [schema.data]);

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
