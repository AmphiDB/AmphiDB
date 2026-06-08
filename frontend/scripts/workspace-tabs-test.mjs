import assert from 'node:assert/strict';
import { filterTabsForDatabase } from '../src/views/workspaceTabs.js';

const tabs = [
  { id: 'data:superspu.orders', database: 'superspu', table: 'orders' },
  { id: 'schema-view:mig_express.mig_transports', database: 'mig_express', table: 'mig_transports' },
  { id: 'data:mig_express.mig_transports', database: 'mig_express', table: 'mig_transports' },
];

assert.deepEqual(
  filterTabsForDatabase(tabs, 'mig_express').map(tab => tab.id),
  ['schema-view:mig_express.mig_transports', 'data:mig_express.mig_transports'],
  'switching database must remove tabs from the previous database'
);

assert.deepEqual(
  filterTabsForDatabase(tabs, null),
  [],
  'clearing database selection must close object tabs'
);

console.log('workspace tab filtering tests passed');
