// src/router/routes.js

const routes = [
  {
    path: '/',
    component: () => import('layouts/MainLayout.vue'),
    children: [
      { path: '', component: () => import('pages/IndexPage.vue') },
      { path: 'add', component: () => import('pages/AddStreamPage.vue') },
      { path: 'config', component: () => import('pages/ConfigPage.vue') },
      { path: 'log', component: () => import('pages/LogPage.vue') },
    ],
  },
];

export default routes;
