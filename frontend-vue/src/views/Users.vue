<template>
  <section class="users">
    <header class="row page-header">
      <div class="col-8">
        <h1>
          {{ $t('globals.terms.users') }}
          <span v-if="!isNaN(users.length)">({{ users.length }})</span>
        </h1>
      </div>
      <div class="col-4 col-end align-right">
        <b-field v-if="$can('users:manage')">
          <button type="button" data-variant="primary" class="btn-new" @click="showNewForm" data-cy="btn-new">
            <b-icon icon="plus" />
            {{ $t('globals.buttons.new') }}
          </button>
        </b-field>
      </div>
    </header>

    <div class="card page-content">
      <b-table :data="users" :loading="loading.users" checkable :checked-rows.sync="checked"
        default-sort="createdAt" backend-sorting @sort="onSort" @check-all="onTableCheck" @check="onTableCheck">
        <template #top-left>
          <div class="row">
            <div class="col-6">
              <form @submit.prevent="getUsers">
                <fieldset class="group">
                  <input aria-label="Search" v-model="queryParams.query" name="query" ref="query" data-cy="query"
                    placeholder="Search">
                  <button type="submit" data-variant="primary" data-cy="btn-query" aria-label="Search">
                    <b-icon icon="magnify" />
                  </button>
                </fieldset>
              </form>
            </div>
          </div>
        </template>

        <b-table-column v-slot="props" field="username" :label="$t('users.username')" header-class="cy-username"
          sortable :td-attrs="$utils.tdID">
          <a :href="`/users/${props.row.id}`" @click.prevent="showEditForm(props.row)"
            :class="{ 'text-light': props.row.status === 'disabled' }">
            {{ props.row.username }}
          </a>
          <b-tag v-if="props.row.status === 'disabled'" type="disabled">
            {{ $t(`users.status.${props.row.status}`) }}
          </b-tag>
          <b-tag v-if="props.row.type === 'api'" type="api">
            <b-icon icon="code" />
            {{ $t(`users.type.${props.row.type}`) }}
          </b-tag>
          <div class="text-light text-7 mt-2">
            {{ props.row.name }}
          </div>
        </b-table-column>

        <b-table-column v-slot="props" field="status" :label="$tc('users.role')" header-class="cy-status" sortable
          :td-attrs="$utils.tdID">
          <router-link :to="{ name: 'userRoles' }">
            <b-tag v-if="props.row.userRole" :type="props.row.userRole.id === 1 ? 'enabled' : 'primary'">
              <b-icon icon="account-outline" />
              {{ props.row.userRole.name }}
            </b-tag>
          </router-link>
          <router-link :to="{ name: 'listRoles' }">
            <span v-if="props.row.listRole" class="badge secondary">
              <b-icon icon="newspaper-variant-outline" />
              {{ props.row.listRole.name }}
            </span>
          </router-link>
        </b-table-column>

        <b-table-column v-slot="props" field="name" :label="$t('subscribers.email')" header-class="cy-name" sortable
          :td-attrs="$utils.tdID">
          <div>
            <a v-if="props.row.email" :href="`/users/${props.row.id}`" @click.prevent="showEditForm(props.row)"
              :class="{ 'text-light': props.row.status === 'disabled' }">
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

              <b-icon icon="pencil-outline" />

            </a>

            <a v-if="$can('users:manage')" href="#" @click.prevent="deleteUser(props.row)" data-cy="btn-delete"
              :aria-label="$t('globals.buttons.delete')">

              <b-icon icon="trash-can-outline" />

            </a>
          </div>
        </b-table-column>

        <template #empty v-if="!loading.users">
          <empty-placeholder />
        </template>
      </b-table>

      <!-- Add / edit form modal -->
      <b-modal :active.sync="isFormVisible" :width="600" @close="onFormClose">
        <user-form :data="curItem" :is-editing="isEditing" @finished="formFinished" />
      </b-modal>
    </div>
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
      if (this.checked.length === 0) {
        this.checked = [];
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
