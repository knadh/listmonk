<template>
  <form @submit.prevent="onSubmit">
    <div class="modal-card content" style="width: auto">
      <header class="modal-card-head">
        <p v-if="isEditing" class="has-text-grey-light is-size-7">
          {{ $t('globals.fields.id') }}: <copy-text :text="`${data.id}`" />
        </p>
        <h4 v-if="isEditing">
          {{ data.name }}
        </h4>
        <h4 v-else>
          {{ type === 'user' ? $t('users.newUserRole') : $t('users.newListRole') }}
        </h4>
      </header>

      <section expanded class="modal-card-body">
        <b-field :label="$t('globals.fields.name')" label-position="on-border">
          <b-input autofocus :disabled="disabled" :maxlength="200" v-model="form.name" name="name" ref="focus"
            required />
        </b-field>

        <div v-if="type === 'list'" class="box">
          <h5>{{ $t('users.listPerms') }}</h5>
          <div class="mb-5">
            <div class="columns">
              <div class="column is-9">
                <b-select :placeholder="$tc('globals.terms.list')" v-model="form.curList" name="list"
                  :disabled="disabled || filteredLists.length < 1" expanded class="mb-3">
                  <template v-for="l in filteredLists">
                    <option :value="l.id" :key="l.id">
                      {{ l.name }}
                    </option>
                  </template>
                </b-select>
              </div>
              <div class="column">
                <b-button @click="onAddListPerm" :disabled="!form.curList" class="is-primary" expanded>
                  {{ $t('globals.buttons.add') }}
                </b-button>
              </div>
            </div>
            <span
              v-if="form.lists.length > 0 && (form.permissions['lists:get_all'] || form.permissions['lists:manage_all'])"
              class="is-size-6 has-text-danger">
              <b-icon icon="warning-empty" />
              {{ $t('users.listPermsWarning') }}
            </span>
          </div>

          <b-table :data="form.lists">
            <b-table-column v-slot="props" field="name" :label="$tc('globals.terms.list')">
              <router-link :to="`/lists/${props.row.id}`" target="_blank">
                {{ props.row.name }}
              </router-link>
            </b-table-column>

            <b-table-column v-slot="props" field="permissions" :label="$t('users.perms')" width="40%">
              <b-checkbox v-model="props.row.permissions" native-value="list:get">
                {{ $t('globals.buttons.view') }}
              </b-checkbox>
              <b-checkbox v-model="props.row.permissions" native-value="list:manage">
                {{ $t('globals.buttons.manage') }}
              </b-checkbox>
            </b-table-column>

            <b-table-column v-slot="props" width="10%">
              <a href="#" @click.prevent="onDeleteListPerm(props.row.id)" data-cy="btn-delete"
                :aria-label="$t('globals.buttons.delete')">
                <b-tooltip :label="$t('globals.buttons.delete')" type="is-dark">
                  <b-icon icon="trash-can-outline" size="is-small" />
                </b-tooltip>
              </a>
            </b-table-column>
          </b-table>
        </div>

        <template v-if="type === 'user'">
          <div class="columns">
            <div class="column is-7">
              <h5 class="mb-0">
                {{ $t('users.perms') }}
              </h5>
            </div>
            <div class="column has-text-right" v-if="!disabled">
              <a href="#" @click.prevent="onToggleSelect">{{ $t('globals.buttons.toggleSelect') }}</a>
            </div>
          </div>

          <b-table :data="serverConfig.permissions">
            <b-table-column v-slot="props" field="group" :label="$t('users.roleGroup')">
              {{ $tc(`globals.terms.${props.row.group}`) }}
            </b-table-column>

            <b-table-column v-slot="props" field="permissions" label="Permissions">
              <div v-for="p in props.row.permissions" :key="p">
                <b-checkbox v-model="form.permissions" :native-value="p" :disabled="disabled">
                  {{ p }}
                  <a v-if="p === 'subscribers:sql_query'"
                    href="https://listmonk.app/docs/roles-and-permissions/#subscriberssql_query" target="_blank"
                    rel="noopener noreferrer" aria-label="Warning: high risk permission">
                    <b-icon icon="warning-empty" type="is-danger" size="is-small" />
                  </a>
                </b-checkbox>
              </div>
            </b-table-column>
          </b-table>
        </template>
        <a href="https://listmonk.app/docs/roles-and-permissions" target="_blank" rel="noopener noreferrer">
          <b-icon icon="link-variant" /> {{ $t('globals.buttons.learnMore') }}
        </a>
      </section>

      <footer class="modal-card-foot has-text-right">
        <b-button @click="$parent.close()">
          {{ $t('globals.buttons.close') }}
        </b-button>
        <b-button v-if="!disabled" native-type="submit" type="is-primary" :loading="loading.roles" data-cy="btn-save">
          {{ $t('globals.buttons.save') }}
        </b-button>
      </footer>
    </div>
  </form>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import CopyText from '../components/CopyText.vue';

