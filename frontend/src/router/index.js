import Vue from 'vue';
import VueRouter from 'vue-router';

Vue.use(VueRouter);

// The meta.group param is used in App.vue to expand menu group by name.
const routes = [
  {
    path: '/404',
    name: '404_page',
    meta: { title: '404' },
    component: () => import(/* webpackChunkName: "main" */ '../views/404.vue'),
  },
  {
    path: '/',
    name: 'dashboard',
    meta: { title: '' },
    component: () => import(/* webpackChunkName: "main" */ '../views/Dashboard.vue'),
  },
  {
    path: '/lists',
    name: 'lists',
    meta: { title: 'globals.terms.lists', group: 'lists' },
    component: () => import(/* webpackChunkName: "main" */ '../views/Lists.vue'),
  },
  {
    path: '/lists/forms',
    name: 'forms',
    meta: { title: 'forms.title', group: 'lists' },
    component: () => import(/* webpackChunkName: "main" */ '../views/Forms.vue'),
  },
  {
    path: '/lists/:id',
    name: 'list',
    meta: { title: 'globals.terms.lists', group: 'lists' },
    component: () => import(/* webpackChunkName: "main" */ '../views/Lists.vue'),
  },
  {
    path: '/subscribers',
    name: 'subscribers',
    meta: { title: 'globals.terms.subscribers', group: 'subscribers' },
    component: () => import(/* webpackChunkName: "main" */ '../views/Subscribers.vue'),
  },
  {
    path: '/subscribers/import',
    name: 'import',
    meta: { title: 'import.title', group: 'subscribers' },
    component: () => import(/* webpackChunkName: "main" */ '../views/Import.vue'),
  },
  {
    path: '/subscribers/bounces',
    name: 'bounces',
    meta: { title: 'globals.terms.bounces', group: 'subscribers' },
    component: () => import(/* webpackChunkName: "main" */ '../views/Bounces.vue'),
  },
  {
    path: '/subscribers/lists/:listID',
    name: 'subscribers_list',
    meta: { title: 'globals.terms.subscribers', group: 'subscribers' },
    component: () => import(/* webpackChunkName: "main" */ '../views/Subscribers.vue'),
  },
  {
    path: '/subscribers/:id',
    name: 'subscriber',
    meta: { title: 'globals.terms.subscribers', group: 'subscribers' },
    component: () => import(/* webpackChunkName: "main" */ '../views/Subscribers.vue'),
  },
  {
    path: '/campaigns',
    name: 'campaigns',
    meta: { title: 'globals.terms.campaigns', group: 'campaigns' },
    component: () => import(/* webpackChunkName: "main" */ '../views/Campaigns.vue'),
  },
  {
    path: '/campaigns/media',
    name: 'media',
    meta: { title: 'globals.terms.media', group: 'campaigns' },
    component: () => import(/* webpackChunkName: "main" */ '../views/Media.vue'),
  },
  {
    path: '/campaigns/templates',
    name: 'templates',
    meta: { title: 'globals.terms.templates', group: 'campaigns' },
    component: () => import(/* webpackChunkName: "main" */ '../views/Templates.vue'),
  },
  {
    path: '/campaigns/analytics',
    name: 'campaignAnalytics',
    meta: { title: 'analytics.title', group: 'campaigns' },
    component: () => import(/* webpackChunkName: "main" */ '../views/CampaignAnalytics.vue'),
  },
  {
    path: '/campaigns/:id',
    name: 'campaign',
    meta: { title: 'globals.terms.campaign', group: 'campaigns' },
    component: () => import(/* webpackChunkName: "main" */ '../views/Campaign.vue'),
  },
  {
    path: '/settings',
    name: 'settings',
    meta: { title: 'globals.terms.settings', group: 'settings' },
    component: () => import(/* webpackChunkName: "main" */ '../views/Settings.vue'),
  },
  {
    path: '/settings/logs',
    name: 'logs',
    meta: { title: 'logs.title', group: 'settings' },
    component: () => import(/* webpackChunkName: "main" */ '../views/Logs.vue'),
  },
  {
    path: '/settings/maintenance',
    name: 'maintenance',
    meta: { title: 'maintenance.title', group: 'settings' },
    component: () => import(/* webpackChunkName: "main" */ '../views/Maintenance.vue'),
  },
];

const router = new VueRouter({
  mode: 'history',
  base: process.env.BASE_URL,
  routes,

  scrollBehavior(to) {
    if (to.hash) {
      return { selector: to.hash };
    }
    return { x: 0, y: 0 };
  },
});

export default router;
