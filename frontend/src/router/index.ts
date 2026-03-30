import { createRouter, createWebHashHistory, RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'Home',
    redirect: '/connections',
  },
  {
    path: '/connections',
    name: 'Connections',
    component: () => import('../views/ConnectionManager.vue'),
  },
  {
    path: '/workspace',
    name: 'Workspace',
    component: () => import('../views/Workspace.vue'),
  },
  {
    path: '/query',
    name: 'Query',
    component: () => import('../views/QueryEditor.vue'),
  },
  {
    path: '/sync',
    name: 'Sync',
    component: () => import('../views/SchemaSync.vue'),
  },
  {
    path: '/data-sync',
    name: 'DataSync',
    component: () => import('../views/DataSync.vue'),
  },
  {
    path: '/import-export',
    name: 'ImportExport',
    component: () => import('../views/ImportExport.vue'),
  },
  {
    path: '/logs',
    name: 'Logs',
    component: () => import('../views/LogViewer.vue'),
  },
  {
    path: '/mongo/connections',
    name: 'MongoConnections',
    component: () => import('../views/mongo/MongoConnectionManager.vue'),
  },
  {
    path: '/mongo/workspace',
    name: 'MongoWorkspace',
    component: () => import('../views/mongo/MongoWorkspace.vue'),
  },
  {
    path: '/mongo/query',
    name: 'MongoQuery',
    component: () => import('../views/mongo/MongoQueryEditor.vue'),
  },
];

const router = createRouter({
  history: createWebHashHistory(),
  routes,
});

export default router;
