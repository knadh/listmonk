import Vue from 'vue';
import Buefy from 'buefy';
import VueI18n from 'vue-i18n';

import App from './App.vue';
import router from './router';
import store from './store';
import * as api from './api';
import Utils from './utils';

// Internationalisation.
Vue.use(VueI18n);
const i18n = new VueI18n();

Vue.use(Buefy, {});
Vue.config.productionTip = false;

// Setup the router.
router.beforeEach((to, from, next) => {
  if (to.matched.length === 0) {
    next('/404');
  } else {
    next();
  }
});

router.afterEach((to) => {
  Vue.nextTick(() => {
    const t = to.meta.title && i18n.te(to.meta.title) ? `${i18n.tc(to.meta.title, 0)} /` : '';
    document.title = `${t} listmonk`;
  });
});

async function initConfig(app) {
  // Load logged in user profile, server side config, and the language file before mounting the app.
  const [profile, cfg] = await Promise.all([api.getUserProfile(), api.getServerConfig()]);

  const lang = await api.getLang(cfg.lang);
  i18n.locale = cfg.lang;
  i18n.setLocaleMessage(i18n.locale, lang);

  Vue.prototype.$utils = new Utils(i18n);
  Vue.prototype.$api = api;
  Vue.prototype.$events = app;

  // $can('permission:name') is used in the UI to check whether the logged in user
  // has a certain permission to toggle visibility of UI objects and UI functionality.
  Vue.prototype.$can = (...perms) => {
    if (profile.userRole.id === 1) {
      return true;
    }

    // If the perm ends with a wildcard, check whether at least one permission
    // in the group is present. Eg: campaigns:* will return true if at least
    // one of campaigns:get, campaigns:manage etc. are present.
    return perms.some((perm) => {
      if (perm.endsWith('*')) {
        const group = `${perm.split(':')[0]}:`;
        return profile.userRole.permissions.some((p) => p.startsWith(group));
      }

      return profile.userRole.permissions.includes(perm);
    });
  };

  Vue.prototype.$canList = (id, perm) => {
    if (profile.userRole.id === 1) {
      return true;
    }

    return profile.listRole.lists.some((list) => list.id === id && list.permissions.includes(perm));
  };

  // Set the page title after i18n has loaded.
  const to = router.history.current;
  const title = to.meta.title ? `${i18n.tc(to.meta.title, 0)} /` : '';
  document.title = `${title} listmonk`;

  if (app) {
    app.$mount('#app');
  }
}

const v = new Vue({
  router,
  store,
  i18n,
  render: (h) => h(App),

  data: {
    isLoaded: false,
  },

  methods: {
    loadConfig() {
      initConfig();
    },
  },

  mounted() {
    v.isLoaded = true;
  },
});

initConfig(v);
