import Vue from 'vue';
import Buefy from 'buefy';
import humps from 'humps';
import VueI18n from 'vue-i18n';

import App from './App.vue';
import router from './router';
import store from './store';
import * as api from './api';
import { models } from './constants';
import Utils from './utils';

// Internationalisation.
Vue.use(VueI18n);
const i18n = new VueI18n();

Vue.use(Buefy, {});
Vue.config.productionTip = false;

// Globals.
const ut = new Utils(i18n);
Vue.mixin({
  computed: {
    $utils: () => ut,
    $api: () => api,
  },

  methods: {
    $reloadServerConfig: () => {
      // Get the config.js <script> tag, remove it, and re-add it.
      let s = document.querySelector('#server-config');
      const url = s.getAttribute('src');
      s.remove();

      s = document.createElement('script');
      s.setAttribute('src', url);
      s.setAttribute('id', 'server-config');
      s.onload = () => {
        store.commit('setModelResponse',
          { model: models.serverConfig, data: humps.camelizeKeys(window.CONFIG) });
      };
      document.body.appendChild(s);
    },
  },
});


// window.CONFIG is loaded from /api/config.js directly in a <script> tag.
if (window.CONFIG) {
  store.commit('setModelResponse',
    { model: models.serverConfig, data: humps.camelizeKeys(window.CONFIG) });

  // Load language.
  i18n.locale = window.CONFIG.lang['_.code'];
  i18n.setLocaleMessage(i18n.locale, window.CONFIG.lang);
}

new Vue({
  router,
  store,
  i18n,
  render: (h) => h(App),
}).$mount('#app');
