import Vue from 'vue';
import VueRouter from 'vue-router';

Vue.use(VueRouter);

// The meta.group param is used in App.vue to expand menu group by name.
const routes = [
  {
    path: '/404',
    name: '404_page',
    meta: { title: '404' },
    component: () => import('../views/404.vue'),
  },
  {
    path: '/',
    name: 'dashboard',
    meta: { title: '' },
    component: () => import('../views/Dashboard.vue'),
  },
  {
    path: '/lists',
    name: 'lists',
    meta: { title: 'globals.terms.lists', group: 'lists' },
    component: () => import('../views/Lists.vue'),
  },
  {
    path: '/lists/forms',
    name: 'forms',
    meta: { title: 'forms.title', group: 'lists' },
    component: () => import('../views/Forms.vue'),
  },
  {
    path: '/lists/:id',
    name: 'list',
    meta: { title: 'globals.terms.lists', group: 'lists' },
    component: () => import('../views/Lists.vue'),
  },
  {
    path: '/subscribers',
    name: 'subscribers',
    meta: { title: 'globals.terms.subscribers', group: 'subscribers' },
    component: () => import('../views/Subscribers.vue'),
  },
  {
    path: '/subscribers/import',
    name: 'import',
    meta: { title: 'import.title', group: 'subscribers' },
    component: () => import('../views/Import.vue'),
  },
  {
    path: '/sql-snippets',
    name: 'sql-snippets',
    meta: { title: 'sqlSnippets.title', group: 'settings' },
    component: () => import('../views/SqlSnippets.vue'),
  },
  {
    path: '/subscribers/bounces',
    name: 'bounces',
    meta: { title: 'globals.terms.bounces', group: 'subscribers' },
    component: () => import('../views/Bounces.vue'),
  },
  {
    path: '/subscribers/lists/:listID',
    name: 'subscribers_list',
    meta: { title: 'globals.terms.subscribers', group: 'subscribers' },
    component: () => import('../views/Subscribers.vue'),
  },
  {
    path: '/subscribers/:id',
    name: 'subscriber',
    meta: { title: 'globals.terms.subscribers', group: 'subscribers' },
    component: () => import('../views/Subscribers.vue'),
  },
  {
    path: '/campaigns',
    name: 'campaigns',
    meta: { title: 'globals.terms.campaigns', group: 'campaigns' },
    component: () => import('../views/Campaigns.vue'),
  },
  {
    path: '/campaigns/media',
    name: 'media',
    meta: { title: 'globals.terms.media', group: 'campaigns' },
    component: () => import('../views/Media.vue'),
  },
  {
    path: '/campaigns/templates',
    name: 'templates',
    meta: { title: 'globals.terms.templates', group: 'campaigns' },
    component: () => import('../views/Templates.vue'),
  },
  {
    path: '/campaigns/analytics',
    name: 'campaignAnalytics',
    meta: { title: 'analytics.title', group: 'campaigns' },
    component: () => import('../views/CampaignAnalytics.vue'),
  },
  {
    path: '/campaigns/:id',
    name: 'campaign',
    meta: { title: 'globals.terms.campaign', group: 'campaigns' },
    component: () => import('../views/Campaign.vue'),
  },
  {
    path: '/user/profile',
    name: 'userProfile',
    meta: { title: 'users.profile', group: 'settings' },
    component: () => import('../views/UserProfile.vue'),
  },
  {
    path: '/settings',
    name: 'settings',
    meta: { title: 'globals.terms.settings', group: 'settings' },
    component: () => import('../views/Settings.vue'),
  },
  {
    path: '/settings/logs',
    name: 'logs',
    meta: { title: 'logs.title', group: 'settings' },
    component: () => import('../views/Logs.vue'),
  },
  {
    path: '/users',
    name: 'users',
    meta: { title: 'globals.terms.users', group: 'users' },
    component: () => import('../views/Users.vue'),
  },
  {
    path: '/users/roles/users',
    name: 'userRoles',
    meta: { title: 'users.userRoles', group: 'users' },
    component: () => import('../views/Roles.vue'),
  },
  {
    path: '/users/roles/lists',
    name: 'listRoles',
    meta: { title: 'users.listRoles', group: 'users' },
    component: () => import('../views/Roles.vue'),
  },
  {
    path: '/settings/maintenance',
    name: 'maintenance',
    meta: { title: 'maintenance.title', group: 'settings' },
    component: () => import('../views/Maintenance.vue'),
  },
];

const router = new VueRouter({
  mode: 'history',
  base: import.meta.env.BASE_URL,
  routes,

  scrollBehavior(to) {
    if (to.hash) {
      return { selector: to.hash };
    }
    return { x: 0, y: 0 };
  },
});

export default router;
