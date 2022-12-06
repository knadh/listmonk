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

function initConfig(app) {
  // Load server side config and language before mounting the app.
  api.getServerConfig().then((data) => {
    api.getLang(data.lang).then((lang) => {
      i18n.locale = data.lang;
      i18n.setLocaleMessage(i18n.locale, lang);

      Vue.prototype.$utils = new Utils(i18n);
      Vue.prototype.$api = api;

      // Set the page title after i18n has loaded.
      const to = router.history.current;
      const t = to.meta.title ? `${i18n.tc(to.meta.title, 0)} /` : '';
      document.title = `${t} listmonk`;

      if (app) {
        app.$mount('#app');
      }
    });
  });

  api.getSettings();
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
