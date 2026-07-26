import { Environment } from "@marcbachmann/cel-js";

const santaCELEnvironment = new Environment({ unlistedVariablesAreDyn: false })
  .registerType("Ancestor", {
    fields: {
      path: "string",
      signing_id: "string",
      team_id: "string",
      cdhash: "string",
      args: { repeated: true, type: "string" },
    },
  })
  .registerVariable({
    name: "target",
    schema: {
      signing_id: "string",
      signing_time: "dyn",
      secure_signing_time: "dyn",
      is_platform_binary: "bool",
      team_id: "string",
    },
  })
  .registerVariable("path", "string")
  .registerVariable("args", "list<string>")
  .registerVariable("envs", "map<string, string>")
  .registerVariable("euid", "int")
  .registerVariable("cwd", "string")
  .registerVariable("ancestors", "list<Ancestor>")
  .registerConstant("ALLOWLIST", "int", 1n)
  .registerConstant("BLOCKLIST", "int", 2n)
  .registerConstant("SILENT_BLOCKLIST", "int", 3n)
  .registerConstant("ALLOWLIST_COMPILER", "int", 4n);

export function santaCELExpressionError(expression: string): string | undefined {
  if (expression.trim() === "") return undefined;

  const result = santaCELEnvironment.check(expression);
  if (result.valid) return undefined;
  return readableCELError(result.error);
}

function readableCELError(error: unknown) {
  if (error instanceof Error) return firstLine(error.message);
  if (typeof error === "string") return firstLine(error);
  if (error === undefined || error === null) return "Invalid CEL expression.";
  return "Invalid CEL expression.";
}

function firstLine(message: string) {
  return (
    message
      .split("\n")[0]
      ?.trim()
      .replace(/^ParseError:\s*/, "")
      .replace(/^TypeError:\s*/, "") || "Invalid CEL expression."
  );
}
