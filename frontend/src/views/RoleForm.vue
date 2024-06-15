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
          {{ $t('users.newRole') }}
        </h4>
      </header>

      <section expanded class="modal-card-body">
        <b-field :label="$t('globals.fields.name')" label-position="on-border">
          <b-input :maxlength="200" v-model="form.name" name="name" :ref="'focus'" required />
        </b-field>

        <p class="has-text-right">
          <a href="#" @click.prevent="onToggleSelect">{{ $t('globals.buttons.toggleSelect') }}</a>
        </p>

        <b-table :data="serverConfig.permissions">
          <b-table-column v-slot="props" field="group" label="Group">
            {{ $tc(`globals.terms.${props.row.group}`) }}
          </b-table-column>

          <b-table-column v-slot="props" field="permissions" label="Permissions">
            <div v-for="p in props.row.permissions" :key="p">
              <b-checkbox v-model="form.permissions[p]">
                {{ p }}
              </b-checkbox>
            </div>
          </b-table-column>
        </b-table>
      </section>

      <footer class="modal-card-foot has-text-right">
        <b-button @click="$parent.close()">
          {{ $t('globals.buttons.close') }}
        </b-button>
        <b-button native-type="submit" type="is-primary" :loading="loading.roles" data-cy="btn-save">
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
  },

  data() {
    return {
      // Binds form input values.
      form: {
        name: '',
        permissions: {},
      },
      hasToggle: false,
    };
  },

  methods: {
    onSubmit() {
      if (this.isEditing) {
        this.updateRole();
        return;
      }

      this.createRole();
    },

    onToggleSelect() {
      if (this.hasToggle) {
        this.form.permissions = {};
      } else {
        this.form.permissions = this.serverConfig.permissions.reduce((acc, item) => {
          item.permissions.forEach((p) => {
            acc[p] = true;
          });
          return acc;
        }, {});
      }

      this.hasToggle = !this.hasToggle;
    },

    createRole() {
      const form = { ...this.form, permissions: Object.keys(this.form.permissions) };
      this.$api.createRole(form).then((data) => {
        this.$emit('finished');
        this.$utils.toast(this.$t('globals.messages.created', { name: data.name }));
        this.$parent.close();
      });
    },

    updateRole() {
      const form = { id: this.data.id, name: this.form.name, permissions: Object.keys(this.form.permissions) };
      this.$api.updateRole(form).then((data) => {
        this.$emit('finished');
        this.$parent.close();
        this.$utils.toast(this.$t('globals.messages.updated', { name: data.name }));
      });
    },
  },

  computed: {
    ...mapState(['loading', 'serverConfig']),
  },

  mounted() {
    this.form = { ...this.form, name: this.$props.data.name };

    if (this.isEditing) {
      this.form.permissions = this.$props.data.permissions.reduce((acc, key) => {
        acc[key] = true;
        return acc;
      }, {});
    } else {
      const skip = ['admin', 'users'];
      this.form.permissions = this.serverConfig.permissions.reduce((acc, item) => {
        if (skip.includes(item.group)) {
          return acc;
        }
        item.permissions.forEach((p) => {
          if (p !== 'subscribers:sql_query') {
            acc[p] = true;
          }
        });
        return acc;
      }, {});
    }

    this.$nextTick(() => {
      this.$refs.focus.focus();
    });
  },
});
</script>
