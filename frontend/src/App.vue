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

        <b-navbar-dropdown v-else>
          <template v-if="profile" #label>
            <div class="user-avatar">
              <img v-if="profile.avatar" :src="profile.avatar" alt="" />
              <span v-else>{{ profile.username[0].toUpperCase() }}</span>
            </div>
            {{ profile.username }}
          </template>
          <b-navbar-item href="#">
            <router-link :to="`/user/profile`">
              <b-icon icon="account-outline" /> {{ $t('users.profile') }}
            </router-link>
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
        <div class="global-notices" v-if="serverConfig.needs_restart || serverConfig.update">
          <div v-if="serverConfig.needs_restart" class="notification is-danger">
            {{ $t('settings.needsRestart') }}
            &mdash;
            <b-button class="is-primary" size="is-small"
              @click="$utils.confirm($t('settings.confirmRestart'), reloadApp)">
              {{ $t('settings.restart') }}
            </b-button>
          </div>
          <div v-if="serverConfig.update" class="notification is-success">
            {{ $t('settings.updateAvailable', { version: serverConfig.update.version }) }}
            <a :href="serverConfig.update.url" target="_blank" rel="noopener noreferer">View</a>
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
      profile: null,
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
          this.$utils.toast(msg[2], 'is-danger', null, true);
        }
      };
    },
  },

  computed: {
    ...mapState(['serverConfig']),

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

    window.addEventListener('resize', () => {
      this.windowWidth = window.innerWidth;
    });

    this.listenEvents();
    this.$api.getUserProfile().then((d) => {
      this.profile = d;
    });
  },
});
</script>

<style lang="scss">
@import "assets/style.scss";
@import "assets/icons/fontello.css";
</style>
