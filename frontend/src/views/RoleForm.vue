<template>
  <form @submit.prevent="onSubmit">
    <div class="dialog-card content" style="width: auto">
      <header class="dialog-head">
        <p v-if="isEditing" class="text-lighter text-7 ">
          {{ $t('globals.fields.id') }}: <copy-text :text="`${data.id}`" />
        </p>
        <h4 v-if="isEditing">
          {{ data.name }}
        </h4>
        <h4 v-else>
          {{ type === 'user' ? $t('users.newUserRole') : $t('users.newListRole') }}
        </h4>
      </header>

      <section class="dialog-body">
        <oat-field :label="$t('globals.fields.name')">
          <input aria-label="field" :disabled="disabled" :maxlength="200" v-model="form.name" name="name" ref="focus"
            required>
        </oat-field>

        <div v-if="type === 'list'" class="card">
          <h5>{{ $t('users.listPerms') }}</h5>
          <div class="mb-5">
            <div class="row">
              <div class="col-9">
                <select aria-label="field" :placeholder="$tc('globals.terms.list')" v-model="form.curList" name="list"
                  :disabled="disabled || filteredLists.length < 1" class="mb-3">
                  <template v-for="l in filteredLists">
                    <option :value="l.id" :key="l.id">
                      {{ l.name }}
                    </option>
                  </template>
                </select>
              </div>
              <div class="col-12">
                <button type="button" @click="onAddListPerm" :disabled="!form.curList" data-variant="primary">
                  {{ $t('globals.buttons.add') }}
                </button>
              </div>
            </div>
            <span
              v-if="form.lists.length > 0 && (form.permissions['lists:get_all'] || form.permissions['lists:manage_all'])"
              class="text-danger text-6">
              <oat-icon icon="warning-empty" />
              {{ $t('users.listPermsWarning') }}
            </span>
          </div>

          <oat-data-table :data="form.lists">
            <oat-table-column v-slot="props" field="name" :label="$tc('globals.terms.list')">
              <router-link :to="`/lists/${props.row.id}`" target="_blank">
                {{ props.row.name }}
              </router-link>
            </oat-table-column>

            <oat-table-column v-slot="props" field="permissions" :label="$t('users.perms')" width="40%">
              <oat-checkbox v-model="props.row.permissions" native-value="list:get">
                {{ $t('globals.buttons.view') }}
              </oat-checkbox>
              <oat-checkbox v-model="props.row.permissions" native-value="list:manage">
                {{ $t('globals.buttons.manage') }}
              </oat-checkbox>
            </oat-table-column>

            <oat-table-column v-slot="props" width="10%">
              <a href="#" @click.prevent="onDeleteListPerm(props.row.id)" data-cy="btn-delete"
                :aria-label="$t('globals.buttons.delete')">

                  <oat-icon icon="trash-can-outline" />

              </a>
            </oat-table-column>
</oat-data-table>
        </div>

        <template v-if="type === 'user'">
          <div class="row">
            <div class="col-7">
              <h5 class="mb-0">
                {{ $t('users.perms') }}
              </h5>
            </div>
            <div class="col-12 align-right" v-if="!disabled">
              <a href="#" @click.prevent="onToggleSelect">{{ $t('globals.buttons.toggleSelect') }}</a>
            </div>
          </div>

          <oat-data-table :data="serverConfig.permissions">
            <oat-table-column v-slot="props" field="group" :label="$t('users.roleGroup')">
              {{ $tc(`globals.terms.${props.row.group}`) }}
            </oat-table-column>

            <oat-table-column v-slot="props" field="permissions" label="Permissions">
              <div v-for="p in props.row.permissions" :key="p">
                <oat-checkbox v-model="form.permissions" :native-value="p" :disabled="disabled">
                  {{ p }}
                  <a v-if="p === 'subscribers:sql_query'"
                    href="https://listmonk.app/docs/roles-and-permissions/#subscriberssql_query" target="_blank"
                    rel="noopener noreferrer" aria-label="Warning: high risk permission">
                    <oat-icon icon="warning-empty" data-variant="danger" />
                  </a>
                </oat-checkbox>
              </div>
            </oat-table-column>
</oat-data-table>
        </template>
        <a href="https://listmonk.app/docs/roles-and-permissions" target="_blank" rel="noopener noreferrer">
          <oat-icon icon="link-variant" /> {{ $t('globals.buttons.learnMore') }}
        </a>
      </section>

      <footer class="dialog-foot align-right">
        <button type="button" class="outline" @click="$parent.close()">
          {{ $t('globals.buttons.close') }}
        </button>
        <button v-if="!disabled" type="submit" data-variant="primary" :loading="loading.roles" data-cy="btn-save">
          {{ $t('globals.buttons.save') }}
        </button>
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
