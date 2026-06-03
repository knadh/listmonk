<template>
  <nav>
    <ul>
      <li>
        <router-link :to="{ name: 'dashboard' }">
          <b-icon icon="view-dashboard-variant-outline" />
          {{ $t('menu.dashboard') }}
        </router-link>
      </li>

      <li>
        <details name="sidebar-nav" data-cy="lists">
          <summary>
            <b-icon icon="format-list-bulleted-square" />
            {{ $t('globals.terms.lists') }}
          </summary>
          <ul>
            <li>
              <router-link :to="{ name: 'lists' }" data-cy="all-lists">
                {{ $t('menu.allLists') }}
              </router-link>
            </li>
            <li>
              <router-link :to="{ name: 'forms' }" data-cy="forms">
                {{ $t('menu.forms') }}
              </router-link>
            </li>
          </ul>
        </details>
      </li>

      <li v-if="$can('subscribers:*')">
        <details name="sidebar-nav" data-cy="subscribers">
          <summary>
            <b-icon icon="account-multiple" />
            {{ $t('globals.terms.subscribers') }}
          </summary>
          <ul>
            <li v-if="$can('subscribers:get_all', 'subscribers:get')">
              <router-link :to="{ name: 'subscribers' }" data-cy="all-subscribers">
                {{ $t('menu.allSubscribers') }}
              </router-link>
            </li>
            <li v-if="$can('subscribers:import')">
              <router-link :to="{ name: 'import' }" data-cy="import">
                {{ $t('menu.import') }}
              </router-link>
            </li>
            <li v-if="$can('bounces:get')">
              <router-link :to="{ name: 'bounces' }" data-cy="bounces">
                {{ $t('globals.terms.bounces') }}
              </router-link>
            </li>
          </ul>
        </details>
      </li>

      <li v-if="$can('campaigns:*')">
        <details name="sidebar-nav" data-cy="campaigns">
          <summary>
            <b-icon icon="rocket-launch-outline" />
            {{ $t('globals.terms.campaigns') }}
          </summary>
          <ul>
            <li v-if="$can('campaigns:get')">
              <router-link :to="{ name: 'campaigns' }" data-cy="all-campaigns">
                {{ $t('menu.allCampaigns') }}
              </router-link>
            </li>
            <li v-if="$can('campaigns:manage')">
              <router-link :to="{ name: 'campaign', params: { id: 'new' } }" data-cy="new-campaign">
                {{ $t('menu.newCampaign') }}
              </router-link>
            </li>
            <li v-if="$can('media:*')">
              <router-link :to="{ name: 'media' }" data-cy="media">
                {{ $t('menu.media') }}
              </router-link>
            </li>
            <li v-if="$can('templates:get')">
              <router-link :to="{ name: 'templates' }" data-cy="templates">
                {{ $t('globals.terms.templates') }}
              </router-link>
            </li>
            <li v-if="$can('campaigns:get_analytics')">
              <router-link :to="{ name: 'campaignAnalytics' }" data-cy="analytics">
                {{ $t('globals.terms.analytics') }}
              </router-link>
            </li>
          </ul>
        </details>
      </li>

      <li v-if="$can('users:*', 'roles:*')">
        <details name="sidebar-nav" data-cy="users">
          <summary>
            <b-icon icon="account-multiple" />
            {{ $t('globals.terms.users') }}
          </summary>
          <ul>
            <li v-if="$can('users:get')">
              <router-link :to="{ name: 'users' }" data-cy="users">
                {{ $t('globals.terms.users') }}
              </router-link>
            </li>
            <li v-if="$can('roles:get')">
              <router-link :to="{ name: 'userRoles' }" data-cy="userRoles">
                {{ $t('users.userRoles') }}
              </router-link>
            </li>
            <li v-if="$can('roles:get')">
              <router-link :to="{ name: 'listRoles' }" data-cy="listRoles">
                {{ $t('users.listRoles') }}
              </router-link>
            </li>
          </ul>
        </details>
      </li>

      <li v-if="$can('settings:*')">
        <details name="sidebar-nav" data-cy="settings">
          <summary>
            <b-icon icon="cog-outline" />
            {{ $t('menu.settings') }}
          </summary>
          <ul>
            <li v-if="$can('settings:get')">
              <router-link :to="{ name: 'settings' }" data-cy="all-settings">
                {{ $t('menu.settings') }}
              </router-link>
            </li>
            <li v-if="$can('settings:maintain')">
              <router-link :to="{ name: 'maintenance' }" data-cy="maintenance">
                {{ $t('menu.maintenance') }}
              </router-link>
            </li>
            <li v-if="$can('settings:get')">
              <router-link :to="{ name: 'logs' }" data-cy="logs">
                {{ $t('menu.logs') }}
              </router-link>
            </li>
          </ul>
        </details>
      </li>
    </ul>
  </nav>
</template>
