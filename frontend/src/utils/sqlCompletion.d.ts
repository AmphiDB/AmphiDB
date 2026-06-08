export const SQL_FUNCTIONS: string[];
export const SQL_KEYWORDS: string[];

export interface SqlTableRef {
  table: string;
  qualifiedTable: string;
  alias: string;
}

export interface SqlCompletionContext {
  type: 'member' | 'table' | 'default';
  qualifier?: string;
  fragment: string;
  from: number;
}

export function extractTablesFromSql(sqlBeforeCursor: string): SqlTableRef[];
export function getCompletionContext(sqlBeforeCursor: string): SqlCompletionContext;
export function tableForQualifier(qualifier: string | undefined, tableRefs: SqlTableRef[]): string;
export function preferredFieldTables(sqlBeforeCursor: string): string[];
export function filterCompletionValues(values: string[], fragment: string): string[];
