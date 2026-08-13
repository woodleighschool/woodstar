import { StreamLanguage } from "@codemirror/language";
import { shell } from "@codemirror/legacy-modes/mode/shell";

import { CodeEditor } from "./code-editor";

const shellExtensions = [StreamLanguage.define(shell)];

export function ShellScriptEditor({
  value,
  onChange,
  placeholder,
  invalid,
}: {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  invalid?: boolean;
}) {
  return (
    <CodeEditor
      value={value}
      onChange={onChange}
      extensions={shellExtensions}
      minHeight="14rem"
      maxHeight="30rem"
      placeholder={placeholder}
      invalid={invalid}
    />
  );
}
