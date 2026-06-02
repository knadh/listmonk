<template>
  <div id="app" data-sidebar-layout>
    <template v-if="$root.isLoaded">
      <nav data-topnav>
        <div class="row">
          <div class="col-4 branding">
            <button data-sidebar-toggle type="button" aria-label="Toggle sidebar menu" class="small">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
                stroke-linecap="round" role="img" aria-label="Menu icon">
                <line x1="4" y1="7" x2="20" y2="7" />
                <line x1="4" y1="12" x2="20" y2="12" />
                <line x1="4" y1="17" x2="16" y2="17" />
              </svg>
            </button>
            <router-link :to="{ name: 'dashboard' }" class="favicon">
              <img src="@/assets/favicon.png" alt="listmonk" />
            </router-link>
            <router-link :to="{ name: 'dashboard' }" class="logo">
              <img src="@/assets/logo.svg" alt="listmonk" />
            </router-link>
          </div>

          <div class="col-8 justify-end hstack">
            <button type="button" class="ghost small" @click="emitPageRefresh" data-cy="btn-refresh"
              :aria-label="$t('globals.buttons.refresh')" :title="$t('globals.buttons.refresh')">
              <b-icon icon="refresh" />
            </button>

            <ot-dropdown v-if="profile.username">
              <button popovertarget="user-menu" type="button" class="user-nav-button ghost small user-trigger">
                <span class="user-avatar">
                  <img v-if="profile.avatar" :src="profile.avatar" alt="" />
                  <span v-else>{{ profile.username[0].toUpperCase() }}</span>
                </span>
                <span class="user-label">{{ profile.username }}</span>
              </button>
              <menu popover id="user-menu">
                <router-link to="/user/profile" role="menuitem">
                  <b-icon icon="account-outline" /> {{ $t('users.profile') }}
                </router-link>
                <button type="button" role="menuitem" class="ghost" @click="doLogout">
                  <b-icon icon="logout-variant" /> {{ $t('users.logout') }}
                </button>
              </menu>
            </ot-dropdown>
          </div>
        </div>
      </nav>

      <aside data-sidebar>
        <navigation :is-mobile="isMobile" :active-item="activeItem" :active-group="activeGroup"
          @toggleGroup="toggleGroup" />
      </aside>

      <main>
        <div class="container">
          <div class="global-notices" v-if="isGlobalNotices">
            <div v-if="serverConfig.needs_restart" role="alert">
              {{ $t('settings.needsRestart') }}
              &mdash;
              <button type="button" data-variant="danger"
                @click="$utils.confirm($t('settings.confirmRestart'), reloadApp)">
                {{ $t('settings.restart') }}
              </button>
            </div>

            <template v-if="serverConfig.update">
              <div v-if="serverConfig.update.update.is_new" role="status">
                {{ $t('settings.updateAvailable', {
                  version: `${serverConfig.update.update.release_version}
                (${$utils.getDate(serverConfig.update.update.release_date).format('DD MMM YY')})`,
                }) }}
                <a :href="serverConfig.update.update.url" target="_blank" rel="noopener noreferer">View</a>
              </div>

              <template v-if="serverConfig.update.messages && serverConfig.update.messages.length > 0">
                <div v-for="m in serverConfig.update.messages" role="status" :key="m.title">
                  <h3 v-if="m.title"><strong>{{ m.title }}</strong></h3>
                  <p v-if="m.description">{{ m.description }}</p>
                  <a v-if="m.url" :href="m.url" target="_blank" rel="noopener noreferer">View</a>
                </div>
              </template>
            </template>

            <div v-if="serverConfig.has_legacy_user" role="alert">
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
      </main>
    </template>

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
  },

  methods: {
    toggleGroup(group, state) {
      this.activeGroup = state ? { [group]: true } : {};
    },

    emitPageRefresh() {
      this.$root.$emit('page.refresh');
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
          this.$utils.toast(msg[2], '', null, true);
        }
      };
    },
  },

  computed: {
    ...mapState(['serverConfig', 'profile']),

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
    this.$api.getLists({ minimal: true, per_page: 'all', status: 'active' });

    window.addEventListener('resize', () => {
      this.windowWidth = window.innerWidth;
    });

    this.listenEvents();
  },
});
</script>

<style>
@import "assets/vendor/oat.min.css";
@import "assets/icons/fontello.css";
@import "assets/style.css";
</style>