export default Vue.extend({
  name: 'RoleForm',

  components: {
    CopyText,
  },

  props: {
    data: { type: Object, default: () => ({}) },
    isEditing: { type: Boolean, default: false },
    type: { type: String, default: 'user' },
  },

  data() {
    return {
      // Binds form input values.
      form: {
        curList: null,
        lists: [],
        name: null,
        permissions: {},
      },
      hasToggle: false,
      disabled: false,
    };
  },

  methods: {
    onAddListPerm() {
      const list = this.lists.results.find((l) => l.id === this.form.curList);
      this.form.lists.push({ id: list.id, name: list.name, permissions: ['list:get', 'list:manage'] });

      this.form.curList = (this.filteredLists.length > 0) ? this.filteredLists[0].id : null;
    },

    onDeleteListPerm(id) {
      this.form.lists = this.form.lists.filter((p) => p.id !== id);
      this.form.curList = (this.filteredLists.length > 0) ? this.filteredLists[0].id : null;
    },

    onSubmit() {
      if (this.isEditing) {
        this.updateRole();
        return;
      }

      this.createRole();
    },

    onToggleSelect() {
      if (this.hasToggle) {
        this.form.permissions = [];
      } else {
        this.form.permissions = this.serverConfig.permissions.reduce((acc, item) => {
          item.permissions.forEach((p) => {
            acc.push(p);
          });
          return acc;
        }, []);
      }

      this.hasToggle = !this.hasToggle;
    },

    createRole() {
      let fn;
      const form = { name: this.form.name };

      if (this.$props.type === 'user') {
        fn = this.$api.createUserRole;
        form.permissions = this.form.permissions;
      } else {
        fn = this.$api.createListRole;
        form.lists = this.form.lists.reduce((acc, item) => {
          acc.push({ id: item.id, permissions: item.permissions });
          return acc;
        }, []);
      }

      fn(form).then((data) => {
        this.$emit('finished');
        this.$utils.toast(this.$t('globals.messages.created', { name: data.name }));
        this.$parent.close();
      });
    },

    updateRole() {
      let fn;
      const form = { id: this.$props.data.id, name: this.form.name };

      if (this.$props.type === 'user') {
        fn = this.$api.updateUserRole;
        form.permissions = this.form.permissions;
      } else {
        fn = this.$api.updateListRole;
        form.lists = this.form.lists.reduce((acc, item) => {
          acc.push({ id: item.id, permissions: item.permissions });
          return acc;
        }, []);
      }

      fn(form).then((data) => {
        this.$emit('finished');
        this.$utils.toast(this.$t('globals.messages.updated', { name: data.name }));
        this.$parent.close();
      });
    },
  },

  computed: {
    ...mapState(['loading', 'serverConfig', 'lists']),

    // Return the list of unselected lists.
    filteredLists() {
      if (!this.lists.results || this.type !== 'list') {
        return [];
      }

      const subIDs = this.form.lists.reduce((obj, item) => ({ ...obj, [item.id]: true }), {});
      return this.lists.results.filter((l) => (!(l.id in subIDs)));
    },

  },

  mounted() {
    if (this.isEditing) {
      this.form = { ...this.form, ...this.$props.data };

      // It's the superadmin role. Disable the form.
      if (this.$props.data.id === 1 || !this.$can('roles:manage')) {
        this.disabled = true;
      }
    } else {
      const skip = ['admin', 'users'];
      this.form.permissions = this.serverConfig.permissions.reduce((acc, item) => {
        if (skip.includes(item.group)) {
          return acc;
        }
        item.permissions.forEach((p) => {
          if (p !== 'subscribers:sql_query' && !p.startsWith('lists:') && !p.startsWith('settings:')) {
            acc.push(p);
          }
        });
        return acc;
      }, []);
    }

    this.$nextTick(() => {
      if (this.filteredLists.length > 0) {
        this.form.curList = this.filteredLists[0].id;
      }
      this.$refs.focus.focus();
    });
  },
});
</script>
