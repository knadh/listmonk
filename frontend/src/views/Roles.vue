<template>
  <section class="roles">
    <header class="columns page-header">
      <div class="column is-10">
        <h1 class="title is-4">
          {{ $t('users.roles') }}
          <span v-if="!isNaN(roles.length)">({{ roles.length }})</span>
        </h1>
      </div>
      <div class="column has-text-right">
        <b-field expanded>
          <b-button expanded type="is-primary" icon-left="plus" class="btn-new" @click="showNewForm" data-cy="btn-new">
            {{ $t('globals.buttons.new') }}
          </b-button>
        </b-field>
      </div>
    </header>

    <b-table :data="roles" :loading="loading.roles" hoverable>
      <b-table-column v-slot="props" field="role" :label="$tc('users.role')" sortable>
        <a href="#" @click.prevent="showEditForm(props.row)">
          <b-tag v-if="props.row.id === 1" class="enabled">
            {{ props.row.name }}
          </b-tag>
          <template v-else>{{ props.row.name }}</template>
        </a>
      </b-table-column>

      <b-table-column v-slot="props" field="created_at" :label="$t('globals.fields.createdAt')"
        header-class="cy-created_at" sortable>
        {{ $utils.niceDate(props.row.createdAt) }}
      </b-table-column>

      <b-table-column v-slot="props" field="updated_at" :label="$t('globals.fields.updatedAt')"
        header-class="cy-updated_at" sortable>
        {{ $utils.niceDate(props.row.updatedAt) }}
      </b-table-column>

      <b-table-column v-slot="props" cell-class="actions" align="right">
        <a href="#" @click.prevent="$utils.prompt($t('globals.buttons.clone'),
          {
            placeholder: $t('globals.fields.name'),
            value: $t('campaigns.copyOf', { name: props.row.name }),
          },
          (name) => onCloneRole(name, props.row))" data-cy="btn-clone" :aria-label="$t('globals.buttons.clone')">
          <b-tooltip :label="$t('globals.buttons.clone')" type="is-dark">
            <b-icon icon="file-multiple-outline" size="is-small" />
          </b-tooltip>
        </a>

        <template v-if="props.row.id !== 1">
          <a href="#" @click.prevent="showEditForm(props.row)" data-cy="btn-edit"
            :aria-label="$t('globals.buttons.edit')">
            <b-tooltip :label="$t('globals.buttons.edit')" type="is-dark">
              <b-icon icon="pencil-outline" size="is-small" />
            </b-tooltip>
          </a>

          <a href="#" @click.prevent="onDeleteRole(props.row)" data-cy="btn-delete"
            :aria-label="$t('globals.buttons.delete')">
            <b-tooltip :label="$t('globals.buttons.delete')" type="is-dark">
              <b-icon icon="trash-can-outline" size="is-small" />
            </b-tooltip>
          </a>
        </template>
      </b-table-column>

      <template #empty v-if="!loading.users">
        <empty-placeholder />
      </template>
    </b-table>

    <!-- Add / edit form modal -->
    <b-modal scroll="keep" :aria-modal="true" :active.sync="isFormVisible" :width="700" @close="onFormClose">
      <role-form :data="curItem" :is-editing="isEditing" @finished="formFinished" />
    </b-modal>
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
      isEditing: false,
      isFormVisible: false,
    };
  },

  methods: {
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
      this.$api.getRoles();
    },

    onFormClose() {
      if (this.$route.params.id) {
        this.$router.push({ name: 'users' });
      }
    },

    onCloneRole(name, item) {
      this.$api.createRole({ name, permissions: item.permissions }).then(() => {
        this.$api.getRoles();
      });
    },

    onDeleteRole(item) {
      this.$utils.confirm(
        this.$t('globals.messages.confirm'),
        () => {
          this.$api.deleteRole(item.id).then(() => {
            this.$api.getRoles();

            this.$utils.toast(this.$t('globals.messages.deleted', { name: item.name }));
          });
        },
      );
    },

  },

  computed: {
    ...mapState(['loading', 'roles']),
  },

  mounted() {
    this.$api.getRoles();
  },
});
</script>
