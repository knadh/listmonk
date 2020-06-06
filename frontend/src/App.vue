<template>
  <div id="app">
    <section class="sidebar">
      <b-sidebar
        type="is-white"
        position="static"
        mobile="reduce"
        :fullheight="true"
        :open="true"
        :can-cancel="false"
      >
        <div>
          <div class="logo">
            <a href="/"><img class="full" src="@/assets/logo.svg"/></a>
            <img class="favicon" src="@/assets/favicon.png"/>
            <p class="is-size-7 has-text-grey version">{{ version }}</p>
          </div>
          <b-menu :accordion="false">
            <b-menu-list>
              <b-menu-item :to="{name: 'dashboard'}" tag="router-link"
                :active="activeItem.dashboard"
                icon="view-dashboard-variant-outline" label="Dashboard">
              </b-menu-item><!-- dashboard -->

              <b-menu-item :expanded="activeGroup.lists"
                icon="format-list-bulleted-square" label="Lists">
                <b-menu-item :to="{name: 'lists'}" tag="router-link"
                  :active="activeItem.lists"
                  icon="format-list-bulleted-square" label="All lists"></b-menu-item>

                <b-menu-item :to="{name: 'forms'}" tag="router-link"
                  :active="activeItem.forms"
                  icon="newspaper-variant-outline" label="Forms"></b-menu-item>
              </b-menu-item><!-- lists -->

              <b-menu-item :expanded="activeGroup.subscribers"
                icon="account-multiple" label="Subscribers">
                <b-menu-item :to="{name: 'subscribers'}" tag="router-link"
                  :active="activeItem.subscribers"
                  icon="account-multiple" label="All subscribers"></b-menu-item>

                <b-menu-item :to="{name: 'import'}" tag="router-link"
                  :active="activeItem.import"
                  icon="file-upload-outline" label="Import"></b-menu-item>
              </b-menu-item><!-- subscribers -->

              <b-menu-item :expanded="activeGroup.campaigns"
                  icon="rocket-launch-outline" label="Campaigns">
                <b-menu-item :to="{name: 'campaigns'}" tag="router-link"
                  :active="activeItem.campaigns"
                  icon="rocket-launch-outline" label="All campaigns"></b-menu-item>

                <b-menu-item :to="{name: 'campaign', params: {id: 'new'}}" tag="router-link"
                  :active="activeItem.campaign"
                  icon="plus" label="Create new"></b-menu-item>

                <b-menu-item :to="{name: 'media'}" tag="router-link"
                  :active="activeItem.media"
                  icon="image-outline" label="Media"></b-menu-item>

                <b-menu-item :to="{name: 'templates'}" tag="router-link"
                  :active="activeItem.templates"
                  icon="file-image-outline" label="Templates"></b-menu-item>
              </b-menu-item><!-- campaigns -->

              <!-- <b-menu-item :to="{name: 'settings'}" tag="router-link"
                :active="activeItem.settings"
                icon="cog-outline" label="Settings"></b-menu-item> -->
            </b-menu-list>
          </b-menu>
        </div>
      </b-sidebar>
    </section>
    <!-- sidebar-->

    <!-- body //-->
    <div class="main">
      <router-view :key="$route.fullPath" />
    </div>
  </div>
</template>

<script>
import Vue from 'vue';

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
      }
    },
  },

  mounted() {
    // Lists is required across different views. On app load, fetch the lists
    // and have them in the store.
    this.$api.getLists();
  },

  computed: {
    version() {
      return process.env.VUE_APP_VERSION;
    },
  },
});
</script>

<style lang="scss">
  @import "assets/style.scss";
  @import "assets/icons/fontello.css";
</style>
