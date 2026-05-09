import osqueryFleetTablesJSON from "../../../schema/osquery_fleet_schema.json";

export interface OsquerySchemaColumn {
  name: string;
  type: string;
  description?: string;
  required?: boolean;
  hidden?: boolean;
}

export interface OsquerySchemaTable {
  name: string;
  description?: string;
  platforms?: string[];
  hidden?: boolean;
  columns: OsquerySchemaColumn[];
}

const queryTables = osqueryFleetTablesJSON as OsquerySchemaTable[];

export const osqueryTables = [...queryTables].sort((a, b) => a.name.localeCompare(b.name));

export const osqueryTablesAvailable = osqueryTables.filter((table) => !table.hidden);

export const osqueryTableNames = osqueryTablesAvailable.map((table) => table.name);

export const osqueryTableColumnNames = osqueryTables.flatMap((table) => {
  if (table.hidden) return [];
  return table.columns.flatMap((column) => (column.hidden ? [] : column.name));
});

export function selectedTableColumns(selectedTables: string[]) {
  return osqueryTables.flatMap((table) => {
    if (table.hidden) return [];
    if (selectedTables.length > 0 && !selectedTables.includes(table.name)) return [];
    return table.columns.filter((column) => !column.hidden);
  });
}
