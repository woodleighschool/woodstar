import { Parser } from "@sgress454/node-sql-parser/umd/sqlite.umd.js";

export const invalidSQLSyntaxMessage = "Syntax error. Please review before saving.";

const parser = new Parser();

export function sqlSyntaxError(query: string): string | undefined {
  if (!query.trim()) return undefined;

  try {
    parser.astify(query);
    return undefined;
  } catch {
    return invalidSQLSyntaxMessage;
  }
}

export function validSQLSyntax(query: string) {
  return sqlSyntaxError(query) === undefined;
}
