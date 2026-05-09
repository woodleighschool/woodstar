export interface OsqueryColumn {
  name: string;
  type: string;
  description?: string;
  required?: boolean;
  hidden?: boolean;
  index?: boolean;
  platforms?: string[];
}

export interface OsqueryExample {
  description?: string;
  query?: string;
}

export interface OsqueryTable {
  name: string;
  description?: string;
  url?: string;
  platforms?: string[];
  evented?: boolean;
  cacheable?: boolean;
  notes?: string;
  examples?: string | OsqueryExample[];
  columns: OsqueryColumn[];
  hidden?: boolean;
  [key: string]: unknown;
}
