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
    path: '/campaigns/analytics',
    name: 'campaignAnalytics',
    meta: { title: 'analytics.title', group: 'campaigns' },
    component: () => import('../views/CampaignAnalytics.vue'),
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
