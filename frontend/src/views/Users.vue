<template>
  <section class="users">
    <header class="columns page-header">
      <div class="column is-10">
        <h1 class="title is-4">
          {{ $t('globals.terms.users') }}
          <span v-if="!isNaN(users.length)">({{ users.length }})</span>
        </h1>
      </div>
      <div class="column has-text-right">
        <b-field v-if="$can('users:manage')" expanded>
          <b-button expanded type="is-primary" icon-left="plus" class="btn-new" @click="showNewForm" data-cy="btn-new">
            {{ $t('globals.buttons.new') }}
          </b-button>
        </b-field>
      </div>
    </header>

    <b-table :data="users" :loading="loading.users" hoverable checkable :checked-rows.sync="checked"
      default-sort="createdAt" backend-sorting @sort="onSort" @check-all="onTableCheck" @check="onTableCheck">
      <template #top-left>
        <div class="columns">
          <div class="column is-6">
            <form @submit.prevent="getUsers">
              <div>
                <b-field>
                  <b-input v-model="queryParams.query" name="query" expanded icon="magnify" ref="query"
                    data-cy="query" />
                  <p class="controls">
                    <b-button native-type="submit" type="is-primary" icon-left="magnify" data-cy="btn-query" />
                  </p>
                </b-field>
              </div>
            </form>
          </div>
        </div>
      </template>

      <b-table-column v-slot="props" field="username" :label="$t('users.username')" header-class="cy-username" sortable
        :td-attrs="$utils.tdID">
        <a :href="`/users/${props.row.id}`" @click.prevent="showEditForm(props.row)"
          :class="{ 'has-text-grey': props.row.status === 'disabled' }">
          {{ props.row.username }}
        </a>
        <b-tag v-if="props.row.status === 'disabled'">
          {{ $t(`users.status.${props.row.status}`) }}
        </b-tag>
        <b-tag v-if="props.row.type === 'api'" class="api">
          <b-icon icon="code" />
          {{ $t(`users.type.${props.row.type}`) }}
        </b-tag>
        <div class="has-text-grey is-size-7 mt-2">
          {{ props.row.name }}
        </div>
      </b-table-column>

      <b-table-column v-slot="props" field="status" :label="$tc('users.role')" header-class="cy-status" sortable
        :td-attrs="$utils.tdID">
        <router-link :to="{ name: 'userRoles' }">
          <b-tag :class="props.row.userRole.id === 1 ? 'enabled' : 'primary'">
            <b-icon icon="account-outline" />
            {{ props.row.userRole.name }}
          </b-tag>
        </router-link>
        <router-link :to="{ name: 'listRoles' }">
          <b-tag v-if="props.row.listRole">
            <b-icon icon="newspaper-variant-outline" />
            {{ props.row.listRole.name }}
          </b-tag>
        </router-link>
      </b-table-column>

      <b-table-column v-slot="props" field="name" :label="$t('subscribers.email')" header-class="cy-name" sortable
        :td-attrs="$utils.tdID">
        <div>
          <a v-if="props.row.email" :href="`/users/${props.row.id}`" @click.prevent="showEditForm(props.row)"
            :class="{ 'has-text-grey': props.row.status === 'disabled' }">
            {{ props.row.email }}
          </a>
          <template v-else>
            —
          </template>
        </div>
      </b-table-column>

      <b-table-column v-slot="props" field="created_at" :label="$t('globals.fields.createdAt')"
        header-class="cy-created_at" sortable>
        {{ $utils.niceDate(props.row.createdAt) }}
      </b-table-column>

      <b-table-column v-slot="props" field="updated_at" :label="$t('globals.fields.updatedAt')"
        header-class="cy-updated_at" sortable>
        {{ $utils.niceDate(props.row.updatedAt) }}
      </b-table-column>

      <b-table-column v-slot="props" field="last_login" :label="$t('users.lastLogin')" header-class="cy-updated_at"
        sortable>
        {{ props.row.loggedinAt ? $utils.niceDate(props.row.loggedinAt, true) : '—' }}
      </b-table-column>

      <b-table-column v-slot="props" cell-class="actions" align="right">
        <div>
          <a v-if="$can('users:manage')" href="#" @click.prevent="showEditForm(props.row)" data-cy="btn-edit"
            :aria-label="$t('globals.buttons.edit')">
            <b-tooltip :label="$t('globals.buttons.edit')" type="is-dark">
              <b-icon icon="pencil-outline" size="is-small" />
            </b-tooltip>
          </a>

          <a v-if="$can('users:manage')" href="#" @click.prevent="deleteUser(props.row)" data-cy="btn-delete"
            :aria-label="$t('globals.buttons.delete')">
            <b-tooltip :label="$t('globals.buttons.delete')" type="is-dark">
              <b-icon icon="trash-can-outline" size="is-small" />
            </b-tooltip>
          </a>
        </div>
      </b-table-column>

      <template #empty v-if="!loading.users">
        <empty-placeholder />
      </template>
    </b-table>

    <!-- Add / edit form modal -->
    <b-modal scroll="keep" :aria-modal="true" :active.sync="isFormVisible" :width="600" @close="onFormClose">
      <user-form :data="curItem" :is-editing="isEditing" @finished="formFinished" />
    </b-modal>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import EmptyPlaceholder from '../components/EmptyPlaceholder.vue';

import UserForm from './UserForm.vue';

export default Vue.extend({
  components: {
    EmptyPlaceholder,
    UserForm,
  },

  data() {
    return {
      curItem: null,
      isEditing: false,
      isFormVisible: false,
      users: [],
      checked: [],
      queryParams: {
        page: 1,
        query: '',
        orderBy: 'id',
        order: 'asc',
      },
    };
  },

  methods: {
    onSort(field, direction) {
      this.queryParams.orderBy = field;
      this.queryParams.order = direction;
      this.getUsers();
    },

    onTableCheck() {
      // Disable bulk.all selection if there are no rows checked in the table.
      if (this.bulk.checked.length !== this.subscribers.total) {
        this.bulk.all = false;
      }
    },

    // Show the edit form.
    showEditForm(item) {
      this.curItem = item;
      this.isFormVisible = true;
      this.isEditing = true;
    },

    // Show the new form.
    showNewForm() {
      this.curItem = {};
      this.isFormVisible = true;
      this.isEditing = false;
    },

    formFinished() {
      this.getUsers();
    },

    onFormClose() {
      if (this.$route.params.id) {
        this.$router.push({ name: 'users' });
      }
    },

    getUsers() {
      this.$api.queryUsers({
        query: this.queryParams.query.replace(/[^\p{L}\p{N}\s]/gu, ' '),
        order_by: this.queryParams.orderBy,
        order: this.queryParams.order,
      }).then((resp) => {
        this.users = resp;
      });
    },

    deleteUser(item) {
      this.$utils.confirm(
        this.$t('globals.messages.confirm'),
        () => {
          this.$api.deleteUser(item.id).then(() => {
            this.getUsers();

            this.$utils.toast(this.$t('globals.messages.deleted', { name: item.name }));
          });
        },
      );
    },
  },

  computed: {
    ...mapState(['loading', 'settings']),
  },

  created() {
    this.$root.$on('page.refresh', this.getUsers);
  },

  destroyed() {
    this.$root.$off('page.refresh', this.getUsers);
  },

  mounted() {
    if (this.$route.params.id) {
      this.$api.getUser(parseInt(this.$route.params.id, 10)).then((data) => {
        this.showEditForm(data);
      });
    } else {
      this.getUsers();
    }
  },
});
</script>
