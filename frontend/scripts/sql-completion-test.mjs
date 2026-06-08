import assert from 'node:assert/strict';
import {
  SQL_FUNCTIONS,
  extractTablesFromSql,
  filterCompletionValues,
  getCompletionContext,
  preferredFieldTables,
  tableForQualifier,
} from '../src/utils/sqlCompletion.js';

assert.equal(
  getCompletionContext('SELECT * FROM ').type,
  'table',
  'FROM 后应该进入表名补全上下文'
);

const sql = 'SELECT u. FROM users AS u JOIN orders o ON o.user_id = u.id WHERE ';
const refs = extractTablesFromSql(sql);

assert.deepEqual(
  refs.map(ref => ({ table: ref.table, alias: ref.alias })),
  [
    { table: 'users', alias: 'u' },
    { table: 'orders', alias: 'o' },
  ],
  '应该识别 FROM/JOIN 表名和别名'
);

assert.equal(
  tableForQualifier('u', refs),
  'users',
  '别名 u. 应映射到 users 字段'
);

assert.deepEqual(
  preferredFieldTables('SELECT * FROM users u WHERE '),
  ['users'],
  '输入表名后默认字段建议应优先来自该表'
);

assert.equal(
  getCompletionContext('SELECT u.').type,
  'member',
  '表名或别名加点后应该进入字段补全上下文'
);

assert.deepEqual(
  filterCompletionValues(['id', 'user_id', 'created_at'], getCompletionContext('SELECT u.user').fragment),
  ['user_id'],
  '字段输入片段应该能过滤字段候选'
);

assert.ok(
  filterCompletionValues(SQL_FUNCTIONS, 'COU').includes('COUNT()'),
  '普通上下文应该能匹配 SQL 函数候选'
);

console.log('sql completion context tests passed');
