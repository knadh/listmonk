<template>
  <nav>
    <ul>
      <li>
        <router-link :to="{ name: 'dashboard' }" :aria-current="activeItem.dashboard ? 'page' : null">
          <oat-icon icon="view-dashboard-variant-outline" />
          {{ $t('menu.dashboard') }}
        </router-link>
      </li>

      <li>
        <details :open="activeGroup.lists" data-cy="lists" @toggle="onToggle($event, 'lists')">
          <summary>
            <oat-icon icon="format-list-bulleted-square" />
            {{ $t('globals.terms.lists') }}
          </summary>
          <ul>
            <li>
              <router-link :to="{ name: 'lists' }" :aria-current="activeItem.lists ? 'page' : null" data-cy="all-lists">
                {{ $t('menu.allLists') }}
              </router-link>
            </li>
            <li>
              <router-link :to="{ name: 'forms' }" :aria-current="activeItem.forms ? 'page' : null" data-cy="forms">
                {{ $t('menu.forms') }}
              </router-link>
            </li>
          </ul>
        </details>
      </li>

      <li v-if="$can('subscribers:*')">
        <details :open="activeGroup.subscribers" data-cy="subscribers" @toggle="onToggle($event, 'subscribers')">
          <summary>
            <oat-icon icon="account-multiple" />
            {{ $t('globals.terms.subscribers') }}
          </summary>
          <ul>
            <li v-if="$can('subscribers:get_all', 'subscribers:get')">
              <router-link :to="{ name: 'subscribers' }" :aria-current="activeItem.subscribers ? 'page' : null"
                data-cy="all-subscribers">
                {{ $t('menu.allSubscribers') }}
              </router-link>
            </li>
            <li v-if="$can('subscribers:import')">
              <router-link :to="{ name: 'import' }" :aria-current="activeItem.import ? 'page' : null" data-cy="import">
                {{ $t('menu.import') }}
              </router-link>
            </li>
            <li v-if="$can('bounces:get')">
              <router-link :to="{ name: 'bounces' }" :aria-current="activeItem.bounces ? 'page' : null"
                data-cy="bounces">
                {{ $t('globals.terms.bounces') }}
              </router-link>
            </li>
          </ul>
        </details>
      </li>

      <li v-if="$can('campaigns:*')">
        <details :open="activeGroup.campaigns" data-cy="campaigns" @toggle="onToggle($event, 'campaigns')">
          <summary>
            <oat-icon icon="rocket-launch-outline" />
            {{ $t('globals.terms.campaigns') }}
          </summary>
          <ul>
            <li v-if="$can('campaigns:get')">
              <router-link :to="{ name: 'campaigns' }" :aria-current="activeItem.campaigns ? 'page' : null"
                data-cy="all-campaigns">
                {{ $t('menu.allCampaigns') }}
              </router-link>
            </li>
            <li v-if="$can('campaigns:manage')">
              <router-link :to="{ name: 'campaign', params: { id: 'new' } }"
                :aria-current="activeItem.campaign ? 'page' : null" data-cy="new-campaign">
                {{ $t('menu.newCampaign') }}
              </router-link>
            </li>
            <li v-if="$can('media:*')">
              <router-link :to="{ name: 'media' }" :aria-current="activeItem.media ? 'page' : null" data-cy="media">
                {{ $t('menu.media') }}
              </router-link>
            </li>
            <li v-if="$can('templates:get')">
              <router-link :to="{ name: 'templates' }" :aria-current="activeItem.templates ? 'page' : null"
                data-cy="templates">
                {{ $t('globals.terms.templates') }}
              </router-link>
            </li>
            <li v-if="$can('campaigns:get_analytics')">
              <router-link :to="{ name: 'campaignAnalytics' }"
                :aria-current="activeItem.campaignAnalytics ? 'page' : null" data-cy="analytics">
                {{ $t('globals.terms.analytics') }}
              </router-link>
            </li>
          </ul>
        </details>
      </li>

      <li v-if="$can('users:*', 'roles:*')">
        <details :open="activeGroup.users" data-cy="users" @toggle="onToggle($event, 'users')">
          <summary>
            <oat-icon icon="account-multiple" />
            {{ $t('globals.terms.users') }}
          </summary>
          <ul>
            <li v-if="$can('users:get')">
              <router-link :to="{ name: 'users' }" :aria-current="activeItem.users ? 'page' : null" data-cy="users">
                {{ $t('globals.terms.users') }}
              </router-link>
            </li>
            <li v-if="$can('roles:get')">
              <router-link :to="{ name: 'userRoles' }" :aria-current="activeItem.userRoles ? 'page' : null"
                data-cy="userRoles">
                {{ $t('users.userRoles') }}
              </router-link>
            </li>
            <li v-if="$can('roles:get')">
              <router-link :to="{ name: 'listRoles' }" :aria-current="activeItem.listRoles ? 'page' : null"
                data-cy="listRoles">
                {{ $t('users.listRoles') }}
              </router-link>
            </li>
          </ul>
        </details>
      </li>

      <li v-if="$can('settings:*')">
        <details :open="activeGroup.settings" data-cy="settings" @toggle="onToggle($event, 'settings')">
          <summary>
            <oat-icon icon="cog-outline" />
            {{ $t('menu.settings') }}
          </summary>
          <ul>
            <li v-if="$can('settings:get')">
              <router-link :to="{ name: 'settings' }" :aria-current="activeItem.settings ? 'page' : null"
                data-cy="all-settings">
                {{ $t('menu.settings') }}
              </router-link>
            </li>
            <li v-if="$can('settings:maintain')">
              <router-link :to="{ name: 'maintenance' }" :aria-current="activeItem.maintenance ? 'page' : null"
                data-cy="maintenance">
                {{ $t('menu.maintenance') }}
              </router-link>
            </li>
            <li v-if="$can('settings:get')">
              <router-link :to="{ name: 'logs' }" :aria-current="activeItem.logs ? 'page' : null" data-cy="logs">
                {{ $t('menu.logs') }}
              </router-link>
            </li>
          </ul>
        </details>
      </li>
    </ul>
  </nav>
</template>

<script>
export default {
  name: 'Navigation',
  props: {
    activeItem: { type: Object, default: () => { } },
    activeGroup: { type: Object, default: () => { } },
    isMobile: Boolean,
  },
  methods: {
    onToggle(e, group) {
      this.$emit('toggleGroup', group, e.target.open);
    },
  },
};
</script>
