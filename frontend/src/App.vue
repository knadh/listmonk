<template>
  <div id="app">
    <b-navbar :fixed-top="true" v-if="$root.isLoaded">
        <template slot="brand">
          <div class="logo">
            <router-link :to="{name: 'dashboard'}">
              <img class="full" src="@/assets/logo.svg"/>
              <img class="favicon" src="@/assets/favicon.png"/>
            </router-link>
          </div>
        </template>
        <template slot="end">
            <b-navbar-item tag="div">
              <a href="#" @click.prevent="doLogout">{{ $t('users.logout') }}</a>
            </b-navbar-item>
        </template>
    </b-navbar>

    <div class="wrapper" v-if="$root.isLoaded">
      <section class="sidebar">
        <b-sidebar
          position="static"
          mobile="reduce"
          :fullheight="true"
          :open="true"
          :can-cancel="false"
        >
          <div>
            <b-menu :accordion="false">
              <b-menu-list>
                <b-menu-item :to="{name: 'dashboard'}" tag="router-link"
                  :active="activeItem.dashboard"
                  icon="view-dashboard-variant-outline" :label="$t('menu.dashboard')">
                </b-menu-item><!-- dashboard -->

                <b-menu-item :expanded="activeGroup.lists"
                  :active="activeGroup.lists" data-cy="lists"
                  v-on:update:active="(state) => toggleGroup('lists', state)"
                  icon="format-list-bulleted-square" :label="$t('globals.terms.lists')">
                  <b-menu-item :to="{name: 'lists'}" tag="router-link"
                    :active="activeItem.lists" data-cy="all-lists"
                    icon="format-list-bulleted-square" :label="$t('menu.allLists')"></b-menu-item>

                  <b-menu-item :to="{name: 'forms'}" tag="router-link"
                    :active="activeItem.forms" class="forms"
                    icon="newspaper-variant-outline" :label="$t('menu.forms')"></b-menu-item>
                </b-menu-item><!-- lists -->

                <b-menu-item :expanded="activeGroup.subscribers"
                  :active="activeGroup.subscribers" data-cy="subscribers"
                  v-on:update:active="(state) => toggleGroup('subscribers', state)"
                  icon="account-multiple" :label="$t('globals.terms.subscribers')">
                  <b-menu-item :to="{name: 'subscribers'}" tag="router-link"
                    :active="activeItem.subscribers" data-cy="all-subscribers"
                    icon="account-multiple" :label="$t('menu.allSubscribers')"></b-menu-item>

                  <b-menu-item :to="{name: 'import'}" tag="router-link"
                    :active="activeItem.import" data-cy="import"
                    icon="file-upload-outline" :label="$t('menu.import')"></b-menu-item>

                  <b-menu-item :to="{name: 'bounces'}" tag="router-link"
                    :active="activeItem.bounces" data-cy="bounces"
                    icon="email-bounce" :label="$t('globals.terms.bounces')"></b-menu-item>
                </b-menu-item><!-- subscribers -->

                <b-menu-item :expanded="activeGroup.campaigns"
                  :active="activeGroup.campaigns" data-cy="campaigns"
                  v-on:update:active="(state) => toggleGroup('campaigns', state)"
                  icon="rocket-launch-outline" :label="$t('globals.terms.campaigns')">
                  <b-menu-item :to="{name: 'campaigns'}" tag="router-link"
                    :active="activeItem.campaigns" data-cy="all-campaigns"
                    icon="rocket-launch-outline" :label="$t('menu.allCampaigns')"></b-menu-item>

                  <b-menu-item :to="{name: 'campaign', params: {id: 'new'}}" tag="router-link"
                    :active="activeItem.campaign" data-cy="new-campaign"
                    icon="plus" :label="$t('menu.newCampaign')"></b-menu-item>

                  <b-menu-item :to="{name: 'media'}" tag="router-link"
                    :active="activeItem.media" data-cy="media"
                    icon="image-outline" :label="$t('menu.media')"></b-menu-item>

                  <b-menu-item :to="{name: 'templates'}" tag="router-link"
                    :active="activeItem.templates" data-cy="templates"
                    icon="file-image-outline" :label="$t('globals.terms.templates')"></b-menu-item>

                  <b-menu-item :to="{name: 'campaignAnalytics'}" tag="router-link"
                    :active="activeItem.campaignAnalytics" data-cy="analytics"
                    icon="chart-bar" :label="$t('globals.terms.analytics')"></b-menu-item>
                </b-menu-item><!-- campaigns -->

                <b-menu-item :expanded="activeGroup.settings"
                  :active="activeGroup.settings" data-cy="settings"
                  v-on:update:active="(state) => toggleGroup('settings', state)"
                  icon="cog-outline" :label="$t('menu.settings')">

                  <b-menu-item :to="{name: 'settings'}" tag="router-link"
                    :active="activeItem.settings" data-cy="all-settings"
                    icon="cog-outline" :label="$t('menu.settings')"></b-menu-item>

                  <b-menu-item :to="{name: 'logs'}" tag="router-link"
                    :active="activeItem.logs" data-cy="logs"
                    icon="newspaper-variant-outline" :label="$t('menu.logs')"></b-menu-item>
                </b-menu-item><!-- settings -->
              </b-menu-list>
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
            <a :href="serverConfig.update.url" target="_blank">View</a>
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

export default Vue.extend({
  name: 'App',

  data() {
    return {
      activeItem: {},
      activeGroup: {},
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

    doLogout() {
      const http = new XMLHttpRequest();

      const u = uris.root.substr(-1) === '/' ? uris.root : `${uris.root}/`;
      http.open('get', `${u}api/logout`, false, 'logout_non_user', 'logout_non_user');
      http.onload = () => {
        document.location.href = uris.root;
      };
      http.onerror = () => {
        document.location.href = uris.root;
      };
      http.send();
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
  },

  computed: {
    ...mapState(['serverConfig']),

    version() {
      return process.env.VUE_APP_VERSION;
    },
  },

  mounted() {
    // Lists is required across different views. On app load, fetch the lists
    // and have them in the store.
    this.$api.getLists({ minimal: true });
  },
});
</script>

<style lang="scss">
  @import "assets/style.scss";
  @import "assets/icons/fontello.css";
</style>
