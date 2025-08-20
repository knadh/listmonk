import { createApp, nextTick } from 'vue';
import Buefy from 'buefy';
import Oruga from '@oruga-ui/oruga-next';
import { createI18n } from 'vue-i18n';

import App from './App.vue';
import router from './router';
import store from './store';
import * as api from './api';
import Utils from './utils';

// Create i18n instance
const i18n = createI18n({
  legacy: false,
  locale: 'en',
  messages: {},
});

// Setup the router.
router.beforeEach((to, from, next) => {
  if (to.matched.length === 0) {
    next('/404');
  } else {
    next();
  }
});

router.afterEach((to) => {
  nextTick(() => {
    const t = to.meta.title && i18n.global.te(to.meta.title) ? `${i18n.global.t(to.meta.title, 0)} /` : '';
    document.title = `${t} listmonk`;
  });
});

async function initConfig(app) {
  // Load logged in user profile, server side config, and the language file before mounting the app.
  const [profile, cfg] = await Promise.all([api.getUserProfile(), api.getServerConfig()]);

  const lang = await api.getLang(cfg.lang);
  i18n.global.locale.value = cfg.lang;
  i18n.global.setLocaleMessage(cfg.lang, lang);

  const utils = new Utils(i18n.global);

  // Set global properties
  // eslint-disable-next-line no-param-reassign
  app.config.globalProperties.$utils = utils;
  // eslint-disable-next-line no-param-reassign
  app.config.globalProperties.$api = api;
  // eslint-disable-next-line no-param-reassign
  app.config.globalProperties.$events = app;

  // Set up global toast/dialog functions for backward compatibility
  // eslint-disable-next-line no-param-reassign
  window.showToast = ({
    message, type, duration, position,
  }) => {
    console.log(`Toast: ${type || 'info'} - ${message}`);
  };

  // eslint-disable-next-line no-param-reassign
  window.toastError = (message) => {
    console.log(`Error: ${message}`);
  };

  // eslint-disable-next-line no-param-reassign
  window.showConfirm = ({ message, onConfirm, onCancel }) => {
    if (confirm(message)) { // eslint-disable-line no-alert
      if (onConfirm) onConfirm();
    } else if (onCancel) {
      onCancel();
    }
  };

  // eslint-disable-next-line no-param-reassign
  window.showPrompt = ({
    message, onConfirm, onCancel, inputAttrs,
  }) => {
    const result = prompt(message, inputAttrs?.value || ''); // eslint-disable-line no-alert
    if (result !== null && onConfirm) {
      onConfirm(result);
    } else if (result === null && onCancel) {
      onCancel();
    }
  };

  // $can('permission:name') is used in the UI to check whether the logged in user
  // has a certain permission to toggle visibility of UI objects and UI functionality.
  // eslint-disable-next-line no-param-reassign
  app.config.globalProperties.$can = (...perms) => {
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

  // eslint-disable-next-line no-param-reassign
  app.config.globalProperties.$canList = (id, perm) => {
    if (profile.userRole.id === 1) {
      return true;
    }

    return profile.listRole.lists.some((list) => list.id === id && list.permissions.includes(perm));
  };

  // Set the page title after i18n has loaded.
  const to = router.currentRoute.value;
  const title = to.meta.title ? `${i18n.global.t(to.meta.title, 0)} /` : '';
  document.title = `${title} listmonk`;

  if (app) {
    // eslint-disable-next-line no-param-reassign
    app.config.globalProperties.$isLoaded = true;
    app.mount('#app');
  }
}

// Create Vue 3 app
const app = createApp(App);
app.use(Buefy);

// Use plugins
app.use(router);
app.use(store);
app.use(i18n);
app.use(Oruga, {
  // Oruga config options can go here
});

// Create a reactive data object for global state
app.config.globalProperties.$isLoaded = false;

// Method to reload config
app.config.globalProperties.$loadConfig = () => {
  initConfig();
};

// Initialize the app
initConfig(app);
