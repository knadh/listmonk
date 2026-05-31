<template>
  <section class="roles">
    <header class="row page-header">
      <div class="col-8">
        <h1>
          {{ $t(isUser ? 'users.userRoles' : 'users.listRoles') }}
          <span v-if="!isNaN(roles.length)">({{ roles.length }})</span>
        </h1>
      </div>
      <div class="col-4 col-end align-right">
        <oat-field v-if="$can('users:manage')">
          <button type="button" data-variant="primary" class="btn-new" @click="showNewForm('user')" data-cy="btn-new">
            {{ $t('globals.buttons.new') }}
          </button>
        </oat-field>
      </div>
    </header>
    <div class="card page-content">
      <oat-data-table :data="roles" :loading="isLoading()">
        <oat-table-column v-slot="props" field="role" :label="$tc('users.role')" sortable>
          <a href="#" @click.prevent="showEditForm(props.row, 'user')">
            <oat-badge v-if="props.row.id === 1" type="enabled">
              {{ props.row.name }}
            </oat-badge>
            <template v-else>{{ props.row.name }}</template>
          </a>
        </oat-table-column>

        <oat-table-column v-slot="props" field="created_at" :label="$t('globals.fields.createdAt')"
          header-class="cy-created_at" sortable>
          {{ $utils.niceDate(props.row.createdAt) }}
        </oat-table-column>

        <oat-table-column v-slot="props" field="updated_at" :label="$t('globals.fields.updatedAt')"
          header-class="cy-updated_at" sortable>
          {{ $utils.niceDate(props.row.updatedAt) }}
        </oat-table-column>

        <oat-table-column v-slot="props" cell-class="actions align-right">
          <template v-if="$can('roles:manage')">
            <a href="#" @click.prevent="$utils.prompt($t('globals.buttons.clone'),
              {
                placeholder: $t('globals.fields.name'),
                value: $t('campaigns.copyOf', { name: props.row.name }),
              },
              (name) => onCloneRole(name, props.row))" data-cy="btn-clone" :aria-label="$t('globals.buttons.clone')">

              <oat-icon icon="file-multiple-outline" />

            </a>

            <template v-if="props.row.id !== 1">
              <a href="#" @click.prevent="showEditForm(props.row, 'user')" data-cy="btn-edit"
                :aria-label="$t('globals.buttons.edit')">

                <oat-icon icon="pencil-outline" />

              </a>

              <a href="#" @click.prevent="onDeleteRole(props.row)" data-cy="btn-delete"
                :aria-label="$t('globals.buttons.delete')">

                <oat-icon icon="trash-can-outline" />

              </a>
            </template>
          </template>
        </oat-table-column>

        <template #empty v-if="!isLoading()">
          <empty-placeholder />
        </template>
      </oat-data-table>

      <!-- Add / edit form modal -->
      <oat-modal :active.sync="isFormVisible" :width="700" @close="onFormClose">
        <role-form :data="curItem" :type="curType" :is-editing="isEditing" @finished="formFinished" />
      </oat-modal>
    </div>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import EmptyPlaceholder from '../components/EmptyPlaceholder.vue';
import RoleForm from './RoleForm.vue';

export default Vue.extend({
  components: {
    EmptyPlaceholder,
    RoleForm,
  },

  data() {
    return {
      curItem: null,
      curType: null,
      isEditing: false,
      isFormVisible: false,
    };
  },

  methods: {
    isLoading() {
      return this.curType === 'user' ? this.loading.userRoles : this.loading.listRoles;
    },

    fetchRoles() {
      if (this.isUser) {
        this.$api.getUserRoles();
      } else {
        this.$api.getListRoles();
      }
    },

    // Show the edit form.
    showEditForm(item) {
      this.curItem = item;
      this.curType = this.isUser ? 'user' : 'list';
      this.isFormVisible = true;
      this.isEditing = true;
    },

    // Show the new form.
    showNewForm() {
      this.isEditing = false;
      this.isFormVisible = true;
    },

    formFinished() {
      this.fetchRoles();
    },

    onFormClose() {
      if (this.$route.params.id) {
        this.$router.push({ name: 'users' });
      }
    },

    onCloneRole(name, item) {
      const form = { name };
      let fn;
      if (this.isUser) {
        fn = this.$api.createUserRole;
        form.permissions = item.permissions;
      } else {
        fn = this.$api.createListRole;
        form.lists = item.lists;
      }

      fn(form).then(() => {
        this.fetchRoles();
        this.$utils.toast(this.$t('globals.messages.created', { name }));
      });
    },

    onDeleteRole(item) {
      this.$utils.confirm(
        this.$t('globals.messages.confirm'),
        () => {
          this.$api.deleteRole(item.id).then(() => {
            this.fetchRoles();

            this.$utils.toast(this.$t('globals.messages.deleted', { name: item.name }));
          });
        },
      );
    },

  },

  computed: {
    ...mapState(['loading', 'userRoles', 'listRoles']),

    isUser() {
      return this.curType === 'user';
    },

    isList() {
      return this.curType === 'list';
    },

    roles() {
      return this.isUser ? this.userRoles : this.listRoles;
    },
  },

  mounted() {
    this.curType = this.$route.name === 'userRoles' ? 'user' : 'list';
    this.fetchRoles();
  },
});
</script>
