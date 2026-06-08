export const SQL_FUNCTIONS = [
  'COUNT()', 'COUNT(*)', 'COUNT(DISTINCT )', 'SUM()', 'AVG()', 'MIN()', 'MAX()',
  'COALESCE()', 'IFNULL()', 'IF()', 'CONCAT()', 'SUBSTRING()', 'LENGTH()',
  'TRIM()', 'UPPER()', 'LOWER()', 'NOW()', 'DATE()', 'YEAR()', 'MONTH()',
  'DAY()', 'UNIX_TIMESTAMP()', 'FROM_UNIXTIME()', 'DATE_FORMAT()',
  'ROUND()', 'FLOOR()', 'CEIL()', 'ABS()', 'CAST()', 'CONVERT()',
  'JSON_EXTRACT()', 'JSON_UNQUOTE()',
];

export const SQL_KEYWORDS = [
  'SELECT', 'FROM', 'WHERE', 'AND', 'OR', 'NOT', 'IN', 'LIKE', 'BETWEEN',
  'ORDER BY', 'GROUP BY', 'HAVING', 'LIMIT', 'OFFSET', 'JOIN', 'LEFT JOIN',
  'RIGHT JOIN', 'INNER JOIN', 'ON', 'AS', 'DISTINCT', 'INSERT INTO', 'UPDATE',
  'SET', 'DELETE FROM', 'IS NULL', 'IS NOT NULL', 'ASC', 'DESC',
];

const identifier = '[`"\\[]?[\\w$.-]+[`"\\]]?';
const tableClausePattern = new RegExp(
  `\\b(?:FROM|JOIN|UPDATE|INTO|DELETE\\s+FROM)\\s+(${identifier})(?:\\s+(?:AS\\s+)?([\\w$]+))?`,
  'gi'
);

const normalizeIdentifier = (value = '') => {
  return value.trim().replace(/^[`"\[]|[`"\]]$/g, '');
};

export const extractTablesFromSql = (sqlBeforeCursor) => {
  const tables = [];
  let match;
  tableClausePattern.lastIndex = 0;
  while ((match = tableClausePattern.exec(sqlBeforeCursor))) {
    const tableName = normalizeIdentifier(match[1]);
    const alias = match[2] ? normalizeIdentifier(match[2]) : '';
    if (!tableName) continue;
    tables.push({
      table: tableName.includes('.') ? tableName.split('.').pop() : tableName,
      qualifiedTable: tableName,
      alias,
    });
  }
  return tables;
};

export const getCompletionContext = (sqlBeforeCursor) => {
  const memberMatch = sqlBeforeCursor.match(/([`"\[]?[\w$]+[`"\]]?)\.\s*([`"\[]?[\w$]*)$/);
  if (memberMatch) {
    return {
      type: 'member',
      qualifier: normalizeIdentifier(memberMatch[1]),
      fragment: normalizeIdentifier(memberMatch[2]),
      from: sqlBeforeCursor.length - memberMatch[2].length,
    };
  }

  const wordMatch = sqlBeforeCursor.match(/([`"\[]?[\w$]*)$/);
  const fragment = normalizeIdentifier(wordMatch?.[1] || '');
  const recent = sqlBeforeCursor.slice(0, sqlBeforeCursor.length - (wordMatch?.[1]?.length || 0));
  const afterTableKeyword = /\b(FROM|JOIN|UPDATE|INTO|DELETE\s+FROM)\s+$/i.test(recent);

  return {
    type: afterTableKeyword ? 'table' : 'default',
    fragment,
    from: sqlBeforeCursor.length - (wordMatch?.[1]?.length || 0),
  };
};

export const tableForQualifier = (qualifier, tableRefs) => {
  const normalized = normalizeIdentifier(qualifier).toLowerCase();
  const match = [...tableRefs].reverse().find((ref) => {
    return ref.alias?.toLowerCase() === normalized || ref.table.toLowerCase() === normalized;
  });
  return match?.table || '';
};

export const preferredFieldTables = (sqlBeforeCursor) => {
  return [...extractTablesFromSql(sqlBeforeCursor)].reverse().map(ref => ref.table);
};

export const filterCompletionValues = (values, fragment) => {
  const needle = normalizeIdentifier(fragment).toLowerCase();
  const unique = [...new Set(values.filter(Boolean))];
  if (!needle) return unique;
  return unique.filter(value => value.toLowerCase().includes(needle));
};
