<template>
  <div id="app">
    <b-navbar :fixed-top="true" v-if="$root.isLoaded">
      <template #brand>
        <div class="logo">
          <router-link :to="{ name: 'dashboard' }">
            <img class="full" src="@/assets/logo.svg" alt="" />
            <img class="favicon" src="@/assets/favicon.png" alt="" />
          </router-link>
        </div>
      </template>
      <template #end>
        <navigation v-if="isMobile" :is-mobile="isMobile" :active-item="activeItem" :active-group="activeGroup"
          @toggleGroup="toggleGroup" @doLogout="doLogout" />

        <b-navbar-dropdown class="user" tag="div" right v-else>
          <template v-if="profile.username" #label>
            <span class="user-avatar">
              <img v-if="profile.avatar" :src="profile.avatar" alt="" />
              <span v-else>{{ profile.username[0].toUpperCase() }}</span>
            </span>
          </template>

          <b-navbar-item class="user-name" tag="router-link" to="/user/profile">
            <strong>{{ profile.username }}</strong>
            <div class="is-size-7">{{ profile.name }}</div>
          </b-navbar-item>

          <b-navbar-item href="#">
            <router-link to="/user/profile">
              <b-icon icon="account-outline" /> {{ $t('users.profile') }}
            </router-link>
          </b-navbar-item>
          <b-navbar-item href="#">
            <a href="#" @click.prevent="toggleDarkMode">
              <span class="icon">
                <svg v-if="isDarkMode" viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
                  <!-- eslint-disable-next-line max-len -->
                  <path
                    d="M3.55 19.09L4.96 20.5L6.76 18.71L5.34 17.29M12 6C8.69 6 6 8.69 6 12S8.69 18 12 18 18 15.31 18 12C18 8.68 15.31 6 12 6M20 13H23V11H20M17.24 18.71L19.04 20.5L20.45 19.09L18.66 17.29M20.45 5L19.04 3.6L17.24 5.39L18.66 6.81M13 1H11V4H13M6.76 5.39L4.96 3.6L3.55 5L5.34 6.81L6.76 5.39M1 13H4V11H1M13 20H11V23H13"
                  />
                </svg>
                <svg v-else viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
                  <!-- eslint-disable-next-line max-len -->
                  <path
                    d="M17.75,4.09L15.22,6.03L16.13,9.09L13.5,7.28L10.87,9.09L11.78,6.03L9.25,4.09L12.44,4L13.5,1L14.56,4L17.75,4.09M21.25,11L19.61,12.25L20.2,14.23L18.5,13.06L16.8,14.23L17.39,12.25L15.75,11L17.81,10.95L18.5,9L19.19,10.95L21.25,11M18.97,15.95C19.8,15.87 20.69,17.05 20.16,17.8C19.84,18.25 19.5,18.67 19.08,19.07C15.17,23 8.84,23 4.94,19.07C1.03,15.17 1.03,8.83 4.94,4.93C5.34,4.53 5.76,4.17 6.21,3.85C6.96,3.32 8.14,4.21 8.06,5.04C7.79,7.9 8.75,10.87 10.95,13.06C13.14,15.26 16.1,16.22 18.97,15.95M17.33,17.97C14.5,17.81 11.7,16.64 9.53,14.5C7.36,12.31 6.2,9.5 6.04,6.68C3.23,9.82 3.34,14.64 6.35,17.66C9.37,20.67 14.19,20.78 17.33,17.97Z"
                  />
                </svg>
              </span>
              {{ isDarkMode ? 'Light Mode' : 'Dark Mode' }}
            </a>
          </b-navbar-item>
          <b-navbar-item href="#">
            <a href="#" @click.prevent="doLogout"><b-icon icon="logout-variant" /> {{ $t('users.logout') }}</a>
          </b-navbar-item>
        </b-navbar-dropdown>
      </template>
    </b-navbar>

    <div class="wrapper" v-if="$root.isLoaded">
      <section class="sidebar">
        <b-sidebar position="static" mobile="hide" :fullheight="true" :open="true" :can-cancel="false">
          <div>
            <b-menu :accordion="false">
              <navigation v-if="!isMobile" :is-mobile="isMobile" :active-item="activeItem" :active-group="activeGroup"
                @toggleGroup="toggleGroup" />
            </b-menu>
          </div>
        </b-sidebar>
      </section>
      <!-- sidebar-->

      <!-- body //-->
      <div class="main">
        <div class="global-notices" v-if="isGlobalNotices">
          <div v-if="serverConfig.needs_restart" class="notification is-danger">
            {{ $t('settings.needsRestart') }}
            &mdash;
            <b-button class="is-primary" size="is-small"
              @click="$utils.confirm($t('settings.confirmRestart'), reloadApp)">
              {{ $t('settings.restart') }}
            </b-button>
          </div>

          <template v-if="serverConfig.update">
            <div v-if="serverConfig.update.update.is_new" class="notification is-success">
              {{ $t('settings.updateAvailable', {
                version: `${serverConfig.update.update.release_version}
              (${$utils.getDate(serverConfig.update.update.release_date).format('DD MMM YY')})`,
              }) }}
              <a :href="serverConfig.update.update.url" target="_blank" rel="noopener noreferer">View</a>
            </div>

            <template v-if="serverConfig.update.messages && serverConfig.update.messages.length > 0">
              <div v-for="m in serverConfig.update.messages" class="notification"
                :class="{ [m.priority === 'high' ? 'is-danger' : 'is-info']: true }" :key="m.title">
                <h3 class="is-size-5" v-if="m.title"><strong>{{ m.title }}</strong></h3>
                <p v-if="m.description">{{ m.description }}</p>
                <a v-if="m.url" :href="m.url" target="_blank" rel="noopener noreferer">View</a>
              </div>
            </template>
          </template>

          <div v-if="serverConfig.has_legacy_user" class="notification is-danger">
            <b-icon icon="warning-empty" />
            Remove the <code>admin_username</code> and <code>admin_password</code> fields from the TOML
            configuration file or environment variables. If you are using APIs, create and use new API credentials
            before removing them. Visit
            <router-link :to="{ name: 'users' }">
              Admin -> Settings -> Users
            </router-link> dashboard. <a href="https://listmonk.app/docs/upgrade/#upgrading-to-v4xx" target="_blank"
              rel="noopener noreferer">Learn more.</a>
          </div>
        </div>

        <router-view :key="$route.fullPath" />
      </div>
    </div>

    <b-loading v-if="!$root.isLoaded" active />
  </div>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import { uris } from './constants';

