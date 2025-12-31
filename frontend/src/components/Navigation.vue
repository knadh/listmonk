<template>
  <b-menu-list>
    <b-menu-item :to="{ name: 'dashboard' }" tag="router-link" :active="activeItem.dashboard"
      icon="view-dashboard-variant-outline" :label="$t('menu.dashboard')" /><!-- dashboard -->

    <b-menu-item :expanded="activeGroup.lists" :active="activeGroup.lists" data-cy="lists"
      @update:active="(state) => toggleGroup('lists', state)" icon="format-list-bulleted-square"
      :label="$t('globals.terms.lists')">
      <b-menu-item :to="{ name: 'lists' }" tag="router-link" :active="activeItem.lists" data-cy="all-lists"
        icon="format-list-bulleted-square" :label="$t('menu.allLists')" />
      <b-menu-item :to="{ name: 'forms' }" tag="router-link" :active="activeItem.forms" class="forms"
        icon="newspaper-variant-outline" :label="$t('menu.forms')" />
    </b-menu-item><!-- lists -->

    <b-menu-item v-if="$can('subscribers:*')" :expanded="activeGroup.subscribers" :active="activeGroup.subscribers"
      data-cy="subscribers" @update:active="(state) => toggleGroup('subscribers', state)" icon="account-multiple"
      :label="$t('globals.terms.subscribers')">
      <b-menu-item v-if="$can('subscribers:get_all', 'subscribers:get')" :to="{ name: 'subscribers' }" tag="router-link"
        :active="activeItem.subscribers" data-cy="all-subscribers" icon="account-multiple"
        :label="$t('menu.allSubscribers')" />
      <b-menu-item v-if="$can('subscribers:import')" :to="{ name: 'import' }" tag="router-link"
        :active="activeItem.import" data-cy="import" icon="file-upload-outline" :label="$t('menu.import')" />
      <b-menu-item v-if="$can('bounces:get')" :to="{ name: 'bounces' }" tag="router-link" :active="activeItem.bounces"
        data-cy="bounces" icon="email-bounce" :label="$t('globals.terms.bounces')" />
    </b-menu-item><!-- subscribers -->

    <b-menu-item v-if="$can('campaigns:*')" :expanded="activeGroup.campaigns" :active="activeGroup.campaigns"
      data-cy="campaigns" @update:active="(state) => toggleGroup('campaigns', state)" icon="rocket-launch-outline"
      :label="$t('globals.terms.campaigns')">
      <b-menu-item v-if="$can('campaigns:get')" :to="{ name: 'campaigns' }" tag="router-link"
        :active="activeItem.campaigns" data-cy="all-campaigns" icon="rocket-launch-outline"
        :label="$t('menu.allCampaigns')" />
      <b-menu-item v-if="$can('campaigns:manage')" :to="{ name: 'campaign', params: { id: 'new' } }" tag="router-link"
        :active="activeItem.campaign" data-cy="new-campaign" icon="plus" :label="$t('menu.newCampaign')" />
      <b-menu-item v-if="$can('media:*')" :to="{ name: 'media' }" tag="router-link" :active="activeItem.media"
        data-cy="media" icon="image-outline" :label="$t('menu.media')" />
      <b-menu-item v-if="$can('templates:get')" :to="{ name: 'templates' }" tag="router-link"
        :active="activeItem.templates" data-cy="templates" icon="file-image-outline"
        :label="$t('globals.terms.templates')" />
      <b-menu-item v-if="$can('campaigns:get_analytics')" :to="{ name: 'campaignAnalytics' }" tag="router-link"
        :active="activeItem.campaignAnalytics" data-cy="analytics" icon="chart-bar"
        :label="$t('globals.terms.analytics')" />
    </b-menu-item><!-- campaigns -->

    <b-menu-item v-if="$can('users:*', 'roles:*')" :expanded="activeGroup.users" :active="activeGroup.users"
      data-cy="users" @update:active="(state) => toggleGroup('users', state)" icon="account-multiple"
      :label="$t('globals.terms.users')">
      <b-menu-item v-if="$can('users:get')" :to="{ name: 'users' }" tag="router-link" :active="activeItem.users"
        data-cy="users" icon="account-multiple" :label="$t('globals.terms.users')" />
      <b-menu-item v-if="$can('roles:get')" :to="{ name: 'userRoles' }" tag="router-link" :active="activeItem.userRoles"
        data-cy="userRoles" icon="newspaper-variant-outline" :label="$t('users.userRoles')" />
      <b-menu-item v-if="$can('roles:get')" :to="{ name: 'listRoles' }" tag="router-link" :active="activeItem.listRoles"
        data-cy="listRoles" icon="format-list-bulleted-square" :label="$t('users.listRoles')" />
    </b-menu-item><!-- users -->

    <b-menu-item v-if="$can('settings:*')" :expanded="activeGroup.settings" :active="activeGroup.settings"
      data-cy="settings" @update:active="(state) => toggleGroup('settings', state)" icon="cog-outline"
      :label="$t('menu.settings')">
      <b-menu-item v-if="$can('settings:get')" :to="{ name: 'settings' }" tag="router-link"
        :active="activeItem.settings" data-cy="all-settings" icon="cog-outline" :label="$t('menu.settings')" />
      <b-menu-item v-if="$can('settings:maintain')" :to="{ name: 'maintenance' }" tag="router-link"
        :active="activeItem.maintenance" data-cy="maintenance" icon="wrench-outline" :label="$t('menu.maintenance')" />
      <b-menu-item v-if="$can('settings:get')" :to="{ name: 'logs' }" tag="router-link" :active="activeItem.logs"
        data-cy="logs" icon="format-list-bulleted-square" :label="$t('menu.logs')" />
    </b-menu-item><!-- settings -->
  </b-menu-list>
</template>

<script>
import { mapState } from 'vuex';

export default {
  name: 'Navigation',

  props: {
    activeItem: { type: Object, default: () => { } },
    activeGroup: { type: Object, default: () => { } },
    isMobile: Boolean,
  },

  methods: {
    toggleGroup(group, state) {
      this.$emit('toggleGroup', group, state);
    },

    doLogout() {
      this.$emit('doLogout');
    },
  },

  computed: {
    ...mapState(['profile']),
  },

  mounted() {
    // A hack to close the open accordion burger menu items on click.
    // Buefy does not have a way to do this.
    if (this.isMobile) {
      document.querySelectorAll('.navbar li a[href]').forEach((e) => {
        e.onclick = () => {
          document.querySelector('.navbar-burger').click();
        };
      });
    }
  },
};

</script>
