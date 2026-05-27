import type { CompletionContext, CompletionResult } from "@codemirror/autocomplete";

import type { OsqueryColumn, OsqueryTable } from "@/lib/schema";

const SQL_KEYWORDS = [
  "SELECT",
  "FROM",
  "WHERE",
  "JOIN",
  "LEFT",
  "RIGHT",
  "INNER",
  "OUTER",
  "ON",
  "AND",
  "OR",
  "NOT",
  "IN",
  "LIKE",
  "BETWEEN",
  "GROUP BY",
  "ORDER BY",
  "LIMIT",
  "OFFSET",
  "AS",
  "DISTINCT",
  "UNION",
  "COUNT",
  "SUM",
  "AVG",
  "MAX",
  "MIN",
];

interface SchemaIndex {
  tables: OsqueryTable[];
  byName: Map<string, OsqueryTable>;
}

export function indexSchema(tables: OsqueryTable[]): SchemaIndex {
  return {
    tables,
    byName: new Map(tables.map((table) => [table.name, table])),
  };
}

// Returns table alias → table object for all FROM/JOIN references before the cursor.
function referencedTables(sql: string, index: SchemaIndex): Map<string, OsqueryTable> {
  const out = new Map<string, OsqueryTable>();
  const cleaned = sql.replace(/\s+/g, " ").toLowerCase();
  const tableMatches = cleaned.matchAll(/\b(?:from|join)\s+([a-z0-9_]+)(?:\s+(?:as\s+)?([a-z0-9_]+))?/g);
  for (const match of tableMatches) {
    const name = match[1];
    // optional regex group: TS types it `string`, but it's `undefined` when unmatched
    const alias = match[2] || name;
    const table = index.byName.get(name);
    if (table) {
      out.set(alias, table);
    }
  }
  return out;
}

const KEYWORD_OPTIONS = SQL_KEYWORDS.map((label) => ({ label, type: "keyword" }));

function tableOptions(tables: OsqueryTable[]) {
  return tables.map((table) => ({
    label: table.name,
    type: "type",
    info: table.description,
  }));
}

function columnOptions(columns: OsqueryColumn[]) {
  return columns.map((column) => ({
    label: column.name,
    type: "property",
    info: `${column.type}${column.description ? " — " + column.description : ""}`,
  }));
}

export function completionsFromSchema(tables: OsqueryTable[]) {
  const index = indexSchema(tables);
  return (context: CompletionContext): CompletionResult | null => {
    const word = context.matchBefore(/[\w.]*/);
    if (!word) return null;
    if (word.from === word.to && !context.explicit) return null;

    const before = context.state.sliceDoc(0, context.pos).toLowerCase();
    const referenced = referencedTables(before, index);

    const qualified = word.text.match(/^([a-z0-9_]+)\.([a-z0-9_]*)$/i);
    if (qualified) {
      const alias = qualified[1].toLowerCase();
      const table = referenced.get(alias) ?? index.byName.get(alias);
      if (!table) return null;
      return {
        from: word.from + alias.length + 1,
        options: columnOptions(table.columns),
      };
    }

    if (/\b(?:from|join)\s+\S*$/.test(before)) {
      return { from: word.from, options: tableOptions(index.tables) };
    }

    const options = [
      ...KEYWORD_OPTIONS,
      ...tableOptions(index.tables),
      ...[...referenced.values()].flatMap((table) => columnOptions(table.columns)),
    ];
    return { from: word.from, options };
  };
}