import Navigation from './components/Navigation.vue';

export default Vue.extend({
  name: 'App',

  components: {
    Navigation,
  },

  data() {
    return {
      activeItem: {},
      activeGroup: {},
      windowWidth: window.innerWidth,
    };
  },

  watch: {
    $route(to) {
      // Set the current route name to true for active+expanded keys in the
      // menu to pick up.
      this.activeItem = { [to.name]: true };
      if (to.meta.group) {
        this.activeGroup = { [to.meta.group]: true };
      } else {
        // Reset activeGroup to collapse menu items on navigating
        // to non group items from sidebar
        this.activeGroup = {};
      }
    },

    isDarkMode(newVal) {
      this.applyDarkMode(newVal);
    },
  },

  methods: {
    toggleGroup(group, state) {
      this.activeGroup = state ? { [group]: true } : {};
    },

    reloadApp() {
      this.$api.reloadApp().then(() => {
        this.$utils.toast('Reloading app ...');

        // Poll until there's a 200 response, waiting for the app
        // to restart and come back up.
        const pollId = setInterval(() => {
          this.$api.getHealth().then(() => {
            clearInterval(pollId);
            document.location.reload();
          });
        }, 500);
      });
    },

    doLogout() {
      this.$api.logout().then(() => {
        document.location.href = uris.root;
      });
    },

    toggleDarkMode() {
      const newValue = !this.isDarkMode;
      this.$store.commit('setDarkMode', newValue);
      this.$utils.setPref('darkMode', newValue);
    },

    applyDarkMode(isDark) {
      const appElement = document.getElementById('app');
      if (isDark) {
        appElement.classList.add('dark-mode');
      } else {
        appElement.classList.remove('dark-mode');
      }
    },

    listenEvents() {
      const reMatchLog = /(.+?)\.go:\d+:(.+?)$/im;
      const evtSource = new EventSource(uris.errorEvents, { withCredentials: true });
      let numEv = 0;
      evtSource.onmessage = (e) => {
        if (numEv > 50) {
          return;
        }
        numEv += 1;

        const d = JSON.parse(e.data);
        if (d && d.type === 'error') {
          const msg = reMatchLog.exec(d.message.trim());
          this.$utils.toast(msg[2], 'is-danger', null, true);
        }
      };
    },
  },

  computed: {
    ...mapState(['serverConfig', 'profile', 'isDarkMode']),

    isGlobalNotices() {
      return (this.serverConfig.needs_restart
        || this.serverConfig.has_legacy_user
        || (this.serverConfig.update
        && this.serverConfig.update.messages
        && this.serverConfig.update.messages.length > 0));
    },

    version() {
      return import.meta.env.VUE_APP_VERSION;
    },

    isMobile() {
      return this.windowWidth <= 768;
    },
  },

  mounted() {
    // Lists is required across different views. On app load, fetch the lists
    // and have them in the store.
    this.$api.getLists({ minimal: true, per_page: 'all' });

    // Initialize dark mode from localStorage
    const savedDarkMode = this.$utils.getPref('darkMode');
    if (savedDarkMode !== null) {
      this.$store.commit('setDarkMode', savedDarkMode);
    }
    this.applyDarkMode(this.isDarkMode);

    window.addEventListener('resize', () => {
      this.windowWidth = window.innerWidth;
    });

    this.listenEvents();
  },
});
</script>

<style lang="scss">
@import "assets/style.scss";
@import "assets/icons/fontello.css";
</style>
